package sidecar

import (
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
)

// SetVolumeOwnershipAndPermissionsIfEmpty will set the ownership and
// permissions on a sidecar volume if (and only if) the volume is empty.
func SetVolumeOwnershipAndPermissionsIfEmpty(name string, ownership *filesystem.OwnershipSpecification, mode filesystem.Mode) error {
	// Open the volumes directory and defer its closure.
	volumes, _, err := filesystem.OpenDirectory(volumeMountParent, false)
	if err != nil {
		return fmt.Errorf("unable to open volumes directory: %w", err)
	}
	defer volumes.Close()

	// Open the volume mount point and defer its closure.
	volume, err := volumes.OpenDirectory(name)
	if err != nil {
		return fmt.Errorf("unable to open volume root: %w", err)
	}
	defer volume.Close()

	// Check if the volume is empty. If not, then we're done.
	if contentNames, err := volume.ReadContentNames(); err != nil {
		return fmt.Errorf("unable to read volume contents: %w", err)
	} else if len(contentNames) != 0 {
		return nil
	}

	// Set permissions on the volume.
	return volumes.SetPermissions(name, ownership, mode)
}
