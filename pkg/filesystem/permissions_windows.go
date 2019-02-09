package filesystem

import (
	"os"
	userpkg "os/user"

	"github.com/pkg/errors"

	"golang.org/x/sys/windows"

	"github.com/hectane/go-acl"
	aclapi "github.com/hectane/go-acl/api"
)

// OwnershipSpecification is an opaque type that encodes specification of file
// and/or directory ownership.
type OwnershipSpecification struct {
	// userSid encodes the Windows user SID associated with the ownership
	// specification. A nil value indicates the absence of specification.
	userSID *windows.SID
	// groupSid encodes the Windows group SID associated with the ownership
	// specification. A nil value indicates the absence of specification.
	groupSID *windows.SID
}

// NewOwnershipSpecification parsers user and group specifications and resolves
// their system-level identifiers.
func NewOwnershipSpecification(user, group string) (*OwnershipSpecification, error) {
	// Attempt to parse and look up user, if specified.
	var userSID *windows.SID
	if user != "" {
		switch kind, identifier := ParseOwnershipIdentifier(user); kind {
		case OwnershipIdentifierKindInvalid:
			return nil, errors.New("invalid user specification")
		case OwnershipIdentifierKindPOSIXID:
			return nil, errors.New("POSIX IDs not supported on Windows systems")
		case OwnershipIdentifierKindWindowsSID:
			if userObject, err := userpkg.LookupId(identifier); err != nil {
				return nil, errors.Wrap(err, "unable to lookup user by ID")
			} else if u, err := windows.StringToSid(userObject.Uid); err != nil {
				return nil, errors.Wrap(err, "unable to convert SID string to object")
			} else {
				userSID = u
			}
		case OwnershipIdentifierKindName:
			if userObject, err := userpkg.Lookup(identifier); err != nil {
				return nil, errors.Wrap(err, "unable to lookup user by ID")
			} else if u, err := windows.StringToSid(userObject.Uid); err != nil {
				return nil, errors.Wrap(err, "unable to convert SID string to object")
			} else {
				userSID = u
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
			if groupObject, err := userpkg.LookupId(identifier); err != nil {
				return nil, errors.Wrap(err, "unable to lookup group by ID")
			} else if g, err := windows.StringToSid(groupObject.Uid); err != nil {
				return nil, errors.Wrap(err, "unable to convert SID string to object")
			} else {
				groupSID = g
			}
		case OwnershipIdentifierKindName:
			if groupObject, err := userpkg.Lookup(identifier); err != nil {
				return nil, errors.Wrap(err, "unable to lookup group by ID")
			} else if g, err := windows.StringToSid(groupObject.Uid); err != nil {
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
		userSID:  userSID,
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
	if ownership != nil && (ownership.userSID != nil || ownership.groupSID != nil) {
		// Compute the information that we're going to set.
		var information uint32
		if ownership.userSID != nil {
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
			ownership.userSID,
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
		if err := acl.Chmod(path, os.FileMode(mode)); err != nil {
			return errors.Wrap(err, "unable to set permission bits")
		}
	}

	// Success.
	return nil
}
