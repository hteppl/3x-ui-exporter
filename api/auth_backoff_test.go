package api

import (
	"testing"
	"time"
)

func TestAuthRetryBackoffStaysUnderPanelLoginLimit(t *testing.T) {
	authCache.Lock()
	defer authCache.Unlock()
	resetAuthFailures()

	want := []time.Duration{
		30 * time.Second, time.Minute, 2 * time.Minute, 4 * time.Minute,
		8 * time.Minute, 15 * time.Minute, 15 * time.Minute, 15 * time.Minute,
	}
	for i, w := range want {
		if got := registerAuthFailure(); got != w {
			t.Fatalf("failure %d: got %s, want %s", i+1, got, w)
		}
	}

	// The panel blocks after 5 failures in a 5-minute window; never reach that.
	resetAuthFailures()
	var at []time.Duration
	var now time.Duration
	for i := 0; i < 12; i++ {
		at = append(at, now)
		now += registerAuthFailure()
	}
	for i := range at {
		n := 0
		for _, t2 := range at {
			if t2 > at[i]-5*time.Minute && t2 <= at[i] {
				n++
			}
		}
		if n >= 5 {
			t.Fatalf("%d attempts in the 5m window ending at %s: would lock the account", n, at[i])
		}
	}
	t.Logf("attempt times: %v", at)

	resetAuthFailures()
	if authCache.Failures != 0 || !authCache.NextRetryAt.IsZero() {
		t.Fatal("reset did not clear backoff state")
	}
}
