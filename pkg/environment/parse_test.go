package environment

import (
	"testing"
)

func TestParseNil(t *testing.T) {
	if parsed, err := Parse(nil); err != nil {
		t.Fatal("unable to parse nil environment:", err)
	} else if len(parsed) != 0 {
		t.Error("parsed environment not empty when parsing from nil")
	}
}

func TestParseEmpty(t *testing.T) {
	if parsed, err := Parse([]string{}); err != nil {
		t.Fatal("unable to parse empty environment:", err)
	} else if len(parsed) != 0 {
		t.Error("parsed environment not empty when parsing from empty environment")
	}
}

func TestParseInvalid(t *testing.T) {
	if _, err := Parse([]string{""}); err == nil {
		t.Fatal("parsing didn't fail for invalid environment")
	}
}

func TestParse(t *testing.T) {
	// Create a faux environment to test.
	native := []string{
		"=",
		"=something",
		"=something2=other",
		"a=b",
		"WASHINGTON=george",
		"WASHINGTON=george2",
		"Lincoln=abraham",
		"ADAMS=JOHN=QUINCY",
		"JEFFERSON=tHoMaS!\n",
	}
	expected := map[string]string{
		"a":          "b",
		"WASHINGTON": "george2",
		"Lincoln":    "abraham",
		"ADAMS":      "JOHN=QUINCY",
		"JEFFERSON":  "tHoMaS!\n",
	}

	// Parse it.
	parsed, err := Parse(native)
	if err != nil {
		t.Fatal("unable to parse environment:", err)
	}

	// Ensure the length is as expected.
	if len(parsed) != len(expected) {
		t.Error("parsed environment does not match expected length")
	}

	// Ensure values are as expected.
	for k, ev := range expected {
		if pv, ok := parsed[k]; !ok {
			t.Error("parsed environment missing key:", k)
		} else if pv != ev {
			t.Error("parsed environment value doesn't match expected:", pv, "!=", ev)
		}
	}
}

func TestParseBlock(t *testing.T) {
	// Create a test block environment.
	environment := "=\n=something\r\n=something2=other\na=b\r\nWASHINGTON=george\nWASHINGTON=george2\r\r\r\n"
	expected := map[string]string{
		"a":          "b",
		"WASHINGTON": "george2",
	}

	// Parse it.
	parsed, err := ParseBlock(environment)
	if err != nil {
		t.Fatal("unable to parse block environment:", err)
	}

	// Ensure the length is as expected.
	if len(parsed) != len(expected) {
		t.Error("parsed environment does not match expected length")
	}

	// Ensure values are as expected.
	for k, ev := range expected {
		if pv, ok := parsed[k]; !ok {
			t.Error("parsed environment missing key:", k)
		} else if pv != ev {
			t.Error("parsed environment value doesn't match expected:", pv, "!=", ev)
		}
	}
}
