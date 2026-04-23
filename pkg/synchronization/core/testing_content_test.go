package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/must"
)

// testingContentManager generates and removes test content on disk using
// Entry-based definitions and Transition. For testing content that can't be
// expressed using these definitions (e.g. intentionally problematic content
// used to test error handling), custom functions can be specified to tweak
// baseline content by operating directly on the underlying filesystem.
type testingContentManager struct {
	// storage is the path where temporary directories (that serve as parent
	// directories to content) will be created. This will be used as the first
	// argument to os.MkdirTemp and may be empty to indicate that the default
	// temporary directory should be used.
	storage string
	// parent is the temporary directory where content currently exists on disk.
	// It is an internal member and should not be set or modified directly.
	parent string
	// baseline is the baseline content to be generated.
	baseline *Entry
	// baselineContentMap is the content map for baseline.
	baselineContentMap testingContentMap
	// tweak is an optional callback that will be provided with the path to the
	// generated baseline content root when generating content. It can perform
	// arbitrary filesystem modifications.
	tweak func(string) error
	// untweak is an optional callback that will be provided with the path to
	// the generated baseline content root when removing content. By default,
	// removal uses os.RemoveAll to remove on-disk content, but for content that
	// is created with intentionally problematic characteristics by a tweak
	// operation (or perhaps by the test using the generator), os.RemoveAll may
	// fail. If untweak is specified, it will be called with the path to the
	// generated content root and can attempt to remove content or at least make
	// it suitable for removal by os.RemoveAll.
	untweak func(string) error

	logger *logging.Logger
}

// generate generates test content on disk. It returns the path to the generated
// content root.
func (g *testingContentManager) generate() (string, error) {
	// If there's already on-disk content, then don't regenerate.
	if g.parent != "" {
		return "", errors.New("on-disk content already generated")
	}

	// Track success.
	var successful bool

	// Create a temporary directory to serve as the parent of the content root.
	if p, err := os.MkdirTemp(g.storage, "mutagen_testing_content"); err != nil {
		return "", fmt.Errorf("unable to generate temporary directory: %w", err)
	} else {
		g.parent = p
	}

	// If we don't generate content successfully, then remove and unset the
	// parent directory when we return.
	defer func() {
		if !successful {
			must.Succeed(os.RemoveAll(g.parent),
				fmt.Sprintf("remove all files from '%s'", g.parent),
				g.logger,
			)
			g.parent = ""
		}
	}()

	// Compute the path to the content root.
	root := filepath.Join(g.parent, "root")

	// Transition the baseline content into existence.
	//
	// HACK: We're relying on internal knowledge of Transition's behavior given
	// the provided argument, specifically the fact that it doesn't validate the
	// symbolic link mode on a per-platform basis like Scan does, the fact that
	// POSIX raw mode will work fine for a pure creation operation on Windows,
	// and the fact that Unicode composition doesn't enter the picture for pure
	// creation operations. This allows us to make the interface for and usage
	// of testingContentManager much simpler. Fortunately, if any of these
	// invariants change, it'll be picked up immediately by tests.
	creation := &Change{New: g.baseline}
	provider := &testingProvider{
		storage:    g.parent,
		contentMap: g.baselineContentMap,
		hasher:     newTestingHasher(),
		logger:     g.logger,
	}
	results, problems, missingFiles := Transition(
		context.Background(),
		root,
		[]*Change{creation},
		nil,
		SymbolicLinkMode_SymbolicLinkModePOSIXRaw,
		0600,
		0700,
		nil,
		false,
		provider,
		g.logger,
	)
	if missingFiles {
		return "", errors.New("content map missing file definitions")
	} else if len(problems) > 0 {
		return "", errors.New("problems encountered during creation")
	} else if len(results) != 1 {
		return "", errors.New("invalid number of results from Transition")
	} else if !results[0].Equal(g.baseline, true) {
		return "", errors.New("generated content did not match baseline")
	}

	// Run the tweak operation, if specified.
	if g.tweak != nil {
		if err := g.tweak(root); err != nil {
			return "", fmt.Errorf("unable to tweak content root: %w", err)
		}
	}

	// Success.
	successful = true
	return root, nil
}

// remove removes test content from disk.
func (g *testingContentManager) remove() error {
	// If there's no on-disk content, then there's nothing to remove.
	if g.parent == "" {
		return errors.New("no on-disk content generated")
	}

	// Unset the parent (because it'll be invalid after this point anyway).
	parent := g.parent
	g.parent = ""

	// Compute the content root path.
	root := filepath.Join(parent, "root")

	// Run the untweak operation, if specified. Even if it fails, we'll still
	// try a removal operation to attempt cleanup.
	if g.untweak != nil {
		if err := g.untweak(root); err != nil {
			must.Succeed(os.RemoveAll(parent),
				fmt.Sprintf("remove all files from '%s'", parent),
				g.logger,
			)
			return fmt.Errorf("unable to untweak content root: %w", err)
		}
	}

	// Perform the removal operation.
	if err := os.RemoveAll(parent); err != nil {
		return fmt.Errorf("unable to remove content: %w", err)
	}

	// Success.
	return nil
}
