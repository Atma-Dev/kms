package m2mclient

import (
	"context"
	stderrors "errors"
	"api-kms/errors"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func (r *Repository) GetAll(ctx context.Context) ([]M2MClient, error) {
	var clients []M2MClient
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&clients).Error; err != nil {
		return nil, errors.InternalServerError("failed to get m2m clients").WithCause(err)
	}
	return clients, nil
}

func (r *Repository) GetById(ctx context.Context, id string) (*M2MClient, error) {
	var client M2MClient
	if err := r.db.WithContext(ctx).First(&client, "id = ?", id).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.EmptyDataException("m2m client not found").WithCause(err)
		}
		return nil, errors.InternalServerError("failed to get m2m client by id").WithCause(err)
	}
	return &client, nil
}

func (r *Repository) AddNewData(ctx context.Context, client *M2MClient) error {
	if err := r.db.WithContext(ctx).Create(client).Error; err != nil {
		return translateDuplicateClient(err, "failed to create m2m client")
	}
	return nil
}

func (r *Repository) UpdateById(ctx context.Context, client *M2MClient) error {
	if err := r.db.WithContext(ctx).Save(client).Error; err != nil {
		return translateDuplicateClient(err, "failed to update m2m client")
	}
	return nil
}

// translateDuplicateClient mirrors errors.TranslateDuplicateCode for the
// `client` column instead of `code`.
func translateDuplicateClient(err error, fallbackMessage string) error {
	if stderrors.Is(err, gorm.ErrDuplicatedKey) {
		return errors.DataAlreadyExistException("client already exists").
			WithErrors(map[string][]string{"client": {"client already exists"}}).
			WithCause(err)
	}
	return errors.InternalServerError(fallbackMessage).WithCause(err)
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}
