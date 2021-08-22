package profile

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
)

// Profile manages a CPU and heap profile.
type Profile struct {
	// name is the name of the profile.
	name string
	// cpuProfile is the output file for the CPU profile.
	cpuProfile *os.File
}

// New creates a new profile instance. The profiling begins immediately.
func New(name string) (*Profile, error) {
	// Open the CPU profile output.
	cpuProfile, err := os.Create(fmt.Sprintf("%s_cpu.prof", name))
	if err != nil {
		return nil, fmt.Errorf("unable to create CPU profile: %w", err)
	}

	// Start CPU profiling.
	if err := pprof.StartCPUProfile(cpuProfile); err != nil {
		cpuProfile.Close()
		return nil, fmt.Errorf("unable to start CPU profile: %w", err)
	}

	// Success.
	return &Profile{
		name:       name,
		cpuProfile: cpuProfile,
	}, nil
}

// Finalize terminates a profile and writes its measurements to disk in the
// current working directory.
func (p *Profile) Finalize() error {
	// Close out the CPU profile.
	pprof.StopCPUProfile()
	if err := p.cpuProfile.Close(); err != nil {
		return fmt.Errorf("unable to close CPU profile: %w", err)
	}

	// Run a GC cycle to update the heap profile statistics.
	runtime.GC()

	// Write a heap profile.
	heapProfile, err := os.Create(fmt.Sprintf("%s_heap.prof", p.name))
	if err != nil {
		return fmt.Errorf("unable to create heap profile: %w", err)
	}
	if err := pprof.WriteHeapProfile(heapProfile); err != nil {
		heapProfile.Close()
		return fmt.Errorf("unable to write heap profile: %w", err)
	}
	if err := heapProfile.Close(); err != nil {
		return fmt.Errorf("unable to close heap profile: %w", err)
	}

	// Success.
	return nil
}
