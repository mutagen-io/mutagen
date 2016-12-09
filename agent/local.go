package agent

import (
	"net"
	"os"
	"os/exec"
	"runtime"

	"github.com/pkg/errors"

	"google.golang.org/grpc"

	"github.com/havoc-io/mutagen"
	"github.com/havoc-io/mutagen/connectivity"
	"github.com/havoc-io/mutagen/grpcutil"
)

func installLocal() error {
	// Find the appropriate agent binary.
	agent, err := executableForPlatform(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return errors.Wrap(err, "unable to get agent for platform")
	}

	// Invoke its installation.
	if err := exec.Command(agent, "--install").Run(); err != nil {
		os.Remove(agent)
		return errors.Wrap(err, "unable to invoke agent installation")
	}

	// Success.
	return nil
}

func connectLocal() (net.Conn, bool, error) {
	// Compute the path where the agent should be installed.
	agent, err := installPath()
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to compute agent install path")
	}

	// Create an instance of the agent.
	process := exec.Command(agent)

	// Create pipes to the process.
	stdin, err := process.StdinPipe()
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to redirect agent input")
	}
	stdout, err := process.StdoutPipe()
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to redirect agent output")
	}

	// Start the process.
	if err = process.Start(); err != nil {
		if os.IsNotExist(err) {
			return nil, true, errors.New("command not found")
		}
		return nil, false, errors.Wrap(err, "unable to start agent process")
	}

	// Confirm that the process started correctly by performing a version
	// handshake.
	if versionMatch, err := mutagen.ReceiveAndCompareVersion(stdout); err != nil {
		return nil, false, errors.Wrap(err, "unable to handshake with agent process")
	} else if !versionMatch {
		return nil, true, errors.New("version mismatch")
	}

	// Create a connection.
	// HACK: We don't register the standard output pipe as a closer, even though
	// we could, because it might have undesirable blocking behavior. In any
	// case, there's no NEED to close it, because it happens automatically when
	// the process dies, and closing standard input will be sufficient to
	// indicate to the child process that it should exit (and the blocking
	// behavior of standard input won't conflict with closing in our use cases).
	connection, _ := connectivity.NewIOConnection(stdout, stdin, stdin)
	return &processConnection{connection, process}, false, nil
}

func dialLocal() (*grpc.ClientConn, error) {
	// Attempt a connection. If this fails, but it's a failure that justfies
	// attempting an install, then continue, otherwise fail.
	if connection, install, err := connectLocal(); err == nil {
		return grpcutil.NewNonRedialingClientConnection(connection), nil
	} else if !install {
		return nil, errors.Wrap(err, "unable to connect to agent")
	}

	// Attempt to install.
	if err := installLocal(); err != nil {
		return nil, errors.Wrap(err, "unable to install agent")
	}

	// Re-attempt connectivity.
	if connection, _, err := connectLocal(); err != nil {
		return nil, errors.Wrap(err, "unable to connect to agent")
	} else {
		return grpcutil.NewNonRedialingClientConnection(connection), nil
	}
}
