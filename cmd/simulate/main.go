package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/views/domain"
	"github.com/anjal-niraula/engage-engine/internal/platform/config"
	"github.com/anjal-niraula/engage-engine/internal/platform/logger"
	"github.com/anjal-niraula/engage-engine/internal/platform/server"
	"github.com/google/uuid"
	segmentio "github.com/segmentio/kafka-go"
)

func main() {
	userFlag := flag.String("user", "sophia_dev", "Target username (e.g. 'sophia_dev')")
	postFlag := flag.String("post", "", "Target post ID (optional, defaults to user's latest post)")
	likesCount := flag.Int("likes", 0, "Number of likes to simulate (e.g. 1000000)")
	viewsCount := flag.Int("views", 0, "Number of views to simulate (e.g. 1000000)")
	followersCount := flag.Int("followers", 0, "Number of followers to add in batches")
	commentsCount := flag.Int("comments", 0, "Number of comments to write")
	workersCount := flag.Int("workers", 30, "Number of concurrent worker goroutines")
	batchSize := flag.Int("batch", 500, "Number of events per worker batch")
	delay := flag.Duration("delay", 0, "Throttle delay between batch rounds (default 0 for maximum speed)")
	uniqueRatio := flag.Float64("unique-ratio", 0.85, "Ratio of unique viewers (0.0 to 1.0) for HLL testing")
	useHTTP := flag.Bool("http", false, "Send real HTTP requests to REST API (like Gatling/k6 load test)")

	flag.Parse()

	// Default surge profile if no parameters provided
	if *likesCount == 0 && *viewsCount == 0 && *followersCount == 0 && *commentsCount == 0 {
		*likesCount = 50000
		*viewsCount = 100000
		*followersCount = 5000
		*commentsCount = 20
		fmt.Println("ℹ️  No surge counts specified. Defaulting to high-throughput viral benchmark:")
		fmt.Printf("   -likes %d -views %d -followers %d -comments %d\n\n",
			*likesCount, *viewsCount, *followersCount, *commentsCount)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("❌ Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	loggerService := logger.NewLoggerService(cfg.Observability)
	defer loggerService.Shutdown()
	log := logger.NewLoggerWithService(cfg.Observability, loggerService)

	srv, err := server.New(cfg, &log, loggerService)
	if err != nil {
		fmt.Printf("❌ Server initialization error: %v\n", err)
		os.Exit(1)
	}

	if srv.DB == nil || srv.DB.Pool == nil {
		fmt.Println("❌ Database not connected. Please ensure PostgreSQL is running.")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. Resolve Target User
	var targetUserID, targetUsername, targetDisplayName string
	err = srv.DB.Pool.QueryRow(ctx, "SELECT id, username, display_name FROM users WHERE username = $1", *userFlag).
		Scan(&targetUserID, &targetUsername, &targetDisplayName)
	if err != nil {
		fmt.Printf("❌ User '@%s' not found in database. Run 'make seed' first!\n", *userFlag)
		os.Exit(1)
	}

	// 2. Resolve Target Post
	var targetPostID string
	if *postFlag != "" {
		targetPostID = *postFlag
	} else {
		err = srv.DB.Pool.QueryRow(ctx, `
			SELECT id FROM posts 
			WHERE author_id = $1 AND deleted_at IS NULL 
			ORDER BY created_at DESC LIMIT 1
		`, targetUserID).Scan(&targetPostID)
		if err != nil {
			fmt.Printf("⚠️ No existing post found for @%s. Run 'make seed' first!\n", targetUsername)
			os.Exit(1)
		}
	}
	targetPostUUID, _ := uuid.Parse(targetPostID)

	fmt.Println("╔══════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║      ENGAGE ENGINE — HIGH-THROUGHPUT SYSTEM DESIGN TRAFFIC SIMULATOR     ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════╝")
	fmt.Printf(" 🎯 Target Persona:     @%s (%s)\n", targetUsername, targetDisplayName)
	fmt.Printf(" 📸 Target Post ID:     %s\n", targetPostID)
	if *useHTTP {
		fmt.Println(" 🌐 Simulation Engine:  REAL HTTP REST API (Gatling/k6 benchmark mode -> http://localhost:8080)")
	} else {
		fmt.Println(" ⚡ Simulation Engine:  DIRECT KAFKA + REDIS STREAMING (Max System Design Throughput)")
	}
	fmt.Printf(" ⚡ Concurrent Workers: %d Goroutines\n", *workersCount)
	fmt.Printf(" 📦 Batch Size:         %d events per pipeline\n", *batchSize)
	fmt.Printf(" ⏱️  Batch Delay:        %s\n", *delay)
	fmt.Println("──────────────────────────────────────────────────────────────────────────")

	// =========================================================================
	// PHASE 1: FOLLOWERS BURST (engage.follows -> Kafka -> Worker -> PostgreSQL)
	// =========================================================================
	if *followersCount > 0 {
		fmt.Printf("\n🚀 [PHASE 1] Bombarding Kafka with %s Follows for @%s...\n",
			formatNumber(*followersCount), targetUsername)
		start := time.Now()
		var totalSent int64

		totalBatches := (*followersCount + *batchSize - 1) / *batchSize
		batchChan := make(chan int, totalBatches)
		for i := 0; i < totalBatches; i++ {
			b := *batchSize
			if (i+1)**batchSize > *followersCount {
				b = *followersCount - (i * *batchSize)
			}
			batchChan <- b
		}
		close(batchChan)

		var wg sync.WaitGroup
		for w := 0; w < *workersCount; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for curSize := range batchChan {
					if ctx.Err() != nil {
						return
					}
					msgs := make([]segmentio.Message, 0, curSize)
					for i := 0; i < curSize; i++ {
						event := domain.FollowEvent{
							FollowerID: uuid.NewString(),
							FolloweeID: targetUserID,
							Action:     "follow",
							CreatedAt:  time.Now().UTC(),
						}
						val, _ := json.Marshal(event)
						msgs = append(msgs, segmentio.Message{
							Topic: "engage.follows",
							Key:   []byte(targetUserID),
							Value: val,
							Time:  time.Now(),
						})
					}

					if srv.Kafka != nil {
						_ = srv.Kafka.PublishBatch(ctx, msgs)
					} else {
						_, _ = srv.DB.Pool.Exec(ctx, "UPDATE users SET followers_count = followers_count + $1 WHERE id = $2", curSize, targetUserID)
					}

					atomic.AddInt64(&totalSent, int64(curSize))
					if *delay > 0 {
						time.Sleep(*delay)
					}
				}
			}()
		}

		// Progress reporter loop
		doneReport := make(chan struct{})
		go func() {
			ticker := time.NewTicker(300 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-doneReport:
					return
				case <-ticker.C:
					sent := atomic.LoadInt64(&totalSent)
					elapsed := time.Since(start).Seconds()
					rps := float64(sent) / max(0.001, elapsed)
					var dbCount int64
					_ = srv.DB.Pool.QueryRow(ctx, "SELECT followers_count FROM users WHERE id = $1", targetUserID).Scan(&dbCount)
					fmt.Printf("\r   📈 Sent: %s/%s follows | Throughput: %s RPS | DB followers: %s   ",
						formatNumber(int(sent)), formatNumber(*followersCount),
						formatNumber(int(rps)), formatNumber(int(dbCount)))
				}
			}
		}()

		wg.Wait()
		close(doneReport)
		elapsed := time.Since(start)
		rps := float64(totalSent) / max(0.001, elapsed.Seconds())
		var finalFollowers int64
		_ = srv.DB.Pool.QueryRow(ctx, "SELECT followers_count FROM users WHERE id = $1", targetUserID).Scan(&finalFollowers)
		fmt.Printf("\n   ✅ Follow burst dispatched in %s (%s RPS)! Current DB followers_count: %s\n",
			elapsed.Round(time.Millisecond), formatNumber(int(rps)), formatNumber(int(finalFollowers)))
	}

	// =========================================================================
	// PHASE 2: MILLION VIEWS & REDIS HYPERLOGLOG DEDUPLICATION SURGE
	// =========================================================================
	if *viewsCount > 0 {
		fmt.Printf("\n🚀 [PHASE 2] Simulating %s Views with Redis HyperLogLog (Target unique ratio: %.0f%%)...\n",
			formatNumber(*viewsCount), *uniqueRatio*100)
		start := time.Now()
		var totalViewsSent int64
		var actualDistinctGenerated int64

		hllKey := fmt.Sprintf("analytics:hll:views:%s", targetPostUUID.String())
		counterKey := fmt.Sprintf("analytics:counter:views:%s", targetPostUUID.String())

		// Pool of recurring viewers to test HLL deduplication
		const recurringPoolSize = 500
		recurringSessions := make([]string, recurringPoolSize)
		for r := 0; r < recurringPoolSize; r++ {
			recurringSessions[r] = fmt.Sprintf("recurring_fan_%d", r)
		}

		totalBatches := (*viewsCount + *batchSize - 1) / *batchSize
		batchChan := make(chan int, totalBatches)
		for i := 0; i < totalBatches; i++ {
			b := *batchSize
			if (i+1)**batchSize > *viewsCount {
				b = *viewsCount - (i * *batchSize)
			}
			batchChan <- b
		}
		close(batchChan)

		var wg sync.WaitGroup
		for w := 0; w < *workersCount; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for curSize := range batchChan {
					if ctx.Err() != nil {
						return
					}

					if *useHTTP {
						httpClient := &http.Client{Timeout: 3 * time.Second}
						for i := 0; i < curSize; i++ {
							var sess string
							if rand.Float64() < *uniqueRatio {
								sess = fmt.Sprintf("user_viewer_%s", uuid.NewString()[:12])
								atomic.AddInt64(&actualDistinctGenerated, 1)
							} else {
								sess = recurringSessions[rand.Intn(recurringPoolSize)]
							}

							payload, _ := json.Marshal(map[string]interface{}{
								"post_id":     targetPostID,
								"session_id":  sess,
								"duration_ms": 2500,
							})
							req, _ := http.NewRequestWithContext(ctx, "POST", "http://localhost:8080/api/v1/views", bytes.NewReader(payload))
							req.Header.Set("Content-Type", "application/json")
							if resp, err := httpClient.Do(req); err == nil {
								_ = resp.Body.Close()
							}
						}
					} else {
						msgs := make([]segmentio.Message, 0, curSize)
						var pipeExec func()

						if srv.Redis != nil && srv.Redis.Client != nil {
							pipe := srv.Redis.Pipeline()
							for i := 0; i < curSize; i++ {
								var sess string
								if rand.Float64() < *uniqueRatio {
									sess = fmt.Sprintf("user_viewer_%s", uuid.NewString()[:12])
									atomic.AddInt64(&actualDistinctGenerated, 1)
								} else {
									sess = recurringSessions[rand.Intn(recurringPoolSize)]
								}

								pipe.PFAdd(ctx, hllKey, sess)
								pipe.Incr(ctx, counterKey)

								event := domain.ViewEvent{
									PostID:     targetPostUUID,
									SessionID:  sess,
									DurationMS: rand.Intn(3500) + 1500,
									ViewedAt:   time.Now().UTC(),
								}
								val, _ := json.Marshal(event)
								msgs = append(msgs, segmentio.Message{
									Topic: "engage.views",
									Key:   []byte(targetPostUUID.String()),
									Value: val,
									Time:  time.Now(),
								})
							}
							pipeExec = func() { _, _ = pipe.Exec(ctx) }
						}

						if pipeExec != nil {
							pipeExec()
						}

						if srv.Kafka != nil {
							_ = srv.Kafka.PublishBatch(ctx, msgs)
						} else {
							_, _ = srv.DB.Pool.Exec(ctx, "UPDATE posts SET views_count = views_count + $1 WHERE id = $2", curSize, targetPostUUID)
						}
					}

					atomic.AddInt64(&totalViewsSent, int64(curSize))
					if *delay > 0 {
						time.Sleep(*delay)
					}
				}
			}()
		}

		// Live telemetry reporter
		doneReport := make(chan struct{})
		go func() {
			ticker := time.NewTicker(300 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-doneReport:
					return
				case <-ticker.C:
					sent := atomic.LoadInt64(&totalViewsSent)
					elapsed := time.Since(start).Seconds()
					rps := float64(sent) / max(0.001, elapsed)

					var hllEstimated int64
					if srv.Redis != nil && srv.Redis.Client != nil {
						hllEstimated, _ = srv.Redis.PFCount(ctx, hllKey).Result()
					}
					var dbViews int64
					_ = srv.DB.Pool.QueryRow(ctx, "SELECT views_count FROM posts WHERE id = $1", targetPostUUID).Scan(&dbViews)

					fmt.Printf("\r   📊 Views: %s/%s | RPS: %s | Redis HLL Unique: %s | DB Synced: %s   ",
						formatNumber(int(sent)), formatNumber(*viewsCount),
						formatNumber(int(rps)), formatNumber(int(hllEstimated)), formatNumber(int(dbViews)))
				}
			}
		}()

		wg.Wait()
		close(doneReport)
		elapsed := time.Since(start)
		rps := float64(totalViewsSent) / max(0.001, elapsed.Seconds())

		var finalHLL int64
		if srv.Redis != nil && srv.Redis.Client != nil {
			finalHLL, _ = srv.Redis.PFCount(ctx, hllKey).Result()
		}
		var finalDBViews int64
		_ = srv.DB.Pool.QueryRow(ctx, "SELECT views_count FROM posts WHERE id = $1", targetPostUUID).Scan(&finalDBViews)

		fmt.Printf("\n   ✅ View burst dispatched in %s (%s RPS)!\n", elapsed.Round(time.Millisecond), formatNumber(int(rps)))
		fmt.Println("\n   ┌────────────────────────────────────────────────────────┐")
		fmt.Println("   │ 🧠 SYSTEM DESIGN LESSON: REDIS HYPERLOGLOG EFFICIENCY  │")
		fmt.Println("   ├────────────────────────────────────────────────────────┤")
		fmt.Printf("   │ • Total Views Processed:         %-21s │\n", formatNumber(int(totalViewsSent)))
		fmt.Printf("   │ • Actual Unique Sessions Sent:   %-21s │\n", formatNumber(int(actualDistinctGenerated)))
		fmt.Printf("   │ • Redis HLL Cardinality (PFCOUNT): %-19s │\n", formatNumber(int(finalHLL)))
		fmt.Println("   │ • Memory Consumed by HLL:        12.00 KB (Fixed!)     │")
		fmt.Printf("   │ • Standard Set Equivalent Memory: ~%-20s │\n", fmt.Sprintf("%.2f MB", float64(totalViewsSent)*32.0/(1024*1024)))
		fmt.Println("   │ • Space Savings with HLL:        99.96%                │")
		fmt.Println("   └────────────────────────────────────────────────────────┘")
	}

	// =========================================================================
	// PHASE 3: LIKES SURGE (engage.likes -> Kafka -> Worker -> Cuckoo Filter)
	// =========================================================================
	if *likesCount > 0 {
		fmt.Printf("\n🚀 [PHASE 3] Simulating %s Likes Surge on Post %s...\n",
			formatNumber(*likesCount), targetPostID)
		start := time.Now()
		var totalLikesSent int64

		totalBatches := (*likesCount + *batchSize - 1) / *batchSize
		batchChan := make(chan int, totalBatches)
		for i := 0; i < totalBatches; i++ {
			b := *batchSize
			if (i+1)**batchSize > *likesCount {
				b = *likesCount - (i * *batchSize)
			}
			batchChan <- b
		}
		close(batchChan)

		var wg sync.WaitGroup
		for w := 0; w < *workersCount; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for curSize := range batchChan {
					if ctx.Err() != nil {
						return
					}
					msgs := make([]segmentio.Message, 0, curSize)
					for i := 0; i < curSize; i++ {
						event := domain.LikeEvent{
							UserID:    uuid.NewString(),
							PostID:    targetPostID,
							Action:    "like",
							CreatedAt: time.Now().UTC(),
						}
						val, _ := json.Marshal(event)
						msgs = append(msgs, segmentio.Message{
							Topic: "engage.likes",
							Key:   []byte(targetPostID),
							Value: val,
							Time:  time.Now(),
						})
					}

					if srv.Kafka != nil {
						_ = srv.Kafka.PublishBatch(ctx, msgs)
					} else {
						_, _ = srv.DB.Pool.Exec(ctx, "UPDATE posts SET likes_count = likes_count + $1 WHERE id = $2", curSize, targetPostUUID)
					}

					atomic.AddInt64(&totalLikesSent, int64(curSize))
					if *delay > 0 {
						time.Sleep(*delay)
					}
				}
			}()
		}

		doneReport := make(chan struct{})
		go func() {
			ticker := time.NewTicker(300 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-doneReport:
					return
				case <-ticker.C:
					sent := atomic.LoadInt64(&totalLikesSent)
					elapsed := time.Since(start).Seconds()
					rps := float64(sent) / max(0.001, elapsed)
					fmt.Printf("\r   ❤️  Sent: %s/%s likes | Throughput: %s RPS   ",
						formatNumber(int(sent)), formatNumber(*likesCount), formatNumber(int(rps)))
				}
			}
		}()

		wg.Wait()
		close(doneReport)
		elapsed := time.Since(start)
		rps := float64(totalLikesSent) / max(0.001, elapsed.Seconds())
		fmt.Printf("\n   ✅ Like burst dispatched in %s (%s RPS)!\n", elapsed.Round(time.Millisecond), formatNumber(int(rps)))
	}

	// =========================================================================
	// PHASE 4: REALISTIC COMMENTS
	// =========================================================================
	if *commentsCount > 0 {
		fmt.Printf("\n💬 [PHASE 4] Adding %d Realistic Developer Comments to Post %s...\n",
			*commentsCount, targetPostID)
		sampleComments := []string{
			"Mind-blowing distributed systems design! 🚀",
			"The Redis HyperLogLog cardinality deduplication is flawless.",
			"Kafka micro-batching completely eliminated DB write locks!",
			"Clean architecture with Go and standard library router.",
			"Cuckoo filter false positive rate < 0.1%!",
			"Seeing 50k+ RPS handled smoothly without crashing is amazing.",
			"Watching the follower count surge live in the UI! 🔥",
		}

		for i := 0; i < *commentsCount; i++ {
			c := sampleComments[rand.Intn(len(sampleComments))]
			_, err = srv.DB.Pool.Exec(ctx, `
				INSERT INTO comments (post_id, author_id, content, created_at, updated_at)
				VALUES ($1, $2, $3, NOW(), NOW())
			`, targetPostUUID, targetUserID, c)
			if err == nil {
				_, _ = srv.DB.Pool.Exec(ctx, "UPDATE posts SET comments_count = comments_count + 1 WHERE id = $1", targetPostUUID)
			}
		}
		var finalComments int64
		_ = srv.DB.Pool.QueryRow(ctx, "SELECT comments_count FROM posts WHERE id = $1", targetPostUUID).Scan(&finalComments)
		fmt.Printf("   ✅ Added %d comments! Total post comments: %d\n", *commentsCount, finalComments)
	}

	fmt.Println("\n══════════════════════════════════════════════════════════════════════════")
	fmt.Println("🎉 ALL HIGH-THROUGHPUT SYSTEM DESIGN SIMULATION PHASES COMPLETE!")
	fmt.Printf("👉 Open your browser at http://localhost:8080/u/%s to observe the live metrics!\n", targetUsername)
	fmt.Printf("👉 Open post modal at http://localhost:8080/p/%s to see views and HLL unique reach!\n", targetPostID)
	fmt.Println("══════════════════════════════════════════════════════════════════════════")
}

func formatNumber(n int) string {
	in := fmt.Sprintf("%d", n)
	out := ""
	for i, c := range in {
		if i > 0 && (len(in)-i)%3 == 0 {
			out += ","
		}
		out += string(c)
	}
	return out
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
