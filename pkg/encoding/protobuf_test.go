package encoding

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/havoc-io/mutagen/pkg/url"
)

func TestProtocolBuffersCycle(t *testing.T) {
	// Create an empty temporary file and defer its cleanup.
	file, err := ioutil.TempFile("", "mutagen_encoding")
	if err != nil {
		t.Fatal("unable to create temporary file:", err)
	} else if err = file.Close(); err != nil {
		t.Fatal("unable to close temporary file:", err)
	}
	defer os.Remove(file.Name())

	// Create a Protocol Buffers message that we can test with.
	message := &url.URL{
		Protocol: url.Protocol_SSH,
		Username: "George",
		Hostname: "washington",
		Port:     1776,
		Path:     "/by/land/or/by/sea",
	}
	if err := MarshalAndSaveProtobuf(file.Name(), message); err != nil {
		t.Fatal("unable to marshal and save Protocol Buffers message:", err)
	}

	// Reload the message.
	decoded := &url.URL{}
	if err := LoadAndUnmarshalProtobuf(file.Name(), decoded); err != nil {
		t.Fatal("unable to load and unmarshal Protocol Buffers message:", err)
	}

	// Verify that contents were preserved. We have to explicitly compare
	// members because the Protocol Buffers generated struct contains fields
	// that can't be compared by value.
	match := decoded.Protocol == message.Protocol &&
		decoded.Username == message.Username &&
		decoded.Hostname == message.Hostname &&
		decoded.Port == message.Port &&
		decoded.Path == message.Path
	if !match {
		t.Error("decoded Protocol Buffers message did not match original:", decoded, "!=", message)
	}
}
