package compose

const (
	// mutagenServiceName is the name used for the Mutagen service.
	mutagenServiceName = "mutagen"
)

// mutagenDockerfileLinux is the Dockerfile template for the Mutagen service
// when using a Linux-based Docker daemon.
const mutagenDockerfileLinux = `FROM alpine:latest
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

// mutagenComposeYAML is the Docker Compose configuration template for the
// Mutagen service and any reverse forwarding services.
const mutagenComposeYAML = `version: "{{ .Version }}"
services:
  mutagen:
    build: "{{ .TemporaryDirectory }}/services/mutagen"
    init: true
    # TODO: Add network dependencies
    networks:
    # TODO: Add volume dependencies
    volumes:
  # TODO: Add reverse forwarding services
`
