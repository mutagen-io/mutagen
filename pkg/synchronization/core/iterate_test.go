package core

import (
	"testing"
)

func TestIterateNameUnionEmpty(t *testing.T) {
	if len(nameUnion()) != 0 {
		t.Error("name union of no content maps non-empty")
	}
}

func TestIterateNameUnion(t *testing.T) {
	firstContents := map[string]*Entry{
		"name1": testDirectory1Entry,
		"name2": testFile1Entry,
		"name3": testSymlinkEntry,
	}
	secondContents := map[string]*Entry{
		"name1": testDirectory1Entry,
		"name2": testFile1Entry,
		"name3": testSymlinkEntry,
	}
	thirdContents := map[string]*Entry{
		"name1": testDirectory1Entry,
		"name3": testSymlinkEntry,
		"name4": testFile1Entry,
		"name5": testSymlinkEntry,
	}
	union := nameUnion(firstContents, secondContents, thirdContents)
	names := []string{"name1", "name2", "name3", "name4", "name5"}
	if len(union) != len(names) {
		t.Error("name union does not have expected length:", len(union), "!=", len(names))
	}
	for _, n := range names {
		if _, ok := union[n]; !ok {
			t.Error("name not in union:", n)
		}
	}
}
