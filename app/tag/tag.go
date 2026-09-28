// Package tag is the free vocabulary documents are labelled with (FR-6).
//
// Tags are global rather than per space: a tag is a word, and the same word
// meaning two things in two spaces is a tagging problem, not a schema one.
//
// Anyone who may edit a document may attach tags, and attaching a tag that
// does not exist creates it — that is what "bukan dari daftar tertutup" means.
// The masters.tags.* permissions govern the vocabulary itself (renaming a tag,
// deleting one out of every document at once), which is a different act.
package tag

import "api-kms/app"

type Tag struct {
	app.BaseModel `gorm:"embedded"`
	Name          string `json:"name" gorm:"column:name"`
	// Slug is what makes "Deployment", "deployment" and " deployment "
	// the same tag rather than three.
	Slug string `json:"slug" gorm:"column:slug"`
}

func (Tag) TableName() string {
	return "tags"
}
