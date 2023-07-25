package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/pflag"

	"google.golang.org/protobuf/proto"

	"github.com/dustin/go-humanize"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/cmd/profile"

	"github.com/mutagen-io/mutagen/pkg/filesystem/behavior"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core/ignore"
	dockerignore "github.com/mutagen-io/mutagen/pkg/synchronization/core/ignore/docker"
	mutagenignore "github.com/mutagen-io/mutagen/pkg/synchronization/core/ignore/mutagen"
	"github.com/mutagen-io/mutagen/pkg/synchronization/hashing"
)

const (
	snapshotFile = "snapshot_test"
	cacheFile    = "cache_test"
)

const usage = `scan_bench [-h|--help] [-p|--profile] [-d|--digest=(` + digestFlagOptions + `)]
           [--ignore-syntax=(mutagen|docker)] [-i|--ignore=<pattern>] <path>
`

// ignoreCachesIntersectionEqual compares two ignore caches, ensuring that keys
// which are present in both caches have the same value. It's the closest we can
// get to the core package's testAcceleratedCacheIsSubset without having access
// to the members of IgnoreCacheKey.
func ignoreCachesIntersectionEqual(first, second ignore.IgnoreCache) bool {
	// Check matches from first in second.
	for key, firstValue := range first {
		if secondValue, ok := second[key]; ok && secondValue != firstValue {
			return false
		}
	}

	// Check matches from second in first.
	for key, secondValue := range second {
		if firstValue, ok := first[key]; ok && firstValue != secondValue {
			return false
		}
	}

	// Success.
	return true
}

func main() {
	// Parse command line arguments.
	// TODO: Implement support for ignore syntax specification.
	flagSet := pflag.NewFlagSet("scan_bench", pflag.ContinueOnError)
	flagSet.SetOutput(io.Discard)
	var enableProfile bool
	var digest string
	var ignoreSyntaxName string
	var ignores []string
	flagSet.BoolVarP(&enableProfile, "profile", "p", false, "enable profiling")
	flagSet.StringVarP(&digest, "digest", "d", "sha1", "specify digest algorithm")
	flagSet.StringVar(&ignoreSyntaxName, "ignore-syntax", "mutagen", "specify ignore syntax")
	flagSet.StringSliceVarP(&ignores, "ignore", "i", nil, "specify ignore paths")
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		if err == pflag.ErrHelp {
			fmt.Fprint(os.Stdout, usage)
			return
		} else {
			cmd.Fatal(fmt.Errorf("unable to parse command line: %w", err))
		}
	}
	arguments := flagSet.Args()
	if len(arguments) != 1 {
		cmd.Fatal(errors.New("invalid number of paths specified"))
	}
	path := arguments[0]

	// Create a context for the scan. The main reason for using a custom context
	// instead of using context.Background() is that the latter provides a
	// context that returns a nil result from Done(). To ensure that we fully
	// understand the impact of preemption checks, we want a context that will
	// return a non-nil completion channel. This also allows us to wire up
	// interrupt handling to this context.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Wire up termination signals to context cancellation.
	signalTermination := make(chan os.Signal, 1)
	signal.Notify(signalTermination, cmd.TerminationSignals...)
	go func() {
		<-signalTermination
		cancel()
	}()

	// Parse the hashing algorithm and create a hasher.
	var hashingAlgorithm hashing.Algorithm
	if err := hashingAlgorithm.UnmarshalText([]byte(digest)); err != nil {
		cmd.Fatal(fmt.Errorf("unable to parse hashing algorithm: %w", err))
	} else if hashingAlgorithm.SupportStatus() != hashing.AlgorithmSupportStatusSupported {
		cmd.Fatal(fmt.Errorf("%s hashing not supported", hashingAlgorithm.Description()))
	}
	hasher := hashingAlgorithm.Factory()()

	// Parse the ignore syntax.
	var ignoreSyntax ignore.Syntax
	if err := ignoreSyntax.UnmarshalText([]byte(ignoreSyntaxName)); err != nil {
		cmd.Fatal(fmt.Errorf("unable to parse ignore syntax: %w", err))
	}

	// Create an ignorer.
	var ignorer ignore.Ignorer
	if ignoreSyntax == ignore.Syntax_SyntaxMutagen {
		if i, err := mutagenignore.NewIgnorer(ignores); err != nil {
			cmd.Fatal(fmt.Errorf("unable to create Mutagen-style ignorer: %w", err))
		} else {
			ignorer = i
		}
	} else if ignoreSyntax == ignore.Syntax_SyntaxDocker {
		if i, err := dockerignore.NewIgnorer(ignores); err != nil {
			cmd.Fatal(fmt.Errorf("unable to create Docker-style ignorer: %w", err))
		} else {
			ignorer = i
		}
	} else {
		panic("unhandled ignore syntax")
	}

	// Print information.
	fmt.Println("Analyzing", path)

	// Perform a full (cold) scan. If requested, enable CPU and memory
	// profiling.
	var profiler *profile.Profile
	if enableProfile {
		if p, err := profile.New("scan_full_cold"); err != nil {
			cmd.Fatal(fmt.Errorf("unable to create profiler: %w", err))
		} else {
			profiler = p
		}
	}
	start := time.Now()
	snapshot, cache, ignoreCache, err := core.Scan(
		ctx,
		path,
		nil, nil,
		hasher, nil,
		ignorer, nil,
		behavior.ProbeMode_ProbeModeProbe,
		core.SymbolicLinkMode_SymbolicLinkModePortable,
		core.PermissionsMode_PermissionsModePortable,
	)
	if err != nil {
		cmd.Fatal(fmt.Errorf("unable to perform cold scan: %w", err))
	}
	if snapshot.Content == nil {
		fmt.Println("No content at the specified path!")
	}
	stop := time.Now()
	if enableProfile {
		if err = profiler.Finalize(); err != nil {
			cmd.Fatal(fmt.Errorf("unable to finalize profiler: %w", err))
		}
		profiler = nil
	}
	fmt.Println("Cold scan took", stop.Sub(start))
	fmt.Println("Root preserves executability:", snapshot.PreservesExecutability)
	fmt.Println("Root requires Unicode recomposition:", snapshot.DecomposesUnicode)

	// Perform a full (warm) scan. If requested, enable CPU and memory
	// profiling.
	if enableProfile {
		if profiler, err = profile.New("scan_full_warm"); err != nil {
			cmd.Fatal(fmt.Errorf("unable to create profiler: %w", err))
		}
	}
	start = time.Now()
	newSnapshot, newCache, newIgnoreCache, err := core.Scan(
		ctx,
		path,
		nil, nil,
		hasher, cache,
		ignorer, ignoreCache,
		behavior.ProbeMode_ProbeModeProbe,
		core.SymbolicLinkMode_SymbolicLinkModePortable,
		core.PermissionsMode_PermissionsModePortable,
	)
	if err != nil {
		cmd.Fatal(fmt.Errorf("unable to perform warm scan: %w", err))
	}
	stop = time.Now()
	if enableProfile {
		if err = profiler.Finalize(); err != nil {
			cmd.Fatal(fmt.Errorf("unable to finalize profiler: %w", err))
		}
		profiler = nil
	}
	fmt.Println("Warm scan took", stop.Sub(start))

	// Compare the warm scan results with the baseline results.
	if !newSnapshot.Equal(snapshot) {
		cmd.Fatal(errors.New("snapshot mismatch"))
	} else if !newCache.Equal(cache) {
		cmd.Fatal(errors.New("cache mismatch"))
	} else if len(newIgnoreCache) != len(ignoreCache) {
		cmd.Fatal(errors.New("ignore cache length mismatch"))
	} else if !ignoreCachesIntersectionEqual(newIgnoreCache, ignoreCache) {
		cmd.Fatal(errors.New("ignore cache mismatch"))
	}

	// Perform a second full (warm) scan. If requested, enable CPU and memory
	// profiling. A second warm scan will provide a more accurate real-world
	// assessment of performance because the cold scan won't have wiped out the
	// majority of the filesystem caches.
	if enableProfile {
		if profiler, err = profile.New("scan_second_full_warm"); err != nil {
			cmd.Fatal(fmt.Errorf("unable to create profiler: %w", err))
		}
	}
	start = time.Now()
	newSnapshot, newCache, newIgnoreCache, err = core.Scan(
		ctx,
		path,
		nil, nil,
		hasher, cache,
		ignorer, ignoreCache,
		behavior.ProbeMode_ProbeModeProbe,
		core.SymbolicLinkMode_SymbolicLinkModePortable,
		core.PermissionsMode_PermissionsModePortable,
	)
	if err != nil {
		cmd.Fatal(fmt.Errorf("unable to perform second warm scan: %w", err))
	}
	stop = time.Now()
	if enableProfile {
		if err = profiler.Finalize(); err != nil {
			cmd.Fatal(fmt.Errorf("unable to finalize profiler: %w", err))
		}
		profiler = nil
	}
	fmt.Println("Second warm scan took", stop.Sub(start))

	// Compare the warm scan results with the baseline results.
	if !newSnapshot.Equal(snapshot) {
		cmd.Fatal(errors.New("snapshot mismatch"))
	} else if !newCache.Equal(cache) {
		cmd.Fatal(errors.New("cache mismatch"))
	} else if len(newIgnoreCache) != len(ignoreCache) {
		cmd.Fatal(errors.New("ignore cache length mismatch"))
	} else if !ignoreCachesIntersectionEqual(newIgnoreCache, ignoreCache) {
		cmd.Fatal(errors.New("ignore cache mismatch"))
	}

	// Perform an accelerated scan (with a re-check path). If requested, enable
	// CPU and memory profiling.
	if enableProfile {
		if profiler, err = profile.New("scan_accelerated_recheck"); err != nil {
			cmd.Fatal(fmt.Errorf("unable to create profiler: %w", err))
		}
	}
	start = time.Now()
	newSnapshot, newCache, newIgnoreCache, err = core.Scan(
		ctx,
		path,
		snapshot, map[string]bool{"fake path": true},
		hasher, cache,
		ignorer, ignoreCache,
		behavior.ProbeMode_ProbeModeProbe,
		core.SymbolicLinkMode_SymbolicLinkModePortable,
		core.PermissionsMode_PermissionsModePortable,
	)
	if err != nil {
		cmd.Fatal(fmt.Errorf("unable to perform accelerated scan (with re-check paths): %w", err))
	}
	stop = time.Now()
	if enableProfile {
		if err = profiler.Finalize(); err != nil {
			cmd.Fatal(fmt.Errorf("unable to finalize profiler: %w", err))
		}
		profiler = nil
	}
	fmt.Println("Accelerated scan (with re-check paths) took", stop.Sub(start))

	// Compare the accelerated scan results with the baseline results.
	if !newSnapshot.Equal(snapshot) {
		cmd.Fatal(errors.New("snapshot mismatch"))
	} else if !newCache.Equal(cache) {
		cmd.Fatal(errors.New("cache mismatch"))
	} else if !ignoreCachesIntersectionEqual(newIgnoreCache, ignoreCache) {
		cmd.Fatal(errors.New("ignore cache mismatch"))
	}

	// Perform an accelerated scan (without any re-check paths). If requested,
	// enable CPU and memory profiling.
	if enableProfile {
		if profiler, err = profile.New("scan_accelerated_no_recheck"); err != nil {
			cmd.Fatal(fmt.Errorf("unable to create profiler: %w", err))
		}
	}
	start = time.Now()
	newSnapshot, newCache, newIgnoreCache, err = core.Scan(
		ctx,
		path,
		snapshot, nil,
		hasher, cache,
		ignorer, ignoreCache,
		behavior.ProbeMode_ProbeModeProbe,
		core.SymbolicLinkMode_SymbolicLinkModePortable,
		core.PermissionsMode_PermissionsModePortable,
	)
	if err != nil {
		cmd.Fatal(fmt.Errorf("unable to perform accelerated scan (without re-check paths): %w", err))
	}
	stop = time.Now()
	if enableProfile {
		if err = profiler.Finalize(); err != nil {
			cmd.Fatal(fmt.Errorf("unable to finalize profiler: %w", err))
		}
		profiler = nil
	}
	fmt.Println("Accelerated scan (without re-check paths) took", stop.Sub(start))

	// Compare the accelerated scan results with the baseline results.
	if !newSnapshot.Equal(snapshot) {
		cmd.Fatal(errors.New("snapshot mismatch"))
	} else if !newCache.Equal(cache) {
		cmd.Fatal(errors.New("cache mismatch"))
	} else if !ignoreCachesIntersectionEqual(newIgnoreCache, ignoreCache) {
		cmd.Fatal(errors.New("ignore cache mismatch"))
	}

	// Validate the snapshot.
	start = time.Now()
	if err := snapshot.EnsureValid(); err != nil {
		cmd.Fatal(fmt.Errorf("invalid snapshot: %w", err))
	}
	stop = time.Now()
	fmt.Println("Snapshot validation took", stop.Sub(start))

	// Count snapshot content entries.
	start = time.Now()
	entryCount := snapshot.Content.Count()
	stop = time.Now()
	fmt.Println("Snapshot entry counting took", stop.Sub(start))
	fmt.Println("Snapshot contained", entryCount, "entries")

	// Print snapshot statistics.
	fmt.Println("Snapshot contained", snapshot.Directories, "directories")
	fmt.Println("Snapshot contained", snapshot.Files, "files")
	fmt.Println("Snapshot contained", snapshot.SymbolicLinks, "symbolic links")
	fmt.Println("Snapshot files totaled", humanize.Bytes(snapshot.TotalFileSize))

	// Measure how long phantom reification would take on snapshots of this size
	// and print reified directory counts for comparison.
	start = time.Now()
	_, _, αDirectoryCount, βDirectoryCount := core.ReifyPhantomDirectories(
		snapshot.Content, snapshot.Content, snapshot.Content,
	)
	stop = time.Now()
	fmt.Println("Phantom directory reification took", stop.Sub(start))
	fmt.Println("Reified alpha snapshot contained", αDirectoryCount, "directories")
	fmt.Println("Reified beta snapshot contained", βDirectoryCount, "directories")

	// Perform a deep copy of the snapshot contents.
	start = time.Now()
	snapshot.Content.Copy(core.EntryCopyBehaviorDeep)
	stop = time.Now()
	fmt.Println("Snapshot entry copying took", stop.Sub(start))

	// Serialize the snapshot.
	if enableProfile {
		if profiler, err = profile.New("serialize_snapshot"); err != nil {
			cmd.Fatal(fmt.Errorf("unable to create profiler: %w", err))
		}
	}
	start = time.Now()
	marshaling := proto.MarshalOptions{Deterministic: true}
	serializedSnapshot, err := marshaling.Marshal(snapshot)
	if err != nil {
		cmd.Fatal(fmt.Errorf("unable to serialize snapshot: %w", err))
	}
	stop = time.Now()
	if enableProfile {
		if err = profiler.Finalize(); err != nil {
			cmd.Fatal(fmt.Errorf("unable to finalize profiler: %w", err))
		}
		profiler = nil
	}
	fmt.Println("Snapshot serialization took", stop.Sub(start))

	// Deserialize the snapshot.
	if enableProfile {
		if profiler, err = profile.New("deserialize_snapshot"); err != nil {
			cmd.Fatal(fmt.Errorf("unable to create profiler: %w", err))
		}
	}
	start = time.Now()
	deserializedSnapshot := &core.Snapshot{}
	if err = proto.Unmarshal(serializedSnapshot, deserializedSnapshot); err != nil {
		cmd.Fatal(fmt.Errorf("unable to deserialize snapshot: %w", err))
	}
	stop = time.Now()
	if enableProfile {
		if err = profiler.Finalize(); err != nil {
			cmd.Fatal(fmt.Errorf("unable to finalize profiler: %w", err))
		}
		profiler = nil
	}
	fmt.Println("Snapshot deserialization took", stop.Sub(start))

	// Validate the deserialized snapshot.
	start = time.Now()
	if err = deserializedSnapshot.EnsureValid(); err != nil {
		cmd.Fatal(fmt.Errorf("deserialized snapshot invalid: %w", err))
	}
	stop = time.Now()
	fmt.Println("Snapshot validation took", stop.Sub(start))

	// Write the serialized snapshot to disk.
	start = time.Now()
	if err = os.WriteFile(snapshotFile, serializedSnapshot, 0600); err != nil {
		cmd.Fatal(fmt.Errorf("unable to write snapshot to disk: %w", err))
	}
	stop = time.Now()
	fmt.Println("Snapshot write took", stop.Sub(start))

	// Read the serialized snapshot from disk.
	start = time.Now()
	if _, err = os.ReadFile(snapshotFile); err != nil {
		cmd.Fatal(fmt.Errorf("unable to read snapshot from disk: %w", err))
	}
	stop = time.Now()
	fmt.Println("Snapshot read took", stop.Sub(start))

	// Remove the temporary file.
	if err = os.Remove(snapshotFile); err != nil {
		cmd.Fatal(fmt.Errorf("unable to remove snapshot from disk: %w", err))
	}

	// Print serialized snapshot size.
	fmt.Println("Serialized snapshot size is", len(serializedSnapshot), "bytes")

	// Print whether or not snapshots are equivalent.
	fmt.Println("Original/deserialized snapshots equivalent?", deserializedSnapshot.Equal(snapshot))

	// Checksum it.
	start = time.Now()
	hasher.Reset()
	hasher.Write(serializedSnapshot)
	hasher.Sum(nil)
	stop = time.Now()
	fmt.Println("Snapshot digest took", stop.Sub(start))

	// Serialize the cache.
	if enableProfile {
		if profiler, err = profile.New("serialize_cache"); err != nil {
			cmd.Fatal(fmt.Errorf("unable to create profiler: %w", err))
		}
	}
	start = time.Now()
	serializedCache, err := proto.Marshal(cache)
	if err != nil {
		cmd.Fatal(fmt.Errorf("unable to serialize cache: %w", err))
	}
	stop = time.Now()
	if enableProfile {
		if err = profiler.Finalize(); err != nil {
			cmd.Fatal(fmt.Errorf("unable to finalize profiler: %w", err))
		}
		profiler = nil
	}
	fmt.Println("Cache serialization took", stop.Sub(start))

	// Deserialize the cache.
	if enableProfile {
		if profiler, err = profile.New("deserialize_cache"); err != nil {
			cmd.Fatal(fmt.Errorf("unable to create profiler: %w", err))
		}
	}
	start = time.Now()
	deserializedCache := &core.Cache{}
	if err = proto.Unmarshal(serializedCache, deserializedCache); err != nil {
		cmd.Fatal(fmt.Errorf("unable to deserialize cache: %w", err))
	}
	stop = time.Now()
	if enableProfile {
		if err = profiler.Finalize(); err != nil {
			cmd.Fatal(fmt.Errorf("unable to finalize profiler: %w", err))
		}
		profiler = nil
	}
	fmt.Println("Cache deserialization took", stop.Sub(start))

	// Write the serialized cache to disk.
	start = time.Now()
	if err = os.WriteFile(cacheFile, serializedCache, 0600); err != nil {
		cmd.Fatal(fmt.Errorf("unable to write cache to disk: %w", err))
	}
	stop = time.Now()
	fmt.Println("Cache write took", stop.Sub(start))

	// Read the serialized cache from disk.
	start = time.Now()
	if _, err = os.ReadFile(cacheFile); err != nil {
		cmd.Fatal(fmt.Errorf("unable to read cache from disk: %w", err))
	}
	stop = time.Now()
	fmt.Println("Cache read took", stop.Sub(start))

	// Remove the temporary file.
	if err = os.Remove(cacheFile); err != nil {
		cmd.Fatal(fmt.Errorf("unable to remove cache from disk: %w", err))
	}

	// Print serialized cache size.
	fmt.Println("Serialized cache size is", len(serializedCache), "bytes")

	// Generate a reverse lookup map for the cache.
	start = time.Now()
	if _, err = cache.GenerateReverseLookupMap(); err != nil {
		cmd.Fatal(fmt.Errorf("unable to generate reverse lookup map: %w", err))
	}
	stop = time.Now()
	fmt.Println("Reverse lookup map generation took", stop.Sub(start))
}
