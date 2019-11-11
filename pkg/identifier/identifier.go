package identifier

import (
	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/random"
)

const (
	// PrefixSynchronization is the prefix used for synchronization session
	// identifiers.
	PrefixSynchronization = "sync_"
	// PrefixForwarding is the prefix used for forwarding session identifiers.
	PrefixForwarding = "fwrd_"
	// PrefixProject is the prefix used for project identifiers.
	PrefixProject = "proj_"
)

// New generates a new collision-resistant identifier with the specified prefix.
func New(prefix string) (string, error) {
	// Create the random value.
	random, err := random.New(random.CollisionResistantLength)
	if err != nil {
		return "", err
	}

	// Encode the random value.
	return prefix + encoding.EncodeBase62(random), nil
}
