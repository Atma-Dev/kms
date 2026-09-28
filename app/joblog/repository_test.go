package joblog

import (
	"context"
	"api-kms/internal/testdb"
	"testing"
	"time"
)

// Covers the branch that matters: Prune removes only what's older than
// cutoff, not the whole table.
func TestPrune(t *testing.T) {
	db := testdb.Open(t)
	repository := NewRepository(db)
	ctx := context.Background()

	old := time.Now().Add(-4 * 24 * time.Hour)
	recent := time.Now().Add(-1 * time.Hour)

	rows := []JobLog{
		{Type: TypeSchedule, Name: "test-prune-old", Status: StatusSuccess, StartedAt: old},
		{Type: TypeSchedule, Name: "test-prune-recent", Status: StatusSuccess, StartedAt: recent},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed insert failed: %v", err)
	}
	oldRow, recentRow := rows[0], rows[1]
	defer db.Unscoped().Delete(&JobLog{}, "id = ?", recentRow.ID)

	cutoff := time.Now().Add(-3 * 24 * time.Hour)
	n, err := repository.Prune(ctx, cutoff)
	if err != nil {
		t.Fatalf("Prune failed: %v", err)
	}
	if n < 1 {
		t.Fatalf("expected at least 1 row pruned, got %d", n)
	}

	var count int64
	if err := db.Model(&JobLog{}).Where("id = ?", oldRow.ID).Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected the old row to be pruned")
	}

	if err := db.Model(&JobLog{}).Where("id = ?", recentRow.ID).Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected the recent row to survive pruning")
	}
}
