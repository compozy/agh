package bridgesdk

import (
	"testing"
	"time"
)

func TestDedupCacheSuppressesDuplicatesWithinTTLAndReleasesAfterExpiry(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	cache := NewDedupCache(time.Minute, 10)
	cache.now = func() time.Time { return now }

	if duplicate := cache.Mark("dup-key"); duplicate {
		t.Fatal("first Mark() = duplicate, want false")
	}
	if duplicate := cache.Mark("dup-key"); !duplicate {
		t.Fatal("second Mark() = false, want true")
	}

	now = now.Add(2 * time.Minute)
	if duplicate := cache.Mark("dup-key"); duplicate {
		t.Fatal("Mark() after expiry = true, want false")
	}
}

func TestDedupCacheClearDropsTrackedKeys(t *testing.T) {
	t.Parallel()

	cache := NewDedupCache(time.Minute, 10)
	cache.Mark("dup-key")
	cache.Clear()

	if duplicate := cache.Mark("dup-key"); duplicate {
		t.Fatal("Mark() after Clear() = true, want false")
	}
}

func TestNewDedupCacheAppliesDefaultBounds(t *testing.T) {
	t.Parallel()

	cache := NewDedupCache(0, 0)
	if cache.ttl != defaultDedupTTL {
		t.Fatalf("cache.ttl = %s, want %s", cache.ttl, defaultDedupTTL)
	}
	if cache.maxSize != defaultDedupMaxSize {
		t.Fatalf("cache.maxSize = %d, want %d", cache.maxSize, defaultDedupMaxSize)
	}
}
