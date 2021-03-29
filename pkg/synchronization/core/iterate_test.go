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
		"name1": tD2,
		"name2": tF1,
		"name3": tSA,
	}
	secondContents := map[string]*Entry{
		"name1": tD2,
		"name2": tF1,
		"name3": tSA,
	}
	thirdContents := map[string]*Entry{
		"name1": tD2,
		"name3": tSA,
		"name4": tF1,
		"name5": tSR,
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
