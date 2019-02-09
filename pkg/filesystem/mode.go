package filesystem

const (
	// ModePermissionsMask is a bit mask that isolates portable permission bits.
	ModePermissionsMask = Mode(0777)

	// ModePermissionUserRead is the user readable bit.
	ModePermissionUserRead = Mode(0400)
	// ModePermissionUserWrite is the user writable bit.
	ModePermissionUserWrite = Mode(0200)
	// ModePermissionUserExecute is the user executable bit.
	ModePermissionUserExecute = Mode(0100)
	// ModePermissionGroupRead is the group readable bit.
	ModePermissionGroupRead = Mode(0040)
	// ModePermissionGroupWrite is the group writable bit.
	ModePermissionGroupWrite = Mode(0020)
	// ModePermissionGroupExecute is the group executable bit.
	ModePermissionGroupExecute = Mode(0010)
	// ModePermissionOthersRead is the others readable bit.
	ModePermissionOthersRead = Mode(0004)
	// ModePermissionOthersWrite is the others writable bit.
	ModePermissionOthersWrite = Mode(0002)
	// ModePermissionOthersExecute is the others executable bit.
	ModePermissionOthersExecute = Mode(0001)
)
