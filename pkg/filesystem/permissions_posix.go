// +build !windows

package filesystem

import (
	userpkg "os/user"
	"strconv"

	"github.com/pkg/errors"

	"golang.org/x/sys/unix"
)

// OwnershipSpecification is an opaque type that encodes specification of file
// and/or directory ownership.
type OwnershipSpecification struct {
	// userID encodes the POSIX user ID associated with the ownership
	// specification. A value of -1 indicates the absence of specification. The
	// availability of -1 as a sentinel value for omission is guaranteed by the
	// POSIX definition of chmod.
	userID int
	// groupID encodes the POSIX user ID associated with the ownership
	// specification. A value of -1 indicates the absence of specification. The
	// availability of -1 as a sentinel value for omission is guaranteed by the
	// POSIX definition of chmod.
	groupID int
}

// NewOwnershipSpecification parsers user and group specifications and resolves
// their system-level identifiers.
func NewOwnershipSpecification(user, group string) (*OwnershipSpecification, error) {
	// Attempt to parse and look up user, if specified.
	userID := -1
	if user != "" {
		switch kind, identifier := ParseOwnershipIdentifier(user); kind {
		case OwnershipIdentifierKindInvalid:
			return nil, errors.New("invalid user specification")
		case OwnershipIdentifierKindPOSIXID:
			if _, err := userpkg.LookupId(identifier); err != nil {
				return nil, errors.Wrap(err, "unable to lookup user by ID")
			} else if u, err := strconv.Atoi(identifier); err != nil {
				return nil, errors.Wrap(err, "unable to convert user ID to numeric value")
			} else {
				userID = u
			}
		case OwnershipIdentifierKindWindowsSID:
			return nil, errors.New("Windows SIDs not supported on POSIX systems")
		case OwnershipIdentifierKindName:
			if userObject, err := userpkg.Lookup(identifier); err != nil {
				return nil, errors.Wrap(err, "unable to lookup user by ID")
			} else if u, err := strconv.Atoi(userObject.Uid); err != nil {
				return nil, errors.Wrap(err, "unable to convert user ID to numeric value")
			} else {
				userID = u
			}
		default:
			panic("unhandled ownership identifier kind")
		}
	}

	// Attempt to parse and look up group, if specified.
	groupID := -1
	if group != "" {
		switch kind, identifier := ParseOwnershipIdentifier(group); kind {
		case OwnershipIdentifierKindInvalid:
			return nil, errors.New("invalid group specification")
		case OwnershipIdentifierKindPOSIXID:
			if _, err := userpkg.LookupGroupId(identifier); err != nil {
				return nil, errors.Wrap(err, "unable to lookup group by ID")
			} else if g, err := strconv.Atoi(identifier); err != nil {
				return nil, errors.Wrap(err, "unable to convert group ID to numeric value")
			} else {
				groupID = g
			}
		case OwnershipIdentifierKindWindowsSID:
			return nil, errors.New("Windows SIDs not supported on POSIX systems")
		case OwnershipIdentifierKindName:
			if groupObject, err := userpkg.Lookup(identifier); err != nil {
				return nil, errors.Wrap(err, "unable to lookup group by ID")
			} else if g, err := strconv.Atoi(groupObject.Gid); err != nil {
				return nil, errors.Wrap(err, "unable to convert group ID to numeric value")
			} else {
				groupID = g
			}
		default:
			panic("unhandled ownership identifier kind")
		}
	}

	// Success.
	return &OwnershipSpecification{
		userID:  userID,
		groupID: groupID,
	}, nil
}

// CopyPermissions copies ownership and permission information from a source
// file to a target file. On POSIX systems, this includes copying user ID, group
// ID, and permission bits (including the setuid bit, the setgid bit, and the
// sticky bit).
func CopyPermissions(
	sourceDirectory *Directory, sourceName string,
	targetDirectory *Directory, targetName string,
) error {
	// Verify that both names are valid.
	if err := ensureValidName(sourceName); err != nil {
		return errors.Wrap(err, "source name invalid")
	} else if err = ensureValidName(targetName); err != nil {
		return errors.Wrap(err, "target name invalid")
	}

	// Grab metadata (which will include ownership and permissions) for the
	// source file.
	var metadata unix.Stat_t
	if err := fstatat(sourceDirectory.descriptor, sourceName, &metadata, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return errors.Wrap(err, "unable to read source file metadata")
	}

	// Set ownership information on the target file.
	if err := fchownat(targetDirectory.descriptor, targetName, int(metadata.Uid), int(metadata.Gid), unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return errors.Wrap(err, "unable to set ownership information on target file")
	}

	// Set permissions on the target file, including the setuid bit, the setgid
	// bit, and the sticky bit.
	permissions := metadata.Mode & (unix.S_IRWXU | unix.S_IRWXG | unix.S_IRWXO | unix.S_ISUID | unix.S_ISGID | unix.S_ISVTX)
	if err := fchmodat(targetDirectory.descriptor, targetName, uint32(permissions), unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return errors.Wrap(err, "unable to set permission bits")
	}

	// Success.
	return nil
}
