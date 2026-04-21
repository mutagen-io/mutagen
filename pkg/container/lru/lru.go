// Package lru provides a generic LRU (least recently used) cache.
package lru

import "container/list"

// Cache is a generic LRU cache. It is not safe for concurrent
// access. The zero value is not usable; create instances with New.
type Cache[K comparable, V any] struct {
	// maxEntries is the maximum number of entries before eviction.
	// A value of zero means no limit.
	maxEntries int
	// onEvicted is an optional callback invoked when an entry is
	// evicted from the cache (either due to capacity overflow or
	// explicit removal).
	onEvicted func(key K, value V)
	// entries is the doubly-linked list that maintains recency
	// order. The front of the list is the most recently used.
	entries *list.List
	// index maps keys to their corresponding list elements for
	// O(1) lookup.
	index map[K]*list.Element
}

// entry is a key-value pair stored in the linked list.
type entry[K comparable, V any] struct {
	// key is the cache key.
	key K
	// value is the cached value.
	value V
}

// New creates a new LRU cache with the specified maximum number of
// entries. If maxEntries is zero, the cache has no limit and the
// caller is responsible for managing eviction. The onEvicted
// callback, if non-nil, is called when an entry is evicted.
func New[K comparable, V any](
	maxEntries int,
	onEvicted func(key K, value V),
) *Cache[K, V] {
	return &Cache[K, V]{
		maxEntries: maxEntries,
		onEvicted:  onEvicted,
		entries:    list.New(),
		index:      make(map[K]*list.Element),
	}
}

// Add inserts or updates an entry in the cache. If the key already
// exists, its value is updated and the entry is moved to the front
// (most recently used). If the cache is at capacity, the least
// recently used entry is evicted.
func (c *Cache[K, V]) Add(key K, value V) {
	if e, ok := c.index[key]; ok {
		c.entries.MoveToFront(e)
		e.Value.(*entry[K, V]).value = value
		return
	}
	e := c.entries.PushFront(&entry[K, V]{key, value})
	c.index[key] = e
	if c.maxEntries != 0 && c.entries.Len() > c.maxEntries {
		c.removeOldest()
	}
}

// Get retrieves an entry from the cache. If the key is found, the
// entry is moved to the front (most recently used) and the value
// and true are returned. If not found, the zero value and false
// are returned.
func (c *Cache[K, V]) Get(key K) (value V, ok bool) {
	if e, hit := c.index[key]; hit {
		c.entries.MoveToFront(e)
		return e.Value.(*entry[K, V]).value, true
	}
	return
}

// Remove removes the entry with the specified key from the cache.
// If an eviction callback is set, it is called with the removed
// entry. This is a no-op if the key is not present.
func (c *Cache[K, V]) Remove(key K) {
	if e, hit := c.index[key]; hit {
		c.removeElement(e)
	}
}

// Len returns the number of entries in the cache.
func (c *Cache[K, V]) Len() int {
	return c.entries.Len()
}

// removeOldest removes the least recently used entry.
func (c *Cache[K, V]) removeOldest() {
	if e := c.entries.Back(); e != nil {
		c.removeElement(e)
	}
}

// removeElement removes an element from the cache and invokes the
// eviction callback if set.
func (c *Cache[K, V]) removeElement(e *list.Element) {
	c.entries.Remove(e)
	kv := e.Value.(*entry[K, V])
	delete(c.index, kv.key)
	if c.onEvicted != nil {
		c.onEvicted(kv.key, kv.value)
	}
}
