package main

// This script generates Go code from Protocol Buffers specifications. It uses
// the "gogo" Go code generator (https://github.com/gogo/protobuf). The
// generated Go code depends only on pure Go libraries, so it doesn't need the
// standard C++-based Protocol Buffers installation available to compile. Thus,
// since we check-in the generated code, users can build Mutagen without the
// need to install anything other than Go, and there is no need to run this
// script as part of the normal build process.
//
// If you do want to run this script (say after modifying a .proto file), you'll
// need Protocol Buffers 3+ installed (the C++ version which includes protoc -
// https://github.com/google/protobuf) and the "protoc-gen-gogo" generator
// binary installed (see the "gogo" Go code generator link for installation
// instructions). You will need to ensure that you have a version of the code
// generator installed that corresponds to the vendored runtime package,
// otherwise the generated code may be incompatible.
//
// TODO: Can this script compile the protoc-gen-gogo binary directly from the
// vendor directory (into a temporary directory) and add it to the path of the
// protoc executable? That would remove the need for manually keeping the
// generator installation in sync with the runtime that's vendored.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/cmd"
)

var subdirectories = []struct {
	path  string
	files []string
}{
	{"session", []string{"session.proto"}},
	{"sync", []string{"cache.proto", "entry.proto"}},
	{"url", []string{"url.proto"}},
}

func main() {
	// Compute the path to the Mutagen source directory.
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		cmd.Fatal(errors.New("unable to compute script path"))
	}
	mutagenSource := filepath.Dir(filepath.Dir(file))

	// Compute the vendoring path.
	vendor := filepath.Join(mutagenSource, "vendor")

	// Compute the GOPATH src directory.
	gopathSrc := filepath.Dir(filepath.Dir(filepath.Dir(mutagenSource)))

	// Process subdirectories.
	for _, s := range subdirectories {
		// Compute the subdirectory path.
		subdirectory := filepath.Join(mutagenSource, s.path)

		// Print directory information.
		fmt.Println("Processing", subdirectory)

		// Execute the Protocol Buffers compiler using the gogo generator.
		// HACK: We specify include paths so that we can reference definitions
		// between packages, but this means that we also end up needing to
		// specify -I., because for some reason the Protocol Buffers compiler is
		// too stupid to include this automatically. If you don't believe me,
		// try removing that argument and the compiler will literally print a
		// message telling you how "stupid" it is.
		// HACK: We have to use a special argument to the "gogo" generator to
		// have it point to the "gogo" timestamp implementation. The default
		// timestamp.proto file that protoc imports sets go_package to
		// github.com/golang/protobuf/ptypes/timestamp, so we just override
		// that to point to the message implementation in the "gogo"
		// repository).
		arguments := make([]string, 0, len(s.files)+1)
		arguments = append(arguments, "-I.")
		arguments = append(arguments, fmt.Sprintf("-I%s", vendor))
		arguments = append(arguments, fmt.Sprintf("-I%s", gopathSrc))
		timestampRedirect := "Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types"
		arguments = append(arguments, fmt.Sprintf("--gogo_out=%s:.", timestampRedirect))
		arguments = append(arguments, s.files...)
		protoc := exec.Command("protoc", arguments...)
		protoc.Dir = subdirectory
		protoc.Stdin = os.Stdin
		protoc.Stdout = os.Stdout
		protoc.Stderr = os.Stderr
		if err := protoc.Run(); err != nil {
			cmd.Fatal(errors.Wrap(err, "protoc execution failed"))
		}
	}
}
