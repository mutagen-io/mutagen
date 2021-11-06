package sidecar

import (
	"os"
	"sync"
)

// environmentIsSidecar is the cached result of the sidecar environment check.
var environmentIsSidecar bool

// checkEnvironmentOnce gates access to environmentIsSidecar.
var checkEnvironmentOnce sync.Once

// EnvironmentIsSidecar returns true if the current operating environment is a
// Mutagen sidecar container.
func EnvironmentIsSidecar() bool {
	checkEnvironmentOnce.Do(func() {
		environmentIsSidecar = os.Getenv("MUTAGEN_SIDECAR") == "1"
	})
	return environmentIsSidecar
}
