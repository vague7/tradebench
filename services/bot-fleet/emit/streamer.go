package emit

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bench/shared/proto/gen"
	"github.com/bench/shared/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EmitChannelCap is the buffer capacity for the internal channel between bots and
// the streamer. Matches the telemetry-ingester's ring buffer capacity (FR-4.2).
const EmitChannelCap = 100_000

// maxRetries is the number of times to retry stream.Send on failure.
const maxRetries = 3

// retryBackoff is the backoff duration between stream.Send retries.
const retryBackoff = 500 * time.Millisecond

// Streamer connects to telemetry-ingester, opens the StreamEvents client-side stream,
// and forwards BotEvent records from the bot fleet's internal emit channel to the stream.
type Streamer struct {
	conn    *grpc.ClientConn
	client  gen.TelemetryIngesterClient
	eventCh <-chan types.BotEvent
}

// NewStreamer dials the telemetry-ingester at telemetryAddr and returns a Streamer.
// Returns an error if the dial fails.
func NewStreamer(telemetryAddr string, eventCh <-chan types.BotEvent) (*Streamer, error) {
	conn, err := grpc.Dial(telemetryAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("streamer: dial telemetry-ingester at %s: %w", telemetryAddr, err)
	}

	client := gen.NewTelemetryIngesterClient(conn)

	return &Streamer{
		conn:    conn,
		client:  client,
		eventCh: eventCh,
	}, nil
}

// Run is the main streaming loop. It opens the StreamEvents RPC stream, ranges over
// eventCh, converts each BotEvent to BotEventProto, and sends it on the stream.
// It retries failed sends up to 3 times with 500ms backoff. When ctx is done or
// eventCh is closed, it flushes the stream via CloseAndRecv.
func (s *Streamer) Run(ctx context.Context) error {
	stream, err := s.client.StreamEvents(ctx)
	if err != nil {
		return fmt.Errorf("streamer: open StreamEvents: %w", err)
	}

	for event := range s.eventCh {
		// Check context before sending
		select {
		case <-ctx.Done():
			return s.closeStream(stream)
		default:
		}

		proto := botEventToProto(event)

		if sendErr := s.sendWithRetry(ctx, &stream, proto); sendErr != nil {
			slog.Error("gRPC stream permanently failed after retries", "err", sendErr)
			return fmt.Errorf("streamer: send failed permanently: %w", sendErr)
		}
	}

	// eventCh was closed — flush the stream
	return s.closeStream(stream)
}

// sendWithRetry attempts to send a proto message on the stream, retrying up to
// maxRetries times with retryBackoff on failure. If a retry requires reopening the
// stream, it does so.
func (s *Streamer) sendWithRetry(ctx context.Context, stream *gen.TelemetryIngester_StreamEventsClient, proto *gen.BotEventProto) error {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			slog.Warn("gRPC stream send failed, retrying",
				"attempt", attempt,
				"err", lastErr,
			)
			select {
			case <-ctx.Done():
				return fmt.Errorf("streamer: context cancelled during retry: %w", ctx.Err())
			case <-time.After(retryBackoff):
			}

			// Attempt to reopen the stream
			newStream, err := s.client.StreamEvents(ctx)
			if err != nil {
				lastErr = fmt.Errorf("streamer: reopen stream attempt %d: %w", attempt, err)
				continue
			}
			*stream = newStream
		}

		if err := (*stream).Send(proto); err != nil {
			lastErr = fmt.Errorf("streamer: send attempt %d: %w", attempt, err)
			continue
		}
		return nil
	}
	return lastErr
}

// closeStream flushes the gRPC stream via CloseAndRecv and logs the Ack response.
func (s *Streamer) closeStream(stream gen.TelemetryIngester_StreamEventsClient) error {
	ack, err := stream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("streamer: CloseAndRecv: %w", err)
	}
	slog.Info("streamer: stream closed", "ack", ack.GetOk())
	return nil
}

// Close closes the underlying gRPC connection. Called by main.go during graceful shutdown.
func (s *Streamer) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// botEventToProto converts a types.BotEvent to a gen.BotEventProto.
// This is the exact inverse of the mapping in telemetry-ingester's ingest/server.go.
func botEventToProto(e types.BotEvent) *gen.BotEventProto {
	return &gen.BotEventProto{
		SubmissionId: e.SubmissionID,
		BotId:        e.BotID,
		OrderId:      e.OrderID,
		OrderType:    string(e.OrderType),
		SentAt:       timestamppb.New(e.SentAt),
		AckedAt:      timestamppb.New(e.AckedAt),
		HttpStatus:   int32(e.HTTPStatus),
		ExpectedFill: fillToProto(e.ExpectedFill),
		ActualFill:   fillToProto(e.ActualFill),
	}
}

// fillToProto converts a types.Fill to a gen.Fill protobuf message.
func fillToProto(f types.Fill) *gen.Fill {
	return &gen.Fill{
		Price:    f.Price,
		Quantity: f.Quantity,
		Side:     f.Side,
	}
}
