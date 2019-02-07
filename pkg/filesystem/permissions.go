package filesystem

import (
	"strings"
)

// OwnershipIdentifierKind specifies the type of an identifier provided for
// ownership specification.
type OwnershipIdentifierKind uint8

const (
	// OwnershipIdentifierKindInvalid specifies an invalid identifier kind.
	OwnershipIdentifierKindInvalid OwnershipIdentifierKind = iota
	// OwnershipIdentifierKindPOSIXID specifies a POSIX user or group ID.
	OwnershipIdentifierKindPOSIXID
	// OwnershipIdentifierKindWindowsSID specifies a Windows SID.
	OwnershipIdentifierKindWindowsSID
	// OwnershipIdentifierKindWindowsSID specifies a name-based identifier.
	OwnershipIdentifierKindName
)

// isValidPOSIXID determines whether or not a string represents a valid POSIX
// user or group ID.
func isValidPOSIXID(value string) bool {
	// Ensure that the value is non-empty.
	if len(value) == 0 {
		return false
	}

	// As a special case, allow 0 for root specification. We disallow numeric
	// values that start with a 0 below, so we have to allow this specification
	// explicitly.
	if value == "0" {
		return true
	}

	// Ensure that the string starts with a non-0 digit (just so we know that
	// we're not going to parse in octal mode) and that all digits are between
	// 0 and 9.
	first := true
	for _, r := range value {
		if first {
			if !('1' <= r && r <= '9') {
				return false
			}
			first = false
		} else {
			if !('0' <= r && r <= '9') {
				return false
			}
		}
	}

	// Success.
	return true
}

// isValidWindowsSID determines whether or not a string represents a valid
// Windows SID. It does not validate that the string resolves to a valid SID,
// merely that the formatting is plausible.
func isValidWindowsSID(value string) bool {
	// Ensure that the value is non-empty.
	if len(value) == 0 {
		return false
	}

	// Unfortunately there's not much validation we can do beyond this because
	// Windows supports string constants (e.g. "BA" resolves to built-in
	// administrators) that we have no way of validating (and they seem to
	// change by syste language). So we just assume that the identifier is valid
	// for now. It will fail later on resolution if it's invalid.
	return true
}

// ParseOwnershipIdentifier parses an identifier provided for ownership
// specification.
func ParseOwnershipIdentifier(specification string) (OwnershipIdentifierKind, string) {
	// Ensure that the specification is non-empty.
	if len(specification) == 0 {
		return OwnershipIdentifierKindInvalid, ""
	}

	// Check if this is a POSIX ID.
	if strings.HasPrefix(specification, "id:") {
		if value := specification[3:]; !isValidPOSIXID(value) {
			return OwnershipIdentifierKindInvalid, ""
		} else {
			return OwnershipIdentifierKindPOSIXID, value
		}
	}

	// Check if this is a Windows SID.
	if strings.HasPrefix(specification, "sid:") {
		if value := specification[4:]; !isValidWindowsSID(value) {
			return OwnershipIdentifierKindInvalid, ""
		} else {
			return OwnershipIdentifierKindWindowsSID, value
		}
	}

	// Otherwise assume this is a name-based specification. If it's not valid,
	// it will fail during lookup. Unfortunately there isn't a good set of
	// cross-platform validations that we can perform to ensure that the name is
	// valid. On POSIX it's governed by NAME_REGEX, which is system-dependent
	// and not accessible except via cgo. On Windows, I think it's more of a
	// trial-and-error check.
	return OwnershipIdentifierKindName, specification
}
