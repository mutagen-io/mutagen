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
	"path"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/cmd"
)

var subdirectories = []struct {
	path  string
	files []string
}{
	{
		"github.com/havoc-io/mutagen/pkg/daemon/service",
		[]string{"daemon.proto"},
	},
	{
		"github.com/havoc-io/mutagen/pkg/filesystem",
		[]string{"watch.proto"},
	},
	{
		"github.com/havoc-io/mutagen/pkg/prompt/service",
		[]string{"prompt.proto"},
	},
	{
		"github.com/havoc-io/mutagen/pkg/rsync",
		[]string{"receive.proto"},
	},
	{
		"github.com/havoc-io/mutagen/pkg/session",
		[]string{"configuration.proto", "session.proto", "state.proto"},
	},
	{
		"github.com/havoc-io/mutagen/pkg/session/service",
		[]string{"session.proto"},
	},
	{
		"github.com/havoc-io/mutagen/pkg/sync",
		[]string{
			"archive.proto",
			"cache.proto",
			"change.proto",
			"conflict.proto",
			"entry.proto",
			"ignore.proto",
			"problem.proto",
			"symlink.proto",
		},
	},
	{
		"github.com/havoc-io/mutagen/pkg/url",
		[]string{"url.proto"},
	},
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
	protocEnvironment := os.Environ()
	if existingPath := os.Getenv("PATH"); existingPath != "" {
		protocEnvironment = append(protocEnvironment, fmt.Sprintf(
			"PATH=%s%s%s",
			generatorPath,
			string(os.PathListSeparator),
			existingPath,
		))
	} else {
		protocEnvironment = append(protocEnvironment, fmt.Sprintf(
			"PATH=%s",
			generatorPath,
		))
	}

	// Compute the path to the Mutagen source directory.
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		cmd.Fatal(errors.New("unable to compute script path"))
	}
	mutagenSource := filepath.Dir(filepath.Dir(file))

	// Compute the $GOPATH/src directory.
	gopathSrc := filepath.Dir(filepath.Dir(filepath.Dir(mutagenSource)))

	// Process subdirectories.
	for _, s := range subdirectories {
		// Print directory information.
		fmt.Println("Processing", s.path)

		// Execute the Protocol Buffers compiler using the Go code generator.
		var arguments []string
		arguments = append(arguments, fmt.Sprintf("-I%s", gopathSrc))
		arguments = append(arguments, fmt.Sprintf("--go_out=plugins=grpc:."))
		for _, f := range s.files {
			arguments = append(arguments, path.Join(s.path, f))
		}
		protoc := exec.Command("protoc", arguments...)
		protoc.Dir = gopathSrc
		protoc.Env = protocEnvironment
		protoc.Stdin = os.Stdin
		protoc.Stdout = os.Stdout
		protoc.Stderr = os.Stderr
		if err := protoc.Run(); err != nil {
			cmd.Fatal(errors.Wrap(err, "protoc execution failed"))
		}
	}
}
