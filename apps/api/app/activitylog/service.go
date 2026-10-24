package activitylog

import (
	"api-kms/utils"

	"github.com/gin-gonic/gin"
)

type Service struct {
	repository *Repository
}

func (s *Service) GetAllPaginated(c *gin.Context) ([]ActivityLog, *utils.PaginationMeta, error) {
	return s.repository.GetAllPaginated(c)
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}
