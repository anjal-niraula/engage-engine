package cuckoo_test

import (
	"fmt"
	"testing"

	"github.com/anjal-niraula/engage-engine/internal/shared/cuckoo"
	"github.com/stretchr/testify/assert"
)

func TestCuckooFilter_BasicOperations(t *testing.T) {
	cf := cuckoo.NewFilter(1024)

	userPostKey1 := []byte("user_1:post_100")
	userPostKey2 := []byte("user_1:post_101")
	notLikedKey := []byte("user_1:post_999")

	// 1. Initial lookup should be false
	assert.False(t, cf.Lookup(userPostKey1))
	assert.False(t, cf.Lookup(userPostKey2))
	assert.False(t, cf.Lookup(notLikedKey))

	// 2. Insert items
	err := cf.Insert(userPostKey1)
	assert.NoError(t, err)
	err = cf.Insert(userPostKey2)
	assert.NoError(t, err)
	assert.Equal(t, uint32(2), cf.Count())

	// 3. Lookup inserted items (Guaranteed True)
	assert.True(t, cf.Lookup(userPostKey1))
	assert.True(t, cf.Lookup(userPostKey2))

	// 4. Item not inserted should be false
	assert.False(t, cf.Lookup(notLikedKey))

	// 5. Delete item (Cuckoo filter advantage over Bloom filter!)
	deleted := cf.Delete(userPostKey1)
	assert.True(t, deleted)
	assert.Equal(t, uint32(1), cf.Count())

	// 6. After deletion, lookup must return false
	assert.False(t, cf.Lookup(userPostKey1))
	assert.True(t, cf.Lookup(userPostKey2))
}

func TestCuckooFilter_ZeroFalseNegatives(t *testing.T) {
	numItems := 1000
	cf := cuckoo.NewFilter(2048)

	keys := make([][]byte, numItems)
	for i := 0; i < numItems; i++ {
		keys[i] = []byte(fmt.Sprintf("user_%d:post_%d", i, i*2))
		err := cf.Insert(keys[i])
		assert.NoError(t, err)
	}

	// Mathematical property of Cuckoo filters: ZERO false negatives
	for i := 0; i < numItems; i++ {
		assert.True(t, cf.Lookup(keys[i]), "expected item %d to be found without false negative", i)
	}
}

func TestCuckooFilter_LowFalsePositiveRate(t *testing.T) {
	cf := cuckoo.NewFilter(4096)

	// Insert 1000 items
	for i := 0; i < 1000; i++ {
		key := []byte(fmt.Sprintf("user_alice:liked_post_%d", i))
		_ = cf.Insert(key)
	}

	// Test 1000 items that were NOT inserted
	falsePositives := 0
	testCount := 1000
	for i := 0; i < testCount; i++ {
		key := []byte(fmt.Sprintf("user_alice:unliked_post_%d", i+100000))
		if cf.Lookup(key) {
			falsePositives++
		}
	}

	fpRate := float64(falsePositives) / float64(testCount)
	t.Logf("Observed False Positive Rate: %.4f (%d/%d)", fpRate, falsePositives, testCount)

	// With 16-bit fingerprints, theoretical false positive rate is ~0.003 (0.3%)
	assert.Less(t, fpRate, 0.02, "false positive rate must be below 2%")
}
