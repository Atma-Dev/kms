package scheduler

import (
	"testing"
	"time"
)

func TestParseCronAndMatches(t *testing.T) {
	cases := []struct {
		expr    string
		t       string // "2006-01-02 15:04", assumed to fall on the right weekday
		matches bool
	}{
		{"* * * * *", "2026-09-09 10:15", true},
		{"*/15 * * * *", "2026-09-09 10:15", true},
		{"*/15 * * * *", "2026-09-09 10:16", false},
		{"0 3 * * *", "2026-09-09 03:00", true},
		{"0 3 * * *", "2026-09-09 03:01", false},
		{"30 3 * * *", "2026-09-09 03:30", true},
		{"@daily", "2026-09-09 00:00", true},
		{"@hourly", "2026-09-09 14:00", true},
		{"@hourly", "2026-09-09 14:01", false},
		// 2026-09-09 is a Wednesday (weekday 3); "0 0 1 * *" only matches day-of-month 1.
		{"0 0 1 * *", "2026-09-01 00:00", true},
		{"0 0 1 * *", "2026-09-09 00:00", false},
	}

	for _, tc := range cases {
		cron, err := ParseCron(tc.expr)
		if err != nil {
			t.Fatalf("ParseCron(%q) returned error: %v", tc.expr, err)
		}
		when, err := time.Parse("2006-01-02 15:04", tc.t)
		if err != nil {
			t.Fatalf("bad test fixture time %q: %v", tc.t, err)
		}
		if got := cron.Matches(when); got != tc.matches {
			t.Errorf("Cron(%q).Matches(%s) = %v, want %v", tc.expr, tc.t, got, tc.matches)
		}
	}
}

func TestParseCronInvalid(t *testing.T) {
	invalid := []string{"", "* * * *", "60 * * * *", "* 24 * * *", "* * 0 * *", "* * * 13 *", "* * * * 7"}
	for _, expr := range invalid {
		if _, err := ParseCron(expr); err == nil {
			t.Errorf("ParseCron(%q) expected an error, got nil", expr)
		}
	}
}

func TestCronNext(t *testing.T) {
	cron, err := ParseCron("30 3 * * *")
	if err != nil {
		t.Fatalf("ParseCron returned error: %v", err)
	}

	after, err := time.Parse("2006-01-02 15:04", "2026-09-09 10:00")
	if err != nil {
		t.Fatalf("bad fixture: %v", err)
	}

	next, ok := cron.Next(after)
	if !ok {
		t.Fatal("Next returned ok=false, expected a match")
	}
	want := "2026-09-10 03:30"
	if got := next.Format("2006-01-02 15:04"); got != want {
		t.Errorf("Next(%s) = %s, want %s", after.Format("2006-01-02 15:04"), got, want)
	}
}
