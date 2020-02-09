package identifier

import (
	"errors"
	"regexp"
	"strings"

	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/random"
)

const (
	// PrefixSynchronization is the prefix used for synchronization session
	// identifiers.
	PrefixSynchronization = "sync"
	// PrefixForwarding is the prefix used for forwarding session identifiers.
	PrefixForwarding = "fwrd"
	// PrefixProject is the prefix used for project identifiers.
	PrefixProject = "proj"
	// PrefixPrompter is the prefix used for prompter identifiers.
	PrefixPrompter = "pmtr"

	// requiredPrefixLength is the required length for identifier prefixes.
	requiredPrefixLength = 4
	// collisionResistantLength is the number of random bytes needed to ensure
	// collision-resistance in an identifier.
	collisionResistantLength = 32
	// targetBase62Length is the target length for the Base62-encoded portion of
	// the identifier. This is set to the maximum possible length that a byte
	// array of collisionResistantLength bytes will take to encode in Base62
	// encoding. This length can be computed for n bytes using the formula
	// ceil(n*8*ln(2)/ln(62))).
	targetBase62Length = 43
)

// legacyMatcher is a regular expression that matches Mutagen's legacy
// identifiers (which are lowercase UUIDs).
var legacyMatcher = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")

// matcher is a regular expression that matches Mutagen's identifiers.
var matcher = regexp.MustCompile("^[a-z]{4}_[0-9a-zA-Z]{43}$")

// New generates a new collision-resistant identifier with the specified prefix.
// The prefix should have a length of RequiredPrefixLength.
func New(prefix string) (string, error) {
	// Ensure that the prefix length is correct.
	if len(prefix) != requiredPrefixLength {
		return "", errors.New("incorrect prefix length")
	}

	// Ensure that each prefix character is allowed.
	for _, r := range prefix {
		if !('a' <= r && r <= 'z') {
			return "", errors.New("invalid prefix character")
		}
	}

	// Create the random value.
	random, err := random.New(collisionResistantLength)
	if err != nil {
		return "", err
	}

	// Encode the random value using a Base62 encoding scheme. As a sanity
	// check, ensure that the encoded value doesn't exceed the target length.
	encoded := encoding.EncodeBase62(random)
	if len(encoded) > targetBase62Length {
		panic("encoded random data length longer than expected")
	}

	// Create a string builder.
	builder := &strings.Builder{}

	// Add the identifier prefix.
	builder.WriteString(prefix)

	// Add the separator.
	builder.WriteRune('_')

	// If the encoded value has a length less than the target length, then
	// left-pad it with 0s. Actually, we technically pad it using whatever the
	// zero value is in our Base62 alphabet, but that happens to be '0'.
	for i := targetBase62Length - len(encoded); i > 0; i-- {
		builder.WriteByte(encoding.Base62Alphabet[0])
	}

	// Write the encoded value.
	builder.WriteString(encoded)

	// Success.
	return builder.String(), nil
}

// IsValid determines whether or not a string is a valid identifier.
func IsValid(value string) bool {
	return matcher.MatchString(value) || legacyMatcher.MatchString(value)
}
