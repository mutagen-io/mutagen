package multiplex

import (
	"io"
)

// copyN is a replacement for using io.CopyBuffer with io.LimitedReader. It
// forwards a requested number of bytes from an io.Reader to an io.Writer using
// a provided buffer.
//
// This function is necessary to avoid an allocation on every data forwarding
// operation in the readMultiplexer (which would be very frequent). The problem
// is that we want to use io.CopyBuffer to avoid frequent buffer allocation, but
// the only way to get length-limited behavior in io.CopyBuffer is to use
// io.LimitedReader (this is also used by io.CopyN, which is even worse because
// it allocates a new copy buffer each time). Unfortunately, because it's being
// used to satisfy an interface, the io.LimitedReader has to be allocated on the
// heap (https://github.com/golang/go/issues/19361) (this is confirmed by escape
// analysis). If this was ever fixed, we could use io.LimitedReader in
// conjunction with io.CopyBuffer (though that's not as easy as it seems on its
// surface - e.g. io.Reader is allowed to return io.EOF along with a non-0 byte
// count, and io.CopyBuffer will gobble this up, so we might see a "successful"
// read without any error, and then we'd be relying on the reader to return
// io.EOF again on the next call, but the behavior there is undefined).
func copyN(dst io.Writer, src io.Reader, n int64, buffer []byte) (int64, error) {
	// Count the number of bytes we've copied.
	var copied int64
	var err error

	// Loop while there is data left to copy.
	for n > 0 {
		// Restrict our buffer to deal with what remains.
		if int64(len(buffer)) > n {
			buffer = buffer[:n]
		}

		// Perform a read.
		read, readErr := src.Read(buffer)

		// If any bytes were read, forward them. Any errors which occur here are
		// terminal. We also let write errors take precedence over read errors,
		// which is a somewhat arbitrary decision in some cases and a more
		// reasonable decision in others, but in any case it keeps consistency
		// with the behavior of the io.copyBuffer function.
		if read > 0 {
			written, writeErr := dst.Write(buffer[:read])
			if written > 0 {
				copied += int64(written)
				n -= int64(written)
			}
			if writeErr != nil {
				err = writeErr
				break
			}
			// TODO: This check should not be necessary since io.Writer objects
			// are required to return a non-nil error if they write less than
			// the requested number of bytes, but the io.copyBuffer function
			// makes this check because it was written before io.Writer behavior
			// was fully flushed out. It might be interesting to try to bench
			// the difference with this check removed (it's essentially another
			// comparison on a hot path). I doubt the Go developers would remove
			// the behavior at this point though, because it would probably
			// expose a lot of faulty io.Writer implementations. Anyway, for the
			// same reason, I'll leave it in. It also technically serves as a
			// check that io.Writer doesn't return MORE than the requested
			// number of bytes, though it returns the wrong error in that case.
			if written != read {
				err = io.ErrShortWrite
				break
			}
		}

		// Handle read errors.
		if readErr != nil {
			err = readErr
			break
		}
	}

	// Done.
	return copied, err
}
