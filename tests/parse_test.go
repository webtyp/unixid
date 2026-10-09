package unixid__test

import (
	"testing"
	. "webtyp.com/unixid"
)

func TestParse(t *testing.T) {
	uid, err := NewUnixID()
	if err != nil {
		t.Fatal("Error creating unixid:", err)
	}

	testCases := []struct {
		name              string
		input             string
		expectedTimestamp int64
		expectedReplica   Replica
		expectError       bool
	}{
		{"server ID", "1624397134562544800", 1624397134562544800, 0, false},
		{"client ID", "1624397134562544800.42", 1624397134562544800, 42, false},
		{"client ID with large replica", "1624397134562544800.1234", 1624397134562544800, 1234, false},
		{"invalid multiple dots", "1624397134562544800.42.42", 0, 0, true},
		{"invalid letter", "1624397134562544800a", 0, 0, true},
		{"invalid starts with dot", ".1624397134562544800", 0, 0, true},
		{"invalid ends with dot", "1624397134562544800.", 0, 0, true},
		{"invalid empty", "", 0, 0, true},
		{"invalid zero replica", "1624397134562544800.0", 0, 0, true},
		{"invalid negative replica", "1624397134562544800.-1", 0, 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			timestamp, replica, err := uid.Parse(tc.input)

			if tc.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if timestamp != tc.expectedTimestamp {
				t.Errorf("timestamp: expected %d, got %d", tc.expectedTimestamp, timestamp)
			}
			if replica != tc.expectedReplica {
				t.Errorf("replica: expected %v, got %v", tc.expectedReplica, replica)
			}
		})
	}
}
