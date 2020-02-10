package prompting

import (
	"testing"
)

// TestDetermineResponseModeBinaryQuestionMark tests response mode determination
// for a binary prompt with a question mark suffix.
func TestDetermineResponseModeBinaryQuestionMark(t *testing.T) {
	if m := determineResponseMode("Question? (yes/no)? "); m != ResponseModeBinary {
		t.Error("misclassified binary prompt response mode as", m)
	}
}

// TestDetermineResponseModeBinaryQuestionColon tests response mode
// determination for a binary prompt with a colon suffix.
func TestDetermineResponseModeBinaryQuestionColon(t *testing.T) {
	if m := determineResponseMode("Question? (yes/no): "); m != ResponseModeBinary {
		t.Error("misclassified binary prompt response mode as", m)
	}
}

// TestDetermineResponseModeSecret tests response mode determination for a
// secret prompt.
func TestDetermineResponseModeSecret(t *testing.T) {
	if m := determineResponseMode("Give me your password: "); m != ResponseModeSecret {
		t.Error("misclassified secret prompt response mode as", m)
	}
}
