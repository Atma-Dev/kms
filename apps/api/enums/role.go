package enums

type Role string

const (
	SUPER_ADMIN Role = "Super-Admin"
	ADMIN       Role = "Admin"
	GRAY_USER   Role = "Gray-User"

	// The content ladder from the PRD's role table, in order of reach. Each
	// one can do everything the one above it can, plus one more thing — which
	// is why database/seeders builds their grants by extension rather than
	// listing every permission five times.
	//
	// The PRD's "System Admin" is SUPER_ADMIN: it is not a separate row,
	// because Super-Admin already short-circuits every permission check.
	VIEWER      Role = "Viewer"
	CONTRIBUTOR Role = "Contributor"
	EDITOR      Role = "Editor"
	REVIEWER    Role = "Reviewer"
	SPACE_ADMIN Role = "Space-Admin"
)
