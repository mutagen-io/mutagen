//go:build sspl && cli

package hashing

import (
	"github.com/mutagen-io/mutagen/sspl/pkg/licensing"
)

// xxh128SupportStatus returns XXH128 hashing support status.
func xxh128SupportStatus() AlgorithmSupportStatus {
	licensed, err := licensing.Check(licensing.ProductIdentifierMutagenPro)
	if licensed || err == licensing.ErrNoLicenseManager {
		return AlgorithmSupportStatusSupported
	}
	return AlgorithmSupportStatusRequiresLicense
}
