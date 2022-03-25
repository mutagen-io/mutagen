package main

import (
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/mutagen"
	"github.com/mutagen-io/mutagen/pkg/sidecar"
)

func main() {
	fmt.Printf("%s:%s\n", sidecar.BaseTag, mutagen.Version)
}
