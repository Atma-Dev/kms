package main

import (
	"io"
	"api-kms/config"
	"api-kms/utils"
	"syscall"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// The signal wiring is the part that can fail silently: build it wrong and
// nothing errors, the cache simply never clears.
func TestWatchCacheClearSignal(t *testing.T) {
	config.AppLogger = logrus.New()
	config.AppLogger.SetOutput(io.Discard)

	watchCacheClearSignal()

	loads := 0
	load := func() (int, error) {
		loads++
		return loads, nil
	}
	if _, err := utils.Remember("test:sighup", time.Hour, load); err != nil {
		t.Fatalf("seeding the cache failed: %v", err)
	}

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGHUP); err != nil {
		t.Fatalf("failed to signal self: %v", err)
	}

	// The handler runs on its own goroutine, so give it a moment to land.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, _ = utils.Remember("test:sighup", time.Hour, load); loads > 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("SIGHUP did not clear the cache")
}
