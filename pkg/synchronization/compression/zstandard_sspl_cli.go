//go:build sspl && cli

package compression

import (
	"github.com/mutagen-io/mutagen/sspl/pkg/licensing"
)

// zstandardSupportStatus returns Zstandard compression support status.
func zstandardSupportStatus() AlgorithmSupportStatus {
	licensed, err := licensing.Check(licensing.ProductIdentifierMutagenPro)
	if licensed || err == licensing.ErrNoLicenseManager {
		return AlgorithmSupportStatusSupported
	}
	return AlgorithmSupportStatusRequiresLicense
}
