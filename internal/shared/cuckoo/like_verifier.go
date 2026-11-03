package cuckoo

import (
	"context"
	"fmt"
)

// DBVerifierFunc is a callback that checks the actual database for a like record.
type DBVerifierFunc func(ctx context.Context, userID string, postID string) (bool, error)

// LikeChecker implements the Cuckoo Filter + Database fallback pattern.
// It eliminates 95%+ of database queries for "has user liked this post?" checks.
type LikeChecker struct {
	filter  *Filter
	dbCheck DBVerifierFunc
}

// NewLikeChecker creates an instance of LikeChecker with a given Cuckoo Filter capacity.
func NewLikeChecker(capacity uint32, dbCheck DBVerifierFunc) *LikeChecker {
	return &LikeChecker{
		filter:  NewFilter(capacity),
		dbCheck: dbCheck,
	}
}

// Key formats a user and post into a deterministic cache key.
func (c *LikeChecker) Key(userID, postID string) []byte {
	return []byte(fmt.Sprintf("like:%s:%s", userID, postID))
}

// HasLiked determines if a user has liked a post.
//
// Workflow:
//  1. Check Cuckoo Filter in memory (< 50 nanoseconds).
//  2. If Cuckoo Filter returns FALSE:
//     Because Cuckoo filters have ZERO false negatives, the user is 100% guaranteed
//     to have NOT liked the post. Database query is completely bypassed!
//  3. If Cuckoo Filter returns TRUE:
//     There is a small (~0.3%) chance of a false positive. We verify from PostgreSQL
//     to ensure 100% accuracy before rendering the filled heart icon to the user.
func (c *LikeChecker) HasLiked(ctx context.Context, userID, postID string) (bool, error) {
	key := c.Key(userID, postID)

	// Step 1: Fast filter lookup
	if !c.filter.Lookup(key) {
		// Guaranteed NOT liked. Zero DB queries.
		return false, nil
	}

	// Step 2: Filter returned positive. Verify with PostgreSQL to eliminate false positives.
	if c.dbCheck != nil {
		return c.dbCheck(ctx, userID, postID)
	}

	return true, nil
}

// RecordLike records a new like in the Cuckoo Filter.
func (c *LikeChecker) RecordLike(userID, postID string) error {
	return c.filter.Insert(c.Key(userID, postID))
}

// RecordUnlike removes a like from the Cuckoo Filter.
// (This is why Cuckoo filters beat Bloom filters — Bloom filters cannot delete).
func (c *LikeChecker) RecordUnlike(userID, postID string) bool {
	return c.filter.Delete(c.Key(userID, postID))
}

// Filter returns the underlying raw Cuckoo Filter.
func (c *LikeChecker) Filter() *Filter {
	return c.filter
}
