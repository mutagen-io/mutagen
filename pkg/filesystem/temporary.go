package filesystem

const (
	// TemporaryNamePrefix is the file name prefix used for all temporary files
	// and directories created by Mutagen. Using this prefix guarantees that any
	// such files will be ignored by filesystem watching and synchronization
	// scans. It may be suffixed with additional elements if desired.
	TemporaryNamePrefix = ".mutagen-temporary-"
)
