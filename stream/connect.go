package stream

import (
	"io"
)

func Connect(first, second io.ReadWriter) {
	go io.Copy(first, second)
	go io.Copy(second, first)
}
