package compose

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// mutagenServiceName is the name used for the Mutagen service.
	mutagenServiceName = "mutagen"
)

// mutagenLinuxDockerfileTemplate is the Dockerfile template for the Mutagen
// service when using a Linux-based Docker daemon.
const mutagenLinuxDockerfileTemplate = `FROM alpine:latest
RUN ["mkdir", "/volumes"]
ENTRYPOINT ["tail", "-f", "/dev/null"]
`

// TODO: Implement the Dockerfile definition for the Mutagen service on Windows.
// We'll likely want to use a Windows Server Core base image, but the versioning
// is more complex and tied to the Docker daemon host. We may need to probe for
// additional information from the Docker daemon, for example the KernelVersion
// parameter. On Windows, the Docker daemon populates this parameter using the
// HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion registry key
// and it can likely be translated to a Windows Server Core image version tag.

// needMutagenServiceInitForPlatform returns whether or not the Mutagen service
// needs a wrapper init function for the specified container platform. This
// function should only be called for supported Docker platforms.
// TODO: Add support for Windows.
func needMutagenServiceInitForPlatform(platform string) bool {
	switch platform {
	case "linux":
		return true
	default:
		panic("unsupported Docker platform")
	}
}

// generateMutagenServiceBuildContext generates a Mutagen service build context
// at the specified path. The path must not exist but its parent directory must.
// This function should only be called for supported Docker platforms.
func generateMutagenServiceBuildContext(path, platform string) error {
	// Create the directory.
	if err := os.Mkdir(path, 0700); err != nil {
		return err
	}

	// Determine the Dockerfile template.
	// TODO: Add support for Windows.
	var dockerfileTemplate string
	if platform == "linux" {
		dockerfileTemplate = mutagenLinuxDockerfileTemplate
	} else {
		panic("unsupported Docker platform")
	}

	// Create the Dockerfile and defer its closure.
	dockerfilePath := filepath.Join(path, "Dockerfile")
	dockerfile, err := os.OpenFile(dockerfilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to create Dockerfile: %w", err)
	}
	defer dockerfile.Close()

	// Write the template to the file.
	// TODO: If we need templatized support for Windows, switch to using the
	// text/template package here.
	if count, err := dockerfile.Write([]byte(dockerfileTemplate)); err != nil {
		return fmt.Errorf("unable to write Dockerfile contents: %w", err)
	} else if count != len(dockerfileTemplate) {
		return errors.New("unable to write full Dockerfile contents")
	}

	// Success.
	return nil
}
