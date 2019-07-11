package selection

import (
	"sort"
	"strings"

	"github.com/pkg/errors"

	k8slabels "k8s.io/apimachinery/pkg/labels"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"
)

// LabelSelector is a type that performs matching against a set of labels.
type LabelSelector interface {
	// Matches checks whether or not a set of labels is matched by the selector.
	Matches(labels map[string]string) bool
}

// labelSelector is the internal selector implementation. Internally it uses the
// Kubernetes label selection infrastructure.
type labelSelector struct {
	// k8sLabelSelector is the underlying Kubernetes label selector.
	k8sLabelSelector k8slabels.Selector
}

// Matches implements Selector.Matches.
func (s *labelSelector) Matches(labels map[string]string) bool {
	return s.k8sLabelSelector.Matches(k8slabels.Set(labels))
}

// ParseLabelSelector performs label selector parsing. The syntax is currently
// the same as that for Kubernetes.
func ParseLabelSelector(selector string) (LabelSelector, error) {
	// Parse the selector using the Kubernetes label infrastructure.
	k8sLabelSelector, err := k8slabels.Parse(selector)
	if err != nil {
		return nil, err
	}

	// Wrap up the Kubernetes selector.
	return &labelSelector{k8sLabelSelector}, nil
}

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
	if errs := k8svalidation.IsQualifiedName(key); len(errs) > 0 {
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
	if errs := k8svalidation.IsValidLabelValue(value); len(errs) > 0 {
		return errors.New(strings.Join(errs, ", "))
	}

	// Success.
	return nil
}
