package filesystem

import (
	userpkg "os/user"
	"path/filepath"

	"github.com/pkg/errors"

	"golang.org/x/sys/windows"

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

// CopyPermissions copies ownership and permission information from a source
// file to a target file. On Windows systems, this includes copying the owner,
// primary group, and DACL.
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

	// Query permission information, including user ownership, group ownership,
	// and DACL. If successful, defer the release of the securityDescriptor,
	// which provides the memory backing for the other parameters.
	var userSID, groupSID *windows.SID
	var dacl, securityDescriptor windows.Handle
	if err := aclapi.GetNamedSecurityInfo(
		filepath.Join(sourceDirectory.file.Name(), sourceName),
		aclapi.SE_FILE_OBJECT,
		aclapi.OWNER_SECURITY_INFORMATION|aclapi.GROUP_SECURITY_INFORMATION|aclapi.DACL_SECURITY_INFORMATION,
		&userSID,
		&groupSID,
		&dacl,
		nil,
		&securityDescriptor,
	); err != nil {
		return errors.Wrap(err, "unable to read source file permissions")
	}
	defer windows.LocalFree(securityDescriptor)

	// Set permission information.
	if err := aclapi.SetNamedSecurityInfo(
		filepath.Join(targetDirectory.file.Name(), targetName),
		aclapi.SE_FILE_OBJECT,
		aclapi.OWNER_SECURITY_INFORMATION|aclapi.GROUP_SECURITY_INFORMATION|aclapi.DACL_SECURITY_INFORMATION,
		userSID,
		groupSID,
		dacl,
		0,
	); err != nil {
		return errors.Wrap(err, "unable to read source file permissions")
	}

	// Success.
	return nil
}
