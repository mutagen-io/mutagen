package encoding

import (
	"github.com/eknkc/basex"
)

const (
	// Base62Alphabet is the alphabet used for Base62 encoding.
	Base62Alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// base62 is the Base62 encoder. It is safe for concurrent use.
var base62 *basex.Encoding

func init() {
	// Initialize the Base62 encoder.
	if encoding, err := basex.NewEncoding(Base62Alphabet); err != nil {
		panic("unable to initialize Base62 encoder")
	} else {
		base62 = encoding
	}
}

// EncodeBase62 performs Base62 encoding.
func EncodeBase62(value []byte) string {
	return base62.Encode(value)
}

// DecodeBase62 performs Base62 decoding.
func DecodeBase62(value string) ([]byte, error) {
	return base62.Decode(value)
}
