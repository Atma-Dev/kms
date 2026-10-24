// Package audit persists go-audit records via GORM. Distinct from
// app/activitylog, which logs every HTTP request (method/path/status) —
// this logs per-mutation field changes (what changed, from what to what),
// wired in explicitly at the call sites that need it, not every request.
package audit

import (
	"context"
	"encoding/json"
	"fmt"

	goaudit "github.com/iqbalatma/go-audit"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// GormStorer implements goaudit.Storer on top of this app's *gorm.DB.
type GormStorer struct {
	db     *gorm.DB
	logger *logrus.Logger
	hub    *Hub
}

// hub may be nil — Save just skips the broadcast, so this stays usable
// wherever a live feed isn't wired (tests, a future non-HTTP consumer).
func NewGormStorer(db *gorm.DB, logger *logrus.Logger, hub *Hub) *GormStorer {
	return &GormStorer{db: db, logger: logger, hub: hub}
}

// Save persists the audit and swallows its own failure after logging —
// Execute() calls this from a goroutine with nothing left to do with the
// error, and logging it here (structured, into go.log) beats letting
// go-audit's generic log.Printf write an unstructured line to stdout only.
func (s *GormStorer) Save(ctx context.Context, h goaudit.Header, trails []goaudit.Trail) error {
	if err := s.save(ctx, h, trails); err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"action":             h.Action,
			"entry_object_table": h.EntryObjectTable,
			"entry_object_id":    fmt.Sprint(h.EntryObjectID),
		}).Error("failed to persist audit")
	}
	return nil
}

func (s *GormStorer) save(ctx context.Context, h goaudit.Header, trails []goaudit.Trail) error {
	tag, _ := json.Marshal(h.Tag)
	additional, _ := json.Marshal(h.Additional)

	row := Model{
		Action:           h.Action,
		Message:          h.Message,
		AppName:          h.AppName,
		ActorTable:       h.Actor.Table,
		ActorID:          idString(h.Actor.ID),
		ActorName:        h.Actor.Name,
		ActorEmail:       h.Actor.Email,
		ActorPhone:       h.Actor.Phone,
		IPAddress:        h.Request.IPAddress,
		Method:           h.Request.Method,
		Endpoint:         h.Request.Endpoint,
		UserAgent:        h.Request.UserAgent,
		EntryObjectTable: h.EntryObjectTable,
		EntryObjectID:    idString(h.EntryObjectID),
		Tag:              string(tag),
		Additional:       string(additional),
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		if len(trails) == 0 {
			return nil
		}

		trailRows := make([]TrailModel, 0, len(trails))
		for _, t := range trails {
			before, _ := json.Marshal(t.Before)
			after, _ := json.Marshal(t.After)
			trailTag, _ := json.Marshal(t.Tag)
			trailAdditional, _ := json.Marshal(t.Additional)

			trailRows = append(trailRows, TrailModel{
				AuditID:     row.ID.String(),
				ObjectTable: t.ObjectTable,
				ObjectID:    idString(t.ObjectID),
				Before:      string(before),
				After:       string(after),
				Tag:         string(trailTag),
				Additional:  string(trailAdditional),
			})
		}
		if err := tx.Create(&trailRows).Error; err != nil {
			return err
		}
		// Attach for the broadcast below — Resource needs the trails and
		// nothing reloads row after this transaction commits.
		row.Trails = trailRows
		return nil
	})
	if err != nil {
		return err
	}

	if s.hub != nil {
		s.hub.Broadcast(NewResource(&row))
	}
	return nil
}

// idString avoids fmt.Sprint(nil) turning into the literal string "<nil>"
// when an actor or entry object has no id set.
func idString(id any) string {
	if id == nil {
		return ""
	}
	return fmt.Sprint(id)
}
