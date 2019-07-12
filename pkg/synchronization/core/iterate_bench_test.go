package core

import (
	"testing"
)

func BenchmarkThreeWayNameUnionUnaltered(b *testing.B) {
	// Create sample maps for a three-way merge.
	contents := map[string]*Entry{
		"first":   nil,
		"second":  nil,
		"third":   nil,
		"fourth":  nil,
		"fifth":   nil,
		"sixth":   nil,
		"seventh": nil,
		"eighth":  nil,
		"ninth":   nil,
	}

	// Reset the benchmark timer to exclude the setup time.
	b.ResetTimer()

	// Perform the benchmark.
	for i := 0; i < b.N; i++ {
		nameUnion(contents, contents, contents)
	}
}

func BenchmarkThreeWayNameUnionOneAltered(b *testing.B) {
	// Create sample maps for a three-way merge.
	ancestor := map[string]*Entry{
		"first":   nil,
		"second":  nil,
		"third":   nil,
		"fourth":  nil,
		"fifth":   nil,
		"sixth":   nil,
		"seventh": nil,
		"eighth":  nil,
		"ninth":   nil,
	}
	altered := map[string]*Entry{
		"first":   nil,
		"second":  nil,
		"third":   nil,
		"fourth":  nil,
		"fifth":   nil,
		"sixth":   nil,
		"seventh": nil,
		"eighth":  nil,
		"ninth":   nil,
		"tenth":   nil,
	}

	// Reset the benchmark timer to exclude the setup time.
	b.ResetTimer()

	// Perform the benchmark.
	for i := 0; i < b.N; i++ {
		nameUnion(ancestor, ancestor, altered)
	}
}
