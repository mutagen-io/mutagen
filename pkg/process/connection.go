package process

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// Stream implements io.ReadWriteCloser around the standard input/output of a
// process. It is "closed" by terminating the underlying process. It supports an
// optional "kill delay" which tells the Close method to wait (up to the
// specified duration) for the process to exit on its own before killing it.
type Stream struct {
	// process is the underlying process.
	process *exec.Cmd
	// standardOutput is the source for process output data.
	standardOutput io.Reader
	// standardInput is the destination for process input data.
	standardInput io.Writer
	// killDelayLock restricts access the kill delay parameter.
	killDelayLock sync.Mutex
	// killDelay specifies the duration that the stream should wait for the
	// underlying process to exit on its own before killing the process.
	killDelay time.Duration
}

// NewStream creates a new stream (io.ReadWriteCloser) by wraping a command
// object. It must be called before the corresponding process is started, while
// the resulting stream must only be used after the corresponding process is
// started. This function will panic if killDelay is negative.
func NewStream(process *exec.Cmd, killDelay time.Duration) (*Stream, error) {
	// Validate the kill delay time.
	if killDelay < 0 {
		panic("negative kill delay specified")
	}

	// Redirect the process' standard input.
	standardInput, err := process.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("unable to redirect process input: %w", err)
	}

	// Redirect the process' standard output.
	standardOutput, err := process.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("unable to redirect process output: %w", err)
	}

	// Create the result.
	return &Stream{
		process:        process,
		standardOutput: standardOutput,
		standardInput:  standardInput,
		killDelay:      killDelay,
	}, nil
}

// Read implements io.Reader.Read.
func (s *Stream) Read(buffer []byte) (int, error) {
	return s.standardOutput.Read(buffer)
}

// Write implements io.Writer.Write.
func (s *Stream) Write(buffer []byte) (int, error) {
	return s.standardInput.Write(buffer)
}

// SetKillDelay sets the kill delay for the stream. This function will panic if
// killDelay is negative. This method is safe to call concurrently with Close,
// though if called concurrently, there is no guarantee that the new kill delay
// will be set before Close checks its value.
func (s *Stream) SetKillDelay(killDelay time.Duration) {
	// Validate the kill delay time.
	if killDelay < 0 {
		panic("negative kill delay specified")
	}

	// Lock and defer release of the kill delay lock.
	s.killDelayLock.Lock()
	defer s.killDelayLock.Unlock()

	// Set the kill delay.
	s.killDelay = killDelay
}

// Close closes the stream by terminating the underlying process and waiting for
// it to exit. This is the only portable way to unblock input/output streams, as
// many platforms will block closure of an OS pipe if there are pending read or
// write operations.
//
// If a non-negative/non-zero kill delay has been specified, then this this
// method will wait (up to the specified duration) for the process to exit on
// its own before issuing a kill request. By the time this method returns, the
// underlying process is guaranteed to no longer be running.
func (s *Stream) Close() error {
	// Extract the current kill delay.
	s.killDelayLock.Lock()
	killDelay := s.killDelay
	s.killDelayLock.Unlock()

	// Start a background Goroutine that will wait for the process to exit and
	// return the wait result.
	waitResults := make(chan error, 1)
	go func() {
		waitResults <- s.process.Wait()
	}()

	// If a kill delay has been specified, then wait (up to the specified
	// duration) for the process to exit on its own.
	if killDelay > 0 {
		killDelayTimer := time.NewTimer(killDelay)
		select {
		case err := <-waitResults:
			killDelayTimer.Stop()
			if err != nil {
				return fmt.Errorf("process wait failed: %w", err)
			}
			return nil
		case <-killDelayTimer.C:
		}
	}

	// Send a termination signal to the process. On Windows, we use the Kill
	// method since it has to use TerminateProcess to signal termination. On
	// POSIX, we use the Signal method with SIGTERM, because the Kill method
	// sends SIGKILL and this can lead to zombie processes when the SIGKILL
	// isn't (or (probably) can't be) forwarded to child processes. This was the
	// cause of mutagen-io/mutagen#223. SIGTERM is also a more idiomatic way of
	// doing this, though it does come at the cost of (a) relying on processes
	// not ignoring it and (b) requiring processes to perform proper signal
	// forwarding all the way down the process tree.
	//
	// TODO: We might be able to solve issue (b) by using killpg instead, and we
	// could solve both issues with killpg and SIGKILL, though there's no
	// guarantee that child processes aren't spawning off separate process
	// groups anyway. We might also want to switch to grouping processes as jobs
	// on Windows and performing similar group termination, though that's quite
	// complicated. We don't want to end up implementing init just to handle
	// child processes.
	//
	// NOTE: We don't handle errors here, because there's not much we can do
	// with the information. We need to guarantee that, by the time this method
	// returns, the process is no longer running. That will be enforced by our
	// indefinite wait in the return statement, but it's possible that the
	// termination signal could fail, and that the process could run
	// indefinitely. That's highly unlikely though, and it's safer to block
	// indefinitely in that case than to return with the process still running.
	if runtime.GOOS == "windows" {
		s.process.Process.Kill()
	} else {
		s.process.Process.Signal(syscall.SIGTERM)
	}

	// Wait for the wait operation to complete.
	if err := <-waitResults; err != nil {
		return fmt.Errorf("process wait failed: %w", err)
	}

	// Success.
	return nil
}
