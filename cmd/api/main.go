package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	viewsHandler "github.com/anjal-niraula/engage-engine/internal/modules/views/handler"
	viewsRepo "github.com/anjal-niraula/engage-engine/internal/modules/views/repository"
	viewsUsecase "github.com/anjal-niraula/engage-engine/internal/modules/views/usecase"
	commentHandler "github.com/anjal-niraula/engage-engine/internal/modules/comment/handler"
	commentRepo "github.com/anjal-niraula/engage-engine/internal/modules/comment/repository"
	commentUsecase "github.com/anjal-niraula/engage-engine/internal/modules/comment/usecase"
	followHandler "github.com/anjal-niraula/engage-engine/internal/modules/follow/handler"
	followRepo "github.com/anjal-niraula/engage-engine/internal/modules/follow/repository"
	followUsecase "github.com/anjal-niraula/engage-engine/internal/modules/follow/usecase"
	likeHandler "github.com/anjal-niraula/engage-engine/internal/modules/like/handler"
	likeRepo "github.com/anjal-niraula/engage-engine/internal/modules/like/repository"
	likeUsecase "github.com/anjal-niraula/engage-engine/internal/modules/like/usecase"
	notificationHandler "github.com/anjal-niraula/engage-engine/internal/modules/notification/handler"
	notificationRepo "github.com/anjal-niraula/engage-engine/internal/modules/notification/repository"
	notificationUsecase "github.com/anjal-niraula/engage-engine/internal/modules/notification/usecase"
	postHandler "github.com/anjal-niraula/engage-engine/internal/modules/post/handler"
	postRepo "github.com/anjal-niraula/engage-engine/internal/modules/post/repository"
	postUsecase "github.com/anjal-niraula/engage-engine/internal/modules/post/usecase"
	userHandler "github.com/anjal-niraula/engage-engine/internal/modules/user/handler"
	userRepo "github.com/anjal-niraula/engage-engine/internal/modules/user/repository"
	userUsecase "github.com/anjal-niraula/engage-engine/internal/modules/user/usecase"
	"github.com/anjal-niraula/engage-engine/internal/platform/config"
	"github.com/anjal-niraula/engage-engine/internal/platform/docs"
	"github.com/anjal-niraula/engage-engine/internal/platform/logger"
	"github.com/anjal-niraula/engage-engine/internal/platform/postgres"
	"github.com/anjal-niraula/engage-engine/internal/platform/server"
	"github.com/anjal-niraula/engage-engine/internal/platform/web"
	"github.com/anjal-niraula/engage-engine/internal/platform/worker"
	"github.com/anjal-niraula/engage-engine/internal/shared/cuckoo"
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

	// 3. Initialize server platform (managing PostgreSQL, Redis, Kafka, HTTP server)
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

	// Register interactive API Documentation (/docs and /docs/openapi.json)
	docs.RegisterRoutes(mux)

	// Register Instagram Web UI (/ and /app)
	web.RegisterRoutes(mux)

	// Setup application modules if database is connected
	if srv.DB != nil && srv.DB.Pool != nil {
		if cfg.Primary.Env == "local" || cfg.Primary.Env == "development" {
			migrationCtx, cancelMigration := context.WithTimeout(context.Background(), 10*time.Second)
			if err := postgres.Migrate(migrationCtx, &log, cfg); err != nil {
				log.Warn().Err(err).Msg("database schema migration warning")
			}
			cancelMigration()
		}

		requireAuth := sharedMw.RequireAuth(cfg.Server.JWTSecret)
		optionalAuth := sharedMw.OptionalAuth(cfg.Server.JWTSecret)

		// 1. User Module
		userRepository := userRepo.NewPostgresUserRepository(srv.DB.Pool)
		userUseCase := userUsecase.NewUserUseCase(userRepository, cfg.Server.JWTSecret, 24*time.Hour)
		userHTTPHandler := userHandler.NewUserHandler(userUseCase)
		userHandler.RegisterRoutes(mux, userHTTPHandler, requireAuth, optionalAuth)

		// Post Repository (needed for Cuckoo verification callback)
		postRepository := postRepo.NewPostgresPostRepository(srv.DB.Pool)

		// Initialize Cuckoo Filter for fast Like lookups with DB fallback
		likeChecker := cuckoo.NewLikeChecker(100000, func(ctx context.Context, userID, postID string) (bool, error) {
			likedMap, err := postRepository.BatchCheckLiked(ctx, userID, []string{postID})
			if err != nil {
				return false, err
			}
			return likedMap[postID], nil
		})

		// 2. Post Module (uses likeChecker for cache & Redis enricher for real-time HLL stats)
		statsEnricher := postUsecase.NewRedisStatsEnricher(srv.Redis)
		postUseCase := postUsecase.NewPostUseCase(postRepository, likeChecker, statsEnricher)
		postHTTPHandler := postHandler.NewPostHandler(postUseCase)
		postHandler.RegisterRoutes(mux, postHTTPHandler, requireAuth, optionalAuth)

		// 3. Like Module (updates likeChecker on Like / Unlike)
		likeRepository := likeRepo.NewPostgresLikeRepository(srv.DB.Pool)
		likeUseCase := likeUsecase.NewLikeUseCase(likeRepository, likeChecker)
		likeHTTPHandler := likeHandler.NewLikeHandler(likeUseCase)
		likeHandler.RegisterRoutes(mux, likeHTTPHandler, requireAuth, optionalAuth)

		// 4. Follow Module
		followRepository := followRepo.NewPostgresFollowRepository(srv.DB.Pool)
		followUseCase := followUsecase.NewFollowUseCase(followRepository)
		followHTTPHandler := followHandler.NewFollowHandler(followUseCase)
		followHandler.RegisterRoutes(mux, followHTTPHandler, requireAuth, optionalAuth)

		// 5. Comment Module
		commentRepository := commentRepo.NewPostgresCommentRepo(srv.DB.Pool)
		commentUseCase := commentUsecase.NewCommentUseCase(commentRepository)
		commentHTTPHandler := commentHandler.NewCommentHandler(commentUseCase, &log)
		commentHandler.RegisterRoutes(mux, commentHTTPHandler, requireAuth, optionalAuth)

		// 6. Notification Module
		notificationRepository := notificationRepo.NewPostgresNotificationRepo(srv.DB.Pool)
		notificationUseCase := notificationUsecase.NewNotificationUseCase(notificationRepository)
		notificationHTTPHandler := notificationHandler.NewNotificationHandler(notificationUseCase, &log)
		notificationHandler.RegisterRoutes(mux, notificationHTTPHandler, requireAuth, optionalAuth)

		// 7. Views Module (Impressions, HyperLogLog & Realtime Counters)
		viewsRepository := viewsRepo.NewViewRepository(srv.DB.Pool, srv.Redis)
		var kafkaViewProducer *viewsRepo.KafkaViewProducer
		if srv.Kafka != nil {
			kafkaViewProducer = viewsRepo.NewKafkaViewProducer(srv.Kafka, "engage.views")
		}
		viewsUseCase := viewsUsecase.NewViewUseCase(viewsRepository, kafkaViewProducer)
		viewsHTTPHandler := viewsHandler.NewViewsHandler(viewsUseCase, &log)
		viewsHandler.RegisterRoutes(mux, viewsHTTPHandler, requireAuth, optionalAuth)

		log.Info().Msg("all core modules successfully registered (user, post, like, follow, comment, notification, views)")

		// Automatic background worker: flushes Kafka batches to PostgreSQL in the background
		if srv.Kafka != nil {
			worker.StartAllConsumers(context.Background(), srv, &log)
		}
	}

	srv.SetupHTTPServer(mux)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 5. Start server in a background goroutine
	go func() {
		if err = srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("failed to start server")
		}
	}()

	// 6. Wait for interrupt signal to gracefully shutdown
	<-ctx.Done()
	log.Info().Msg("received shutdown signal, shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), DefaultContextTimeout*time.Second)
	defer cancel()

	if err = srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("server forced to shutdown")
	}

	log.Info().Msg("server exited properly")
}
