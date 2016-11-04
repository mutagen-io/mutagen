package main

import (
	"crypto/sha1"
	"fmt"
	"time"
	"os"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/sync"
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
	serialized, err := snapshot.Marshal()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to serialize snapshot"))
	}
	stop = time.Now()
	fmt.Println("Serialization took", stop.Sub(start))

	// Deserialize it.
	start = time.Now()
	deserialized := &sync.Entry{}
	if err = deserialized.Unmarshal(serialized); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to deserialize snapshot"))
	}
	stop = time.Now()
	fmt.Println("Deserialization took", stop.Sub(start))

	// Print other information.
	fmt.Println("Serialized size was", len(serialized), "bytes")
	fmt.Println(
		"Original/deserialized snapshots equivalent?",
		deserialized.Equal(snapshot),
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

	// Print other information.
	fmt.Println("Serialized cache size was", len(serializedCache), "bytes")
}
