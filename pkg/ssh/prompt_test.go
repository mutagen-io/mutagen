package ssh

import (
	"testing"
)

func TestAddPrompterVariablesNoPrompter(t *testing.T) {
	if e, err := setPrompterVariables([]string{"SSH_ASKPASS=someprogram"}, ""); err != nil {
		t.Fatal("failed to set prompter environment variables:", err)
	} else if len(e) != 0 {
		t.Error("SSH_ASKPASS environment variable not removed in absence of prompter")
	}
}

func TestAddPrompterVariables(t *testing.T) {
	if e, err := setPrompterVariables(nil, "prompter-id"); err != nil {
		t.Fatal("failed to set prompter environment variables:", err)
	} else if len(e) != 3 {
		t.Error("unexpected number of environment variables after adding prompter values")
	}
}
