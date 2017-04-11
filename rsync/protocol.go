package rsync

// request encodes a batch request.
type request struct {
	// Paths are the requested paths.
	Paths []string
	// Signatures are the corresponding base signatures.
	Signatures [][]BlockHash
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
