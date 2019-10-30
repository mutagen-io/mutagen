package tunneling

import (
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
)

// pathForTunnel computes the path to the serialized tunnel for the given tunnel
// identifier. An empty tunnel identifier will return the tunnels directory
// path.
func pathForTunnel(tunnelID string) (string, error) {
	// Compute/create the sessions directory.
	tunnelsDirectoryPath, err := filesystem.Mutagen(true, filesystem.MutagenTunnelsDirectoryName)
	if err != nil {
		return "", errors.Wrap(err, "unable to compute/create tunnels directory")
	}

	// Success.
	return filepath.Join(tunnelsDirectoryPath, tunnelID), nil
}
