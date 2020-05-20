package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func upMain(_ *cobra.Command, arguments []string) {
	// Handle top-level help and version flags.
	handleTopLevelHelp()
	handleTopLevelVersion()

	// TODO: Implement.
	fmt.Println("up not yet implemented")
}

var upCommand = &cobra.Command{
	Use:          "up",
	Run:          upMain,
	SilenceUsage: true,
}

var upConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
	// detach indicates the presence of the -d/--detach flag.
	detach bool
	// noColor indicates the presence of the --no-color flag.
	noColor bool
	// quietPull indicates the presence of the --quiet-pull flag.
	quietPull bool
	// noDeps indicates the presence of the --no-deps flag.
	noDeps bool
	// forceRecreate indicates the presence of the --force-recreate flag.
	forceRecreate bool
	// alwaysRecreateDeps indicates the presence of the --always-recreate-deps
	// flag.
	alwaysRecreateDeps bool
	// noRecreate indicates the presence of the --no-recreate flag.
	noRecreate bool
	// noBuild indicates the presence of the --no-build flag.
	noBuild bool
	// noStart indicates the presence of the --no-start flag.
	noStart bool
	// build indicates the presence of the --build flag.
	build bool
	// abortOnContainerExit indicates the presence of the
	// --abort-on-container-exit flag.
	abortOnContainerExit bool
	// attachDependencies indicates the presence of the --attach-dependencies
	// flag.
	attachDependencies bool
	// timeout stores the value of the -t/--timeout flag.
	timeout string
	// renewAnonVolumes indicates the presence of the -V/--renew-anon-volumes
	// flag.
	renewAnonVolumes bool
	// removeOrphans indicates the presence of the --remove-orphans flag.
	removeOrphans bool
	// exitCodeFrom stores the value of the --exit-code-from flag.
	exitCodeFrom string
	// scale stores the value(s) of the --scale flag(s).
	scale []string
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present and instead just redirect control to the
	// nominal entry point. We'll use the -h/--help flag that we create below to
	// determine when help functionality needs to be displayed.
	upCommand.SetHelpFunc(upMain)

	// Grab a handle for the command line flags.
	flags := upCommand.Flags()

	// Wire up flags. We don't bother specifying usage information since we'll
	// shell out to Docker Compose if we need to display help information.
	flags.BoolVarP(&upConfiguration.help, "help", "h", false, "")
	flags.BoolVarP(&upConfiguration.detach, "detach", "d", false, "")
	flags.BoolVar(&upConfiguration.noColor, "no-color", false, "")
	flags.BoolVar(&upConfiguration.quietPull, "quiet-pull", false, "")
	flags.BoolVar(&upConfiguration.noDeps, "no-deps", false, "")
	flags.BoolVar(&upConfiguration.forceRecreate, "force-recreate", false, "")
	flags.BoolVar(&upConfiguration.alwaysRecreateDeps, "always-recreate-deps", false, "")
	flags.BoolVar(&upConfiguration.noRecreate, "no-recreate", false, "")
	flags.BoolVar(&upConfiguration.noBuild, "no-build", false, "")
	flags.BoolVar(&upConfiguration.noStart, "no-start", false, "")
	flags.BoolVar(&upConfiguration.build, "build", false, "")
	flags.BoolVar(&upConfiguration.abortOnContainerExit, "abort-on-container-exit", false, "")
	flags.BoolVar(&upConfiguration.attachDependencies, "attach-dependencies", false, "")
	flags.StringVarP(&upConfiguration.timeout, "timeout", "t", "", "")
	flags.BoolVarP(&upConfiguration.renewAnonVolumes, "renew-anon-volumes", "V", false, "")
	flags.BoolVar(&upConfiguration.removeOrphans, "remove-orphans", false, "")
	flags.StringVar(&upConfiguration.exitCodeFrom, "exit-code-from", "", "")
	flags.StringSliceVar(&upConfiguration.scale, "scale", nil, "")
}
