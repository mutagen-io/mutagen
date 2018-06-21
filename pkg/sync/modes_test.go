package sync

import (
	"testing"
)

func TestAnyExecutableBitSet(t *testing.T) {
	if AnyExecutableBitSet(0666) {
		t.Error("executable bits detected")
	}
	if !AnyExecutableBitSet(0766) {
		t.Error("user executable bit not detected")
	}
	if !AnyExecutableBitSet(0676) {
		t.Error("group executable bit not detected")
	}
	if !AnyExecutableBitSet(0667) {
		t.Error("others executable bit not detected")
	}
}

func TestStripExecutableBits(t *testing.T) {
	if StripExecutableBits(0777) != 0666 {
		t.Error("executable bits not stripped")
	}
	if StripExecutableBits(0766) != 0666 {
		t.Error("user executable bit not stripped")
	}
	if StripExecutableBits(0676) != 0666 {
		t.Error("group executable bit not stripped")
	}
	if StripExecutableBits(0667) != 0666 {
		t.Error("others executable bit not stripped")
	}
}

func TestMarkExecutableForReaders(t *testing.T) {
	if MarkExecutableForReaders(0222) != 0222 {
		t.Error("erroneous executable bits added")
	}
	if MarkExecutableForReaders(0622) != 0722 {
		t.Error("incorrect executable bits added for user-readable file")
	}
	if MarkExecutableForReaders(0262) != 0272 {
		t.Error("incorrect executable bits added for group-readable file")
	}
	if MarkExecutableForReaders(0226) != 0227 {
		t.Error("incorrect executable bits added for others-readable file")
	}
}
