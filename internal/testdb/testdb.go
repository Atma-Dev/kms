// Package testdb opens the local dev database for tests in any domain package.
// It exists because more than one package needs the same .env-relative setup,
// and a test helper cannot be shared across packages otherwise.
package testdb

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	stderrors "errors"
	"fmt"
	"api-kms/config"
	"api-kms/utils"
	"os"
	"strconv"
	"testing"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// maxRootClimb bounds the search for the repo root, so a test run from
// outside the repo walks a few levels and gives up rather than to /.
const maxRootClimb = 4

// ensureEnvLoaded climbs to find .env (see Open's comment) and loads it,
// skipping the test if none turns up anywhere. Shared by Open and OpenRedis
// so whichever runs first in a test binary does the climb for both.
func ensureEnvLoaded(t *testing.T) {
	t.Helper()

	// Tests run from their package directory, and those sit at different
	// depths — app/<domain>/ but also middleware/ — so climb until .env turns
	// up instead of assuming a fixed number of levels. A hardcoded "../.."
	// used to overshoot for middleware, whose tests therefore found no .env
	// and skipped themselves on every machine without ever saying so.
	//
	// A second caller in the same binary starts from the root this already
	// moved to, finds .env at once and climbs nothing. When there is no .env
	// anywhere the original directory is restored, so the miss costs the next
	// caller nothing.
	if _, err := os.Stat(".env"); err != nil {
		start, _ := os.Getwd()
		for range maxRootClimb {
			if err := os.Chdir(".."); err != nil {
				break
			}
			if _, err := os.Stat(".env"); err == nil {
				break
			}
		}
		if _, err := os.Stat(".env"); err != nil {
			if start != "" {
				_ = os.Chdir(start)
			}
			skipOrFail(t, "no .env")
		}
	}
	config.LoadEnv()
}

// Open returns a connection to the dev database, or skips the test when there
// is nothing to connect to.
func Open(t *testing.T) *gorm.DB {
	t.Helper()
	ensureEnvLoaded(t)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.AppConfig.DbUser, config.AppConfig.DbPassword,
		config.AppConfig.DbHost, config.AppConfig.DbPort, config.AppConfig.DbName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil || db.Exec("SELECT 1").Error != nil {
		skipOrFail(t, "no dev database reachable")
	}
	return db
}

// OpenRedis returns the dev Redis connection (the same instance gofortify's
// JWT blacklist and the ws-token registry use) via config.RDB, connecting it
// first if nothing has yet, or skips the test when nothing is reachable.
func OpenRedis(t *testing.T) *redis.Client {
	t.Helper()
	ensureEnvLoaded(t)

	if config.RDB != nil {
		return config.RDB
	}

	db, err := strconv.Atoi(config.AppConfig.JWTRedisDB)
	if err != nil {
		skipOrFail(t, "invalid JWT_REDIS_DB")
	}
	client := redis.NewClient(&redis.Options{
		Addr:     config.AppConfig.JWTRedisHost + ":" + config.AppConfig.JWTRedisPort,
		Password: config.AppConfig.JWTRedisPassword,
		DB:       db,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		skipOrFail(t, "no dev redis reachable")
	}
	config.RDB = client
	return config.RDB
}

// skipOrFail skips locally but fails when TESTDB_REQUIRED=1.
//
// A skipped test and a passing one look identical in a CI summary, so a
// pipeline with no database would go green having asserted nothing — the very
// thing a test stage is there to prevent. Developers without a local database
// still get the skip.
func skipOrFail(t *testing.T, reason string) {
	t.Helper()
	if os.Getenv("TESTDB_REQUIRED") == "1" {
		t.Fatalf("TESTDB_REQUIRED=1 but %s", reason)
	}
	t.Skip(reason)
}

// HTTPStatus reports the status an *utils.HTTPError carries, or 0 for anything
// else — including nil, so a missing error never looks like the expected one.
func HTTPStatus(err error) int {
	var httpErr *utils.HTTPError
	if stderrors.As(err, &httpErr) {
		return httpErr.StatusCode
	}
	return 0
}

// users.number is NOT NULL and carries users_number_unique, so every inserted
// test user needs a value nothing else is using.
//
// The band sits far above the seeded users (which count from 1) and far below
// the bigint unsigned ceiling, so a row left behind by a crashed test cannot
// collide with real data — and the app's own "highest number + 1" assignment
// still has room above it.
const (
	userNumberBase = 1_000_000_000
	userNumberSpan = 1_000_000_000
)

// UniqueUserNumber returns a users.number no concurrent test will also pick.
//
// Tests used to read COALESCE(MAX(number), 0) + 1 and insert that, which
// races: `go test ./...` runs packages in parallel against this one dev
// database, so two of them read the same maximum and whichever inserts second
// trips users_number_unique. Drawing at random removes the read entirely, and
// with it the window between reading and inserting.
func UniqueUserNumber(t *testing.T) uint64 {
	t.Helper()

	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		t.Fatalf("failed to draw a unique user number: %v", err)
	}
	return userNumberBase + binary.BigEndian.Uint64(buf[:])%userNumberSpan
}
