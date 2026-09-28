// Command seed writes the reference data every database needs — permissions,
// roles and their grants — plus the QA fixture users.
//
//	go run ./cmd/seed
//
// Safe to re-run: every seeder upserts on its natural key. Drop &User{} from
// the registry in database/seeders/seeder.go before running it against a
// production database.
package main

import (
	"api-kms/config"
	"api-kms/database/seeders"
	"log"
)

func main() {
	config.LoadEnv()
	config.ConnectDB()

	log.Println("Running seeders...")
	seeders.RunAll(config.DB)
	log.Println("Seeding complete.")
}
