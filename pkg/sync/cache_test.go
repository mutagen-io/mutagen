package sync

import (
	"testing"

	"github.com/golang/protobuf/ptypes"
)

func TestCacheNilInvalid(t *testing.T) {
	var cache *Cache
	if cache.EnsureValid() == nil {
		t.Error("nil cache considered valid")
	}
}

func TestCacheNilEntryInvalid(t *testing.T) {
	cache := &Cache{Entries: make(map[string]*CacheEntry)}
	cache.Entries["name"] = nil
	if cache.EnsureValid() == nil {
		t.Error("cache containing nil entry considered valid")
	}
}

func TestCacheEntryNilTimeInvalid(t *testing.T) {
	cache := &Cache{Entries: make(map[string]*CacheEntry)}
	cache.Entries["name"] = &CacheEntry{}
	if cache.EnsureValid() == nil {
		t.Error("cache containing entry with nil timestamp considered valid")
	}
}

func TestCacheEmptyValid(t *testing.T) {
	cache := &Cache{Entries: make(map[string]*CacheEntry)}
	if err := cache.EnsureValid(); err != nil {
		t.Error("empty cache failed validation:", err)
	}
}

func TestCacheValid(t *testing.T) {
	cache := &Cache{Entries: make(map[string]*CacheEntry)}
	cache.Entries["name"] = &CacheEntry{
		Mode:             0600,
		ModificationTime: ptypes.TimestampNow(),
		Size:             100,
		Digest:           []byte{0, 1, 2, 3, 4, 5, 6},
	}
	if err := cache.EnsureValid(); err != nil {
		t.Error("valid cache failed validation:", err)
	}
}

// TODO: Add tests for Cache.Equal, even though this is an internal testing
// method.
