package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/platform/config"
	"github.com/anjal-niraula/engage-engine/internal/platform/logger"
	"github.com/anjal-niraula/engage-engine/internal/platform/server"
	"github.com/anjal-niraula/engage-engine/internal/platform/worker"
)

func main() {
	// 1. Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// 2. Initialize logger service
	loggerService := logger.NewLoggerService(cfg.Observability)
	defer loggerService.Shutdown()

	log := logger.NewLoggerWithService(cfg.Observability, loggerService)
	log.Info().
		Str("service", cfg.Observability.ServiceName+"-worker").
		Str("env", cfg.Primary.Env).
		Msg("starting standalone background worker service (Kafka Micro-Batch Consumer)")

	// 3. Initialize server platform (PostgreSQL, Redis, Kafka)
	srv, err := server.New(cfg, &log, loggerService)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize worker server dependencies")
	}

	// 4. Set up root context with graceful cancellation
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if srv.Kafka == nil {
		log.Warn().Msg("Kafka is not connected, worker running in idle standby mode")
		<-ctx.Done()
		return
	}

	// 5. Start Kafka Micro-Batch Consumers for Views, Follows, and Likes
	worker.StartAllConsumers(ctx, srv, &log)

	// 6. Wait for shutdown signal
	<-ctx.Done()
	log.Info().Msg("received shutdown signal: draining background worker tasks...")
	time.Sleep(500 * time.Millisecond)
	log.Info().Msg("worker service stopped cleanly")
}
