// Command sync-permission makes the permissions table match
// enums/permission.go: rows added, descriptions updated, and rows no longer in
// the enum removed along with their grants.
//
//	go run ./cmd/sync-permission
//
// cmd/seed does the same thing as part of seeding everything else. This exists
// for the common case — a permission was added or renamed in code — where the
// territories and the cash and bank accounts have nothing to do with it.
//
// Safe to re-run, and safe against production: the enum is the source of truth,
// so running it is how the database catches up with the code.
package main

import (
	"api-kms/config"
	"api-kms/database/seeders"
	"log"
)

func main() {
	config.LoadEnv()
	config.ConnectDB()

	log.Println("Syncing permissions...")
	if err := seeders.SyncPermissions(config.DB); err != nil {
		log.Fatalf("Permission sync failed: %v", err)
	}
	log.Println("Permission sync complete.")
}
