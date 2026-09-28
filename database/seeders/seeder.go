package seeders

import (
	"log"

	"gorm.io/gorm"
)

type Seeder interface {
	Run(db *gorm.DB) error
}

// registry is the reference data the application needs to work at all, on
// every database including production: the permission list, the roles and
// their grants.
//
// User is fixture QA accounts — a seeder that invents users has no safe
// moment to run once the database holds production data. There is no APP_ENV
// guard; drop it from the registry before running cmd/seed at cutover.
//
// Register new seeders here — order matters.
var registry = []Seeder{
	&Permission{},
	&Role{},
	// After both: needs every permission row and the Super-Admin role row to exist.
	&RolePermission{},
	// After Role: assigns each fixture user's role by name.
	&User{},
}

// RunAll seeds the reference data. It is safe to run against production and
// safe to re-run: every seeder upserts on its natural key.
// SyncPermissions runs just the two seeders that own the permission list, for
// callers who want the permission table matched to enums/permission.go without
// touching the territories or the cash and bank accounts. cmd/seed does this
// too, as part of everything else.
//
// RolePermission has to follow Permission: it replaces Super-Admin's set with
// whatever the table now holds, so a permission added or removed a moment ago
// is only granted or revoked once it runs.
func SyncPermissions(db *gorm.DB) error {
	for _, s := range []Seeder{&Permission{}, &RolePermission{}} {
		log.Printf("running seeder: %T", s)

		if err := s.Run(db); err != nil {
			return err
		}
	}
	return nil
}

func RunAll(db *gorm.DB) {
	for _, s := range registry {
		log.Printf("running seeder: %T", s)

		if err := s.Run(db); err != nil {
			log.Fatalf("Seeder failed: %v", err)
		}
	}
}
