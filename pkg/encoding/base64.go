package encoding

import (
	"encoding/base64"
)

// EncodeBase64 is shorthand for base64.RawURLEncoding.EncodeToString
func EncodeBase64(value []byte) string {
	return base64.RawURLEncoding.EncodeToString(value)
}

// DecodeBase64 is shorthand for base64.RawURLEncoding.DecodeString.
func DecodeBase64(value string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(value)
}
