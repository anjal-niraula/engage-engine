package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/platform/config"
	"github.com/anjal-niraula/engage-engine/internal/platform/logger"
	"github.com/anjal-niraula/engage-engine/internal/platform/server"
)

const DefaultContextTimeout = 30

func main() {
	// 1. Load configuration (koanf + validator)
	cfg, err := config.LoadConfig()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// 2. Initialize New Relic / telemetry logger service
	loggerService := logger.NewLoggerService(cfg.Observability)
	defer loggerService.Shutdown()

	log := logger.NewLoggerWithService(cfg.Observability, loggerService)

	// 3. Initialize server platform (managing PostgreSQL, Redis, HTTP server)
	srv, err := server.New(cfg, &log, loggerService)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize server")
	}

	// 4. HTTP Router and module route registration
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	srv.SetupHTTPServer(mux)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

	// 5. Start server in a background goroutine
	go func() {
		if err = srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("failed to start server")
		}
	}()

	// 6. Wait for interrupt signal to gracefully shutdown
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), DefaultContextTimeout*time.Second)

	if err = srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("server forced to shutdown")
	}
	stop()
	cancel()

	log.Info().Msg("server exited properly")
}
