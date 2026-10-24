package m2mclient

import (
	"context"
	"api-kms/internal/testdb"
	"api-kms/utils"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Covers the two branches that matter: a live client's old secret stops
// working and the new one verifies, and a revoked client is refused outright
// rather than being handed a working credential again.
func TestRegenerateSecret(t *testing.T) {
	db := testdb.Open(t)
	repository := NewRepository(db)
	service := NewService(repository)
	ctx := context.Background()

	oldHash, err := utils.MakeHash("old-secret")
	if err != nil {
		t.Fatalf("MakeHash failed: %v", err)
	}

	client := M2MClient{Client: "TEST-" + uuid.NewString()[:8], Secret: oldHash, Scope: Scopes{ScopeAll}}
	if err := db.Create(&client).Error; err != nil {
		t.Fatalf("failed to seed client: %v", err)
	}
	defer db.Unscoped().Delete(&client)

	updated, plaintext, err := service.RegenerateSecret(ctx, client.ID.String())
	if err != nil {
		t.Fatalf("RegenerateSecret failed: %v", err)
	}

	if utils.CheckHash(updated.Secret, "old-secret") {
		t.Fatal("old secret still verifies after regeneration")
	}
	if !utils.CheckHash(updated.Secret, plaintext) {
		t.Fatal("new plaintext secret does not verify against its own hash")
	}

	revokedAt := time.Now()
	revoked := M2MClient{Client: "TEST-" + uuid.NewString()[:8], Secret: oldHash, RevokedAt: &revokedAt}
	if err := db.Create(&revoked).Error; err != nil {
		t.Fatalf("failed to seed revoked client: %v", err)
	}
	defer db.Unscoped().Delete(&revoked)

	if _, _, err := service.RegenerateSecret(ctx, revoked.ID.String()); err == nil {
		t.Fatal("expected regenerating a revoked client's secret to be refused")
	}
}
