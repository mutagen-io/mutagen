package ssh

import (
	"testing"
)

func TestSCPCommand(t *testing.T) {
	if commandName, err := scpCommandPath(); err != nil {
		t.Fatal("unable to locate SCP command:", err)
	} else if commandName == "" {
		t.Error("SCP command path is empty")
	}
}

func TestSSHCommand(t *testing.T) {
	if commandName, err := sshCommandPath(); err != nil {
		t.Fatal("unable to locate SSH command:", err)
	} else if commandName == "" {
		t.Error("SSH command path is empty")
	}
}
