package document

import (
	"api-kms/enums"
	"api-kms/errors"
)

// Action is one lifecycle move from FR-4. Named rather than inferred from the
// target status, because two actions can land on the same status from
// different places (approve and publish both end at published) and they are
// not the same capability.
type Action string

const (
	ActionSubmitReview Action = "submit-review"
	ActionApprove      Action = "approve"
	ActionReject       Action = "reject"
	ActionPublish      Action = "publish"
	ActionUnpublish    Action = "unpublish"
	ActionArchive      Action = "archive"
	ActionRestore      Action = "restore"
)

// transition is one legal move: which statuses it may be applied from, what it
// lands on, and the permission it answers to.
type transition struct {
	from []Status
	to   Status
	// permission is the gate. An empty permission means "whoever may edit this
	// document", which is decided per row in the service (author vs any) and
	// cannot be expressed as a route-level grant.
	permission enums.Permission
}

// transitions is the whole lifecycle in one table. Keeping it as data rather
// than a switch is what makes "which actions can I take right now" answerable
// for the UI (see Available) without restating the rules a second time.
var transitions = map[Action]transition{
	// Anyone who may edit the document may send it for review — a contributor
	// with no review rights still has to be able to hand their work on.
	ActionSubmitReview: {from: []Status{StatusDraft}, to: StatusInReview},

	ActionApprove: {from: []Status{StatusInReview}, to: StatusPublished, permission: enums.PermissionDocumentsReview},
	ActionReject:  {from: []Status{StatusInReview}, to: StatusDraft, permission: enums.PermissionDocumentsReview},

	// Publishing straight from draft skips review. The PRD gives Reviewer and
	// up "Publish: Ya" as a capability of its own, so it is not the same thing
	// as approving someone else's submission.
	ActionPublish:   {from: []Status{StatusDraft}, to: StatusPublished, permission: enums.PermissionDocumentsReview},
	ActionUnpublish: {from: []Status{StatusPublished}, to: StatusDraft, permission: enums.PermissionDocumentsReview},

	ActionArchive: {from: []Status{StatusDraft, StatusInReview, StatusPublished}, to: StatusArchived, permission: enums.PermissionDocumentsArchive},
	ActionRestore: {from: []Status{StatusArchived}, to: StatusDraft, permission: enums.PermissionDocumentsArchive},
}

// Permission reports the gate an action answers to, and whether it has one at
// all. Used by the route layer; the empty case is handled in the service.
func (a Action) Permission() (enums.Permission, bool) {
	t, ok := transitions[a]
	if !ok || t.permission == "" {
		return "", false
	}
	return t.permission, true
}

// Next validates the move and returns the status to land on.
//
// The refusal is 422 and names both statuses: "cannot publish" alone leaves
// the caller guessing whether they lacked a right or picked the wrong moment,
// and those want different fixes.
func Next(current Status, action Action) (Status, error) {
	t, ok := transitions[action]
	if !ok {
		return "", errors.InvalidStatusException("unknown document action: " + string(action))
	}

	for _, from := range t.from {
		if from == current {
			return t.to, nil
		}
	}

	return "", errors.InvalidStatusException(
		"cannot " + string(action) + " a document that is " + string(current))
}

// Available lists the actions legal from a status, so the client can render
// exactly the buttons that will work instead of guessing from the status name.
// Permission is not considered here — the resource layer filters on that,
// where the caller is known.
func Available(current Status) []Action {
	// Iterating the map would reorder the buttons on every request, so the
	// order is fixed by walking the lifecycle.
	ordered := []Action{
		ActionSubmitReview, ActionApprove, ActionReject,
		ActionPublish, ActionUnpublish, ActionArchive, ActionRestore,
	}

	available := make([]Action, 0, len(ordered))
	for _, action := range ordered {
		if _, err := Next(current, action); err == nil {
			available = append(available, action)
		}
	}
	return available
}
