package stream

import (
	"io"
)

type stream struct {
	io.Reader
	io.Writer
	closers []io.Closer
}

func New(reader io.Reader, writer io.Writer, closers ...io.Closer) io.ReadWriteCloser {
	return &stream{reader, writer, closers}
}

func (s *stream) Close() error {
	// Iterate through the closers, recording only the first error, if any.
	var firstError error
	for _, closer := range s.closers {
		if err := closer.Close(); err != nil && firstError == nil {
			firstError = err
		}
	}
	return firstError
}
