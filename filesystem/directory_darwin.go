package filesystem

import (
	"syscall"

	"github.com/pkg/errors"

	"golang.org/x/text/unicode/norm"
)

func normalizeDirectoryNames(path string, names []string) error {
	// Grab statistics on the filesystem at this path.
	var fsStats syscall.Statfs_t
	if err := syscall.Statfs(path, &fsStats); err != nil {
		return errors.Wrap(err, "unable to load filesystem information")
	}

	// Check if the filesystem is some variant of HFS. If not, then no
	// normalization is required.
	//
	// Well, that's not entirely true, but we have to take that position.
	// Apple's new APFS is normalization-preserving (while being normalization-
	// insensitive), which is great, except for cases where people convert HFS
	// volumes to APFS in-place (the default behavior for macOS 10.12 -> 10.13
	// upgrades), thus "locking-in" the decomposed normalization that HFS
	// enforced. Unfortunately there's not really a good heuristic for
	// determining which decomposed filenames on an APFS volume are due to HFS'
	// behavior. We could assume that all decomposed filenames on APFS volume at
	// like that for this reason, but (a) that's a pretty wild assumption,
	// especially as time goes on, and (b) as HFS dies off and more people
	// switch to APFS (likely though new volume creation), the cross-section of
	// cases where some HFS-induced decomposition is still haunting cross-
	// platform synchronization is going to become vanishingly small. People can
	// also just fix the problem by fixing the file name normalization once and
	// relying on APFS to preserve it.
	//
	// For a complete accounting of APFS' behavior, see the following article:
	// 	https://developer.apple.com/library/content/documentation/FileManagement/Conceptual/APFS_Guide/FAQ/FAQ.html
	// Search for "How does Apple File System handle filenames?" The behavior
	// was a little inconsistent and crazy during the initial deployment on iOS
	// 10.3, but it's pretty much settled down now, and was always sane on
	// macOS, where its deployment occurred later. Even during the crazy periods
	// though, the above logic regarding not doing normalization on APFS still
	// stands.
	//
	// Anyway, we perform this check by checking if the filesystem type name
	// starts with "hfs". This is not the ideal way of checking for HFS volumes,
	// but unfortunately macOS' statvfs and statfs implementations are a bit
	// terrible. According to the man pages, the f_fsid field of the statvfs
	// structure is not meaningful, and it is seemingly not populated. The man
	// pages also say that the f_type field of the statfs structure is reserved,
	// but there is no documentation of its value. Before macOS 10.12, its value
	// was 17 for all HFS variants, but then it changed to 23. The only place
	// this value is available is in the XNU sources (xnu/bsd/vfs/vfs_conf.c),
	// and those aren't even available for 10.12 yet.
	//
	// Other people have solved this by checking for both:
	//  http://stackoverflow.com/questions/39350259
	//  https://trac.macports.org/ticket/52463
	//  https://github.com/jrk/afsctool/commit/1146c90
	//
	// But this is not robust, because it can break at any time with OS updates.
	// Thus, the only sensible recourse is to use f_fstypename field, which is
	// BARELY documented. I suspect this is what's being used by NSWorkspace's
	// getFileSystemInfoForPath... method.
	//
	// This check should cover all HFS variants.
	isHFS := fsStats.Fstypename[0] == 'h' &&
		fsStats.Fstypename[1] == 'f' &&
		fsStats.Fstypename[2] == 's'
	if !isHFS {
		return nil
	}

	// If this is an HFS volume, then we need to convert names to NFC
	// normalization.
	//
	// This conversion might not be perfect, because HFS actually uses a custom
	// variant of NFD, but my understanding is that it's just NFD with certain
	// CJK characters not decomposed. The exact normalization has evolved a lot
	// over time and is way under-documented, so it's difficult to say.
	//
	// This link has a lot of information:
	// 	https://bugzilla.mozilla.org/show_bug.cgi?id=703161
	//
	// In any case, converting to NFC should be a fairly decent approximation
	// because most text will be have been in NFC normalization before HFS
	// forcefully decomposed it. At the end of the day, though, it's just a
	// heuristic.
	for i, n := range names {
		names[i] = norm.NFC.String(n)
	}

	// Success.
	return nil
}
