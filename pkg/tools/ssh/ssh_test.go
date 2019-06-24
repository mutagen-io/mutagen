package ssh

import (
	"testing"
)

func TestSCPCommand(t *testing.T) {
	if commandName, err := scpCommandNameOrPath(); err != nil {
		t.Fatal("unable to locate SCP command:", err)
	} else if commandName == "" {
		t.Error("SCP command name is empty")
	}
}

func TestSSHCommand(t *testing.T) {
	if commandName, err := sshCommandNameOrPath(); err != nil {
		t.Fatal("unable to locate SSH command:", err)
	} else if commandName == "" {
		t.Error("SSH command name is empty")
	}
}
