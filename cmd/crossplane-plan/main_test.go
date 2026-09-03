package main

import "testing"

func TestValidateOnceFlags(t *testing.T) {
	cases := []struct {
		name    string
		once    bool
		pr      int
		wantErr bool
	}{
		{"watcher mode, no pr", false, 0, false},
		{"one-shot with a pr", true, 42, false},
		{"one-shot without a pr", true, 0, true},
		{"one-shot with a negative pr", true, -1, true},
		// The important one: --pr alone must not be ignored. Silently dropping
		// it starts a long-running watcher, so a CI job expecting one pass and
		// an exit hangs until its timeout instead of failing.
		{"pr without one-shot", false, 42, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateOnceFlags(c.once, c.pr)
			if c.wantErr && err == nil {
				t.Fatalf("once=%v pr=%d should be rejected", c.once, c.pr)
			}
			if !c.wantErr && err != nil {
				t.Fatalf("once=%v pr=%d should be accepted, got: %v", c.once, c.pr, err)
			}
		})
	}
}
