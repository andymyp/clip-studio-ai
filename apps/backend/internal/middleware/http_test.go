package middleware

import "testing"

func TestColoredStatus(t *testing.T) {
	tests := []struct {
		status int
		color  string
	}{
		{status: 200, color: "\x1b[32m"},
		{status: 302, color: "\x1b[36m"},
		{status: 404, color: "\x1b[33m"},
		{status: 500, color: "\x1b[31m"},
	}

	for _, test := range tests {
		got := coloredStatus(test.status)
		want := test.color
		if len(got) < len(want) || got[:len(want)] != want {
			t.Fatalf("coloredStatus(%d) = %q, want prefix %q", test.status, got, want)
		}
	}
}
