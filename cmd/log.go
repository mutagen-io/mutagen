package cmd

import (
	"io/ioutil"
	"log"
)

func init() {
	// Silence the default logger.
	log.SetOutput(ioutil.Discard)
}
