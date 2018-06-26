package filesystem

import (
	"os"
)

func GetOwnership(_ os.FileInfo) (int, int, error) {
	return 0, 0, nil
}

func SetOwnership(_ string, _, _ int) error {
	return nil
}
