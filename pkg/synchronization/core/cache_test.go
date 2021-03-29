package core

import (
	"math"
	"testing"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// testInvalidProtocolBuffersTimestamp is an invalid Protocol Buffers timestamp.
var testInvalidProtocolBuffersTimestamp = &timestamppb.Timestamp{
	Seconds: math.MinInt64,
}

// TestCacheEnsureValid tests Cache.EnsureValid.
func TestCacheEnsureValid(t *testing.T) {
	// Define test cases.
	tests := []struct {
		cache    *Cache
		expected bool
	}{
		{nil, false},
		{&Cache{Entries: map[string]*CacheEntry{"name": nil}}, false},
		{&Cache{Entries: map[string]*CacheEntry{"name": {}}}, false},
		{&Cache{}, true},
		{&Cache{Entries: map[string]*CacheEntry{
			"": {
				Mode:             0600,
				ModificationTime: timestamppb.Now(),
				Size:             uint64(len(tF1Content)),
				Digest:           tF1.Digest,
			},
		}}, true},
		{&Cache{Entries: map[string]*CacheEntry{
			"file": {
				Mode:             0600,
				ModificationTime: timestamppb.Now(),
				Size:             uint64(len(tF1Content)),
				Digest:           tF1.Digest,
			},
		}}, true},
		{&Cache{Entries: map[string]*CacheEntry{
			"file": {
				Mode:             0600,
				ModificationTime: testInvalidProtocolBuffersTimestamp,
				Size:             uint64(len(tF1Content)),
				Digest:           tF1.Digest,
			},
		}}, false},
	}

	// Process test cases.
	for i, test := range tests {
		err := test.cache.EnsureValid()
		valid := err == nil
		if valid != test.expected {
			if valid {
				t.Errorf("test index %d: cache incorrectly classified as valid", i)
			} else {
				t.Errorf("test index %d: cache incorrectly classified as invalid: %v", i, err)
			}
		}
	}
}

// TODO: Implement TestCacheEqual. This is purely an internal testing method,
// but it's worth testing for completeness.

// TODO: Implement TestReverseLookupMap.
