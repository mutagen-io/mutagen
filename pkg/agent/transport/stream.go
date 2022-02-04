package transport

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// Stream implements io.ReadWriteCloser using the standard input and output
// streams of an agent process, with closure implemented via termination
// signaling heuristics designed to shut down agent processes reliably. It
// guarantees that its Close method unblocks pending Read and Write calls. It
// also provides optional forwarding of the process' standard error stream.
type Stream struct {
	// process is the underlying process.
	process *exec.Cmd
	// standardInput is the process' standard input stream.
	standardInput io.WriteCloser
	// standardOutput is the process' standard output stream.
	standardOutput io.Reader
	// terminationDelayLock restricts access to terminationDelay.
	terminationDelayLock sync.Mutex
	// terminationDelay specifies the duration that the stream should wait for
	// the underlying process to exit on its own before performing termination.
	terminationDelay time.Duration
}

// NewStream creates a new Stream instance that wraps the specified command
// object. It must be called before the corresponding process is started and no
// other I/O redirection should be performed for the process. If this function
// fails, the command should be considered unusable. If standardErrorReceiver is
// non-nil, then the process' standard error output will be forwarded to it.
func NewStream(process *exec.Cmd, standardErrorReceiver io.Writer) (*Stream, error) {
	// Create a pipe to the process' standard input stream.
	standardInput, err := process.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("unable to redirect process input: %w", err)
	}

	// Create a pipe from the process' standard output stream.
	standardOutput, err := process.StdoutPipe()
	if err != nil {
		standardInput.Close()
		return nil, fmt.Errorf("unable to redirect process output: %w", err)
	}

	// If a standard error receiver has been specified, then create a pipe from
	// the process' standard error stream and forward it to the receiver. We do
	// this manually (instead of just assigning the receiver to process.Stderr)
	// to avoid golang/go#23019. We perform the same closure on the standard
	// error stream as os/exec's standard forwarding Goroutines, a fix designed
	// to avoid golang/go#10400.
	if standardErrorReceiver != nil {
		standardError, err := process.StderrPipe()
		if err != nil {
			standardInput.Close()
			standardOutput.Close()
			return nil, fmt.Errorf("unable to redirect process error output: %w", err)
		}
		go func() {
			io.Copy(standardErrorReceiver, standardError)
			standardError.Close()
		}()
	}

	// Create the result.
	return &Stream{
		process:        process,
		standardInput:  standardInput,
		standardOutput: standardOutput,
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

// SetTerminationDelay sets the termination delay for the stream. This method
// will panic if terminationDelay is negative. This method is safe to call
// concurrently with Close, though, if called concurrently, there is no
// guarantee that the new delay will be set in time for Close to use it.
func (s *Stream) SetTerminationDelay(terminationDelay time.Duration) {
	// Validate the kill delay time.
	if terminationDelay < 0 {
		panic("negative termination delay specified")
	}

	// Lock and defer release of the termination delay lock.
	s.terminationDelayLock.Lock()
	defer s.terminationDelayLock.Unlock()

	// Set the termination delay.
	s.terminationDelay = terminationDelay
}

// Close closes the process' streams and terminates the process using heuristics
// designed for agent transport processes. These heuristics are necessary to
// avoid the problem described in golang/go#23019 and experienced in
// mutagen-io/mutagen#223 and mutagen-io/mutagen-compose#11.
//
// First, if a non-negative, non-zero termination delay has been specified, then
// this method will wait (up to the specified duration) for the process to exit
// on its own. If the process exits on its own, then its standard input, output,
// and error streams are closed and this method returns.
//
// Second, the process' standard input stream will be closed. The process will
// then be given up to one second to exit on its own. If it does, then the
// standard output and error streams are closed and this method returns. Closure
// of the standard input stream is recognized by the Mutagen agent as indicating
// termination and should thus be sufficient to cause termination for transport
// processes that forward this closure correctly.
//
// Third, on POSIX systems only, the process will be sent a SIGTERM signal. The
// process will then be given up to one second to exit on its own. If it does,
// then the standard output and error streams are closed and this method
// returns. Reception of SIGTERM is also recognized by the Mutagen agent as
// indicating termination and should thus be sufficient to cause termination for
// transport processes that correctly forward this signal. Windows lacks a
// directly equivalent termination mechanism (the closest analog would be
// sending WM_CLOSE, but reception and processing of such a message may have
// unpredictable effects in different runtimes).
//
// Finally, the process will be sent a SIGKILL signal (on POSIX) or terminated
// via TerminateProcess (on Windows). This method will then wait for the process
// to exit before closing the standard output and error streams and returning.
//
// This method guarantees that, by the time it returns, the underlying transport
// process has terminated and its associated standard input, output, and error
// stream handles in the current process have been closed. The error returned by
// this function will be that returned by os/exec.Cmd.Wait. Note, however, that
// this method cannot guarantee that any or all child processes spawned by the
// transport process have terminated by the time this method returns. This is
// mostly due to operating system API limitations. Specifically, POSIX provides
// no away to restrict subprocesses to a single process group and therefore
// cannot guarantee that a call to killpg will reach all the subprocesses that
// have been spawned. Even if that were possible, there is no mechanism to wait
// for an entire process group to exit, and it's not well-defined exactly what
// signals or stream closures should be used to signal those processes anyway,
// because Mutagen is not privy to the internals of the transport process(es).
// Windows, while it does provide a "job" API for managing and terminating
// process hierarchies, is even less nuanced in its process signaling mechanism
// (essentially offering only the equivalent of SIGKILL) and it's thus even less
// clear how to signal termination there with arbitrary and opaque process
// hierarchies. We thus rely on a certain level of well-behavedness when it
// comes to transport processes. Specifically, we assume that they know how to
// correctly handle and forward standard input closure and SIGTERM signals, and
// that they'll terminate when their underlying agent process terminates.
func (s *Stream) Close() error {
	// Start a background Goroutine that will wait for the process to exit and
	// return the wait result. We'll rely on this call to Wait to close the
	// standard output and error streams. We don't have to worry about
	// golang/go#23019 in this case because we're only using pipes and thus Wait
	// doesn't have any internal copying Goroutines to wait on.
	waitResults := make(chan error, 1)
	go func() {
		waitResults <- s.process.Wait()
	}()

	// Start by waiting for the process to terminate on its own.
	s.terminationDelayLock.Lock()
	terminationDelay := s.terminationDelay
	s.terminationDelayLock.Unlock()
	waitTimer := time.NewTimer(terminationDelay)
	select {
	case err := <-waitResults:
		waitTimer.Stop()
		return err
	case <-waitTimer.C:
	}

	// Close the process' standard input and wait up to one second for it to
	// terminate on its own.
	s.standardInput.Close()
	waitTimer.Reset(time.Second)
	select {
	case err := <-waitResults:
		waitTimer.Stop()
		return err
	case <-waitTimer.C:
	}

	// If this is a POSIX system, then send SIGTERM to the process and wait up
	// to one second for it to terminate on its own.
	if runtime.GOOS != "windows" {
		s.process.Process.Signal(syscall.SIGTERM)
		waitTimer.Reset(time.Second)
		select {
		case err := <-waitResults:
			waitTimer.Stop()
			return err
		case <-waitTimer.C:
		}
	}

	// Kill the process (via SIGKILL on POSIX and TerminateProcess on Windows)
	// and wait for it to exit.
	s.process.Process.Kill()
	return <-waitResults
}
