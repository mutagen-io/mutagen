//go:build sspl

package daemon

import (
	"google.golang.org/grpc"

	"github.com/mutagen-io/mutagen/pkg/logging"

	"github.com/mutagen-io/mutagen/sspl/pkg/licensing"
	licensingsvc "github.com/mutagen-io/mutagen/sspl/pkg/service/licensing"
)

// licenseManager is the Mutagen Pro license manager.
var licenseManager *licensing.Manager

// initializeLicenseManager initializes the Mutagen Pro license manager. If this
// function succeeds, then any existing license information will have been
// loaded by the time it returns. This function should only be called once.
func initializeLicenseManager(logger *logging.Logger) error {
	// Attempt to initialize the license manager. This constructor guarantees
	// that it will check for existing license information before returning.
	if manager, err := licensing.NewManager(logger, licensing.ProductIdentifierMutagenPro, ""); err != nil {
		return err
	} else {
		licenseManager = manager
	}

	// Register the license manager.
	licensing.RegisterManager(licenseManager)

	// Success.
	return nil
}

// shutdownLicenseManager gracefully terminates the licensing manager. This
// function should only be called if initializeLicenseManager succeeds.
func shutdownLicenseManager() {
	licenseManager.Shutdown()
}

// registerLicensingService registers the licensing service with the gRPC
// server. This function should only be called if initializeLicenseManager
// succeeds.
func registerLicensingService(server *grpc.Server) {
	licensingsvc.RegisterLicensingServer(server, licensingsvc.NewServer(licenseManager))
}
