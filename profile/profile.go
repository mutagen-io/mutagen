package profile

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/pkg/errors"
)

type Profile struct {
	name       string
	cpuProfile *os.File
}

func New(name string) (*Profile, error) {
	// Open the CPU profile output.
	cpuProfile, err := os.Create(fmt.Sprintf("%s_cpu.prof", name))
	if err != nil {
		return nil, errors.Wrap(err, "unable to create CPU profile")
	}

	// Start CPU profiling.
	if err := pprof.StartCPUProfile(cpuProfile); err != nil {
		cpuProfile.Close()
		return nil, errors.Wrap(err, "unable to start CPU profile")
	}

	// Success.
	return &Profile{
		name:       name,
		cpuProfile: cpuProfile,
	}, nil
}

func (p *Profile) Finalize() error {
	// Close out the CPU profile.
	pprof.StopCPUProfile()
	if err := p.cpuProfile.Close(); err != nil {
		return errors.Wrap(err, "unable to close CPU profile")
	}

	// Run a GC cycle to update the heap profile statistics.
	runtime.GC()

	// Write a heap profile.
	heapProfile, err := os.Create(fmt.Sprintf("%s_heap.prof", p.name))
	if err != nil {
		return errors.Wrap(err, "unable to create heap profile")
	}
	if err := pprof.WriteHeapProfile(heapProfile); err != nil {
		heapProfile.Close()
		return errors.Wrap(err, "unable to write heap profile")
	}
	if err := heapProfile.Close(); err != nil {
		return errors.Wrap(err, "unable to close heap profile")
	}

	// Success.
	return nil
}
