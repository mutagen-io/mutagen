package main

import (
	"errors"
	"log"
	"net/http"
	"os"
)

func main() {
	// Grab and validate configuration parameters from the environment.
	root := os.Getenv("SERVER_ROOT")
	if root == "" {
		log.Fatal(errors.New("invalid or unspecified server root"))
	}
	bind := os.Getenv("SERVER_BIND")
	if bind == "" {
		log.Fatal(errors.New("invalid or unspecified server bind"))
	}

	// Set up file serving.
	http.Handle("/", http.FileServer(http.Dir(root)))

	// Serve files.
	log.Fatal(http.ListenAndServe(bind, nil))
}
