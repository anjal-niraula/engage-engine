package cuckoo_test

import (
	"context"
	"testing"

	"github.com/anjal-niraula/engage-engine/internal/shared/cuckoo"
	"github.com/stretchr/testify/assert"
)

func TestLikeChecker_Workflow(t *testing.T) {
	dbCallCount := 0
	mockDB := map[string]bool{
		"like:user_1:post_100": true,
	}

	dbVerifier := func(ctx context.Context, userID, postID string) (bool, error) {
		dbCallCount++
		key := "like:" + userID + ":" + postID
		return mockDB[key], nil
	}

	checker := cuckoo.NewLikeChecker(1024, dbVerifier)

	ctx := context.Background()

	// 1. Check an unliked post:
	// Filter returns false -> DB is NOT called!
	liked, err := checker.HasLiked(ctx, "user_1", "post_999")
	assert.NoError(t, err)
	assert.False(t, liked)
	assert.Equal(t, 0, dbCallCount, "database must NOT be called when Cuckoo Filter returns false")

	// 2. User likes post_100:
	err = checker.RecordLike("user_1", "post_100")
	assert.NoError(t, err)

	// 3. Check liked post:
	// Filter returns true -> DB is called to verify -> returns true
	liked, err = checker.HasLiked(ctx, "user_1", "post_100")
	assert.NoError(t, err)
	assert.True(t, liked)
	assert.Equal(t, 1, dbCallCount, "database is called only to verify positive match")

	// 4. User unlikes post_100:
	mockDB["like:user_1:post_100"] = false
	unliked := checker.RecordUnlike("user_1", "post_100")
	assert.True(t, unliked)

	// 5. Check post_100 again:
	// Filter returns false -> DB is NOT called again!
	liked, err = checker.HasLiked(ctx, "user_1", "post_100")
	assert.NoError(t, err)
	assert.False(t, liked)
	assert.Equal(t, 1, dbCallCount, "database count should remain 1 because filter bypassed the check")
}
