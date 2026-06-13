package ingest

import (
	"io"
	"log/slog"

	"github.com/bench/shared/proto/gen"
	"github.com/bench/shared/types"
	"google.golang.org/grpc"
)

// Server implements the TelemetryIngester gRPC service.
// It receives a BotEvent stream from bot-fleet and pushes events into the ring buffer.
type Server struct {
	gen.UnimplementedTelemetryIngesterServer
	buf *RingBuffer
}

// NewServer creates a new gRPC telemetry ingester server backed by the given ring buffer.
func NewServer(buf *RingBuffer) *Server {
	return &Server{buf: buf}
}

// StreamEvents receives a client-streaming BotEventProto stream from bot-fleet.
// Each received message is converted to types.BotEvent and pushed into the ring buffer.
// If the buffer is full, the event is dropped and a warning is logged.
func (s *Server) StreamEvents(stream grpc.ClientStreamingServer[gen.BotEventProto, gen.TelemetryAck]) error {
	for {
		proto, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&gen.TelemetryAck{Ok: true})
		}
		if err != nil {
			slog.Error("failed to receive event from stream", "err", err)
			return err
		}

		event := protoToBotEvent(proto)

		if !s.buf.Push(event) {
			slog.Warn("buffer full, dropping event",
				"botId", event.BotID,
				"submissionId", event.SubmissionID,
			)
		}
	}
}

// protoToBotEvent converts a BotEventProto message to a types.BotEvent.
func protoToBotEvent(p *gen.BotEventProto) types.BotEvent {
	event := types.BotEvent{
		SubmissionID: p.GetSubmissionId(),
		BotID:        p.GetBotId(),
		OrderID:      p.GetOrderId(),
		OrderType:    types.OrderType(p.GetOrderType()),
		HTTPStatus:   int(p.GetHttpStatus()),
	}

	if p.GetSentAt() != nil {
		event.SentAt = p.GetSentAt().AsTime()
	}
	if p.GetAckedAt() != nil {
		event.AckedAt = p.GetAckedAt().AsTime()
	}

	if ef := p.GetExpectedFill(); ef != nil {
		event.ExpectedFill = types.Fill{
			Price:    ef.GetPrice(),
			Quantity: ef.GetQuantity(),
			Side:     ef.GetSide(),
		}
	}

	if af := p.GetActualFill(); af != nil {
		event.ActualFill = types.Fill{
			Price:    af.GetPrice(),
			Quantity: af.GetQuantity(),
			Side:     af.GetSide(),
		}
	}

	return event
}
