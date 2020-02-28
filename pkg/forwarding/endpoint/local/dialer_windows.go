package local

import (
	"context"
	"net"
	"time"

	"github.com/Microsoft/go-winio"

	"github.com/mutagen-io/mutagen/pkg/forwarding"
)

// npipeCloseWriterConn adapts a net.Conn to support CloseWrite as a no-op. It
// is used for non-message mode Windows named pipes, which don't natively
// support CloseWrite operations.
type npipeCloseWriterConn struct {
	// connection is the underlying connection.
	connection net.Conn
}

// Read implements net.Conn.Read.
func (c *npipeCloseWriterConn) Read(buffer []byte) (int, error) {
	return c.connection.Read(buffer)
}

// Write implements net.Conn.Write.
func (c *npipeCloseWriterConn) Write(data []byte) (int, error) {
	return c.connection.Write(data)
}

// CloseWrite implements CloseWriter.CloseWrite.
func (c *npipeCloseWriterConn) CloseWrite() error {
	return nil
}

// Close implements net.Conn.Close.
func (c *npipeCloseWriterConn) Close() error {
	return c.connection.Close()
}

// LocalAddr implements net.Conn.LocalAddr.
func (c *npipeCloseWriterConn) LocalAddr() net.Addr {
	return c.connection.LocalAddr()
}

// RemoteAddr implements net.Conn.RemoteAddr.
func (c *npipeCloseWriterConn) RemoteAddr() net.Addr {
	return c.connection.RemoteAddr()
}

// SetDeadline implements net.Conn.SetDeadline.
func (c *npipeCloseWriterConn) SetDeadline(t time.Time) error {
	return c.SetDeadline(t)
}

// SetReadDeadline implements net.Conn.SetReadDeadline.
func (c *npipeCloseWriterConn) SetReadDeadline(t time.Time) error {
	return c.SetReadDeadline(t)
}

// SetWriteDeadline implements net.Conn.SetWriteDeadline.
func (c *npipeCloseWriterConn) SetWriteDeadline(t time.Time) error {
	return c.SetWriteDeadline(t)
}

// dialWindowsNamedPipe performs a named pipe dialing operation on Windows. The
// dialing operation (but not any resulting connection) is limited to the
// lifecycle of the provided context.
func dialWindowsNamedPipe(ctx context.Context, address string) (net.Conn, error) {
	// Perform the dialing operation.
	connection, err := winio.DialPipeContext(ctx, address)
	if err != nil {
		return nil, err
	}

	// Check if the connection supports CloseWrite natively. This will only be
	// the case if the target named pipe is in message mode (in which case the
	// go-winio package will encode write closure as a zero-length message).
	// Whether or not the target connection will understand go-winio's semantics
	// is another question (if it doesn't, it'll just ignore these messages),
	// but fortunately Docker (our primary use case for named pipes) uses
	// go-winio with message mode named pipes and thus understands these
	// semantics just fine.
	if _, ok := connection.(forwarding.CloseWriter); ok {
		return connection, err
	}

	// If the connection doesn't support CloseWrite (which is required by our
	// forwarding logic), then wrap it in an adapter that implements CloseWrite
	// as a no-op.
	// TODO: It's a little unclear what the best implementation strategy is
	// here. Using a no-op CloseWrite method is a valid option, but aliasing it
	// to Close (and ensuring Close is only called once) might also be a valid
	// option. Since named pipes don't have a "native" way of indicating write
	// closure, it seems like making CloseWrite a no-op would better align with
	// most use cases, but we might need additional real world feedback. We can
	// make this behavior configurable if the need arises. Fortunately the only
	// real-world use case is probably Docker, in which case this wrapping
	// doesn't enter into the picture.
	return &npipeCloseWriterConn{connection: connection}, nil
}
