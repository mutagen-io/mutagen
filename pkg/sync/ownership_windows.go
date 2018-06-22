package sync

import (
	"os"
)

func getOwnership(_ os.FileInfo) (int, int, error) {
	return 0, 0, nil
}

func setOwnership(_ string, _, _ int) error {
	return nil
}
