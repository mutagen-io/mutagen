package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/mutagen-io/mutagen/cmd"

	"github.com/mutagen-io/mutagen/pkg/filesystem/watching"
)

func main() {
	// Parse arguments.
	if len(os.Args) != 2 {
		cmd.Fatal(errors.New("invalid number of arguments"))
	}
	watchRoot := os.Args[1]

	// Track termination signals.
	signalTermination := make(chan os.Signal, 1)
	signal.Notify(signalTermination, cmd.TerminationSignals...)

	// Create the best available watcher.
	//
	// HACK: We take advantage of the fact that NonRecursiveWatcher implements a
	// superset of the RecursiveWatcher interface (albeit with vastly different,
	// but compatible (for our purposes) semantics), so we can track whatever
	// watcher we establish as a RecursiveWatcher.
	var watcher watching.RecursiveWatcher
	if watching.RecursiveWatchingSupported {
		if w, err := watching.NewRecursiveWatcher(watchRoot); err != nil {
			cmd.Fatal(fmt.Errorf("unable to establish recursive watch: %w", err))
		} else {
			watcher = w
			fmt.Println("Watching", watchRoot, "with recursive watching")
		}
	} else if watching.NonRecursiveWatchingSupported {
		if w, err := watching.NewNonRecursiveWatcher(); err != nil {
			cmd.Fatal(fmt.Errorf("unable to establish non-recursive watch: %w", err))
		} else {
			w.Watch(watchRoot)
			watcher = w
			fmt.Println("Watching", watchRoot, "with non-recursive watching")
		}
	} else {
		cmd.Fatal(errors.New("no supported watching mechanism"))
	}

	// Print events and their paths until watching has terminated.
	for {
		select {
		case path := <-watcher.Events():
			fmt.Printf("\"%s\"\n", path)
		case err := <-watcher.Errors():
			cmd.Fatal(fmt.Errorf("watching failed: %w", err))
		case <-signalTermination:
			fmt.Println("Received termination signal, terminating watching...")
			watcher.Terminate()
		}
	}
}
