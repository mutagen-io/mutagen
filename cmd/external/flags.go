package external

var (
	// DisableDaemonAutostart tells Mutagen cmd packages that they should
	// disable daemon autostart behavior. This is the programmatic equivalent to
	// MUTAGEN_DISABLE_AUTOSTART=1. The resulting behavior is the logical-OR of
	// either condition (i.e. leaving this false does not override the
	// environment variable specification). This variable must be set in an init
	// function.
	DisableDaemonAutostart bool
	// UsePathBasedLookupForDaemonStart tells Mutagen cmd packages that they
	// should use PATH-based lookups to identify the Mutagen executable when
	// trying to start the Mutagen daemon. This is required for start (and
	// autostart) behavior to function correctly if the calling executable is
	// not the Mutagen CLI. This variable must be set in an init function.
	UsePathBasedLookupForDaemonStart bool
)
