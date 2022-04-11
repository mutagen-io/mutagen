package synchronization

import (
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
)

// formatPathForLogging adjusts a path for logging purposes.
func formatPathForLogging(path string) string {
	if path == "" {
		return "<root>"
	}
	return path
}

// formatEntryForLogging formats an entry for logging purposes.
func formatEntryForLogging(entry *core.Entry) string {
	if entry == nil {
		return "<non-existent>"
	} else if entry.Kind == core.EntryKind_Directory {
		return fmt.Sprintf("Directory (+%d content entries)", entry.Count()-1)
	} else if entry.Kind == core.EntryKind_File {
		if entry.Executable {
			return fmt.Sprintf("Executable File (Digest %x)", entry.Digest)
		}
		return fmt.Sprintf("File (Digest %x)", entry.Digest)
	} else if entry.Kind == core.EntryKind_SymbolicLink {
		return fmt.Sprintf("Symbolic Link (Target %s)", entry.Target)
	} else if entry.Kind == core.EntryKind_Untracked {
		return "Untracked content"
	} else if entry.Kind == core.EntryKind_Problematic {
		return fmt.Sprintf("Problematic content (%s)", entry.Problem)
	}
	return "<unknown>"
}
