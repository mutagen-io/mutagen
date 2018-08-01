package process

const (
	// DETACHED_PROCESS specifies that a process should be created in a
	// "detached" state (i.e. not attached to its parent process' console). More
	// information on process creation flags available here:
	// https://msdn.microsoft.com/en-us/library/windows/desktop/ms684863
	DETACHED_PROCESS = 0x00000008
)
