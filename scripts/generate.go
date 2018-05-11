package main

// This script generates Go code from Protocol Buffers specifications in the
// Mutagen source tree. It uses the canonical Go Protocol Buffers code generator
// (https://github.com/golang/protobuf). It builds this generator from the
// vendored sources to ensure it matches the version of the runtime code that
// goes into the final binaries.
//
// The generated Go code depends only on pure Go libraries, so it doesn't need
// the standard C++-based Protocol Buffers installation available to compile.
// Thus, since we check-in the generated code, users can build Mutagen without
// the need to install anything other than Go, and there is no need to run this
// script as part of the normal build process.
//
// If you do want to run this script (say after modifying a .proto file), then
// you'll need the C++ version of Protocol Buffers 3+
// (https://github.com/google/protobuf) installed with the protoc compiler in
// your path.

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/environment"
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
	// Create a temporary directory in which we can build the generator and
	// defer its removal.
	generatorPath, err := ioutil.TempDir("", "mutagen_generate")
	if err != nil {
		cmd.Fatal(errors.New("unable to create directory for generator build"))
	}
	defer os.RemoveAll(generatorPath)

	// Print status.
	fmt.Println("Building generator")

	// Build the generator.
	generatorBuild := exec.Command(
		"go",
		"build",
		"github.com/havoc-io/mutagen/vendor/github.com/golang/protobuf/protoc-gen-go",
	)
	generatorBuild.Dir = generatorPath
	generatorBuild.Stdin = os.Stdin
	generatorBuild.Stdout = os.Stdout
	generatorBuild.Stderr = os.Stderr
	if err := generatorBuild.Run(); err != nil {
		cmd.Fatal(errors.Wrap(err, "generator build failed"))
	}

	// Create an environment with the generator injected into the path.
	protocEnvironmentMap := environment.CopyCurrent()
	if existingPath := protocEnvironmentMap["PATH"]; existingPath != "" {
		protocEnvironmentMap["PATH"] = fmt.Sprintf(
			"%s%s%s",
			generatorPath,
			string(os.PathListSeparator),
			existingPath,
		)
	} else {
		protocEnvironmentMap["PATH"] = generatorPath
	}
	protocEnvironment := environment.Format(protocEnvironmentMap)

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

		// Execute the Protocol Buffers compiler using the Go code generator.
		// HACK: We specify include paths so that we can reference definitions
		// between packages, but this means that we also end up needing to
		// specify -I., because for some reason the Protocol Buffers compiler is
		// too stupid to include this automatically. If you don't believe me,
		// try removing that argument and the compiler will literally print a
		// message telling you how "stupid" it is.
		arguments := make([]string, 0, len(s.files)+1)
		arguments = append(arguments, "-I.")
		arguments = append(arguments, fmt.Sprintf("-I%s", vendor))
		arguments = append(arguments, fmt.Sprintf("-I%s", gopathSrc))
		arguments = append(arguments, fmt.Sprintf("--go_out=."))
		arguments = append(arguments, s.files...)
		protoc := exec.Command("protoc", arguments...)
		protoc.Dir = subdirectory
		protoc.Env = protocEnvironment
		protoc.Stdin = os.Stdin
		protoc.Stdout = os.Stdout
		protoc.Stderr = os.Stderr
		if err := protoc.Run(); err != nil {
			cmd.Fatal(errors.Wrap(err, "protoc execution failed"))
		}
	}
}
