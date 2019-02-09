package sync

import (
	"testing"
)

func TestAnyExecutableBitSet(t *testing.T) {
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
	if !anyExecutableBitSet(0776) {
		t.Error("user executable bits not detected")
	}
	if !anyExecutableBitSet(0677) {
		t.Error("group executable bits not detected")
	}
	if !anyExecutableBitSet(0767) {
		t.Error("others executable bits not detected")
	}
	if !anyExecutableBitSet(0777) {
		t.Error("others executable bits not detected")
	}
}

func TestMarkExecutableForReaders(t *testing.T) {
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
