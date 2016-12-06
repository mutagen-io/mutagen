package agent

import (
	"net"
	"os/exec"
)

type processConnection struct {
	net.Conn
	process *exec.Cmd
}
