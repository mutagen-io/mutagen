package external

// UsePathBasedLookupForDaemonStart tells Mutagen cmd packages that they should
// use PATH-based lookups to identify the Mutagen executable when trying to
// start the Mutagen daemon. This is required for start (and autostart) behavior
// to function correctly if the calling executable is not the Mutagen CLI. This
// variable must be set in an init function.
var UsePathBasedLookupForDaemonStart bool
