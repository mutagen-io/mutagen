package logging

import (
	"log"
	"os"
)

func init() {
	// Set the global logger to use standard output.
	log.SetOutput(os.Stdout)
}
