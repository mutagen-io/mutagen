package cmd

import (
	"fmt"
	"os"
)

func Warning(message string) {
	fmt.Fprintln(os.Stderr, "Warning:", message)
}

func Error(err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
}

func Fatal(err error) {
	Error(err)
	os.Exit(1)
}
