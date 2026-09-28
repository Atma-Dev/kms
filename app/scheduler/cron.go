package scheduler

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Cron is a parsed 5-field cron expression (minute hour day-of-month month
// day-of-week). No external library: this app only ever needs "does this
// minute match" and "what's the next matching minute", both a few dozen
// lines against Go's own time package — see CLAUDE.md's library note.
type Cron struct {
	minute, hour, dom, month, dow field
}

// field is a bitset of allowed values for one cron position (minute 0-59,
// hour 0-23, etc). Values never exceed 59, well within a uint64.
type field struct {
	bits uint64
}

func (f field) has(v int) bool { return f.bits&(1<<uint(v)) != 0 }

func allBits(min, max int) uint64 {
	var b uint64
	for v := min; v <= max; v++ {
		b |= 1 << uint(v)
	}
	return b
}

var aliases = map[string]string{
	"@yearly":   "0 0 1 1 *",
	"@annually": "0 0 1 1 *",
	"@monthly":  "0 0 1 * *",
	"@weekly":   "0 0 * * 0",
	"@daily":    "0 0 * * *",
	"@midnight": "0 0 * * *",
	"@hourly":   "0 * * * *",
}

// ParseCron validates and compiles a cron expression. Used both to run a
// schedule and to reject an invalid one before it's saved.
func ParseCron(expr string) (*Cron, error) {
	expr = strings.TrimSpace(expr)
	if alias, ok := aliases[expr]; ok {
		expr = alias
	}

	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return nil, fmt.Errorf("cron expression must have 5 fields (minute hour day month weekday), got %d", len(parts))
	}

	minute, err := parseField(parts[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("minute: %w", err)
	}
	hour, err := parseField(parts[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("hour: %w", err)
	}
	dom, err := parseField(parts[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("day of month: %w", err)
	}
	month, err := parseField(parts[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("month: %w", err)
	}
	dow, err := parseField(parts[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("day of week: %w", err)
	}

	return &Cron{minute: minute, hour: hour, dom: dom, month: month, dow: dow}, nil
}

// parseField handles a comma-separated list of ranges/steps for one position,
// e.g. "0,15,30,45" or "9-17/2".
func parseField(raw string, min, max int) (field, error) {
	var f field
	for _, part := range strings.Split(raw, ",") {
		if err := parseRange(part, min, max, &f); err != nil {
			return field{}, err
		}
	}
	return f, nil
}

func parseRange(part string, min, max int, f *field) error {
	step := 1
	rangePart := part
	if idx := strings.Index(part, "/"); idx >= 0 {
		rangePart = part[:idx]
		s, err := strconv.Atoi(part[idx+1:])
		if err != nil || s <= 0 {
			return fmt.Errorf("invalid step in %q", part)
		}
		step = s
	}

	lo, hi := min, max
	if rangePart != "*" {
		if idx := strings.Index(rangePart, "-"); idx >= 0 {
			l, err := strconv.Atoi(rangePart[:idx])
			if err != nil {
				return fmt.Errorf("invalid range start in %q", part)
			}
			h, err := strconv.Atoi(rangePart[idx+1:])
			if err != nil {
				return fmt.Errorf("invalid range end in %q", part)
			}
			lo, hi = l, h
		} else {
			v, err := strconv.Atoi(rangePart)
			if err != nil {
				return fmt.Errorf("invalid value %q", part)
			}
			lo, hi = v, v
		}
	}
	if lo < min || hi > max || lo > hi {
		return fmt.Errorf("value out of range in %q (expected %d-%d)", part, min, max)
	}

	for v := lo; v <= hi; v += step {
		f.bits |= 1 << uint(v)
	}
	return nil
}

// Matches reports whether t (minute precision) satisfies the expression.
// Day-of-month and day-of-week are OR'd together when both are restricted,
// matching standard cron semantics.
func (c *Cron) Matches(t time.Time) bool {
	if !c.minute.has(t.Minute()) {
		return false
	}
	if !c.hour.has(t.Hour()) {
		return false
	}
	if !c.month.has(int(t.Month())) {
		return false
	}

	domRestricted := c.dom.bits != allBits(1, 31)
	dowRestricted := c.dow.bits != allBits(0, 6)
	domMatch := c.dom.has(t.Day())
	dowMatch := c.dow.has(int(t.Weekday()))

	switch {
	case domRestricted && dowRestricted:
		return domMatch || dowMatch
	case domRestricted:
		return domMatch
	case dowRestricted:
		return dowMatch
	default:
		return true
	}
}

// Next returns the first minute-aligned time strictly after `after` that
// matches. Bounded to 2 years ahead so an expression that can never legally
// match (e.g. Feb 30) fails fast instead of looping forever.
func (c *Cron) Next(after time.Time) (time.Time, bool) {
	t := after.Truncate(time.Minute).Add(time.Minute)
	limit := after.AddDate(2, 0, 0)
	for t.Before(limit) {
		if c.Matches(t) {
			return t, true
		}
		t = t.Add(time.Minute)
	}
	return time.Time{}, false
}
