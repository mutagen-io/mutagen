package filesystem

import (
	"os"
)

// DeviceID on Windows is a no-op that returns 0 and never fails. It's not
// necessary for our purposes on Windows since directory hierarchies can't span
// devices.
func DeviceID(_ os.FileInfo) (uint64, error) {
	return 0, nil
}
