package sidecar

import (
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/must"
)

// SetVolumeOwnershipAndPermissionsIfEmpty will set the ownership and
// permissions on a sidecar volume if (and only if) the volume is empty.
func SetVolumeOwnershipAndPermissionsIfEmpty(name string, ownership *filesystem.OwnershipSpecification, mode filesystem.Mode, logger *logging.Logger) error {
	// Open the volumes directory and defer its closure.
	volumes, _, err := filesystem.OpenDirectory(volumeMountParent, false, logger)
	if err != nil {
		return fmt.Errorf("unable to open volumes directory: %w", err)
	}
	defer must.Close(volumes, logger)

	// Open the volume mount point and defer its closure.
	volume, err := volumes.OpenDirectory(name, logger)
	if err != nil {
		return fmt.Errorf("unable to open volume root: %w", err)
	}
	defer must.Close(volume, logger)

	// Check if the volume is empty. If not, then we're done.
	if contentNames, err := volume.ReadContentNames(); err != nil {
		return fmt.Errorf("unable to read volume contents: %w", err)
	} else if len(contentNames) != 0 {
		return nil
	}

	// Set permissions on the volume.
	return volumes.SetPermissions(name, ownership, mode)
}
