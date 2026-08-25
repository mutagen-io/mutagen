package project

import (
	"errors"
	"strings"
	"testing"
)

// TestAggregateCreateErrorsEmpty verifies that the helper returns nil when
// no errors were collected. This is the common path: every session
// created successfully, and `mutagen project start` should exit 0.
func TestAggregateCreateErrorsEmpty(t *testing.T) {
	if err := aggregateCreateErrors(nil); err != nil {
		t.Fatalf("expected nil for empty slice, got %v", err)
	}
	if err := aggregateCreateErrors([]error{}); err != nil {
		t.Fatalf("expected nil for empty slice, got %v", err)
	}
}

// TestAggregateCreateErrorsSingle verifies that a single collected error
// is reported with the right count and a message that includes the
// original error's text.
func TestAggregateCreateErrorsSingle(t *testing.T) {
	inner := errors.New("dial tcp 192.0.2.1:22: i/o timeout")
	err := aggregateCreateErrors([]error{inner})
	if err == nil {
		t.Fatal("expected non-nil error, got nil")
	}
	if !strings.Contains(err.Error(), "1 session(s) failed to create") {
		t.Errorf("error message missing count: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "192.0.2.1:22") {
		t.Errorf("error message missing inner text: %q", err.Error())
	}
}

// TestAggregateCreateErrorsMultiple verifies that multiple collected
// errors are joined into a single combined error. Each original error
// must remain inspectable via errors.Is (or at minimum, the combined
// error's message must include the original text).
func TestAggregateCreateErrorsMultiple(t *testing.T) {
	e1 := errors.New("session bad-host: dial tcp 192.0.2.1:22: i/o timeout")
	e2 := errors.New("session edge-1: permission denied")
	e3 := errors.New("session edge-2: connection refused")
	err := aggregateCreateErrors([]error{e1, e2, e3})
	if err == nil {
		t.Fatal("expected non-nil error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "3 session(s) failed to create") {
		t.Errorf("error message missing count: %q", msg)
	}
	for _, want := range []string{"bad-host", "edge-1", "edge-2"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error message missing %q: %q", want, msg)
		}
	}
}

// TestAggregateCreateErrorsUnwrap verifies that the combined error can
// be unwrapped to recover the original errors. This is the contract
// that the implementation uses errors.Join to satisfy.
func TestAggregateCreateErrorsUnwrap(t *testing.T) {
	sentinel := errors.New("upstream failure")
	wrapped := errors.Join(
		errors.New("session a: connection refused"),
		sentinel,
		errors.New("session b: permission denied"),
	)
	err := aggregateCreateErrors([]error{wrapped})
	if err == nil {
		t.Fatal("expected non-nil error, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("errors.Is should reach the sentinel error through join, got: %v", err)
	}
}
