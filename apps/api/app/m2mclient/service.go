package m2mclient

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"api-kms/errors"
	"api-kms/utils"
	"time"

	goaudit "github.com/iqbalatma/go-audit"
)

type Service struct {
	repository *Repository
}

func (s *Service) GetAll(ctx context.Context) ([]M2MClient, error) {
	return s.repository.GetAll(ctx)
}

func (s *Service) GetById(ctx context.Context, id string) (*M2MClient, error) {
	return s.repository.GetById(ctx, id)
}

// AddNew returns the plaintext secret alongside the model: it is generated
// here, hashed for storage, and never recoverable again after this call.
func (s *Service) AddNew(ctx context.Context, request *StoreM2MClientRequest) (*M2MClient, string, error) {
	plaintextSecret, hashedSecret, err := newHashedSecret()
	if err != nil {
		return nil, "", err
	}

	client := M2MClient{Client: request.Client, Secret: hashedSecret, Scope: request.Scope}
	if err := s.repository.AddNewData(ctx, &client); err != nil {
		return nil, "", err
	}

	// Secret/plaintextSecret deliberately never appear in the trail — it's a
	// credential, and the audit table is read by a broader set of people than
	// whoever should ever see it.
	goaudit.Init(ctx, "CREATE_M2M_CLIENT", "create m2m client via m2mclient.Service.AddNew").
		SetEntryObject("m2m_clients", client.ID.String()).
		AddSingleTrail("m2m_clients", client.ID.String(), nil, map[string]any{"client": client.Client, "scope": client.Scope}, nil, nil).
		Execute(ctx)

	return &client, plaintextSecret, nil
}

func (s *Service) UpdateById(ctx context.Context, id string, request *UpdateM2MClientRequest) (*M2MClient, error) {
	client, err := s.repository.GetById(ctx, id)
	if err != nil {
		return nil, err
	}

	before := map[string]any{"client": client.Client, "scope": client.Scope}

	client.Client = request.Client
	client.Scope = request.Scope
	if err := s.repository.UpdateById(ctx, client); err != nil {
		return nil, err
	}

	goaudit.Init(ctx, "UPDATE_M2M_CLIENT", "update m2m client via m2mclient.Service.UpdateById").
		SetEntryObject("m2m_clients", client.ID.String()).
		AddSingleTrail("m2m_clients", client.ID.String(), before, map[string]any{"client": client.Client, "scope": client.Scope}, nil, nil).
		Execute(ctx)

	return client, nil
}

// RevokeById sets revoked_at rather than deleting the row, so a revoked
// client's past requests stay attributable.
func (s *Service) RevokeById(ctx context.Context, id string) error {
	client, err := s.repository.GetById(ctx, id)
	if err != nil {
		return err
	}

	if client.RevokedAt != nil {
		return errors.InvalidAction("m2m client is already revoked")
	}

	now := time.Now()
	client.RevokedAt = &now
	if err := s.repository.UpdateById(ctx, client); err != nil {
		return err
	}

	goaudit.Init(ctx, "REVOKE_M2M_CLIENT", "revoke m2m client via m2mclient.Service.RevokeById").
		SetEntryObject("m2m_clients", client.ID.String()).
		AddSingleTrail("m2m_clients", client.ID.String(), map[string]any{"revoked_at": nil}, map[string]any{"revoked_at": client.RevokedAt}, nil, nil).
		Execute(ctx)

	return nil
}

// RegenerateSecret replaces a client's secret, invalidating the old one
// immediately — the same "shown once, never recoverable" contract as AddNew.
// Refused on a revoked client: revocation is meant to be terminal, and
// issuing it a working secret again would undo that.
func (s *Service) RegenerateSecret(ctx context.Context, id string) (*M2MClient, string, error) {
	client, err := s.repository.GetById(ctx, id)
	if err != nil {
		return nil, "", err
	}

	if client.RevokedAt != nil {
		return nil, "", errors.InvalidAction("cannot regenerate the secret of a revoked m2m client")
	}

	plaintextSecret, hashedSecret, err := newHashedSecret()
	if err != nil {
		return nil, "", err
	}

	client.Secret = hashedSecret
	if err := s.repository.UpdateById(ctx, client); err != nil {
		return nil, "", err
	}

	// Records that a rotation happened, never the secret value itself — same
	// reasoning as AddNew.
	goaudit.Init(ctx, "REGENERATE_M2M_CLIENT_SECRET", "regenerate m2m client secret via m2mclient.Service.RegenerateSecret").
		SetEntryObject("m2m_clients", client.ID.String()).
		AddSingleTrail("m2m_clients", client.ID.String(), map[string]any{"secret_regenerated": false}, map[string]any{"secret_regenerated": true}, nil, nil).
		Execute(ctx)

	return client, plaintextSecret, nil
}

func newHashedSecret() (plaintext string, hashed string, err error) {
	plaintext, err = generateSecret()
	if err != nil {
		return "", "", errors.InternalServerError("failed to generate client secret").WithCause(err)
	}

	hashed, err = utils.MakeHash(plaintext)
	if err != nil {
		return "", "", errors.InternalServerError("failed to hash client secret").WithCause(err)
	}

	return plaintext, hashed, nil
}

func generateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func NewService(repository *Repository) *Service {
	return &Service{repository}
}
