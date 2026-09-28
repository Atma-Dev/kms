package seeders

import (
	"api-kms/app"
	"api-kms/app/rbac"
	"api-kms/app/user"
	"api-kms/enums"
	"api-kms/utils"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// userNamespace gives a stable UUIDv5 per seed code so re-running this seeder
// reuses the same row instead of erroring on the unique email/number
// constraints.
var userNamespace = uuid.MustParse("9b8f868a-4f75-4bb4-af0d-75a1fd5ac001")

func userID(code string) uuid.UUID {
	return uuid.NewSHA1(userNamespace, []byte(code))
}

type userSeed struct {
	Code      string
	FirstName string
	LastName  string
	Email     string
	Number    uint64
	Role      enums.Role
	// Password overrides seedPassword for this one account. Blank means
	// seedPassword.
	Password string
}

// seedPassword is the password every fixture account is created with. Plain
// and shared on purpose — these are local test logins, and the seeder is not
// meant to reach a database where that matters (see the registry comment in
// seeder.go).
const seedPassword = "password"

// UserSeeder is fixture data — one account per role, so a fresh database has
// someone to log in as without hand-writing SQL for every manual test.
//
// A seeder that invents users has no safe moment to run once the database
// holds production data, and there is no APP_ENV guard here: drop &User{}
// from the registry in seeder.go before running `cmd/seed` against one.
var UserSeeder = []userSeed{
	{Code: "super-admin", FirstName: "QA", LastName: "Super Admin", Email: "qa.super-admin@kms.test", Number: 900001, Role: enums.SUPER_ADMIN},
	{Code: "admin", FirstName: "QA", LastName: "Admin", Email: "qa.admin@kms.test", Number: 900002, Role: enums.ADMIN},
	// A real address rather than a .test one, so it is the developer's own
	// login and not a fixture identity — all the more reason the note above
	// about dropping this seeder before production applies.
	{Code: "iqbal", FirstName: "Iqbal", LastName: "Atma", Email: "iqbalatma@gmail.com", Number: 900003, Role: enums.SUPER_ADMIN, Password: "admin"},
}

type User struct{}

func (s *User) Run(db *gorm.DB) error {
	for _, detail := range UserSeeder {
		lastName := detail.LastName
		number := detail.Number

		password := detail.Password
		if password == "" {
			password = seedPassword
		}

		hashed, err := utils.MakeHash(password)
		if err != nil {
			return err
		}

		u := user.User{
			BaseModel:        app.BaseModel{ID: userID(detail.Code)},
			Password:         &hashed,
			FirstName:        detail.FirstName,
			LastName:         &lastName,
			Email:            detail.Email,
			Number:           &number,
			MembershipStatus: user.MembershipStatusActive,
		}

		if err := db.Where("email = ?", detail.Email).FirstOrCreate(&u).Error; err != nil {
			return err
		}

		// FirstOrCreate leaves an existing row untouched, so a fixture account
		// created before this seeder set passwords would keep a NULL one and
		// never be able to log in. Only filled when empty — re-running the
		// seeder must not reset a password someone deliberately changed.
		if u.Password == nil {
			if err := db.Model(&u).Update("password", hashed).Error; err != nil {
				return err
			}
		}

		var role rbac.Role
		if err := db.Where("name = ?", string(detail.Role)).First(&role).Error; err != nil {
			return err
		}
		if err := db.Model(&u).Association("Roles").Replace(&role); err != nil {
			return err
		}

		log.Printf("seeded user: %s (%s) password=%s", detail.Email, detail.Role, password)
	}

	return nil
}
