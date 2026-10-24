package audit

import (
	"api-kms/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// GetAllPaginated lists audits, newest first, with each row's trails
// preloaded — a trail-less row is a header with no field change recorded
// (a relation sync with an unchanged set, or a create/delete before/after
// that AddSingleTrail decided not to add).
func (r *Repository) GetAllPaginated(c *gin.Context) ([]Model, *utils.PaginationMeta, error) {
	var rows []Model
	db := r.db.WithContext(c).
		Preload("Trails").
		Order("created_at DESC")

	if entryObjectTable := c.Query("entry_object_table"); entryObjectTable != "" {
		db = db.Where("entry_object_table = ?", entryObjectTable)
	}
	if entryObjectID := c.Query("entry_object_id"); entryObjectID != "" {
		db = db.Where("entry_object_id = ?", entryObjectID)
	}
	if actorID := c.Query("actor_id"); actorID != "" {
		db = db.Where("actor_id = ?", actorID)
	}
	if action := c.Query("action"); action != "" {
		db = db.Where("action = ?", action)
	}
	// Same "whole day inclusive" range convention as activitylog's list filter.
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		db = db.Where("created_at >= ?", dateFrom)
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		db = db.Where("created_at < ?::date + INTERVAL '1 day'", dateTo)
	}

	meta, err := utils.Paginate(c, db, &rows)
	return rows, meta, err
}
