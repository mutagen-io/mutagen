package session

import (
	"sort"
	"strings"

	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/util/validation"
)

// ExtractAndSortLabelKeys extracts a list of keys from the label set and sorts
// them.
func ExtractAndSortLabelKeys(labels map[string]string) []string {
	// Avoid allocation in the event that there are no labels.
	if len(labels) == 0 {
		return nil
	}

	// Create and populate the key slice.
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}

	// Sort keys.
	sort.Strings(keys)

	// Done.
	return keys
}

// EnsureLabelKeyValid verifies that a key conforms to label key requirements.
// These requirements are currently the same as those for Kubernetes label keys.
func EnsureLabelKeyValid(key string) error {
	// Perform validation.
	if errs := validation.IsQualifiedName(key); len(errs) > 0 {
		return errors.New(strings.Join(errs, ", "))
	}

	// Success.
	return nil
}

// EnsureLabelValueValid verifies that a value conforms to label value
// requirements. These requirements are currently the same as those for
// Kubernetes label values.
func EnsureLabelValueValid(value string) error {
	// Perform validation.
	if errs := validation.IsValidLabelValue(value); len(errs) > 0 {
		return errors.New(strings.Join(errs, ", "))
	}

	// Success.
	return nil
}
