package queue

import (
	"testing"
	"time"
)

func TestBackoff(t *testing.T) {
	base := 10 * time.Second
	max := 10 * time.Minute

	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{0, base}, // clamped to attempt 1
		{1, base}, // 10s
		{2, 20 * time.Second},
		{3, 40 * time.Second},
		{4, 80 * time.Second},
		{20, max}, // capped
	}

	for _, tc := range cases {
		if got := Backoff(tc.attempt, base, max); got != tc.want {
			t.Errorf("Backoff(%d, %s, %s) = %s, want %s", tc.attempt, base, max, got, tc.want)
		}
	}
}
