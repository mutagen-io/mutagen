package url

import (
	"testing"
)

func TestIsWindowsPathLowercase(t *testing.T) {
	if !isWindowsPath(`c:\something`) {
		t.Error("Windows path not classified as such")
	}
}

func TestIsWindowsPathLowercaseForwardSlash(t *testing.T) {
	if !isWindowsPath(`c:/something`) {
		t.Error("Windows path not classified as such")
	}
}

func TestIsWindowsPath(t *testing.T) {
	if !isWindowsPath(`C:\something`) {
		t.Error("Windows path not classified as such")
	}
}

func TestIsWindowsPathForwardSlash(t *testing.T) {
	if !isWindowsPath(`C:/something`) {
		t.Error("Windows path not classified as such")
	}
}

func TestIsWindowsPathLengthTwoDrive(t *testing.T) {
	if isWindowsPath(`CD:\something`) {
		t.Error("non-Windows path classified as such")
	}
}
