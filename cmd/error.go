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

func Fatal(err error) {
	Error(err)
	os.Exit(1)
}
