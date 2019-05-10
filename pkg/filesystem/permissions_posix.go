// +build !windows

package filesystem

import (
	"os"
	userpkg "os/user"
	"strconv"

	"github.com/pkg/errors"
)

// OwnershipSpecification is an opaque type that encodes specification of file
// and/or directory ownership.
type OwnershipSpecification struct {
	// ownerID encodes the POSIX user ID associated with the ownership
	// specification. A value of -1 indicates the absence of specification. The
	// availability of -1 as a sentinel value for omission is guaranteed by the
	// POSIX definition of chmod.
	ownerID int
	// groupID encodes the POSIX user ID associated with the ownership
	// specification. A value of -1 indicates the absence of specification. The
	// availability of -1 as a sentinel value for omission is guaranteed by the
	// POSIX definition of chmod.
	groupID int
}

// NewOwnershipSpecification parsers owner and group specifications and resolves
// their system-level identifiers.
func NewOwnershipSpecification(owner, group string) (*OwnershipSpecification, error) {
	// Attempt to parse and look up owner user, if specified.
	ownerID := -1
	if owner != "" {
		switch kind, identifier := ParseOwnershipIdentifier(owner); kind {
		case OwnershipIdentifierKindInvalid:
			return nil, errors.New("invalid user specification")
		case OwnershipIdentifierKindPOSIXID:
			if _, err := userpkg.LookupId(identifier); err != nil {
				return nil, errors.Wrap(err, "unable to lookup user by ID")
			} else if u, err := strconv.Atoi(identifier); err != nil {
				return nil, errors.Wrap(err, "unable to convert user ID to numeric value")
			} else if u < 0 {
				return nil, errors.New("negative user ID")
			} else {
				ownerID = u
			}
		case OwnershipIdentifierKindWindowsSID:
			return nil, errors.New("Windows SIDs not supported on POSIX systems")
		case OwnershipIdentifierKindName:
			if userObject, err := userpkg.Lookup(identifier); err != nil {
				return nil, errors.Wrap(err, "unable to lookup user by ID")
			} else if u, err := strconv.Atoi(userObject.Uid); err != nil {
				return nil, errors.Wrap(err, "unable to convert user ID to numeric value")
			} else if u < 0 {
				return nil, errors.New("negative user ID retrieved")
			} else {
				ownerID = u
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
			} else if g < 0 {
				return nil, errors.New("negative group ID")
			} else {
				groupID = g
			}
		case OwnershipIdentifierKindWindowsSID:
			return nil, errors.New("Windows SIDs not supported on POSIX systems")
		case OwnershipIdentifierKindName:
			if groupObject, err := userpkg.LookupGroup(identifier); err != nil {
				return nil, errors.Wrap(err, "unable to lookup group by ID")
			} else if g, err := strconv.Atoi(groupObject.Gid); err != nil {
				return nil, errors.Wrap(err, "unable to convert group ID to numeric value")
			} else if g < 0 {
				return nil, errors.New("negative group ID retrieved")
			} else {
				groupID = g
			}
		default:
			panic("unhandled ownership identifier kind")
		}
	}

	// Success.
	return &OwnershipSpecification{
		ownerID: ownerID,
		groupID: groupID,
	}, nil
}

// SetPermissionsByPath sets the permissions on the content at the specified
// path. Ownership information is set first, followed by permissions extracted
// from the mode using ModePermissionsMask. Ownership setting can be skipped
// completely by providing a nil OwnershipSpecification or a specification with
// both components unset. An OwnershipSpecification may also include only
// certain components, in which case only those components will be set.
// Permission setting can be skipped by providing a mode value that yields 0
// after permission bit masking.
func SetPermissionsByPath(path string, ownership *OwnershipSpecification, mode Mode) error {
	// Set ownership information, if specified.
	if ownership != nil && (ownership.ownerID != -1 || ownership.groupID != -1) {
		if err := os.Chown(path, ownership.ownerID, ownership.groupID); err != nil {
			return errors.Wrap(err, "unable to set ownership information")
		}
	}

	// Set permissions, if specified.
	mode = mode & ModePermissionsMask
	if mode != 0 {
		if err := os.Chmod(path, os.FileMode(mode)); err != nil {
			return errors.Wrap(err, "unable to set permission bits")
		}
	}

	// Success.
	return nil
}
