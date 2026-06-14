package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/bench/bot-fleet/config"
	"github.com/bench/bot-fleet/emit"
	"github.com/bench/bot-fleet/fleet"
	gen "github.com/bench/shared/proto/gen"
	"github.com/bench/shared/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BotFleetServer implements the gen.BotFleetServer gRPC interface.
type BotFleetServer struct {
	gen.UnimplementedBotFleetServer

	cfg     *config.Config
	profile fleet.LoadProfile

	// activeBenchmarks tracks running benchmark cancel functions per submissionID.
	// Accessed from gRPC handler goroutines, so sync.Map is required.
	activeBenchmarks sync.Map // map[string]context.CancelFunc

	// streamerDone is closed when the streamer goroutine finishes.
	streamerDone chan struct{}
}

// StartBenchmark is the RPC called by Engineer 1's api-gateway when a submission
// reaches RUNNING status. It launches the benchmark asynchronously and returns immediately.
func (s *BotFleetServer) StartBenchmark(ctx context.Context, req *gen.BenchmarkConfig) (*gen.BotFleetAck, error) {
	if req.GetSubmissionId() == "" || req.GetTargetHost() == "" {
		slog.Error("StartBenchmark called with missing fields",
			"submissionId", req.GetSubmissionId(),
			"targetHost", req.GetTargetHost(),
		)
		return nil, status.Error(codes.InvalidArgument, "submission_id and target_host are required")
	}

	submissionID := req.GetSubmissionId()
	targetURL := "http://" + req.GetTargetHost()

	// Create a new emit channel for this benchmark
	emitCh := make(chan types.BotEvent, emit.EmitChannelCap)

	// Create a new coordinator for this benchmark
	coordinator := fleet.NewCoordinator(s.cfg, s.profile, emitCh)

	// Create a cancellable context for this benchmark, derived from background
	// so it doesn't die when the RPC context ends
	benchCtx, benchCancel := context.WithCancel(context.Background())

	// Store the cancel function so StopBenchmark can cancel it
	s.activeBenchmarks.Store(submissionID, benchCancel)

	// Create and start a streamer for this benchmark
	streamer, err := emit.NewStreamer(s.cfg.TelemetryGRPCAddr, emitCh)
	if err != nil {
		benchCancel()
		slog.Error("failed to connect to telemetry-ingester",
			"addr", s.cfg.TelemetryGRPCAddr,
			"err", err,
		)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to connect to telemetry-ingester: %v", err))
	}

	// Launch streamer goroutine for this benchmark
	streamerDone := make(chan struct{})
	go func() {
		defer close(streamerDone)
		if err := streamer.Run(benchCtx); err != nil {
			slog.Error("streamer error", "submissionId", submissionID, "err", err)
		}
	}()

	// Launch the benchmark in a background goroutine
	go func() {
		defer func() {
			s.activeBenchmarks.Delete(submissionID)
			// Wait for streamer to finish after coordinator closes emitCh
			<-streamerDone
			if closeErr := streamer.Close(); closeErr != nil {
				slog.Error("streamer close error", "submissionId", submissionID, "err", closeErr)
			}
		}()

		if runErr := coordinator.Run(benchCtx, submissionID, targetURL); runErr != nil {
			slog.Error("benchmark run error", "submissionId", submissionID, "err", runErr)
		}
	}()

	slog.Info("benchmark started",
		"submissionId", req.GetSubmissionId(),
		"targetHost", req.GetTargetHost(),
		"botCount", req.GetBotCount(),
		"durationSec", req.GetDurationSec(),
	)

	return &gen.BotFleetAck{Ok: true}, nil
}

// StopBenchmark is the RPC called by Engineer 1's admin stop endpoint.
// It cancels the running benchmark context for the given SubmissionID.
func (s *BotFleetServer) StopBenchmark(_ context.Context, req *gen.StopRequest) (*gen.BotFleetAck, error) {
	submissionID := req.GetSubmissionId()

	slog.Info("benchmark stop requested", "submissionId", submissionID)

	if cancel, ok := s.activeBenchmarks.Load(submissionID); ok {
		cancel.(context.CancelFunc)()
	}

	return &gen.BotFleetAck{Ok: true}, nil
}

func main() {
	// Structured JSON logger — PRD Section 9.1
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg := config.Load()

	slog.Info("bot-fleet starting",
		"fleetGRPCPort", cfg.FleetGRPCPort,
		"telemetryGRPCAddr", cfg.TelemetryGRPCAddr,
		"defaultBotCount", cfg.DefaultBotCount,
		"timeoutMs", cfg.TimeoutMs,
		"targetTPS", cfg.TargetTPS,
	)

	// Construct default load profile
	profile := fleet.DefaultProfile(cfg)

	slog.Info("load profile configured",
		"phases", len(profile),
		"totalDuration", profile.TotalDuration().String(),
	)

	// Create the gRPC server
	grpcServer := grpc.NewServer()

	botFleetServer := &BotFleetServer{
		cfg:     cfg,
		profile: profile,
	}

	gen.RegisterBotFleetServer(grpcServer, botFleetServer)

	// Start listening on the configured port
	lis, err := net.Listen("tcp", cfg.FleetGRPCPort)
	if err != nil {
		slog.Error("failed to listen", "port", cfg.FleetGRPCPort, "err", err)
		os.Exit(1)
	}

	// Handle SIGINT/SIGTERM for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		slog.Info("shutdown signal received", "signal", sig.String())

		// Cancel all active benchmarks
		botFleetServer.activeBenchmarks.Range(func(key, value any) bool {
			value.(context.CancelFunc)()
			return true
		})

		// Gracefully stop the gRPC server
		grpcServer.GracefulStop()
	}()

	slog.Info("bot-fleet gRPC server listening", "port", cfg.FleetGRPCPort)

	if err := grpcServer.Serve(lis); err != nil {
		slog.Error("gRPC server error", "err", err)
		os.Exit(1)
	}

	slog.Info("bot-fleet shutdown complete")
}
