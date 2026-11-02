//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	analyticsHandler "github.com/anjal-niraula/engage-engine/internal/modules/analytics/handler"
	analyticsRepo "github.com/anjal-niraula/engage-engine/internal/modules/analytics/repository"
	analyticsUsecase "github.com/anjal-niraula/engage-engine/internal/modules/analytics/usecase"
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
	"github.com/anjal-niraula/engage-engine/internal/platform/postgres"
	"github.com/anjal-niraula/engage-engine/internal/shared/middleware"
	"github.com/anjal-niraula/engage-engine/internal/shared/response"
	"github.com/rs/zerolog"
)

func TestEndToEndAPI_Integration(t *testing.T) {
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}

	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	// 1. Run migrations to ensure latest schema
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := postgres.Migrate(ctx, &logger, cfg); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// 2. Initialize PostgreSQL connection pool
	db, err := postgres.New(cfg, &logger, nil)
	if err != nil {
		t.Fatalf("failed to connect to postgres: %v", err)
	}
	defer db.Close()

	// 3. Setup all modules
	mux := http.NewServeMux()
	requireAuth := middleware.RequireAuth(cfg.Server.JWTSecret)
	optionalAuth := middleware.OptionalAuth(cfg.Server.JWTSecret)

	// User
	userR := userRepo.NewPostgresUserRepository(db.Pool)
	userUC := userUsecase.NewUserUseCase(userR, cfg.Server.JWTSecret, 24*time.Hour)
	userH := userHandler.NewUserHandler(userUC)
	userHandler.RegisterRoutes(mux, userH, requireAuth, optionalAuth)

	// Post
	postR := postRepo.NewPostgresPostRepository(db.Pool)
	postUC := postUsecase.NewPostUseCase(postR)
	postH := postHandler.NewPostHandler(postUC)
	postHandler.RegisterRoutes(mux, postH, requireAuth, optionalAuth)

	// Like
	likeR := likeRepo.NewPostgresLikeRepository(db.Pool)
	likeUC := likeUsecase.NewLikeUseCase(likeR)
	likeH := likeHandler.NewLikeHandler(likeUC)
	likeHandler.RegisterRoutes(mux, likeH, requireAuth, optionalAuth)

	// Follow
	followR := followRepo.NewPostgresFollowRepository(db.Pool)
	followUC := followUsecase.NewFollowUseCase(followR)
	followH := followHandler.NewFollowHandler(followUC)
	followHandler.RegisterRoutes(mux, followH, requireAuth, optionalAuth)

	// Comment
	commentR := commentRepo.NewPostgresCommentRepo(db.Pool)
	commentUC := commentUsecase.NewCommentUseCase(commentR)
	commentH := commentHandler.NewCommentHandler(commentUC, &logger)
	commentHandler.RegisterRoutes(mux, commentH, requireAuth, optionalAuth)

	// Notification
	notifR := notificationRepo.NewPostgresNotificationRepo(db.Pool)
	notifUC := notificationUsecase.NewNotificationUseCase(notifR)
	notifH := notificationHandler.NewNotificationHandler(notifUC, &logger)
	notificationHandler.RegisterRoutes(mux, notifH, requireAuth, optionalAuth)

	// Analytics
	analyticsR := analyticsRepo.NewAnalyticsRepository(db.Pool, nil)
	analyticsUC := analyticsUsecase.NewAnalyticsUseCase(analyticsR, nil)
	analyticsH := analyticsHandler.NewAnalyticsHandler(analyticsUC, &logger)
	analyticsHandler.RegisterRoutes(mux, analyticsH, requireAuth, optionalAuth)

	uniqueSuffix := time.Now().UnixNano()
	user1Name := fmt.Sprintf("userone%d", uniqueSuffix%1000000)
	user2Name := fmt.Sprintf("usertwo%d", uniqueSuffix%1000000)

	// Helper to register and login
	authenticateUser := func(uname, email string) (string, string) {
		regBody := fmt.Sprintf(`{"username":"%s","email":"%s","password":"Password123!","display_name":"Test %s"}`, uname, email, uname)
		req := httptest.NewRequest("POST", "/api/v1/users/register", bytes.NewBufferString(regBody))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("registration failed for %s: %s", uname, rec.Body.String())
		}

		var regResp struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &regResp)

		loginBody := fmt.Sprintf(`{"identity":"%s","password":"Password123!"}`, uname)
		reqLog := httptest.NewRequest("POST", "/api/v1/users/login", bytes.NewBufferString(loginBody))
		recLog := httptest.NewRecorder()
		mux.ServeHTTP(recLog, reqLog)
		if recLog.Code != http.StatusOK {
			t.Fatalf("login failed for %s: %s", uname, recLog.Body.String())
		}

		var logResp struct {
			Data struct {
				AccessToken string `json:"access_token"`
			} `json:"data"`
		}
		_ = json.Unmarshal(recLog.Body.Bytes(), &logResp)

		return regResp.Data.ID, logResp.Data.AccessToken
	}

	user1ID, user1Token := authenticateUser(user1Name, user1Name+"@example.com")
	_, user2Token := authenticateUser(user2Name, user2Name+"@example.com")

	// 1. User 2 follows User 1
	reqFollow := httptest.NewRequest("POST", "/api/v1/users/"+user1ID+"/follow", nil)
	reqFollow.Header.Set("Authorization", "Bearer "+user2Token)
	recFollow := httptest.NewRecorder()
	mux.ServeHTTP(recFollow, reqFollow)
	if recFollow.Code != http.StatusOK {
		t.Fatalf("follow failed: %d: %s", recFollow.Code, recFollow.Body.String())
	}

	// 2. User 1 creates a post
	postBody := `{"caption":"My first vacation post","media_urls":["https://cdn.example.com/img1.jpg"],"media_type":"image","location":"Hawaii"}`
	reqPost := httptest.NewRequest("POST", "/api/v1/posts", bytes.NewBufferString(postBody))
	reqPost.Header.Set("Authorization", "Bearer "+user1Token)
	reqPost.Header.Set("Content-Type", "application/json")
	recPost := httptest.NewRecorder()
	mux.ServeHTTP(recPost, reqPost)
	if recPost.Code != http.StatusCreated {
		t.Fatalf("post creation failed: %d: %s", recPost.Code, recPost.Body.String())
	}

	var postCreated struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recPost.Body.Bytes(), &postCreated)
	postID := postCreated.Data.ID

	// 3. User 2 likes the post
	reqLike := httptest.NewRequest("POST", "/api/v1/posts/"+postID+"/likes", nil)
	reqLike.Header.Set("Authorization", "Bearer "+user2Token)
	recLike := httptest.NewRecorder()
	mux.ServeHTTP(recLike, reqLike)
	if recLike.Code != http.StatusOK {
		t.Fatalf("like post failed: %d: %s", recLike.Code, recLike.Body.String())
	}

	// 4. User 2 comments on the post
	commentBody := `{"content":"Amazing photo!"}`
	reqComment := httptest.NewRequest("POST", "/api/v1/posts/"+postID+"/comments", bytes.NewBufferString(commentBody))
	reqComment.Header.Set("Authorization", "Bearer "+user2Token)
	reqComment.Header.Set("Content-Type", "application/json")
	recComment := httptest.NewRecorder()
	mux.ServeHTTP(recComment, reqComment)
	if recComment.Code != http.StatusCreated {
		t.Fatalf("comment failed: %d: %s", recComment.Code, recComment.Body.String())
	}

	// 5. Query comments
	reqListComments := httptest.NewRequest("GET", "/api/v1/posts/"+postID+"/comments", nil)
	recListComments := httptest.NewRecorder()
	mux.ServeHTTP(recListComments, reqListComments)
	if recListComments.Code != http.StatusOK {
		t.Fatalf("list comments failed: %d: %s", recListComments.Code, recListComments.Body.String())
	}

	var commentsList response.Envelope
	_ = json.Unmarshal(recListComments.Body.Bytes(), &commentsList)
	if !commentsList.Success {
		t.Fatalf("expected success true for comments")
	}

	// 6. Track view
	viewBody := fmt.Sprintf(`{"post_id":"%s","session_id":"sess_test_1","duration_ms":4500}`, postID)
	reqView := httptest.NewRequest("POST", "/api/v1/views", bytes.NewBufferString(viewBody))
	reqView.Header.Set("Content-Type", "application/json")
	recView := httptest.NewRecorder()
	mux.ServeHTTP(recView, reqView)
	if recView.Code != http.StatusAccepted {
		t.Fatalf("track view failed: %d: %s", recView.Code, recView.Body.String())
	}

	// 7. Get Post Stats
	reqStats := httptest.NewRequest("GET", "/api/v1/posts/"+postID+"/stats", nil)
	recStats := httptest.NewRecorder()
	mux.ServeHTTP(recStats, reqStats)
	if recStats.Code != http.StatusOK {
		t.Fatalf("get post stats failed: %d: %s", recStats.Code, recStats.Body.String())
	}

	// 8. Get Post Analytics (as owner User 1)
	reqAnalytics := httptest.NewRequest("GET", "/api/v1/posts/"+postID+"/analytics", nil)
	reqAnalytics.Header.Set("Authorization", "Bearer "+user1Token)
	recAnalytics := httptest.NewRecorder()
	mux.ServeHTTP(recAnalytics, reqAnalytics)
	if recAnalytics.Code != http.StatusOK {
		t.Fatalf("get post analytics failed: %d: %s", recAnalytics.Code, recAnalytics.Body.String())
	}
}
