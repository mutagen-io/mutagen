package core

import (
	"testing"
)

// TestReifyPhantomDirectories tests ReifyPhantomDirectories.
func TestReifyPhantomDirectories(t *testing.T) {
	// Define test cases.
	tests := []struct {
		ancestor                    *Entry
		alpha                       *Entry
		beta                        *Entry
		expectedAlpha               *Entry
		expectedBeta                *Entry
		expectedAlphaDirectoryCount uint64
		expectedBetaDirectoryCount  uint64
	}{
		{tN, tF1, tF2, tF1, tF2, 0, 0},
		{tDM, tF1, tF2, tF1, tF2, 0, 0},
		{tN, tF1, tD0, tF1, tD0, 0, 1},
		{tN, tD1, tF1, tD1, tF1, 1, 0},

		{tN, tPD0, tPD0, tU, tU, 0, 0},
		{tF1, tPD0, tPD0, tU, tU, 0, 0},
		{tD0, tPD0, tPD0, tD0, tD0, 1, 1},

		{tN, tPD0, tN, tU, tN, 0, 0},
		{tF1, tN, tPD0, tN, tU, 0, 0},
		{tD0, tPD0, tN, tD0, tN, 1, 0},
		{tD0, tN, tPD0, tN, tD0, 0, 1},

		{tN, tPD0, tPD1, tD0, tD1, 1, 1},
		{tF1, tPD1, tPD0, tD1, tD0, 1, 1},
		{tD0, tPD0, tPD1, tD0, tD1, 1, 1},
		{tD0, tPD1, tPD0, tD1, tD0, 1, 1},

		{tN, tF1, tPD0, tF1, tU, 0, 0},
		{tN, tPD0, tF1, tU, tF1, 0, 0},

		{tN, tD1, tPD0, tD1, tD0, 1, 1},
		{tN, tPD0, tD1, tD0, tD1, 1, 1},
		{tF1, tD1, tPD0, tD1, tD0, 1, 1},
		{tF1, tPD0, tD1, tD0, tD1, 1, 1},
		{tD0, tD1, tPD0, tD1, tD0, 1, 1},
		{tD0, tPD0, tD1, tD0, tD1, 1, 1},

		{tN, tDM, tPD0, tDM, tD0, 3, 1},
		{tN, tPD0, tDM, tD0, tDM, 1, 3},
		{tF1, tDM, tPD0, tDM, tD0, 3, 1},
		{tF1, tPD0, tDM, tD0, tDM, 1, 3},
		{tD0, tDM, tPD0, tDM, tD0, 3, 1},
		{tD0, tPD0, tDM, tD0, tDM, 1, 3},

		{tN, tD1, tPDU, tD1, tDU, 1, 1},
		{tN, tPDU, tD1, tDU, tD1, 1, 1},
		{tF1, tD1, tPDU, tD1, tDU, 1, 1},
		{tF1, tPDU, tD1, tDU, tD1, 1, 1},
		{tD0, tD1, tPDU, tD1, tDU, 1, 1},
		{tD0, tPDU, tD1, tDU, tD1, 1, 1},

		{tN, tDM, tPDU, tDM, tDU, 3, 1},
		{tN, tPDU, tDM, tDU, tDM, 1, 3},
		{tF1, tDM, tPDU, tDM, tDU, 3, 1},
		{tF1, tPDU, tDM, tDU, tDM, 1, 3},
		{tD0, tDM, tPDU, tDM, tDU, 3, 1},
		{tD0, tPDU, tDM, tDU, tDM, 1, 3},

		{tD0, tPD0, tPD0, tD0, tD0, 1, 1},
		{tD0, tN, tPD0, tN, tD0, 0, 1},
		{tD0, tPD0, tN, tD0, tN, 1, 0},
		{tD1, tPD0, tPD0, tD0, tD0, 1, 1},
		{tD1, tN, tPD0, tN, tD0, 0, 1},
		{tD1, tPD0, tN, tD0, tN, 1, 0},

		{tF1, tN, tPD0, tN, tU, 0, 0},
		{tF1, tPD0, tN, tU, tN, 0, 0},

		{tN, tPD0, tPDU, tU, tU, 0, 0},
		{tN, tPDU, tPD0, tU, tU, 0, 0},
		{tF1, tPD0, tPDU, tU, tU, 0, 0},
		{tF1, tPDU, tPD0, tU, tU, 0, 0},
		{tD0, tPD0, tPDU, tD0, tDU, 1, 1},
		{tD0, tPDU, tPD0, tDU, tD0, 1, 1},

		{tN, tPD0, tPDP1, tD0, tDP1, 1, 1},
		{tN, tPDP1, tPD0, tDP1, tD0, 1, 1},
		{tF1, tPD0, tPDP1, tD0, tDP1, 1, 1},
		{tF1, tPDP1, tPD0, tDP1, tD0, 1, 1},
		{tD0, tPD0, tPDP1, tD0, tDP1, 1, 1},
		{tD0, tPDP1, tPD0, tDP1, tD0, 1, 1},

		{tN, tPD0, tPDPD0, tU, tU, 0, 0},
		{tN, tPDPD0, tPD0, tU, tU, 0, 0},
		{tF1, tPD0, tPDPD0, tU, tU, 0, 0},
		{tF1, tPDPD0, tPD0, tU, tU, 0, 0},

		{tN, tPD0, tPDD0, tD0, tDD0, 1, 2},
		{tN, tPDD0, tPD0, tDD0, tD0, 2, 1},
		{tF1, tPD0, tPDD0, tD0, tDD0, 1, 2},
		{tF1, tPDD0, tPD0, tDD0, tD0, 2, 1},
	}

	// Process test cases.
	for i, test := range tests {
		alpha, beta, alphaDirectoryCount, betaDirectoryCount := ReifyPhantomDirectories(
			test.ancestor, test.alpha, test.beta,
		)
		if !alpha.Equal(test.expectedAlpha, true) {
			t.Errorf("test index %d: alpha does not match expected: %v != %v",
				i, alpha, test.expectedAlpha,
			)
		}
		if !beta.Equal(test.expectedBeta, true) {
			t.Errorf("test index %d: beta does not match expected: %v != %v",
				i, beta, test.expectedBeta,
			)
		}
		if alphaDirectoryCount != test.expectedAlphaDirectoryCount {
			t.Errorf("test index %d: alpha directory count does not match expected: %d != %d",
				i, alphaDirectoryCount, test.expectedAlphaDirectoryCount,
			)
		}
		if betaDirectoryCount != test.expectedBetaDirectoryCount {
			t.Errorf("test index %d: beta directory count does not match expected: %d != %d",
				i, betaDirectoryCount, test.expectedBetaDirectoryCount,
			)
		}
	}
}
