package lru

import "testing"

// TestBasicAddAndGet tests basic cache insertion and retrieval.
func TestBasicAddAndGet(t *testing.T) {
	cache := New[string, int](2, nil)
	cache.Add("a", 1)
	cache.Add("b", 2)

	if v, ok := cache.Get("a"); !ok || v != 1 {
		t.Fatalf("expected (1, true), got (%d, %t)", v, ok)
	}
	if v, ok := cache.Get("b"); !ok || v != 2 {
		t.Fatalf("expected (2, true), got (%d, %t)", v, ok)
	}
	if _, ok := cache.Get("c"); ok {
		t.Fatal("expected miss for key 'c'")
	}
}

// TestEviction tests that the least recently used entry is evicted
// when the cache exceeds its capacity.
func TestEviction(t *testing.T) {
	cache := New[string, int](2, nil)
	cache.Add("a", 1)
	cache.Add("b", 2)
	cache.Add("c", 3)

	if _, ok := cache.Get("a"); ok {
		t.Fatal("expected 'a' to be evicted")
	}
	if v, ok := cache.Get("b"); !ok || v != 2 {
		t.Fatalf("expected (2, true), got (%d, %t)", v, ok)
	}
	if v, ok := cache.Get("c"); !ok || v != 3 {
		t.Fatalf("expected (3, true), got (%d, %t)", v, ok)
	}
}

// TestEvictionCallback tests that the eviction callback is called
// with the correct key and value when an entry is evicted.
func TestEvictionCallback(t *testing.T) {
	var evictedKey string
	var evictedValue int
	onEvicted := func(k string, v int) {
		evictedKey = k
		evictedValue = v
	}

	cache := New[string, int](2, onEvicted)
	cache.Add("a", 1)
	cache.Add("b", 2)
	cache.Add("c", 3)

	if evictedKey != "a" || evictedValue != 1 {
		t.Fatalf(
			"expected eviction of (a, 1), got (%s, %d)",
			evictedKey, evictedValue,
		)
	}
}

// TestRecencyPromotion tests that accessing an entry promotes it
// to the front of the cache, preventing its eviction.
func TestRecencyPromotion(t *testing.T) {
	cache := New[string, int](2, nil)
	cache.Add("a", 1)
	cache.Add("b", 2)

	// Access "a" to promote it.
	cache.Get("a")

	// Adding "c" should evict "b" (the least recently used), not
	// "a" (which was just accessed).
	cache.Add("c", 3)

	if _, ok := cache.Get("a"); !ok {
		t.Fatal("expected 'a' to survive (was promoted)")
	}
	if _, ok := cache.Get("b"); ok {
		t.Fatal("expected 'b' to be evicted")
	}
}

// TestUpdateExisting tests that adding an entry with an existing
// key updates the value and promotes the entry.
func TestUpdateExisting(t *testing.T) {
	cache := New[string, int](2, nil)
	cache.Add("a", 1)
	cache.Add("b", 2)

	// Update "a" to a new value.
	cache.Add("a", 10)

	if v, ok := cache.Get("a"); !ok || v != 10 {
		t.Fatalf("expected (10, true), got (%d, %t)", v, ok)
	}

	// "a" was promoted by the update, so "b" should be evicted
	// when a third entry is added.
	cache.Add("c", 3)
	if _, ok := cache.Get("b"); ok {
		t.Fatal("expected 'b' to be evicted")
	}
}

// TestExplicitRemove tests that explicitly removed entries trigger
// the eviction callback.
func TestExplicitRemove(t *testing.T) {
	var evictedKey string
	onEvicted := func(k string, v int) {
		evictedKey = k
	}

	cache := New[string, int](10, onEvicted)
	cache.Add("a", 1)
	cache.Remove("a")

	if evictedKey != "a" {
		t.Fatalf("expected eviction of 'a', got '%s'", evictedKey)
	}
	if _, ok := cache.Get("a"); ok {
		t.Fatal("expected miss after removal")
	}
	if cache.Len() != 0 {
		t.Fatalf("expected length 0, got %d", cache.Len())
	}
}

// TestRemoveNonExistent tests that removing a non-existent key is
// a no-op.
func TestRemoveNonExistent(t *testing.T) {
	cache := New[string, int](10, nil)
	cache.Remove("nonexistent")
}

// TestLen tests the Len method.
func TestLen(t *testing.T) {
	cache := New[string, int](10, nil)
	if cache.Len() != 0 {
		t.Fatalf("expected length 0, got %d", cache.Len())
	}
	cache.Add("a", 1)
	cache.Add("b", 2)
	if cache.Len() != 2 {
		t.Fatalf("expected length 2, got %d", cache.Len())
	}
	cache.Remove("a")
	if cache.Len() != 1 {
		t.Fatalf("expected length 1, got %d", cache.Len())
	}
}

// TestZeroMaxEntries tests that a cache with zero maxEntries has
// no eviction limit.
func TestZeroMaxEntries(t *testing.T) {
	cache := New[string, int](0, nil)
	for i := range 1000 {
		cache.Add(string(rune('a'+i)), i)
	}
	if cache.Len() != 1000 {
		t.Fatalf("expected length 1000, got %d", cache.Len())
	}
}
