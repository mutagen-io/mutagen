//go:build !sspl

package daemon

import (
	"google.golang.org/grpc"

	"github.com/mutagen-io/mutagen/pkg/logging"
)

// initializeLicenseManager initializes the Mutagen Pro license manager. If this
// function succeeds, then any existing license information will have been
// loaded by the time it returns. This function should only be called once. In
// non-SSPL builds, this function is a no-op.
func initializeLicenseManager(logger *logging.Logger) error {
	logger.Info("No licensing infrastructure present")
	return nil
}

// shutdownLicenseManager gracefully terminates the licensing manager. This
// function should only be called if initializeLicenseManager succeeds. In
// non-SSPL builds, this function is a no-op.
func shutdownLicenseManager() {}

// registerLicensingService registers the licensing service with the gRPC
// server. This function should only be called if initializeLicenseManager
// succeeds. In non-SSPL builds, this function is a no-op.
func registerLicensingService(_ *grpc.Server) {}
