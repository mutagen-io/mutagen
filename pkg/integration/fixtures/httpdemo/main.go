package main

import (
	"io"
	"log"
	"net/http"

	"github.com/mutagen-io/mutagen/pkg/integration/fixtures/constants"
)

func main() {
	// Create a handler.
	handler := func(response http.ResponseWriter, request *http.Request) {
		io.WriteString(response, constants.HTTPDemoResponse)
	}

	// Register the handler.
	http.HandleFunc("/", handler)

	// Serve requests.
	log.Fatal(http.ListenAndServe(constants.HTTPDemoBindAddress, nil))
}
