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
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/cmd"
)

const (
	// buildDirectoryName is the name of the build directory to create inside
	// the root of the Mutagen source tree.
	buildDirectoryName = "build"

	// generatorBuildSubdirectoryName is the name of the build subdirectory
	// where the Protocol Buffers generator should be built.
	generatorBuildSubdirectoryName = "generator"

	// pkgDirectoryName is the name of the pkg directory in the Mutagen source
	// tree.
	pkgDirectoryName = "pkg"

	// generatorName is the name of the Protocol Buffers generator to use.
	generatorName = "go"

	// generatorPath is the path to use to build the Protocol Buffers generator.
	generatorPath = "github.com/golang/protobuf/protoc-gen-go"
)

var subdirectories = []struct {
	path  string
	files []string
}{
	{"daemon/service", []string{"daemon.proto"}},
	{"filesystem", []string{"watch.proto"}},
	{"prompt/service", []string{"prompt.proto"}},
	{"rsync", []string{"receive.proto"}},
	{"session", []string{"configuration.proto", "session.proto", "state.proto"}},
	{"session/service", []string{"session.proto"}},
	{
		"sync",
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
	{"url", []string{"url.proto"}},
}

// mutagenSourceDirectoryPath computes the path to the Mutagen source directory.
func mutagenSourceDirectoryPath() (string, error) {
	// Compute the path to this script.
	_, scriptPath, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("unable to compute script path")
	}

	// Compute the path to the Mutagen source directory.
	return filepath.Dir(filepath.Dir(scriptPath)), nil
}

func main() {
	// Compute the path to the Mutagen source directory.
	mutagenSourcePath, err := mutagenSourceDirectoryPath()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to compute Mutagen source path"))
	}

	// Verify that we're running inside the Mutagen source directory, otherwise
	// we can't rely on Go modules working.
	workingDirectory, err := os.Getwd()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to compute working directory"))
	}
	workingDirectoryRelativePath, err := filepath.Rel(mutagenSourcePath, workingDirectory)
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to determine working directory relative path"))
	}
	if strings.Contains(workingDirectoryRelativePath, "..") {
		cmd.Fatal(errors.Wrap(err, "build script run outside Mutagen source tree"))
	}

	// Compute the path to the build directory and ensure that it exists.
	buildPath := filepath.Join(mutagenSourcePath, buildDirectoryName)
	if err := os.MkdirAll(buildPath, 0700); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to create build directory"))
	}

	// Create the necessary build directory hierarchy.
	generatorBuildSubdirectoryPath := filepath.Join(buildPath, generatorBuildSubdirectoryName)
	if err := os.MkdirAll(generatorBuildSubdirectoryPath, 0700); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to create generator build subdirectory"))
	}

	// Print status.
	fmt.Println("Building generator")

	// Set up an environment for building the generator which forces the use of
	// Go modules.
	generatorBuildEnvironment := os.Environ()
	generatorBuildEnvironment = append(generatorBuildEnvironment, "GO111MODULE=on")

	// Build the generator.
	generatorBuild := exec.Command("go", "build", generatorPath)
	generatorBuild.Dir = generatorBuildSubdirectoryPath
	generatorBuild.Env = generatorBuildEnvironment
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
			generatorBuildSubdirectoryPath,
			string(os.PathListSeparator),
			existingPath,
		))
	} else {
		protocEnvironment = append(protocEnvironment, fmt.Sprintf(
			"PATH=%s",
			generatorBuildSubdirectoryPath,
		))
	}

	// Compute the path to the Mutagen pkg directory.
	pkgDirectoryPath := filepath.Join(mutagenSourcePath, pkgDirectoryName)

	// Process subdirectories.
	for _, s := range subdirectories {
		// Print directory information.
		fmt.Println("Processing", s.path)

		// Execute the Protocol Buffers compiler using the Go code generator.
		var arguments []string
		arguments = append(arguments, fmt.Sprintf("-I%s", pkgDirectoryPath))
		arguments = append(arguments, fmt.Sprintf("--%s_out=plugins=grpc:.", generatorName))
		for _, f := range s.files {
			arguments = append(arguments, path.Join(s.path, f))
		}
		protoc := exec.Command("protoc", arguments...)
		protoc.Dir = pkgDirectoryPath
		protoc.Env = protocEnvironment
		protoc.Stdin = os.Stdin
		protoc.Stdout = os.Stdout
		protoc.Stderr = os.Stderr
		if err := protoc.Run(); err != nil {
			cmd.Fatal(errors.Wrap(err, "protoc execution failed"))
		}
	}
}
