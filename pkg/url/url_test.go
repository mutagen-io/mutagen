package url

import (
	"testing"
)

func TestProtocolSupported(t *testing.T) {
	// Ensure that the local protocol is marked as supported.
	if !Protocol_Local.supported() {
		t.Error("local protocol marked as unsupported")
	}

	// Ensure that the SSH protocol is marked as supported.
	if !Protocol_SSH.supported() {
		t.Error("SSH protocol marked as unsupported")
	}

	// Ensure that one above the highest value protocol is marked as
	// unsupported. This should help keep enumeration values and test code in
	// sync.
	if (Protocol_SSH+1).supported() {
		t.Error("unknown protocol marked as supported")
	}
}

func TestURLEnsureValid(t *testing.T) {
	// Ensure that a nil URL is invalid.
	var invalid *URL
	if invalid.EnsureValid() == nil {
		t.Error("nil URL marked as valid")
	}

	// Ensure that a URL with an unsupported protocol is invalid. This should
	// also keep enumeration values and test code in sync.
	invalid = &URL{
		Protocol: (Protocol_SSH+1),
		Username: "george",
		Hostname: "washington",
		Port: 22,
		Path: "~/path",
	}
	if invalid.EnsureValid() == nil {
		t.Error("nil URL marked as valid")
	}

	// Ensure that a sane URL is valid.
	valid := &URL{
		Protocol: Protocol_SSH,
		Username: "george",
		Hostname: "washington",
		Port: 22,
		Path: "~/path",
	}
	if err := valid.EnsureValid(); err != nil {
		t.Error("valid URL marked as invalid:", err)
	}
}
