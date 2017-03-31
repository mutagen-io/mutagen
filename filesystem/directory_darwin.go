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
	// We perform this check by checking if the filesystem type name starts with
	// "hfs". This is not the ideal way of checking for HFS volumes, but
	// unfortunately macOS' statvfs and statfs implementations are a bit
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
	// But this doesn't seem ideal, especially with APFS coming soon. Thus, the
	// only sensible recourse is to use f_fstypename field, which is BARELY
	// documented. I suspect this is what's being used by NSWorkspace's
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
	// forcefully decomposed it. It's admittedly not perfect, though.
	//
	// Once Apple decides to take HFS out behind the shed and shoot it, this
	// should be less of an issue. The new Apple File System (APFS) is a bit
	// better - it just treats file names as bags-of-bytes (just like other
	// filesystems). The one problem is that Apple is doing in-place conversion
	// of HFS volumes as they're rolling out APFS, and it's not clear if they're
	// converting to NFC as part of that (I seriously doubt it). If they're not,
	// we'll probably still need to perform this normalization.
	//
	// TODO: Once APFS rolls out, if the converter doesn't perform NFC
	// normalization, then I'd just make the NFC normalization unconditional on
	// macOS until HFS has become a thing of the past (at which point I'd remove
	// the normalization). Perhaps in the intermediate period, we can make the
	// normalization configurable.
	for i, n := range names {
		names[i] = norm.NFC.String(n)
	}

	// Success.
	return nil
}
