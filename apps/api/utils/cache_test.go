package utils

import (
	"errors"
	"testing"
	"time"
)

// The four branches worth having: a hit skips load, an expired entry reloads, a
// failed load caches nothing, and Forget drops by prefix only.
func TestRemember(t *testing.T) {
	Forget("")

	calls := 0
	load := func() (int, error) {
		calls++
		return calls, nil
	}

	if got, _ := Remember("test:a", time.Minute, load); got != 1 {
		t.Fatalf("first call returned %d, want 1", got)
	}
	if got, _ := Remember("test:a", time.Minute, load); got != 1 || calls != 1 {
		t.Fatalf("second call returned %d after %d loads, want 1 after 1", got, calls)
	}

	// Stored already expired, so the next read has to load again.
	if got, _ := Remember("test:expired", -time.Second, load); got != 2 {
		t.Fatalf("first call returned %d, want 2", got)
	}
	if got, _ := Remember("test:expired", time.Minute, load); got != 3 {
		t.Fatalf("expired entry returned %d, want a reload", got)
	}

	failed := 0
	if _, err := Remember("test:b", time.Minute, func() (int, error) {
		failed++
		return 0, errors.New("boom")
	}); err == nil {
		t.Fatal("a failing load should surface its error")
	}
	if _, _ = Remember("test:b", time.Minute, func() (int, error) {
		failed++
		return 0, errors.New("boom")
	}); failed != 2 {
		t.Fatalf("a failed load was cached: %d attempts, want 2", failed)
	}

	Remember("other:a", time.Minute, load) // calls == 4
	Forget("test:")
	if got, _ := Remember("other:a", time.Minute, load); got != 4 {
		t.Fatalf("Forget(\"test:\") dropped an unrelated prefix, got %d", got)
	}
	if got, _ := Remember("test:a", time.Minute, load); got != 5 {
		t.Fatalf("Forget(\"test:\") left its own key behind, got %d", got)
	}
}

// CacheEntries is what the maintenance screen reads, so it has to hide expired
// keys (already dead to Remember) and size a cached slice correctly.
func TestCacheEntries(t *testing.T) {
	Forget("")

	if _, err := Remember("test:live", time.Hour, func() ([]string, error) {
		return []string{"a", "b", "c"}, nil
	}); err != nil {
		t.Fatalf("seeding failed: %v", err)
	}
	if _, err := Remember("test:dead", -time.Second, func() ([]string, error) {
		return []string{"x"}, nil
	}); err != nil {
		t.Fatalf("seeding failed: %v", err)
	}

	entries := CacheEntries()
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want only the live one: %+v", len(entries), entries)
	}
	if entries[0].Key != "test:live" {
		t.Fatalf("got key %q, want test:live", entries[0].Key)
	}
	if entries[0].Count != 3 {
		t.Fatalf("got count %d, want 3", entries[0].Count)
	}
	if entries[0].ExpiresInSeconds <= 0 {
		t.Fatalf("got %d seconds left, want a positive countdown", entries[0].ExpiresInSeconds)
	}
}
