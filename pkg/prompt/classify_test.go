package prompt

import (
	"testing"
)

func TestClassifyBinaryQuestionMark(t *testing.T) {
	if k := Classify("Question? (yes/no)? "); k != PromptKindBinary {
		t.Error("misclassified binary prompt as", k)
	}
}

func TestClassifyBinaryQuestionColon(t *testing.T) {
	if k := Classify("Question? (yes/no): "); k != PromptKindBinary {
		t.Error("misclassified binary prompt as", k)
	}
}

func TestClassifySecret(t *testing.T) {
	if k := Classify("Give me your password: "); k != PromptKindSecret {
		t.Error("misclassified secret prompt as", k)
	}
}
