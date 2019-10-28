package mutagenio

import (
	"errors"
	"fmt"

	"github.com/dgrijalva/jwt-go"
)

const (
	// schemeClaimName is the name of the scheme claim.
	schemeClaimName = "scheme"
	// TokenSchemeAPI is the scheme name for API tokens.
	TokenSchemeAPI = "api"
)

// ExtractTokenScheme parses a mutagen.io token (without validation) and returns
// the token scheme.
func ExtractTokenScheme(token string) (string, error) {
	// Set up the claims receiver.
	claims := make(jwt.MapClaims)

	// Perform parsing.
	parser := &jwt.Parser{}
	_, _, err := parser.ParseUnverified(token, claims)
	if err != nil {
		return "", fmt.Errorf("unable to parse token: %w", err)
	}

	// Extract the scheme and convert it to a string.
	if s, ok := claims[schemeClaimName]; !ok {
		return "", errors.New("no scheme in token")
	} else if scheme, ok := s.(string); !ok {
		return "", errors.New("token scheme has invalid type")
	} else if scheme == "" {
		return "", errors.New("empty token scheme")
	} else {
		return scheme, nil
	}
}
