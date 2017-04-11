package rsync

// request encodes a stream request. It is used in the internal client/server
// protocol.
type request struct {
	// Path is the path of the file relative to the root.
	Path string
	// Signature is the signature of the base file at this path.
	Signature []BlockHash
}

// response encodes a stream response. It is used in the internal client/server
// protocol.
type response struct {
	// Done indicates that the response stream for this request is finished. If
	// set, there will be no operation in the response, but there may be an
	// error.
	Done bool
	// Operation is the next operation in the stream.
	Operation Operation
	// Error indicates that a non-terminal error has occurred. It will only be
	// present if Done is true.
	Error string
}
