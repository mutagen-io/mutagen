package sync

import (
	"github.com/spf13/cobra"
)

// Commands are the synchronization session commands that used to exist at the
// root of the Mutagen command structure. For backward compatibility, we still
// register them at the root of the command structure (and hide them in help
// output). In order to avoid the need to export the commands, we create a list
// of them.
var Commands = []*cobra.Command{
	createCommand,
	listCommand,
	monitorCommand,
	flushCommand,
	pauseCommand,
	resumeCommand,
	resetCommand,
	terminateCommand,
}
