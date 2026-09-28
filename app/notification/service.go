package notification

import (
	"context"
	"api-kms/utils"

	"github.com/gin-gonic/gin"
)

type Service struct {
	repository *Repository
}

func (s *Service) GetAllPaginated(c *gin.Context, userID string) ([]Notification, *utils.PaginationMeta, error) {
	return s.repository.GetAllPaginated(c, userID)
}

func (s *Service) MarkRead(ctx context.Context, id string, userID string) error {
	return s.repository.MarkRead(ctx, id, userID)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repository.MarkAllRead(ctx, userID)
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}
