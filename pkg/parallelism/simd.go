package parallelism

import (
	"runtime"
	"sync"
)

// SIMDWork is the interface for SIMD workloads.
type SIMDWork interface {
	// Do is invoked by SIMD worker Goroutines. It provides the index of the
	// Goroutine in the worker array and the size of the array.
	Do(index, size int) error
}

// SIMDWorkerArray encapsulates an array of worker Goroutines that can perform
// SIMD-style workloads.
type SIMDWorkerArray struct {
	// lock serializes access to the worker array.
	lock sync.Mutex
	// size is the array size.
	size int
	// terminated tracks whether or not the array has been terminated.
	terminated bool
	// submit is a slice of channels used to submit workloads to workers. These
	// channels are closed to signal termination.
	submit []chan SIMDWork
	// errors is a slice of channels used to track completion of workloads.
	// These channels are closed once worker Goroutines have exited.
	errors []chan error
}

// NewSIMDWorkerArray creates a new worker array. If size is zero or negative, a
// size corresponding to the number of system CPUs is created.
func NewSIMDWorkerArray(size int) *SIMDWorkerArray {
	// Handle the case of a default size.
	if size < 1 {
		size = runtime.NumCPU()
		if size < 1 {
			panic("invalid number of CPUs")
		}
	}

	// Create the array.
	array := &SIMDWorkerArray{
		size:   size,
		submit: make([]chan SIMDWork, size),
		errors: make([]chan error, size),
	}

	// Create the communication channels and start the worker Goroutines.
	for i := 0; i < size; i++ {
		array.submit[i] = make(chan SIMDWork)
		array.errors[i] = make(chan error)
		go array.work(i)
	}

	// Done.
	return array
}

// work is the work loop for worker Goroutines.
func (a *SIMDWorkerArray) work(index int) {
	// Loop and perform work until termination.
	for work := range a.submit[index] {
		a.errors[index] <- work.Do(index, a.size)
	}

	// Signal completion.
	close(a.errors[index])
}

// Do performs SIMD-style work with the array. This method is safe for
// concurrent invocation by multiple Goroutines (though their workloads will be
// serialized). It must not be called concurrently with or after Terminate. It
// returns the first non-nil error returned by the workload, if any.
func (a *SIMDWorkerArray) Do(work SIMDWork) error {
	// Lock the array and defer its release.
	a.lock.Lock()
	defer a.lock.Unlock()

	// If the array has been terminated, then the caller is in error.
	if a.terminated {
		panic("work submitted to terminated array")
	}

	// Submit the work to the worker Goroutines.
	for i := 0; i < a.size; i++ {
		a.submit[i] <- work
	}

	// Wait for the worker Goroutines to complete their work.
	var firstErr error
	for i := 0; i < a.size; i++ {
		if err := <-a.errors[i]; err != nil && firstErr == nil {
			firstErr = err
		}
	}

	// Done.
	return firstErr
}

// Terminate terminates the array's workers.
func (a *SIMDWorkerArray) Terminate() {
	// Lock the array and defer its release.
	a.lock.Lock()
	defer a.lock.Unlock()

	// Terminate the array's workers and wait for them to complete.
	for i := 0; i < a.size; i++ {
		close(a.submit[i])
		<-a.errors[i]
	}

	// Mark the array as terminated.
	a.terminated = true
}
