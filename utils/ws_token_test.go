// External test package: internal/testdb already imports api-kms/utils
// (for HTTPStatus), so a same-package (white-box) test here that also
// imports testdb would cycle. utils_test breaks that, same as any other
// consumer of testdb would sit outside this package.
package utils_test

import (
	"context"
	"api-kms/internal/testdb"
	"api-kms/utils"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Covers the round trip AuthMiddleware actually depends on: a registered jti
// reads back as WS-only, and an unrelated jti — the case every normal
// request hits — does not.
func TestWSOnlyTokenRegistry(t *testing.T) {
	testdb.OpenRedis(t)
	ctx := context.Background()

	jti := uuid.NewString()
	if utils.IsWSOnlyToken(ctx, jti) {
		t.Fatal("a jti that was never registered came back as ws-only")
	}

	if err := utils.RegisterWSOnlyToken(ctx, jti, time.Minute); err != nil {
		t.Fatalf("RegisterWSOnlyToken failed: %v", err)
	}

	if !utils.IsWSOnlyToken(ctx, jti) {
		t.Fatal("expected the registered jti to read back as ws-only")
	}

	if utils.IsWSOnlyToken(ctx, uuid.NewString()) {
		t.Fatal("a different, never-registered jti came back as ws-only")
	}
}
