package rsync

import (
	"hash"
	"io"
)

// TODO: Remove these methods once we enable engine re-use in endpoints.

func Signature(base io.Reader) ([]BlockHash, error) {
	engine := NewDefaultEngine()
	return engine.Signature(base)
}

func BytesSignature(base []byte) []BlockHash {
	engine := NewDefaultEngine()
	return engine.BytesSignature(base)
}

func Deltafy(target io.Reader, baseSignature []BlockHash, transmit OperationTransmitter) error {
	engine := NewDefaultEngine()
	return engine.Deltafy(target, baseSignature, transmit)
}

func DeltafyBytes(target []byte, baseSignature []BlockHash) []Operation {
	engine := NewDefaultEngine()
	return engine.DeltafyBytes(target, baseSignature)
}

func Patch(destination io.Writer, base io.ReadSeeker, receive OperationReceiver, digest hash.Hash) error {
	engine := NewDefaultEngine()
	return engine.Patch(destination, base, receive, digest)
}

func PatchBytes(base []byte, delta []Operation, digest hash.Hash) ([]byte, error) {
	engine := NewDefaultEngine()
	return engine.PatchBytes(base, delta, digest)
}
