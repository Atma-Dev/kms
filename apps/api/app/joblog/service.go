package joblog

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

func (s *Service) GetAllPaginated(c *gin.Context) ([]JobLog, *utils.PaginationMeta, error) {
	return s.repository.GetAllPaginated(c)
}

func (s *Service) GetStats(c *gin.Context) (*Stats, error) {
	return s.repository.GetStats(c.Request.Context(), c.Query("range"), c.Query("type"))
}
