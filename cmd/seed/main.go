package main

import (
	"context"
	"fmt"
	"os"

	"github.com/anjal-niraula/engage-engine/internal/modules/views/domain"
	"github.com/anjal-niraula/engage-engine/internal/platform/config"
	"github.com/anjal-niraula/engage-engine/internal/platform/logger"
	"github.com/anjal-niraula/engage-engine/internal/platform/server"
	"golang.org/x/crypto/bcrypt"
)

type Persona struct {
	Username    string
	Email       string
	DisplayName string
	Bio         string
	AvatarURL   string
	Posts       []SeedPost
}

type SeedPost struct {
	Caption  string
	MediaURL string
	Location string
}

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("❌ Failed to load config: %v\n", err)
		os.Exit(1)
	}

	loggerService := logger.NewLoggerService(cfg.Observability)
	defer loggerService.Shutdown()
	log := logger.NewLoggerWithService(cfg.Observability, loggerService)

	srv, err := server.New(cfg, &log, loggerService)
	if err != nil {
		fmt.Printf("❌ Failed to initialize dependencies: %v\n", err)
		os.Exit(1)
	}

	if srv.DB == nil || srv.DB.Pool == nil {
		fmt.Println("❌ Database is not connected. Please ensure PostgreSQL is running.")
		os.Exit(1)
	}

	ctx := context.Background()

	personas := []Persona{
		{
			Username:    "sophia_dev",
			Email:       "sophia_dev@example.com",
			DisplayName: "Sophia Chen",
			Bio:         "Distributed systems in Go & Rust 💻\nCoffee, Linux, & dark mode.",
			AvatarURL:   "https://images.unsplash.com/photo-1517841905240-472988babdf9?w=300",
			Posts: []SeedPost{
				{
					Caption:  "Profiling Kafka partition balancing and Redis HyperLogLog reach! 🚀 Distributed tracing is pure art. #systemdesign #golang",
					MediaURL: "https://images.unsplash.com/photo-1517694712202-14dd9538aa97?w=900",
					Location: "San Francisco, CA",
				},
				{
					Caption:  "Late night debugging with terminal dark mode and hot espresso. ☕💻 #developerlife #coffee",
					MediaURL: "https://images.unsplash.com/photo-1501386761578-eac5c94b800a?w=900",
					Location: "North Beach Coffee",
				},
				{
					Caption:  "Weekend mountain hike to reset the cache. 🏔️🌲 No Wi-Fi, just trees and fresh air.",
					MediaURL: "https://images.unsplash.com/photo-1464822759023-fed622ff2c3b?w=900",
					Location: "Yosemite National Park",
				},
			},
		},
		{
			Username:    "alex_travels",
			Email:       "alex_travels@example.com",
			DisplayName: "Alex Rivera",
			Bio:         "Travel photographer & nomad 🌍\nLiving out of a backpack.",
			AvatarURL:   "https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=300",
			Posts: []SeedPost{
				{
					Caption:  "Golden hour over the Amalfi coast. Italy has my heart forever. 🌅🇮🇹 #wanderlust #travelphotography",
					MediaURL: "https://images.unsplash.com/photo-1533105079780-92b9be482077?w=900",
					Location: "Positano, Italy",
				},
				{
					Caption:  "Sunrise above the clouds in the Swiss Alps. Crisp alpine morning! ❄️🏔️ #alps #adventure",
					MediaURL: "https://images.unsplash.com/photo-1506744038136-46273834b3fb?w=900",
					Location: "Zermatt, Switzerland",
				},
			},
		},
		{
			Username:    "chef_marco",
			Email:       "chef_marco@example.com",
			DisplayName: "Marco Rossi",
			Bio:         "Michelin-trained chef 🍝\nHandcrafted pasta & Italian wine.",
			AvatarURL:   "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=300",
			Posts: []SeedPost{
				{
					Caption:  "Fresh handmade tagliatelle with summer black truffles. Simplicity is perfection. 🍝✨ #pasta #italianfood",
					MediaURL: "https://images.unsplash.com/photo-1555396273-367ea4eb4db5?w=900",
					Location: "Osteria Rossi, Florence",
				},
				{
					Caption:  "Wood-fired sourdough pizza straight from the oven. 48-hour fermented dough. 🍕🔥 #artisan #pizza",
					MediaURL: "https://images.unsplash.com/photo-1513104890138-7c749659a591?w=900",
					Location: "Naples, Italy",
				},
			},
		},
		{
			Username:    "elena_fit",
			Email:       "elena_fit@example.com",
			DisplayName: "Elena Rostova",
			Bio:         "Strength coach & endurance athlete 🏋️‍♀️\nTrain with intention.",
			AvatarURL:   "https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=300",
			Posts: []SeedPost{
				{
					Caption:  "Morning 10k pace run along the coastline. Consistency over intensity every single time. 🏃‍♀️💨 #runner #fitness",
					MediaURL: "https://images.unsplash.com/photo-1517838277536-f5f99be501cd?w=900",
					Location: "Pacific Coast Highway",
				},
			},
		},
		{
			Username:    "neo_tokyo",
			Email:       "neo_tokyo@example.com",
			DisplayName: "Kenji Sato",
			Bio:         "Tokyo night street photography 📸\nNeon & rain aesthetic.",
			AvatarURL:   "https://images.unsplash.com/photo-1500648767791-00dcc994a43e?w=300",
			Posts: []SeedPost{
				{
					Caption:  "Midnight reflections through Shinjuku alleys after the rain. 🌧️⚡ #tokyonights #cyberpunk",
					MediaURL: "https://images.unsplash.com/photo-1503899036084-c55cdd92da26?w=900",
					Location: "Shinjuku, Tokyo",
				},
			},
		},
	}

	fmt.Println("🚀 Seeding Instagram personas, posts, initial views, and social graph...")

	hash, _ := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.DefaultCost)
	passHash := string(hash)

	userIDs := make(map[string]string)
	var allPostIDs []string

	// 1. Upsert personas
	for _, p := range personas {
		var uid string
		err := srv.DB.Pool.QueryRow(ctx, `
			INSERT INTO users (username, email, password_hash, display_name, bio, avatar_url, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
			ON CONFLICT (username) 
			DO UPDATE SET display_name = EXCLUDED.display_name, bio = EXCLUDED.bio, avatar_url = EXCLUDED.avatar_url
			RETURNING id
		`, p.Username, p.Email, passHash, p.DisplayName, p.Bio, p.AvatarURL).Scan(&uid)
		if err != nil {
			fmt.Printf("⚠️ User %s error: %v\n", p.Username, err)
			continue
		}
		userIDs[p.Username] = uid
		fmt.Printf("  👤 Seeded user: @%s (%s)\n", p.Username, uid)

		// 2. Seed posts for this persona
		for _, post := range p.Posts {
			var pid string
			err := srv.DB.Pool.QueryRow(ctx, `
				INSERT INTO posts (author_id, caption, media_urls, media_type, location, created_at, updated_at)
				VALUES ($1, $2, $3, 'image', $4, NOW(), NOW())
				RETURNING id
			`, uid, post.Caption, []string{post.MediaURL}, post.Location).Scan(&pid)
			if err != nil {
				continue
			}
			allPostIDs = append(allPostIDs, pid)

			// Seed initial views: 15 views from 6 unique viewers
			initViews := 15
			for v := 0; v < initViews; v++ {
				sess := fmt.Sprintf("user_seed_viewer_%d", v%6)
				if srv.Redis != nil && srv.Redis.Client != nil {
					hllKey := fmt.Sprintf("analytics:hll:views:%s", pid)
					counterKey := fmt.Sprintf("analytics:counter:views:%s", pid)
					srv.Redis.PFAdd(ctx, hllKey, sess)
					srv.Redis.Incr(ctx, counterKey)
				}
			}

			_, _ = srv.DB.Pool.Exec(ctx, `UPDATE posts SET views_count = $1 WHERE id = $2`, initViews, pid)
			_, _ = srv.DB.Pool.Exec(ctx, `UPDATE users SET posts_count = posts_count + 1 WHERE id = $1`, uid)
			fmt.Printf("    📸 Seeded post: %s (with %d initial views)\n", pid, initViews)
		}
	}

	// 3. Seed social connections (follows)
	usernames := []string{"sophia_dev", "alex_travels", "chef_marco", "elena_fit", "neo_tokyo"}
	for i, u1 := range usernames {
		id1 := userIDs[u1]
		for j, u2 := range usernames {
			if i != j {
				id2 := userIDs[u2]
				_, _ = srv.DB.Pool.Exec(ctx, `
					INSERT INTO follows (follower_id, followee_id)
					VALUES ($1, $2)
					ON CONFLICT (follower_id, followee_id) DO NOTHING
				`, id1, id2)
			}
		}
		// Update follower & following counts
		_, _ = srv.DB.Pool.Exec(ctx, `
			UPDATE users 
			SET followers_count = (SELECT COUNT(*) FROM follows WHERE followee_id = $1),
			    following_count = (SELECT COUNT(*) FROM follows WHERE follower_id = $1)
			WHERE id = $1
		`, id1)
	}

	// 4. Seed initial comments & likes
	sampleComments := []string{
		"Incredible setup! 🚀",
		"Love the color grading on this! 📸",
		"Teach me your ways 🙌",
		"This looks so delicious! 🍝",
		"Great perspective, keep it up!",
	}

	for _, pid := range allPostIDs {
		// Like from a couple personas
		for _, u := range usernames[:3] {
			uid := userIDs[u]
			_, _ = srv.DB.Pool.Exec(ctx, `
				INSERT INTO likes (user_id, post_id)
				VALUES ($1, $2)
				ON CONFLICT (user_id, post_id) DO NOTHING
			`, uid, pid)
		}
		_, _ = srv.DB.Pool.Exec(ctx, `
			UPDATE posts 
			SET likes_count = (SELECT COUNT(*) FROM likes WHERE post_id = $1)
			WHERE id = $1
		`, pid)

		// A comment
		_, _ = srv.DB.Pool.Exec(ctx, `
			INSERT INTO comments (post_id, author_id, content, created_at, updated_at)
			VALUES ($1, $2, $3, NOW(), NOW())
		`, pid, userIDs["alex_travels"], sampleComments[len(pid)%len(sampleComments)])
		_, _ = srv.DB.Pool.Exec(ctx, `
			UPDATE posts 
			SET comments_count = (SELECT COUNT(*) FROM comments WHERE post_id = $1 AND deleted_at IS NULL)
			WHERE id = $1
		`, pid)
	}

	// Unused event struct check to ensure import safety
	_ = domain.ViewEvent{}

	fmt.Println("✅ Database successfully seeded with rich Instagram personas, posts, initial engagement, and views!")
}

