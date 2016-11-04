package main

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/sync"
)

const (
	cacheFile = "cache_test"
)

var usage = `snapshot_bench [-h|--help] <path>
`

func main() {
	// Parse arguments.
	flagSet := cmd.NewFlagSet("snapshot_bench", usage, []int{1})
	path := flagSet.ParseOrDie(os.Args[1:])[0]

	// Print information.
	fmt.Println("Analyzing", path)

	// Create a snapshot without any cache.
	start := time.Now()
	snapshot, cache, err := sync.Snapshot(path, sha1.New(), nil)
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to create snapshot"))
	}
	stop := time.Now()
	fmt.Println("Cold snapshot took", stop.Sub(start))

	// Create a snapshot with a cache.
	start = time.Now()
	snapshot, _, err = sync.Snapshot(path, sha1.New(), cache)
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to create snapshot"))
	}
	stop = time.Now()
	fmt.Println("Warm snapshot took", stop.Sub(start))

	// Serialize it.
	start = time.Now()
	serializedSnapshot, err := snapshot.Marshal()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to serialize snapshot"))
	}
	stop = time.Now()
	fmt.Println("Snapshot serialization took", stop.Sub(start))

	// Deserialize it.
	start = time.Now()
	deserializedSnapshot := &sync.Entry{}
	if err = deserializedSnapshot.Unmarshal(serializedSnapshot); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to deserialize snapshot"))
	}
	stop = time.Now()
	fmt.Println("Snapshot deserialization took", stop.Sub(start))

	// Print other information.
	fmt.Println("Serialized snapshot size is", len(serializedSnapshot), "bytes")
	fmt.Println(
		"Original/deserialized snapshots equivalent?",
		deserializedSnapshot.Equal(snapshot),
	)

	// Serialize the cache.
	start = time.Now()
	serializedCache, err := cache.Marshal()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to serialize cache"))
	}
	stop = time.Now()
	fmt.Println("Cache serialization took", stop.Sub(start))

	// Deserialize the cache.
	start = time.Now()
	deserializedCache := &sync.Cache{}
	if err = deserializedCache.Unmarshal(serializedCache); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to deserialize cache"))
	}
	stop = time.Now()
	fmt.Println("Cache deserialization took", stop.Sub(start))

	// Write the serialized cache to disk.
	start = time.Now()
	if err = ioutil.WriteFile(cacheFile, serializedCache, 0600); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to write cache"))
	}
	stop = time.Now()
	fmt.Println("Cache write took", stop.Sub(start))

	// Read the serialized cache from disk.
	start = time.Now()
	if _, err = ioutil.ReadFile(cacheFile); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to write cache"))
	}
	stop = time.Now()
	fmt.Println("Cache read took", stop.Sub(start))

	// Wipe the temporary file.
	if err = os.Remove(cacheFile); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to remove cache"))
	}

	// Print other information.
	fmt.Println("Serialized cache size is", len(serializedCache), "bytes")
}
