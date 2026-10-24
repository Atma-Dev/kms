package audit

import (
	"api-kms/utils"

	"github.com/gin-gonic/gin"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) GetAllPaginated(c *gin.Context) ([]Model, *utils.PaginationMeta, error) {
	return s.repository.GetAllPaginated(c)
}
