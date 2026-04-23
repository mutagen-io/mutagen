package must

import (
	"fmt"
	"io"
	"net"
	"os"

	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func Fprint(w io.Writer, logger *logging.Logger, a ...any) {
	s := fmt.Sprint(a...)
	n, err := fmt.Fprint(w, s)
	if err != nil {
		logger.Warnf("Unable to Fprint '%s'; %s", s, err.Error())
	}
	if n < len(s) {
		logger.Warnf("Unable to Fprint all of '%s'; printed only %d of %d bytes", s, n, len(s))
	}
}

func Close(c io.Closer, logger *logging.Logger) {
	err := c.Close()
	if err != nil {
		logger.Warnf("Unable to close: %s", err.Error())
	}
}

func Serve(ws interface{ Serve(net.Listener) error }, nl net.Listener, logger *logging.Logger) {
	err := ws.Serve(nl)
	if err != nil {
		logger.Warnf("Unable to serve '%s': %s", nl.Addr(), err.Error())
	}
}

func WriteString(ws interface{ WriteString(string) (int, error) }, s string, logger *logging.Logger) {
	n, err := ws.WriteString(s)
	if err != nil {
		logger.Warnf("Unable to write string '%s': %s", s, err.Error())
	}
	if n < len(s) {
		logger.Warnf("Unable to write all of string '%s'; only wrote %d of %d bytes", s, n, len(s))
	}
}

func CloseWrite(cw interface{ CloseWrite() error }, logger *logging.Logger) {
	err := cw.CloseWrite()
	if err != nil {
		logger.Warnf("Unable to CloseWrite: %s", err.Error())
	}
}

func Signal(s interface{ Signal(os.Signal) error }, sig os.Signal, logger *logging.Logger) {
	err := s.Signal(sig)
	if err != nil {
		logger.Warnf("Unable to signal '%s': %s", sig, err.Error())
	}
}

func Terminate(s interface{ Terminate() error }, logger *logging.Logger) {
	err := s.Terminate()
	if err != nil {
		logger.Warnf("Unable to terminate: %s", err.Error())
	}
}

func Finalize(s interface{ Finalize() error }, logger *logging.Logger) {
	err := s.Finalize()
	if err != nil {
		logger.Warnf("Unable to finalize: %s", err.Error())
	}
}

func Remove(r interface{ Remove(string) error }, path string, logger *logging.Logger) {
	err := r.Remove(path)
	if err != nil {
		logger.Warnf("Unable to remove '%s': %s", path, err.Error())
	}
}

func Unlock(locker interface{ Unlock() error }, logger *logging.Logger) {
	err := locker.Unlock()
	if err != nil {
		logger.Warnf("Unable to unlock locker: %s", err.Error())
	}
}

func OSRemove(name string, logger *logging.Logger) {
	err := os.Remove(name)
	if err != nil {
		logger.Warnf("Unable to remove '%s': %s", name, err.Error())
	}
}

func Truncate(t interface{ Truncate(int64) error }, size int64, logger *logging.Logger) {
	err := t.Truncate(size)
	if err != nil {
		logger.Warnf("Unable to truncate to size %d: %s", size, err.Error())
	}
}

func Kill(s interface{ Kill() error }, logger *logging.Logger) {
	err := s.Kill()
	if err != nil {
		logger.Warnf("Unable to Kill: %s", err.Error())
	}
}

func IOCopy(dst io.Writer, src io.Reader, logger *logging.Logger) {
	_, err := io.Copy(dst, src)
	if err != nil {
		logger.Warnf("Unable to copy from source to destination: %s", err.Error())
	}
}

func CommandHelp(c *cobra.Command, logger *logging.Logger) {
	err := c.Help()
	if err != nil {
		logger.Warnf("Unable to help: %s", err.Error())
	}
}

func RemoveFile(rf interface{ RemoveFile(string) error }, name string, logger *logging.Logger) {
	err := rf.RemoveFile(name)
	if err != nil {
		logger.Warnf("Unable to remove file '%s': %s", name, err.Error())
	}
}

func Shutdown(sd interface{ Shutdown() error }, logger *logging.Logger) {
	err := sd.Shutdown()
	if err != nil {
		logger.Warnf("Unable to shutdown: %s", err.Error())
	}
}

func Flush(sd interface{ Flush() error }, logger *logging.Logger) {
	err := sd.Flush()
	if err != nil {
		logger.Warnf("Unable to flush: %s", err.Error())
	}
}

func Release(r interface{ Release() error }, logger *logging.Logger) {
	err := r.Release()
	if err != nil {
		logger.Warnf("Unable to release: %s", err.Error())
	}
}

func ProtoEncode(e interface {
	Encode(message proto.Message) error
}, message proto.Message, logger *logging.Logger) {
	err := e.Encode(message)
	if err != nil {
		logger.Warnf("Unable to proto encode %s: %s", message, err.Error())
	}
}

func Encode(e interface {
	Encode(e any) error
}, value any, logger *logging.Logger) {
	err := e.Encode(value)
	if err != nil {
		logger.Warnf("Unable to encode %v: %s", value, err.Error())
	}
}

func Succeed(err error, task string, logger *logging.Logger) {
	if err != nil {
		logger.Warnf("Unable to succeed at %s; %s", task, err.Error())
	}
}
