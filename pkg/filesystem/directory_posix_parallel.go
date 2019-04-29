// +build !windows

package filesystem

import (
	"os"
	"runtime"
)

const (
	// maximumReadContentMetadataWorkers provides an upper-bound on the number
	// of Directory.ReadContentMetadata worker Goroutines. It is chosen somewhat
	// arbitrarily, set high enough to allow parallelism but not so high that
	// performance scaling breaks down or that many-core systems are
	// overwhelmed.
	maximumContentMetadataWorkers = 4
)

// batchReadContentMetadataRequest is a request type encoding a batch of
// Directory.ReadContentMetadata operations to be performed serially.
type batchReadContentMetadataRequest struct {
	// directory is the directory object for which content metadata reads should
	// be performed.
	directory *Directory
	// names are the names of entries in the directory for which metadata should
	// be read.
	names []string
	// results is the pre-allocated storage space into which results should be
	// placed. It must have a length equal to name. In the case of entries that
	// are missing (i.e. those for which the fstatat operation returns an
	// os.IsNotExist error), a nil pointer should be stored.
	results []*Metadata
}

// batchReadContentMetadataResponse is a response type encoding the outcome of a
// batchReadContentMetadataRequest.
type batchReadContentMetadataResponse struct {
	// sawMissingEntry indicates whether or not any of the operations in the
	// batch saw an missing entry and set a corresponding nil pointer in the
	// results.
	sawMissingEntry bool
	// fatalError stores any fatal error (i.e. something other than an
	// os.IsNotExist error) that occurred for the batch. In this case, results
	// should be considered invalid.
	fatalError error
}

// handleBatchReadContentMetadataRequests is the entry point for
// ReadContentMetadata worker Goroutines. It processes requests until the
// requests channel is closed, at which point it closes its responses channel.
func handleBatchReadContentMetadataRequests(
	requests chan batchReadContentMetadataRequest,
	responses chan batchReadContentMetadataResponse,
) {
	// Loop until the requests channel is closed.
	for request := range requests {
		// Track missing entries and errors.
		var sawMissingEntry bool
		var fatalError error

		// Process each name in the request.
		for n, name := range request.names {
			if m, err := request.directory.ReadContentMetadata(name); err != nil {
				if os.IsNotExist(err) {
					request.results[n] = nil
					sawMissingEntry = true
					continue
				} else {
					fatalError = err
					break
				}
			} else {
				request.results[n] = m
			}
		}

		// Send a response.
		responses <- batchReadContentMetadataResponse{
			sawMissingEntry: sawMissingEntry,
			fatalError:      fatalError,
		}
	}

	// Indicate completion by closing our responses channel.
	close(responses)
}

// parallelReadContentMetadataRequest is a request type encoding a batch of
// Directory.ReadContentMetadata operations to be performed in parallel.
type parallelReadContentMetadataRequest struct {
	// directory is the directory object for which content metadata reads should
	// be performed.
	directory *Directory
	// names are the names of entries in the directory for which metadata should
	// be read.
	names []string
}

// parallelReadContentMetadataResponse is a response type encoding the outcome
// of a parallelReadContentMetadataRequest.
type parallelReadContentMetadataResponse struct {
	// results are the results for the request. They may be fewer than the names
	// in the corresponding request since entries which don't exist are ignored.
	results []*Metadata
	// fatalError stores any fatal error (i.e. something other than an
	// os.IsNotExist error) that occurred when processing the request. In this
	// case, results should be considered invalid.
	fatalError error
}

// parallelReadContentMetadataRequests is a queue for parallel
// Directory.ReadContentMetadata operations. Once a write to this channel
// succeeds, the writer knows that it should expect a response on
// parallelReadContentMetadataResponses.
var parallelReadContentMetadataRequests = make(chan parallelReadContentMetadataRequest)

// parallelReadContentMetadataResponses is the result channel for requests
// submitted to parallelReadContentMetadataRequests.
var parallelReadContentMetadataResponses = make(chan parallelReadContentMetadataResponse)

// handleParallelReadContentMetadataRequests is the entry point for parallel
// ReadContentMetadata request handling.
func handleParallelReadContentMetadataRequests() {
	// Compute the worker count.
	workerCount := runtime.NumCPU()
	if workerCount > maximumContentMetadataWorkers {
		workerCount = maximumContentMetadataWorkers
	}

	// Create request/response channels and start workers.
	batchRequestQueues := make([]chan batchReadContentMetadataRequest, workerCount)
	batchResponseQueues := make([]chan batchReadContentMetadataResponse, workerCount)
	for w := 0; w < workerCount; w++ {
		batchRequestQueues[w] = make(chan batchReadContentMetadataRequest, 1)
		batchResponseQueues[w] = make(chan batchReadContentMetadataResponse, 1)
		go handleBatchReadContentMetadataRequests(
			batchRequestQueues[w],
			batchResponseQueues[w],
		)
	}

	// Handle requests indefinitely.
	for request := range parallelReadContentMetadataRequests {
		// Allocate result storage.
		results := make([]*Metadata, len(request.names))

		// Compute the batch size. The additive term in the numerator of this
		// formula ensures that batchSize * workerCount >= len(names), while
		// still giving ~equal distribution across workers. Because of this
		// additive term, we have to check explicitly in the dispatch loop below
		// that we don't exceed the boundaries of the name slice.
		batchSize := (len(request.names) + (workerCount - 1)) / workerCount

		// Dispatch batch requests and track how many workers we use.
		workersUsed := 0
		for batchStart := 0; batchStart < len(request.names); batchStart += batchSize {
			batchStop := batchStart + batchSize
			if batchStop > len(request.names) {
				batchStop = len(request.names)
			}
			batchRequestQueues[workersUsed] <- batchReadContentMetadataRequest{
				directory: request.directory,
				names:     request.names[batchStart:batchStop],
				results:   results[batchStart:batchStop],
			}
			workersUsed++
		}

		// Wait for responses from all of the workers that we used.
		var fatalError error
		var sawMissingEntry bool
		for _, responseQueue := range batchResponseQueues[:workersUsed] {
			response := <-responseQueue
			sawMissingEntry = sawMissingEntry || response.sawMissingEntry
			if response.fatalError != nil && fatalError == nil {
				fatalError = response.fatalError
			}
		}

		// Handle the error case.
		if fatalError != nil {
			parallelReadContentMetadataResponses <- parallelReadContentMetadataResponse{
				fatalError: fatalError,
			}
			continue
		}

		// If missing entries were detected, shift them out of the slice.
		if sawMissingEntry {
			filteredResults := results[:0]
			for _, r := range results {
				if r != nil {
					filteredResults = append(filteredResults, r)
				}
			}
			results = filteredResults
		}

		// Success.
		parallelReadContentMetadataResponses <- parallelReadContentMetadataResponse{
			results: results,
		}
	}
}

// parallelMetadataDisabled tracks whether or not parallel metadata query
// operations have been disabled.
var parallelMetadataDisabled bool

func init() {
	// Disable parallel metadata query operations if this platform isn't
	// whitelisted for support. For now this whitelist is just the set of POSIX
	// platforms for which we run tests, because POSIX doesn't explicitly
	// guarantee that fstatat is safe for concurrent invocation on the same file
	// descriptor and we require confirmation of this behavior through
	// observation. We'll expand this list as we bring more builders online and
	// this code has more shakedown time.
	parallelMetadataDisabled = !(runtime.GOOS == "linux" || runtime.GOOS == "darwin")

	// Disable parallel metadata query operations if we don't have multiple
	// CPUs, because for a single-core system it only adds overhead.
	if runtime.NumCPU() < 2 {
		parallelMetadataDisabled = true
	}

	// If parallel metadata query operations aren't disabled, then start the
	// worker Goroutines.
	if !parallelMetadataDisabled {
		go handleParallelReadContentMetadataRequests()
	}
}
