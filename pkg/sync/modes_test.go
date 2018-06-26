package sync

import (
	"testing"
)

func TestanyExecutableBitSet(t *testing.T) {
	if anyExecutableBitSet(0666) {
		t.Error("executable bits detected")
	}
	if !anyExecutableBitSet(0766) {
		t.Error("user executable bit not detected")
	}
	if !anyExecutableBitSet(0676) {
		t.Error("group executable bit not detected")
	}
	if !anyExecutableBitSet(0667) {
		t.Error("others executable bit not detected")
	}
}

func TeststripExecutableBits(t *testing.T) {
	if stripExecutableBits(0777) != 0666 {
		t.Error("executable bits not stripped")
	}
	if stripExecutableBits(0766) != 0666 {
		t.Error("user executable bit not stripped")
	}
	if stripExecutableBits(0676) != 0666 {
		t.Error("group executable bit not stripped")
	}
	if stripExecutableBits(0667) != 0666 {
		t.Error("others executable bit not stripped")
	}
}

func TestmarkExecutableForReaders(t *testing.T) {
	if markExecutableForReaders(0222) != 0222 {
		t.Error("erroneous executable bits added")
	}
	if markExecutableForReaders(0622) != 0722 {
		t.Error("incorrect executable bits added for user-readable file")
	}
	if markExecutableForReaders(0262) != 0272 {
		t.Error("incorrect executable bits added for group-readable file")
	}
	if markExecutableForReaders(0226) != 0227 {
		t.Error("incorrect executable bits added for others-readable file")
	}
}
