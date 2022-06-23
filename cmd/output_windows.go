package cmd

const (
	// statusLineFormat is the format string to use for status line printing. On
	// Windows systems, we truncate and pad messages (with spaces) so that the
	// printed content is exactly 79 characters. This ensures that (a) all
	// content from the previous line is overwritten, (b) the cursor is not
	// flashing back and forth to different positions at the end of the printed
	// content, and (c) that the content doesn't overflow the terminal. Of
	// course, the last condition is contingent on the terminal being at least
	// 80 characters wide, and newlines will occur if that's not the case, but
	// 80 characters is the default width of the console on most versions of
	// Windows, and modern versions are even wider. The reason we have to limit
	// ourselves to 79 characters of content instead of 80 is that carriage
	// return wipes don't work if the cursor has already printed a character in
	// the last position of the line on Windows.
	statusLineFormat = "\r%-79.79s"
	// statusLineClearFormat is the format string to use for printing an empty
	// string to clear the status line. It adds a carriage return to return the
	// cursor to the beginning of the line.
	statusLineClearFormat = statusLineFormat + "\r"
)
