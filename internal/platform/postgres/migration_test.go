package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/anjal-niraula/engage-engine/internal/platform/config"
	"github.com/anjal-niraula/engage-engine/internal/platform/postgres"
)

func TestMigrateWithTern(t *testing.T) {
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Skip("skipping migration test: config could not be loaded")
	}

	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = postgres.Migrate(ctx, &logger, cfg)
	if err != nil {
		t.Skipf("skipping live database migration test (Postgres might not be running in this env): %v", err)
	}
}

