// Root-level test (justified): drives the unexported clock to prove ids stay monotonic when the wall clock goes backwards.

package unixid

import (
	"testing"
)

func TestClockRegression(t *testing.T) {
	// Save the original now and restore it after the test
	originalNow := now
	defer func() { now = originalNow }()

	// First time request will return 100
	currentTime := int64(100)

	now = func() int64 {
		return currentTime
	}

	uid, err := NewUnixID()
	if err != nil {
		t.Fatal(err)
	}

	id1 := uid.NewID()
	timestamp1, _, err := uid.Parse(id1)
	if err != nil {
		t.Fatal(err)
	}

	if timestamp1 != 100 {
		t.Errorf("expected timestamp 100, got %d", timestamp1)
	}

	// Move the clock backwards to 50
	currentTime = 50

	id2 := uid.NewID()
	timestamp2, _, err := uid.Parse(id2)
	if err != nil {
		t.Fatal(err)
	}

	// The timestamp should be monotonic, previous + 1 = 101
	if timestamp2 != 101 {
		t.Errorf("expected timestamp 101, got %d", timestamp2)
	}
}
