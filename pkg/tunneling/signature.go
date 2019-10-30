package tunneling

import (
	"crypto/hmac"
	"hash"
)

// signOffer computes an HMAC offer signature.
func signOffer(offer []byte, hash func() hash.Hash, secret []byte) []byte {
	signer := hmac.New(hash, secret)
	signer.Write(offer)
	return signer.Sum(nil)
}

// verifyOfferSignature verifies an HMAC offer signature without leaking timing
// information.
func verifyOfferSignature(offer []byte, hash func() hash.Hash, secret, signature []byte) bool {
	expected := signOffer(offer, hash, secret)
	return hmac.Equal(signature, expected)
}
