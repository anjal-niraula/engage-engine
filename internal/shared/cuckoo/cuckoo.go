package cuckoo

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"hash/fnv"
	"sync"
)

const (
	// DefaultBucketSize is the number of fingerprint slots per bucket (standard b=4)
	DefaultBucketSize = 4
	// MaxCuckooKicks is the maximum number of displacements before declaring the table full
	MaxCuckooKicks = 500
)

var (
	ErrFilterFull = errors.New("cuckoo filter is full, maximum kicks exceeded")
	ErrNotFound   = errors.New("item not found in filter")
)

type bucket [DefaultBucketSize]uint16

// Filter represents a concurrent-safe, memory-efficient Cuckoo Filter.
type Filter struct {
	mu         sync.RWMutex
	buckets    []bucket
	numBuckets uint32
	count      uint32
	mask       uint32
}

// NewFilter creates a new Cuckoo Filter with the given capacity.
// Capacity will be rounded up to the nearest power of 2 for fast bitwise masking.
func NewFilter(capacity uint32) *Filter {
	if capacity < 8 {
		capacity = 8
	}

	// Round up to nearest power of 2
	n := uint32(1)
	for n < capacity {
		n <<= 1
	}

	return &Filter{
		buckets:    make([]bucket, n),
		numBuckets: n,
		mask:       n - 1,
	}
}

// fingerprint generates a non-zero 16-bit fingerprint from an item.
func (f *Filter) fingerprint(data []byte) uint16 {
	h := fnv.New64a()
	h.Write(data)
	sum := h.Sum64()
	fp := uint16(sum ^ (sum >> 16) ^ (sum >> 32) ^ (sum >> 48))
	if fp == 0 {
		return 1
	}
	return fp
}

// hashIndices computes the primary and secondary bucket indices for an item.
func (f *Filter) hashIndices(data []byte, fp uint16) (uint32, uint32) {
	h := fnv.New32a()
	h.Write(data)
	i1 := h.Sum32() & f.mask

	// Alternate index derived via i2 = i1 XOR hash(fp)
	fpBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(fpBytes, fp)
	h.Reset()
	h.Write(fpBytes)
	i2 := (i1 ^ h.Sum32()) & f.mask

	return i1, i2
}

// altIndex calculates the alternate bucket index given an index and fingerprint.
func (f *Filter) altIndex(index uint32, fp uint16) uint32 {
	h := fnv.New32a()
	fpBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(fpBytes, fp)
	h.Write(fpBytes)
	return (index ^ h.Sum32()) & f.mask
}

// Insert adds an item to the Cuckoo Filter.
// Returns nil on success, or ErrFilterFull if capacity is exhausted.
func (f *Filter) Insert(item []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	fp := f.fingerprint(item)
	i1, i2 := f.hashIndices(item, fp)

	// 1. Try to insert into i1 if empty slot exists
	for s := 0; s < DefaultBucketSize; s++ {
		if f.buckets[i1][s] == 0 {
			f.buckets[i1][s] = fp
			f.count++
			return nil
		}
	}

	// 2. Try to insert into i2 if empty slot exists
	for s := 0; s < DefaultBucketSize; s++ {
		if f.buckets[i2][s] == 0 {
			f.buckets[i2][s] = fp
			f.count++
			return nil
		}
	}

	// 3. Both buckets are full: kick an existing fingerprint out (Cuckoo eviction)
	currIndex := i1
	if randomBool() {
		currIndex = i2
	}
	currFP := fp

	for k := 0; k < MaxCuckooKicks; k++ {
		// Pick random slot to displace
		slot := randomSlot()
		currFP, f.buckets[currIndex][slot] = f.buckets[currIndex][slot], currFP

		currIndex = f.altIndex(currIndex, currFP)

		// Check if the alternate bucket has space
		for s := 0; s < DefaultBucketSize; s++ {
			if f.buckets[currIndex][s] == 0 {
				f.buckets[currIndex][s] = currFP
				f.count++
				return nil
			}
		}
	}

	return ErrFilterFull
}

// Lookup tests whether an item might be in the filter.
// Guaranteed: If it returns false, the item is 100% NOT in the set (zero false negatives).
// If it returns true, there is a small false-positive probability (~1%).
func (f *Filter) Lookup(item []byte) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	fp := f.fingerprint(item)
	i1, i2 := f.hashIndices(item, fp)

	// Check bucket i1
	for s := 0; s < DefaultBucketSize; s++ {
		if f.buckets[i1][s] == fp {
			return true
		}
	}

	// Check bucket i2
	for s := 0; s < DefaultBucketSize; s++ {
		if f.buckets[i2][s] == fp {
			return true
		}
	}

	return false
}

// Delete removes an item from the filter.
// Unlike Bloom filters, Cuckoo filters support deletion!
func (f *Filter) Delete(item []byte) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	fp := f.fingerprint(item)
	i1, i2 := f.hashIndices(item, fp)

	for s := 0; s < DefaultBucketSize; s++ {
		if f.buckets[i1][s] == fp {
			f.buckets[i1][s] = 0
			f.count--
			return true
		}
	}

	for s := 0; s < DefaultBucketSize; s++ {
		if f.buckets[i2][s] == fp {
			f.buckets[i2][s] = 0
			f.count--
			return true
		}
	}

	return false
}

// Count returns the number of active items in the filter.
func (f *Filter) Count() uint32 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.count
}

// Reset clears all items in the filter.
func (f *Filter) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.buckets {
		f.buckets[i] = bucket{}
	}
	f.count = 0
}

func randomBool() bool {
	var b [1]byte
	_, _ = rand.Read(b[:])
	return b[0]&1 == 1
}

func randomSlot() int {
	var b [1]byte
	_, _ = rand.Read(b[:])
	return int(b[0] % DefaultBucketSize)
}
