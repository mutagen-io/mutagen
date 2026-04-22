package selection

import (
	"strings"
	"testing"
)

// TestParseLabelSelector tests that ParseLabelSelector correctly parses a
// variety of valid and invalid selector strings.
func TestParseLabelSelector(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		// description is a human-readable description of the test case.
		description string
		// selector is the selector string to parse.
		selector string
		// expectFailure indicates whether parsing should fail.
		expectFailure bool
	}{
		// Empty selector (matches everything).
		{"empty selector", "", false},

		// Equality-based selectors.
		{"simple equality", "key=value", false},
		{"double-equals equality", "key==value", false},
		{"equality with empty value", "key=", false},

		// Inequality-based selectors.
		{"simple inequality", "key!=value", false},
		{"inequality with empty value", "key!=", false},

		// Set-based selectors.
		{"set-based in", "key in (v1,v2)", false},
		{"set-based notin", "key notin (v1,v2)", false},
		{"set-based in with single value", "key in (v1)", false},
		{"set-based notin with single value", "key notin (v1)", false},

		// Existence selectors.
		{"existence", "key", false},
		{"non-existence", "!key", false},

		// Multiple requirements.
		{"two equality requirements", "key1=val1,key2=val2", false},
		{
			"complex mixed requirements",
			"env in (staging,prod),tier!=frontend",
			false,
		},
		{
			"existence and equality combined",
			"app,version=v1",
			false,
		},

		// Invalid selectors.
		{"bare equals sign", "=", true},
		{"bare comma", ",", true},
		{"unbalanced open paren", "key in (v1,v2", true},
		{"missing open paren", "key in v1,v2)", true},
		{"in without value set", "key in", true},
	}

	// Process test cases.
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			_, err := ParseLabelSelector(testCase.selector)
			if err != nil && !testCase.expectFailure {
				t.Errorf(
					"selector %q failed parsing unexpectedly: %v",
					testCase.selector, err,
				)
			} else if err == nil && testCase.expectFailure {
				t.Errorf(
					"selector %q passed parsing unexpectedly",
					testCase.selector,
				)
			}
		})
	}
}

// TestLabelSelectorMatches tests that LabelSelector.Matches correctly
// determines whether a set of labels satisfies a parsed selector.
func TestLabelSelectorMatches(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		// description is a human-readable description of the test case.
		description string
		// selector is the selector string to parse before matching.
		selector string
		// labels is the label set to match against.
		labels map[string]string
		// expectMatch indicates whether the selector should match.
		expectMatch bool
	}{
		// Empty selector matches everything.
		{"empty selector matches nil labels", "", nil, true},
		{
			"empty selector matches empty labels",
			"", map[string]string{}, true,
		},
		{
			"empty selector matches non-empty labels",
			"", map[string]string{"a": "b"}, true,
		},

		// Equality-based matching.
		{
			"equality matches correct value",
			"app=web",
			map[string]string{"app": "web"},
			true,
		},
		{
			"equality rejects incorrect value",
			"app=web",
			map[string]string{"app": "api"},
			false,
		},
		{
			"equality rejects missing key",
			"app=web",
			map[string]string{},
			false,
		},
		{
			"equality rejects nil labels",
			"app=web", nil, false,
		},

		// Inequality-based matching.
		{
			"inequality matches different value",
			"app!=web",
			map[string]string{"app": "api"},
			true,
		},
		{
			"inequality rejects matching value",
			"app!=web",
			map[string]string{"app": "web"},
			false,
		},
		{
			"inequality matches missing key",
			"app!=web",
			map[string]string{},
			true,
		},
		{
			"inequality matches nil labels",
			"app!=web", nil, true,
		},

		// Set-based in matching.
		{
			"in matches first value",
			"app in (web,api)",
			map[string]string{"app": "web"},
			true,
		},
		{
			"in matches second value",
			"app in (web,api)",
			map[string]string{"app": "api"},
			true,
		},
		{
			"in rejects non-member value",
			"app in (web,api)",
			map[string]string{"app": "db"},
			false,
		},
		{
			"in rejects missing key",
			"app in (web,api)",
			map[string]string{},
			false,
		},

		// Set-based notin matching.
		{
			"notin matches non-member value",
			"app notin (web,api)",
			map[string]string{"app": "db"},
			true,
		},
		{
			"notin rejects member value",
			"app notin (web,api)",
			map[string]string{"app": "web"},
			false,
		},
		{
			"notin matches missing key",
			"app notin (web,api)",
			map[string]string{},
			true,
		},

		// Existence matching.
		{
			"existence matches present key with value",
			"app",
			map[string]string{"app": "web"},
			true,
		},
		{
			"existence matches present key with empty value",
			"app",
			map[string]string{"app": ""},
			true,
		},
		{
			"existence rejects missing key",
			"app",
			map[string]string{},
			false,
		},

		// Non-existence matching.
		{
			"non-existence matches empty labels",
			"!app",
			map[string]string{},
			true,
		},
		{
			"non-existence rejects present key",
			"!app",
			map[string]string{"app": "web"},
			false,
		},
		{
			"non-existence matches unrelated key only",
			"!app",
			map[string]string{"other": "val"},
			true,
		},

		// Multiple requirements (AND semantics).
		{
			"multiple requirements all satisfied",
			"app=web,tier=frontend",
			map[string]string{
				"app":  "web",
				"tier": "frontend",
			},
			true,
		},
		{
			"multiple requirements one unsatisfied",
			"app=web,tier=frontend",
			map[string]string{
				"app":  "web",
				"tier": "backend",
			},
			false,
		},
		{
			"multiple requirements missing key",
			"app=web,tier=frontend",
			map[string]string{"app": "web"},
			false,
		},

		// Extra labels are allowed.
		{
			"extra labels do not prevent match",
			"app=web",
			map[string]string{
				"app":     "web",
				"version": "v2",
			},
			true,
		},
	}

	// Process test cases.
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			// Parse the selector.
			selector, err := ParseLabelSelector(testCase.selector)
			if err != nil {
				t.Fatalf(
					"failed to parse selector %q: %v",
					testCase.selector, err,
				)
			}

			// Check the match result.
			if match := selector.Matches(testCase.labels); match != testCase.expectMatch {
				t.Errorf(
					"selector %q with labels %v: "+
						"got match = %t, want %t",
					testCase.selector,
					testCase.labels,
					match,
					testCase.expectMatch,
				)
			}
		})
	}
}

// TestEnsureLabelKeyValid tests that EnsureLabelKeyValid correctly validates
// label keys against Kubernetes qualified name requirements.
func TestEnsureLabelKeyValid(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		// key is the label key to validate.
		key string
		// expectFailure indicates whether validation should fail.
		expectFailure bool
	}{
		// Valid simple keys.
		{"app", false},
		{"a", false},
		{"A", false},
		{"my-key", false},
		{"my.key", false},
		{"my_key", false},
		{"MyValue123", false},

		// Valid qualified keys (with prefix).
		{"app.kubernetes.io/name", false},

		// Valid key at the 63-character name segment limit.
		{strings.Repeat("a", 63), false},

		// Invalid: empty string.
		{"", true},

		// Invalid: starts with a dash.
		{"-starts-with-dash", true},

		// Invalid: contains spaces.
		{"has space", true},

		// Invalid: exceeds 63-character name segment limit.
		{strings.Repeat("a", 64), true},
	}

	// Process test cases.
	for _, testCase := range testCases {
		err := EnsureLabelKeyValid(testCase.key)
		if err != nil && !testCase.expectFailure {
			t.Errorf(
				"key %q failed validation unexpectedly: %v",
				testCase.key, err,
			)
		} else if err == nil && testCase.expectFailure {
			t.Errorf(
				"key %q passed validation unexpectedly",
				testCase.key,
			)
		}
	}
}

// TestEnsureLabelValueValid tests that EnsureLabelValueValid correctly
// validates label values against Kubernetes label value requirements.
func TestEnsureLabelValueValid(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		// value is the label value to validate.
		value string
		// expectFailure indicates whether validation should fail.
		expectFailure bool
	}{
		// Valid values.
		{"", false},
		{"my-value", false},
		{"MyValue123", false},
		{"a.b_c-d", false},
		{"a", false},

		// Valid value at the 63-character limit.
		{strings.Repeat("a", 63), false},

		// Invalid: starts with a dash.
		{"-starts-with-dash", true},

		// Invalid: ends with a dash.
		{"ends-with-dash-", true},

		// Invalid: contains spaces.
		{"has space", true},

		// Invalid: exceeds the 63-character limit.
		{strings.Repeat("a", 64), true},
	}

	// Process test cases.
	for _, testCase := range testCases {
		err := EnsureLabelValueValid(testCase.value)
		if err != nil && !testCase.expectFailure {
			t.Errorf(
				"value %q failed validation unexpectedly: %v",
				testCase.value, err,
			)
		} else if err == nil && testCase.expectFailure {
			t.Errorf(
				"value %q passed validation unexpectedly",
				testCase.value,
			)
		}
	}
}

// TestExtractAndSortLabelKeys tests that ExtractAndSortLabelKeys correctly
// extracts and sorts keys from label maps.
func TestExtractAndSortLabelKeys(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		// description is a human-readable description of the test case.
		description string
		// labels is the label map from which to extract keys.
		labels map[string]string
		// expected is the expected sorted key slice.
		expected []string
	}{
		// Nil map returns nil.
		{"nil map", nil, nil},

		// Empty map returns nil.
		{"empty map", map[string]string{}, nil},

		// Single key.
		{
			"single key",
			map[string]string{"app": "web"},
			[]string{"app"},
		},

		// Multiple keys returned in sorted order.
		{
			"multiple keys sorted",
			map[string]string{
				"tier":    "frontend",
				"app":     "web",
				"version": "v1",
			},
			[]string{"app", "tier", "version"},
		},

		// Keys with various allowed characters.
		{
			"keys with various characters",
			map[string]string{
				"app.kubernetes.io/name": "myapp",
				"a-b-c":                  "val",
				"z_key":                  "val",
			},
			[]string{"a-b-c", "app.kubernetes.io/name", "z_key"},
		},
	}

	// Process test cases.
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			result := ExtractAndSortLabelKeys(testCase.labels)

			// Check length.
			if len(result) != len(testCase.expected) {
				t.Fatalf(
					"expected %d keys, got %d: %v",
					len(testCase.expected), len(result), result,
				)
			}

			// Check individual elements.
			for i, key := range result {
				if key != testCase.expected[i] {
					t.Errorf(
						"key at index %d: got %q, want %q",
						i, key, testCase.expected[i],
					)
				}
			}
		})
	}
}
