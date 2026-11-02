package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/platform/config"
	kafkaPkg "github.com/anjal-niraula/engage-engine/internal/platform/kafka"
	loggerPkg "github.com/anjal-niraula/engage-engine/internal/platform/logger"
	"github.com/anjal-niraula/engage-engine/internal/platform/postgres"
	redisPkg "github.com/anjal-niraula/engage-engine/internal/platform/redis"
	"github.com/rs/zerolog"
)

type Server struct {
	Config        *config.Config
	Logger        *zerolog.Logger
	LoggerService *loggerPkg.LoggerService
	DB            *postgres.Database
	Redis         *redisPkg.Client
	Kafka         *kafkaPkg.Client
	httpServer    *http.Server
}

func New(cfg *config.Config, logger *zerolog.Logger, loggerService *loggerPkg.LoggerService) (*Server, error) {
	db, err := postgres.New(cfg, logger, loggerService)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to initialize database, continuing without active DB pool")
	}

	// Redis client initialization with connection pooling and telemetry
	redisClient, err := redisPkg.New(cfg.Redis, logger, loggerService)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to initialize redis client, continuing without active Redis")
	}

	// Kafka client initialization
	kafkaClient, err := kafkaPkg.New(cfg.Kafka, logger)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to initialize kafka client, continuing without active Kafka")
	} else {
		ctxPing, cancelPing := context.WithTimeout(context.Background(), 2*time.Second)
		if err := kafkaClient.Ping(ctxPing); err != nil {
			logger.Warn().Err(err).Msg("failed to reach Kafka broker, continuing without active Kafka")
		}
		cancelPing()
	}

	server := &Server{
		Config:        cfg,
		Logger:        logger,
		LoggerService: loggerService,
		DB:            db,
		Redis:         redisClient,
		Kafka:         kafkaClient,
	}

	return server, nil
}

func (s *Server) SetupHTTPServer(handler http.Handler) {
	s.httpServer = &http.Server{
		Addr:         ":" + s.Config.Server.Port,
		Handler:      handler,
		ReadTimeout:  time.Duration(s.Config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.Config.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(s.Config.Server.IdleTimeout) * time.Second,
	}
}

func (s *Server) Start() error {
	if s.httpServer == nil {
		return errors.New("HTTP server not initialized")
	}

	s.Logger.Info().
		Str("port", s.Config.Server.Port).
		Str("env", s.Config.Primary.Env).
		Msg("starting server")

	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("failed to shutdown HTTP server: %w", err)
		}
	}

	if s.DB != nil {
		if err := s.DB.Close(); err != nil {
			return fmt.Errorf("failed to close database connection: %w", err)
		}
	}

	if s.Redis != nil {
		if err := s.Redis.Close(); err != nil {
			s.Logger.Warn().Err(err).Msg("failed to cleanly close redis client")
		}
	}

	if s.Kafka != nil {
		if err := s.Kafka.Close(); err != nil {
			s.Logger.Warn().Err(err).Msg("failed to cleanly close kafka writer")
		}
	}

	return nil
}
