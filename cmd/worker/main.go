package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/platform/config"
	"github.com/anjal-niraula/engage-engine/internal/platform/logger"
)

func main() {
	// 1. Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// 2. Initialize logger
	loggerService := logger.NewLoggerService(cfg.Observability)
	defer loggerService.Shutdown()

	log := logger.NewLoggerWithService(cfg.Observability, loggerService)
	log.Info().
		Str("service", cfg.Observability.ServiceName).
		Str("env", cfg.Primary.Env).
		Msg("starting background worker service")

	// 3. Set up root context with graceful cancellation
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Info().Msg("worker service initialized and listening for tasks/events")

	// 4. Worker event loop / wait for cancellation
	<-ctx.Done()

	log.Info().Msg("received shutdown signal: draining background worker tasks...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	<-shutdownCtx.Done()

	log.Info().Msg("worker service stopped cleanly")
}
