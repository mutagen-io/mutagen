package sync

import (
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

const (
	// maximumPortableSymlinkTargetLength is the maximum symlink target length
	// that we can synchronize in a portable fashion. It is limited by Windows,
	// where a path length of 248 characters or longer will be converted to
	// extended path format (i.e. prefixed with "\\?\"), thus not allowing us to
	// reliably round-trip it to disk. For more information, see
	// Go's src/os/path_windows.go (fixLongPath function). The length should be
	// fine on all other modern systems.
	maximumPortableSymlinkTargetLength = 247
)

// normalizeSymlinkAndEnsurePortable normalizes a symlink target and verifies
// that it's valid safe for portable propagation. This requires that the symlink
// be relative, not escape the synchronization root, and be composed of only
// portable characters. These are the only types of symlinks that we can safely
// (and sanely) synchronize between systems.
//
// Effectively this means that the target needs to be components of the form
// "<name>", "..", or ".", separated by '/', and not an absolute path of any
// form. On POSIX, verifying that the path is not absolute is relatively easy.
// On Windows, it's significantly more difficult. See the "Remarks" section
// here:
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa363866(v=vs.85).aspx
// It lists some of the possible Windows path formats accepted by
// CreateSymbolicLinkW.
func normalizeSymlinkAndEnsurePortable(path, target string) (string, error) {
	// If the target is empty, it's invalid on most (all?) platforms.
	if target == "" {
		return "", errors.New("target empty")
	}

	// If the target is longer than the maximum allowed symlink length, we can't
	// propagate it.
	if len(target) > maximumPortableSymlinkTargetLength {
		return "", errors.New("target too long")
	}

	// Ensure that the target path doesn't contain a colon. On POSIX, colons are
	// allowed to occur in filenames (and hence paths) (they aren't allowed in
	// $PATH because ':' is used as the path separator). On Windows, colons are
	// not allowed in filenames, but they are allowed in paths, where they have
	// very different meanings. They can appear in absolute paths (i.e. those of
	// the form "C:\...") or working directory-relative paths (i.e. those of the
	// form "C:File.txt" (which maps to "<working directory>\File.txt")). If we
	// find a colon in a target path on POSIX, we can't reliably propagate it,
	// and if we find a colon in a target path on Windows, it's a type of target
	// path that we can't propagate.
	if strings.Index(target, ":") != -1 {
		return "", errors.New("colon in target (absolute or unsupported path)")
	}

	// If we're on a Windows system, convert all backslashes to forward slashes.
	// Windows only supports using backslashes in symlink targets, so Go
	// performs this conversion when creating them. If we're on a POSIX system,
	// backslashes are allowed in filenames (and hence paths), and they don't
	// act as a path separator. That being said, we won't be able to round-trip
	// them to a Windows system, so we have to avoid their presence.
	if runtime.GOOS == "windows" {
		target = strings.ReplaceAll(target, "\\", "/")
	} else if strings.Index(target, "\\") != -1 {
		return "", errors.New("backslash in target")
	}

	// Watch for an absolute path specification. This may be either an absolute
	// POSIX path (one beginning with "/"), an extended-length Windows path
	// (which would be prefixed with \\?\), a "root relative" Windows path (i.e.
	// one of the form "\x" (which maps to "C:\x")), some other UNC Windows path
	// (which would be prefixed with "\\"). None of these can be propagated.
	if target[0] == '/' {
		return "", errors.New("target is absolute")
	}

	// Compute the depth of the symlink inside the synchronization root and
	// iterate through the target components, ensuring that the target never
	// references a location outside of the synchronization root. Note that we
	// don't add 1 to our calculation of pathDepth because the act of
	// dereferencing the symlink removes one element of path depth.
	pathDepth := strings.Count(path, "/")
	for _, component := range strings.Split(target, "/") {
		// Update the depth.
		if component == "." {
			// No change to depth.
		} else if component == ".." {
			pathDepth--
		} else {
			pathDepth++
		}

		// Verify that we haven't escaped the synchronization root.
		if pathDepth < 0 {
			return "", errors.New("target references location outside synchronization root")
		}
	}

	// Success.
	return target, nil
}
