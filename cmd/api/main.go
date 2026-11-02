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
	"github.com/anjal-niraula/engage-engine/internal/platform/postgres"
	"github.com/anjal-niraula/engage-engine/internal/platform/server"
	userHandler "github.com/anjal-niraula/engage-engine/internal/modules/user/handler"
	userRepo "github.com/anjal-niraula/engage-engine/internal/modules/user/repository"
	userUsecase "github.com/anjal-niraula/engage-engine/internal/modules/user/usecase"
	sharedMw "github.com/anjal-niraula/engage-engine/internal/shared/middleware"
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

	// Setup User Module if database is connected
	if srv.DB != nil && srv.DB.Pool != nil {
		if cfg.Primary.Env == "local" || cfg.Primary.Env == "development" {
			migrationCtx, cancelMigration := context.WithTimeout(context.Background(), 10*time.Second)
			if err := postgres.Migrate(migrationCtx, &log, cfg); err != nil {
				log.Warn().Err(err).Msg("database schema migration warning")
			}
			cancelMigration()
		}

		userRepository := userRepo.NewPostgresUserRepository(srv.DB.Pool)
		userUseCase := userUsecase.NewUserUseCase(userRepository, cfg.Server.JWTSecret, 24*time.Hour)
		userHTTPHandler := userHandler.NewUserHandler(userUseCase)

		requireAuth := sharedMw.RequireAuth(cfg.Server.JWTSecret)
		optionalAuth := sharedMw.OptionalAuth(cfg.Server.JWTSecret)

		userHandler.RegisterRoutes(mux, userHTTPHandler, requireAuth, optionalAuth)
		log.Info().Msg("user module routes registered")
	}

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
