package document

import (
	"api-kms/app"
	"api-kms/app/user"
	"api-kms/enums"

	"github.com/google/uuid"
)

// Access is one per-document grant (FR-9).
//
// The rule is **restrictive, not additive**: a document with no rows follows
// the ordinary permissions, and a document with rows is visible only to the
// subjects listed. That is what the PRD's own example asks for — "dokumen
// sensitif hanya untuk beberapa user tertentu meski mereka Viewer di space
// itu". Adding the first grant narrows a document; it never widens one.
//
// Holders of documents.access.manage are never locked out, or a mistake in
// this table would need a database client to undo.
type Access struct {
	app.BaseModel `gorm:"embedded"`
	DocumentID    uuid.UUID   `json:"document_id" gorm:"column:document_id"`
	SubjectType   SubjectType `json:"subject_type" gorm:"column:subject_type"`
	SubjectID     uuid.UUID   `json:"subject_id" gorm:"column:subject_id"`
	Permission    AccessLevel `json:"permission" gorm:"column:permission"`
}

func (Access) TableName() string {
	return "document_access"
}

// SubjectType is who a grant names. A grant on a role reaches whoever holds it
// at read time, which is the point — staffing changes should not need the
// grants rewritten.
type SubjectType string

const (
	SubjectUser SubjectType = "user"
	SubjectRole SubjectType = "role"
)

// AccessLevel is ordered: manage implies edit implies view. Comparing through
// accessRank is what keeps "may they edit" from having to list every level
// that counts as yes.
type AccessLevel string

const (
	AccessView   AccessLevel = "view"
	AccessEdit   AccessLevel = "edit"
	AccessManage AccessLevel = "manage"
)

var accessRank = map[AccessLevel]int{AccessView: 1, AccessEdit: 2, AccessManage: 3}

// AtLeast reports whether this level covers the one asked for.
func (a AccessLevel) AtLeast(required AccessLevel) bool {
	return accessRank[a] >= accessRank[required]
}

// Valid guards the column against a level the ordering does not know, which
// would otherwise rank as 0 and silently deny everything.
func (a AccessLevel) Valid() bool {
	_, ok := accessRank[a]
	return ok
}

// grantFor returns the highest level the user is granted by these rows, or ""
// when none of them name them.
//
// A user matched both directly and through a role gets the stronger of the
// two: the grants are permissions, and the narrower one was not meant as a cap.
func grantFor(grants []Access, u *user.User) AccessLevel {
	best := AccessLevel("")

	roleIDs := make(map[uuid.UUID]bool, len(u.Roles))
	for _, role := range u.Roles {
		roleIDs[role.ID] = true
	}

	for _, grant := range grants {
		matches := (grant.SubjectType == SubjectUser && grant.SubjectID == u.ID) ||
			(grant.SubjectType == SubjectRole && roleIDs[grant.SubjectID])
		if matches && accessRank[grant.Permission] > accessRank[best] {
			best = grant.Permission
		}
	}
	return best
}

// allows answers the whole rule for one document's grants.
//
// No grants at all means the document is unrestricted, so the caller's
// ordinary permissions decide. Otherwise they must be named at `required` or
// above — unless they may manage access, which is the escape hatch that keeps
// a bad grant from being unfixable through the API.
func allows(grants []Access, u *user.User, required AccessLevel) bool {
	if len(grants) == 0 {
		return true
	}
	if u.HasPermission(enums.PermissionDocumentsAccessManage) {
		return true
	}
	return grantFor(grants, u).AtLeast(required)
}
