package core

import (
	"bytes"
	"testing"

	"github.com/mutagen-io/mutagen/pkg/comparison"
)

// TestTransitionDependencies tests TransitionDependencies.
func TestTransitionDependencies(t *testing.T) {
	// Define test cases.
	tests := []struct {
		transitions     []*Change
		expectedPaths   []string
		expectedDigests [][]byte
	}{
		{nil, nil, nil},
		{[]*Change{{New: nil}}, nil, nil},
		{[]*Change{{New: tD0}}, nil, nil},
		{[]*Change{{New: tSA}}, nil, nil},
		{[]*Change{{New: tU}}, nil, nil},
		{[]*Change{{New: tP1}}, nil, nil},
		{[]*Change{{New: tF1}}, []string{""}, [][]byte{tF1.Digest}},
		{[]*Change{{New: tD1}}, []string{"file"}, [][]byte{tF1.Digest}},
		{[]*Change{{Old: tF3, New: tF3E}}, nil, nil},
		{[]*Change{{Old: tF3E, New: tF3}}, nil, nil},
	}

	// Process test cases.
	for i, test := range tests {
		// Compute transition dependencies and perform basic validation.
		paths, digests := TransitionDependencies(test.transitions)
		if len(paths) != len(digests) {
			t.Errorf("test index %d: path and digest counts differ: %d != %d",
				i, len(paths), len(digests),
			)
		}

		// Validate that paths match expected.
		if !comparison.StringSlicesEqual(paths, test.expectedPaths) {
			t.Errorf("test index %d: paths do not match expected", i)
		}

		// Validate that digest match expected.
		if len(digests) != len(test.expectedDigests) {
			t.Errorf("test index %d: digests do not match expected", i)
		} else {
			for d, digest := range digests {
				if !bytes.Equal(digest, test.expectedDigests[d]) {
					t.Errorf("test index %d: digests do not match expected", i)
					break
				}
			}
		}
	}
}
