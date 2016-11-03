package cmd

import (
	"fmt"
	"os"
)

func Warning(message string) {
	fmt.Fprintln(os.Stderr, "warning:", message)
}

func Error(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
}

func Die(errored bool) {
	if errored {
		os.Exit(1)
	}
	os.Exit(0)
}

func Fatal(err error) {
	Error(err)
	Die(true)
}
