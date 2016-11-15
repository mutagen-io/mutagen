package filesystem

import (
	"syscall"

	"github.com/pkg/errors"

	"golang.org/x/text/unicode/norm"
)

const (
	// This constant is VERY hard to find and only applies on Darwin systems
	// (statfs or statvfs might need different values on other systems). It
	// applies to all HFS variants, and its definition can be found in the XNU
	// sources (xnu/bsd/vfs/vfs_conf.c).
	statfsTypeHFS = 17
)

func normalizeDirectoryNames(path string, names []string) error {
	// Check the path type.
	var fsStats syscall.Statfs_t
	if err := syscall.Statfs(path, &fsStats); err != nil {
		return errors.Wrap(err, "unable to determine filesystem type")
	}

	// If we're not dealing with HFS, then we don't need to renormalize.
	if fsStats.Type != statfsTypeHFS {
		return nil
	}

	// Otherwise we need to convert names to NFC normalization. This might not
	// be perfect, because HFS actually uses a custom variant of NFD, but my
	// understanding is that it's just NFD with certain CJK characters not
	// decomposed. It's evolved a lot over time though and is under-documented,
	// so it is difficult to say. This link has a lot of information:
	// https://bugzilla.mozilla.org/show_bug.cgi?id=703161
	// In any case, converting to NFC should be a fairly decent approximation
	// because almost every other system will use NFC for Unicode filenames.
	// Well, actually, that's not true, they usually don't enforce a
	// normalization, they just store the code points that they get, so in
	// theory we could see NFD or other normalization coming from other systems,
	// but that is less likely and this is really the best we can do. Once Apple
	// decides to take HFS out behind the shed and shoot it, this should be less
	// of an issue (unless they end up propagating this behavior to AppleFS).
	for i, n := range names {
		names[i] = norm.NFC.String(n)
	}

	// Success.
	return nil
}
