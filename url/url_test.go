package url

import (
	"testing"
)

func TestAccessorsNil(t *testing.T) {
	var url *URL
	if url.GetProtocol() != Protocol_Local {
		t.Error("protocol accessor value mismatch for nil message")
	}
	if url.GetUsername() != "" {
		t.Error("username accessor value mismatch for nil message")
	}
	if url.GetHostname() != "" {
		t.Error("hostname accessor value mismatch for nil message")
	}
	if url.GetPort() != 0 {
		t.Error("port accessor value mismatch for nil message")
	}
	if url.GetPath() != "" {
		t.Error("path accessor value mismatch for nil message")
	}
}

func TestAccessors(t *testing.T) {
	url := &URL{
		Protocol: Protocol_SSH,
		Username: "user",
		Hostname: "host",
		Port:     23,
		Path:     "/test/path",
	}
	if url.GetProtocol() != url.Protocol {
		t.Error("protocol accessor value mismatch")
	}
	if url.GetUsername() != url.Username {
		t.Error("username accessor value mismatch")
	}
	if url.GetHostname() != url.Hostname {
		t.Error("hostname accessor value mismatch")
	}
	if url.GetPort() != url.Port {
		t.Error("port accessor value mismatch")
	}
	if url.GetPath() != url.Path {
		t.Error("path accessor value mismatch")
	}
}

func TestReset(t *testing.T) {
	url := &URL{
		Protocol: Protocol_SSH,
		Username: "user",
		Hostname: "host",
		Port:     23,
		Path:     "/test/path",
	}
	url.Reset()
	if url.Protocol != Protocol_Local {
		t.Error("reset did not reset protocol")
	}
	if url.Username != "" {
		t.Error("reset did not reset username")
	}
	if url.Hostname != "" {
		t.Error("reset did not reset hostname")
	}
	if url.Port != 0 {
		t.Error("reset did not reset port")
	}
	if url.Path != "" {
		t.Error("reset did not reset path")
	}
}

// TestProtocolBuffersMethods tests miscellaneous generated methods that we
// don't actually invoke to (a) make sure they don't panic and (b) stop their
// existence from deflating test coverage.
func TestProtocolBuffersMethods(t *testing.T) {
	// Test Protocol methods.
	protocol := Protocol_Local
	_ = protocol.String()
	_, _ = protocol.EnumDescriptor()

	// Test URL methods.
	url := &URL{
		Protocol: Protocol_SSH,
		Username: "user",
		Hostname: "host",
		Port:     12345,
		Path:     "/some/path",
	}
	_ = url.String()
	url.ProtoMessage()
	_, _ = url.Descriptor()
	encoded, err := url.Marshal()
	if err != nil {
		t.Error("unable to marshal URL:", err)
	}
	if err = url.Unmarshal(encoded); err != nil {
		t.Error("unable to unmarshal URL:", err)
	}
}
