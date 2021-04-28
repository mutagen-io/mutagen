package ssh

import (
	"testing"
)

func TestAddLocaleVariables(t *testing.T) {
	if e := addLocaleVariables(nil); len(e) != 1 {
		t.Fatal("no locale variables added")
	} else if e[0] != "LC_ALL=C" {
		t.Error("incorrect locale variables added")
	}
}
