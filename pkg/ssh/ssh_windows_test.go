package ssh

import (
	"testing"
)

func TestCommandNamedNotExist(t *testing.T) {
	if _, err := commandNamed("non-existent-command"); err == nil {
		t.Fatal("non-existent command found")
	}
}
