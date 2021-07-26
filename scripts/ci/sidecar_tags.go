package main

import (
	"fmt"
	"strings"

	"github.com/mutagen-io/mutagen/pkg/mutagen"
	"github.com/mutagen-io/mutagen/pkg/sidecar"
)

func main() {
	// Track the tags that we need to push.
	var tags []string

	// Always create a tag for the full version.
	tags = append(tags, fmt.Sprintf("%s:%s", sidecar.BaseTag, mutagen.Version))

	// If this is a proper release version (i.e. it doesn't have a version tag),
	// then add other tags as appropriate. In this case, we don't need to add
	// the patch version, because that will be the full version added above.
	if mutagen.VersionTag == "" {
		// Add the minor version.
		tags = append(tags, fmt.Sprintf("%s:%d.%d", sidecar.BaseTag, mutagen.VersionMajor, mutagen.VersionMinor))

		// Add the major version if we're post-1.0.
		if mutagen.VersionMajor >= 1 {
			tags = append(tags, fmt.Sprintf("%s:%d", sidecar.BaseTag, mutagen.VersionMajor))
		}
	}

	// Compute the combined tag list.
	fmt.Println(strings.Join(tags, ","))
}
