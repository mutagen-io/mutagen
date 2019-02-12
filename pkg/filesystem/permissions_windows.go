package filesystem

import (
	"os"
	userpkg "os/user"

	"github.com/pkg/errors"

	"golang.org/x/sys/windows"

	aclapi "github.com/hectane/go-acl/api"
)

// OwnershipSpecification is an opaque type that encodes specification of file
// and/or directory ownership.
type OwnershipSpecification struct {
	// ownerSID encodes the Windows owner SID associated with the ownership
	// specification. It may represent either a user SID or a group SID. A nil
	// value indicates the absence of specification.
	ownerSID *windows.SID
	// groupSid encodes the Windows group SID associated with the ownership
	// specification. A nil value indicates the absence of specification.
	groupSID *windows.SID
}

// NewOwnershipSpecification parsers owner and group specifications and resolves
// their system-level identifiers.
func NewOwnershipSpecification(owner, group string) (*OwnershipSpecification, error) {
	// Attempt to parse and look up owner, if specified. On Windows, an owner
	// can be either a user or a group.
	var ownerSID *windows.SID
	if owner != "" {
		switch kind, identifier := ParseOwnershipIdentifier(owner); kind {
		case OwnershipIdentifierKindInvalid:
			return nil, errors.New("invalid owner specification")
		case OwnershipIdentifierKindPOSIXID:
			return nil, errors.New("POSIX IDs not supported on Windows systems")
		case OwnershipIdentifierKindWindowsSID:
			// Verify that this SID represents either a user or a group.
			var retrievedSID string
			if userObject, err := userpkg.LookupId(identifier); err == nil {
				retrievedSID = userObject.Uid
			} else if groupObject, err := userpkg.LookupGroupId(identifier); err == nil {
				retrievedSID = groupObject.Gid
			} else {
				return nil, errors.New("unable to find user or group with specified owner SID")
			}

			// Convert the retrieved SID to a string.
			if s, err := windows.StringToSid(retrievedSID); err != nil {
				return nil, errors.Wrap(err, "unable to convert SID string to object")
			} else {
				ownerSID = s
			}
		case OwnershipIdentifierKindName:
			// Verify that this name represents either a user or a group and
			// retrieve the associated SID.
			var retrievedSID string
			if userObject, err := userpkg.Lookup(identifier); err == nil {
				retrievedSID = userObject.Uid
			} else if groupObject, err := userpkg.LookupGroup(identifier); err == nil {
				retrievedSID = groupObject.Gid
			} else {
				return nil, errors.New("unable to find user or group with specified owner name")
			}

			// Convert the retrieved SID to a string.
			if s, err := windows.StringToSid(retrievedSID); err != nil {
				return nil, errors.Wrap(err, "unable to convert SID string to object")
			} else {
				ownerSID = s
			}
		default:
			panic("unhandled ownership identifier kind")
		}
	}

	// Attempt to parse and look up group, if specified.
	var groupSID *windows.SID
	if group != "" {
		switch kind, identifier := ParseOwnershipIdentifier(group); kind {
		case OwnershipIdentifierKindInvalid:
			return nil, errors.New("invalid group specification")
		case OwnershipIdentifierKindPOSIXID:
			return nil, errors.New("POSIX IDs not supported on Windows systems")
		case OwnershipIdentifierKindWindowsSID:
			if groupObject, err := userpkg.LookupGroupId(identifier); err != nil {
				return nil, errors.Wrap(err, "unable to lookup group by ID")
			} else if g, err := windows.StringToSid(groupObject.Gid); err != nil {
				return nil, errors.Wrap(err, "unable to convert SID string to object")
			} else {
				groupSID = g
			}
		case OwnershipIdentifierKindName:
			if groupObject, err := userpkg.LookupGroup(identifier); err != nil {
				return nil, errors.Wrap(err, "unable to lookup group by ID")
			} else if g, err := windows.StringToSid(groupObject.Gid); err != nil {
				return nil, errors.Wrap(err, "unable to convert SID string to object")
			} else {
				groupSID = g
			}
		default:
			panic("unhandled ownership identifier kind")
		}
	}

	// Success.
	return &OwnershipSpecification{
		ownerSID: ownerSID,
		groupSID: groupSID,
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
	if ownership != nil && (ownership.ownerSID != nil || ownership.groupSID != nil) {
		// Compute the information that we're going to set.
		var information uint32
		if ownership.ownerSID != nil {
			information |= aclapi.OWNER_SECURITY_INFORMATION
		}
		if ownership.groupSID != nil {
			information |= aclapi.GROUP_SECURITY_INFORMATION
		}

		// Set the information.
		if err := aclapi.SetNamedSecurityInfo(
			path,
			aclapi.SE_FILE_OBJECT,
			information,
			ownership.ownerSID,
			ownership.groupSID,
			0,
			0,
		); err != nil {
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
