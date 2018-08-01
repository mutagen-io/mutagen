package prompt

import (
	"testing"
)

// TestClassifyBinaryQuestionMark tests classification of a binary prompt with a
// question mark suffix.
func TestClassifyBinaryQuestionMark(t *testing.T) {
	if k := Classify("Question? (yes/no)? "); k != PromptKindBinary {
		t.Error("misclassified binary prompt as", k)
	}
}

// TestClassifyBinaryQuestionColon tests classification of a binary prompt with
// a colon suffix.
func TestClassifyBinaryQuestionColon(t *testing.T) {
	if k := Classify("Question? (yes/no): "); k != PromptKindBinary {
		t.Error("misclassified binary prompt as", k)
	}
}

// TestClassifySecret tests classification of a secret prompt.
func TestClassifySecret(t *testing.T) {
	if k := Classify("Give me your password: "); k != PromptKindSecret {
		t.Error("misclassified secret prompt as", k)
	}
}
