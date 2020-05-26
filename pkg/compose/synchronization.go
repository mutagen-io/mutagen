package compose

import (
	"errors"
	"strings"

	"github.com/mutagen-io/mutagen/pkg/docker"
	"github.com/mutagen-io/mutagen/pkg/url"
)

// volumeURLPrefix is the lowercase version of the volume URL prefix.
const volumeURLPrefix = "volume://"

// isVolumeURL checks if raw URL is a Docker Compose volume pseudo-URL.
func isVolumeURL(raw string) bool {
	return strings.HasPrefix(strings.ToLower(raw), volumeURLPrefix)
}

// mountPathForVolumeInMutagenContainer returns the mount path that will be used
// for a volume inside the Mutagen container. The path will be returned without
// a trailing slash. The daemon OS must be supported (as indicated by
// isSupportedDaemonOS) and the volume name non-empty, otherwise this function
// will panic.
func mountPathForVolumeInMutagenContainer(daemonOS, volume string) string {
	// Verify that the volume is non-empty.
	if volume == "" {
		panic("empty volume name")
	}

	// Compute the path based on the daemon OS.
	switch daemonOS {
	case "linux":
		return "/volumes/" + volume
	case "windows":
		return `c:\volumes\` + volume
	default:
		panic("unsupported daemon OS")
	}
}

// parseVolumeURL parses a Docker Compose volume pseudo-URL, converting it to a
// concrete Mutagen Docker URL. It uses the top-level daemon connection flags to
// determine URL parameters and looks for Docker environment variables in the
// fully resolved project environment (which may included variables loaded from
// "dotenv" files). This function also returns the volume dependency for the
// URL. This function must only be called on URLs that have been classified as
// volume URLs by isVolumeURL, otherwise this function may panic.
func parseVolumeURL(
	raw, daemonOS, mutagenContainerName string,
	environment map[string]string,
	daemonFlags docker.DaemonConnectionFlags,
) (*url.URL, string, error) {
	// Strip off the prefix
	raw = raw[len(volumeURLPrefix):]

	// Find the first slash, which will indicate the end of the volume name. If
	// no slash is found, then we assume that the volume itself is the target
	// synchronization root.
	var volume, path string
	if slashIndex := strings.IndexByte(raw, '/'); slashIndex < 0 {
		volume = raw
		path = mountPathForVolumeInMutagenContainer(daemonOS, volume)
	} else if slashIndex == 0 {
		return nil, "", errors.New("empty volume name")
	} else {
		volume = raw[:slashIndex]
		path = mountPathForVolumeInMutagenContainer(daemonOS, volume) + raw[slashIndex:]
	}

	// Store any Docker environment variables that we need to preserve. We only
	// store variables that are actually present, because Docker behavior will
	// vary depending on whether a variable is unset vs. set but empty. Note
	// that unlike standard Docker URL parsing, we load these variables from the
	// project environment (which may include values from "dotenv" files). We
	// also don't support endpoint-specific variants since those don't make
	// sense in the context of Docker Compose.
	urlEnvironment := make(map[string]string)
	for _, variable := range url.DockerEnvironmentVariables {
		if value, present := environment[variable]; present {
			urlEnvironment[variable] = value
		}
	}

	// Create a Docker synchronization URL.
	return &url.URL{
		Kind:        url.Kind_Synchronization,
		Protocol:    url.Protocol_Docker,
		Host:        mutagenContainerName,
		Path:        path,
		Environment: urlEnvironment,
		Parameters:  daemonFlags.ToURLParameters(),
	}, volume, nil
}
