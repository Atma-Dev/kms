package m2mclient

import (
	"api-kms/utils"
	"testing"
)

// Covers the one thing that would silently break auth for every m2m client if
// it regressed: the plaintext secret handed back once must actually verify
// against the hash that gets persisted.
func TestGenerateSecretHashRoundTrip(t *testing.T) {
	secret, err := generateSecret()
	if err != nil {
		t.Fatalf("generateSecret failed: %v", err)
	}
	if len(secret) != 64 {
		t.Fatalf("secret length = %d, want 64 hex chars", len(secret))
	}

	hashed, err := utils.MakeHash(secret)
	if err != nil {
		t.Fatalf("MakeHash failed: %v", err)
	}
	if !utils.CheckHash(hashed, secret) {
		t.Fatal("plaintext secret does not verify against its own hash")
	}

	other, err := generateSecret()
	if err != nil {
		t.Fatalf("generateSecret failed: %v", err)
	}
	if other == secret {
		t.Fatal("two generated secrets collided")
	}
}
