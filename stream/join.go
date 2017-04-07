package stream

import (
	"io"
)

type joinedStream struct {
	io.Reader
	io.Writer
}

func Join(reader io.Reader, writer io.Writer) io.ReadWriter {
	return &joinedStream{reader, writer}
}
