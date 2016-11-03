package filesystem

import "testing"

// TestInsensitiveIdentity verifies that insensitivity detection returns true
// for the identity transformation (i.e. there's no sensitivity to changes that
// don't change paths).
func TestInsensitiveIdentity(t *testing.T) {
	if insensitive, err := insensitive("", "foo", "foo"); err != nil {
		t.Fatal("unable to check identity sensitivity:", err)
	} else if !insensitive {
		t.Error("unexpected identity sensitivity")
	}
}

// TestInsensitiveNonEquality verifies that insensitivity detection returns
// false for paths that no system would consider to be equal.
func TestInsensitiveNonEquality(t *testing.T) {
	if insensitive, err := insensitive("", "foo", "bar"); err != nil {
		t.Fatal("unable to check non-equality sensitivity:", err)
	} else if insensitive {
		t.Error("unexpected non-equality insensitivity")
	}
}

// TestInsensitiveInvalidRoot verifies that insensitivity detection fails on a
// non-existent root.
func TestInsensitiveInvalidRoot(t *testing.T) {
	if _, err := insensitive("NONEXISTENT/PATH", "foo", "bar"); err == nil {
		t.Error("expected error for insensitivity test on non-existent path")
	}
}
