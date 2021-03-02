package syscall

// FileAttributeTagInfo is the Go representation of FILE_ATTRIBUTE_TAG_INFO.
type FileAttributeTagInfo struct {
	// FileAttributes are the file attributes.
	FileAttributes uint32
	// ReparseTag is the file reparse tag.
	ReparseTag uint32
}
