package document

import (
	"api-kms/app/protection"
	"api-kms/app/tag"
	"api-kms/app/user"
	"api-kms/enums"
	"api-kms/errors"
	"api-kms/utils"
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	goaudit "github.com/iqbalatma/go-audit"
	"gorm.io/gorm"
)

// viewThrottleTTL is the NFR's "one view per user per document per hour".
//
// It keys off its own prefix rather than utils.ReadCachePrefix so clearing the
// read cache — a routine admin action — does not also reset every reader's
// throttle and let the counts be inflated on demand.
const (
	viewThrottlePrefix = "view:"
	viewThrottleTTL    = time.Hour
)

type Service struct {
	repository *Repository
	tags       *tag.Service
	// protection holds Modul 4's rows: the access trail and the share links.
	// This package decides who may do what; that one stores what was done.
	protection *protection.Service
}

func (s *Service) GetAllPaginated(c *gin.Context, spaceID string, viewer *user.User) ([]Document, *utils.PaginationMeta, error) {
	// Restricted documents this caller is not named on are excluded from the
	// query, not filtered out of the results — see FR-9 and the Access doc
	// comment for why a grant narrows rather than widens.
	//
	// Whoever may manage access is never hidden from, exactly as in GetById:
	// a list that omitted a document the same caller can open by URL would be
	// two different answers to "what is in this space", and the one that hides
	// it is the one that makes a bad grant impossible to find and fix.
	var hidden []uuid.UUID
	if !viewer.HasPermission(enums.PermissionDocumentsAccessManage) {
		var err error
		hidden, err = s.repository.RestrictedIDsExcluding(c.Request.Context(), spaceID, viewer)
		if err != nil {
			return nil, nil, err
		}
	}
	return s.repository.GetAllPaginated(c, spaceID, hidden)
}

// GetById loads one document and counts the read, at most once an hour per
// reader (NFR). The count is best effort: a failed increment is logged by the
// repository's error path and never fails the read itself — nobody should be
// shown an error page because a statistic did not save.
func (s *Service) GetById(ctx context.Context, id string, viewer *user.User) (*Document, error) {
	doc, err := s.loadReadable(ctx, id, viewer)
	if err != nil {
		return nil, err
	}

	s.countView(ctx, doc, viewer.ID)
	return doc, nil
}

// loadReadable loads a document and refuses it unless FR-9 lets this caller
// see it. Every endpoint that hands back a document, or anything derived from
// one, goes through here.
//
// It exists because the check used to be written out at each of them, and the
// ones that never got it — the version history — handed the whole content to
// callers the document itself answered 404 to. A restriction with a way around
// it is not a restriction, and the way around it was a route nobody thought of
// as "reading the document".
//
// 404, not 403: telling a restricted document apart from one that does not
// exist confirms it exists to exactly the people kept out of it.
func (s *Service) loadReadable(ctx context.Context, id string, viewer *user.User) (*Document, error) {
	doc, err := s.repository.GetById(ctx, id)
	if err != nil {
		return nil, err
	}

	grants, err := s.repository.GrantsFor(ctx, id)
	if err != nil {
		return nil, err
	}
	if !allows(grants, viewer, AccessView) {
		return nil, errors.EmptyDataException("document not found")
	}
	return doc, nil
}

// RelatedLimit is the PRD's "3-5 item" (FR-4), taken at the top of the range:
// a panel of five is still one glance, and fewer means a reader more often
// sees nothing at all.
const RelatedLimit = 5

// Related answers "what else should I read" for one document (FR-4).
//
// The source document is access-checked exactly as reading it is — a caller
// who may not open it may not mine it for suggestions either — and the results
// are filtered through the same hidden set the space list uses, so a
// restricted document never surfaces as somebody else's "related".
//
// Not counted as a view: nobody read the source because a sidebar loaded.
func (s *Service) Related(ctx context.Context, documentID string, viewer *user.User) ([]Document, error) {
	doc, err := s.loadReadable(ctx, documentID, viewer)
	if err != nil {
		return nil, err
	}

	tags, err := s.tags.GetByDocument(ctx, documentID)
	if err != nil {
		return nil, err
	}
	tagIDs := make([]uuid.UUID, len(tags))
	for i := range tags {
		tagIDs[i] = tags[i].ID
	}

	hidden, err := s.HiddenFrom(ctx, doc.SpaceID.String(), viewer)
	if err != nil {
		return nil, err
	}

	return s.repository.RelatedTo(ctx, doc, tagIDs, hidden, RelatedLimit)
}

// Grants is the /access listing. Reading who may reach a document is itself
// gated on managing access — the list names people, and a restricted document
// should not leak its allowlist to everyone who can open it.
func (s *Service) Grants(ctx context.Context, documentID string) ([]Access, error) {
	if _, err := s.repository.GetById(ctx, documentID); err != nil {
		return nil, err
	}
	return s.repository.GrantsFor(ctx, documentID)
}

func (s *Service) AddGrant(ctx context.Context, documentID string, request *StoreAccessRequest) (*Access, error) {
	doc, err := s.repository.GetById(ctx, documentID)
	if err != nil {
		return nil, err
	}
	if !request.Permission.Valid() {
		return nil, errors.InvalidStatusException("permission must be view, edit or manage").
			WithErrors(map[string][]string{"permission": {"must be view, edit or manage"}})
	}

	grant := Access{
		DocumentID:  doc.ID,
		SubjectType: request.SubjectType,
		SubjectID:   request.SubjectID,
		Permission:  request.Permission,
	}
	if err := s.repository.AddGrant(ctx, &grant); err != nil {
		return nil, err
	}

	goaudit.Init(ctx, "GRANT_DOCUMENT_ACCESS", "grant document access via document.Service.AddGrant").
		SetEntryObject("documents", doc.ID.String()).
		AddSingleTrail("document_access", grant.ID.String(), nil, map[string]any{
			"subject_type": grant.SubjectType, "subject_id": grant.SubjectID, "permission": grant.Permission,
		}, nil, nil).
		Execute(ctx)

	return &grant, nil
}

func (s *Service) RevokeGrant(ctx context.Context, documentID, accessID string) error {
	if _, err := s.repository.GetById(ctx, documentID); err != nil {
		return err
	}
	if err := s.repository.DeleteGrant(ctx, documentID, accessID); err != nil {
		return err
	}

	goaudit.Init(ctx, "REVOKE_DOCUMENT_ACCESS", "revoke document access via document.Service.RevokeGrant").
		SetEntryObject("documents", documentID).
		AddSingleTrail("document_access", accessID, map[string]any{"revoked": false}, nil, nil, nil).
		Execute(ctx)

	return nil
}

// SyncTags replaces a document's tag set. The tag work belongs to app/tag; the
// permission to do it belongs here, because only this package knows who
// authored the row the edit rule turns on.
func (s *Service) SyncTags(ctx context.Context, documentID string, request *tag.SyncDocumentTagsRequest, editor *user.User) ([]tag.Tag, error) {
	doc, err := s.repository.GetById(ctx, documentID)
	if err != nil {
		return nil, err
	}
	if err := s.assertCanEdit(ctx, doc, editor); err != nil {
		return nil, err
	}
	return s.tags.SyncDocument(ctx, doc.ID, request)
}

// TagsFor fills the chips on a document.
func (s *Service) TagsFor(ctx context.Context, documentID string) ([]tag.Tag, error) {
	return s.tags.GetByDocument(ctx, documentID)
}

func (s *Service) countView(ctx context.Context, doc *Document, viewerID uuid.UUID) {
	// Only published documents are counted. A draft's author reloading their
	// own editor is not readership.
	if doc.Status != StatusPublished {
		return
	}

	key := viewThrottlePrefix + doc.ID.String() + ":" + viewerID.String()
	_, _ = utils.Remember(key, viewThrottleTTL, func() (bool, error) {
		// Reached only on a miss, which is exactly "this reader has not been
		// counted for this document in the last hour".
		if err := s.repository.IncrementViewCount(ctx, doc.ID.String()); err != nil {
			return false, err
		}
		doc.ViewCount++
		// The detail row behind the number: same call, same throttle, so
		// COUNT(document_views) and view_count cannot drift apart.
		_ = s.repository.RecordView(ctx, doc.ID, viewerID)
		return true, nil
	})
}

func (s *Service) AddNew(ctx context.Context, spaceID string, request *StoreDocumentRequest, authorID uuid.UUID) (*Document, error) {
	space, err := uuid.Parse(spaceID)
	if err != nil {
		return nil, errors.EmptyDataException("space not found").WithCause(err)
	}

	slug, err := s.uniqueSlug(ctx, spaceID, request.Title, "")
	if err != nil {
		return nil, err
	}

	doc := Document{
		SpaceID:    space,
		CategoryID: request.CategoryID,
		Title:      request.Title,
		Slug:       slug,
		// Everything starts as a draft (FR-3). Publishing is a separate act
		// with its own permission, never a field on the create form.
		Status:       StatusDraft,
		AuthorID:     authorID,
		LastEditorID: &authorID,
	}
	doc.setContent(request.Content)

	if err := s.repository.AddNewData(ctx, &doc); err != nil {
		return nil, err
	}

	goaudit.Init(ctx, "CREATE_DOCUMENT", "create document via document.Service.AddNew").
		SetEntryObject("documents", doc.ID.String()).
		AddSingleTrail("documents", doc.ID.String(), nil, map[string]any{
			"space_id": spaceID, "title": doc.Title, "slug": doc.Slug, "status": doc.Status,
		}, nil, nil).
		Execute(ctx)

	return &doc, nil
}

// UpdateById rewrites the document, snapshotting what it replaces when the
// document has ever been published (FR-5).
//
// The snapshot and the write share one transaction — see
// Repository.SaveWithSnapshot for why they cannot be two calls.
func (s *Service) UpdateById(ctx context.Context, id string, request *UpdateDocumentRequest, editor *user.User) (*Document, error) {
	doc, err := s.repository.GetById(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.assertCanEdit(ctx, doc, editor); err != nil {
		return nil, err
	}
	// An archived document is out of circulation; editing it would quietly
	// change what a restore brings back.
	if doc.Status == StatusArchived {
		return nil, errors.InvalidStatusException("restore this document before editing it")
	}

	before := map[string]any{"title": doc.Title, "slug": doc.Slug, "excerpt": doc.Excerpt}

	// Only a real content change earns a version. PUT also carries the title,
	// the category and the pin, so without this a rename would snapshot
	// content identical to the version before it — and the edits that actually
	// changed something end up buried among rows nothing can tell apart.
	//
	// Compared by meaning, not by bytes and not by flattened text. Bytes fail
	// because MySQL's JSON column sorts keys and strips whitespace, so the
	// stored form never matches what a client sent; flattened text fails the
	// other way, because turning a paragraph into a heading leaves the words
	// alone and is still a real change. The title is not part of this — it is
	// not content, and the audit trail already records it changing.
	contentChanged := !SameContent(doc.Content, request.Content)

	var snapshot *Version
	if doc.HasBeenPublished() && contentChanged {
		snapshot = &Version{
			DocumentID: doc.ID,
			// The content as it stands right now — the state this edit is
			// about to replace.
			Content:    doc.Content,
			EditorID:   &editor.ID,
			ChangeNote: request.ChangeNote,
		}
	}

	if request.Title != doc.Title {
		slug, err := s.uniqueSlug(ctx, doc.SpaceID.String(), request.Title, doc.ID.String())
		if err != nil {
			return nil, err
		}
		doc.Slug = slug
	}

	doc.Title = request.Title
	doc.CategoryID = request.CategoryID
	doc.setContent(request.Content)
	doc.LastEditorID = &editor.ID
	if request.IsPinned != nil {
		doc.IsPinned = *request.IsPinned
	}

	if err := s.repository.SaveWithSnapshot(ctx, doc, snapshot); err != nil {
		return nil, err
	}

	goaudit.Init(ctx, "UPDATE_DOCUMENT", "update document via document.Service.UpdateById").
		SetEntryObject("documents", doc.ID.String()).
		AddSingleTrail("documents", doc.ID.String(), before, map[string]any{
			"title": doc.Title, "slug": doc.Slug, "excerpt": doc.Excerpt,
			"content_changed": contentChanged, "snapshotted": snapshot != nil,
		}, nil, nil).
		Execute(ctx)

	return doc, nil
}

// Transition runs one lifecycle move. The legality of the move is decided by
// the table in transition.go; the permission behind it is decided by the route
// — except for submit-review, which has no permission of its own and belongs
// to whoever may edit the document.
func (s *Service) Transition(ctx context.Context, id string, action Action, request *TransitionRequest, actor *user.User) (*Document, error) {
	doc, err := s.repository.GetById(ctx, id)
	if err != nil {
		return nil, err
	}

	// A gated action answers to its route permission for *whether* the caller
	// may make this move at all, and to the grants for *which documents* they
	// may make it to. Only the second half applies here — see assertGranted.
	//
	// The level is edit, not view: a lifecycle move changes the document, and
	// someone granted read access to a restricted document was not thereby
	// handed the right to publish it to everyone.
	if _, gated := action.Permission(); gated {
		if err := s.assertGranted(ctx, doc, actor, AccessEdit); err != nil {
			return nil, err
		}
	} else if err := s.assertCanEdit(ctx, doc, actor); err != nil {
		return nil, err
	}

	if action == ActionReject && (request.Reason == nil || *request.Reason == "") {
		// FR-4 requires the reason. A rejection with no note leaves the author
		// with nothing to act on, which is the whole point of sending it back.
		return nil, errors.InvalidStatusException("a reason is required when rejecting a document").
			WithErrors(map[string][]string{"reason": {"a reason is required"}})
	}

	next, err := Next(doc.Status, action)
	if err != nil {
		return nil, err
	}

	from := doc.Status
	doc.Status = next
	if next == StatusPublished {
		now := time.Now()
		doc.PublishedAt = &now
	}

	if err := s.repository.Save(ctx, doc); err != nil {
		return nil, err
	}

	goaudit.Init(ctx, "TRANSITION_DOCUMENT", "transition document via document.Service.Transition").
		SetEntryObject("documents", doc.ID.String()).
		AddSingleTrail("documents", doc.ID.String(),
			map[string]any{"status": from},
			map[string]any{"status": doc.Status, "action": action, "reason": request.Reason},
			nil, nil).
		Execute(ctx)

	return doc, nil
}

func (s *Service) DeleteById(ctx context.Context, id string, actor *user.User) error {
	doc, err := s.repository.GetById(ctx, id)
	if err != nil {
		return err
	}
	// documents.destroy says they may delete documents; the grants say which
	// ones. Edit rather than manage, so deleting a restricted document asks
	// for the same standing as archiving it — the authorship half stays out
	// for the same reason it does in Transition.
	if err := s.assertGranted(ctx, doc, actor, AccessEdit); err != nil {
		return err
	}

	if err := s.repository.SoftDeleteById(ctx, id); err != nil {
		return err
	}

	goaudit.Init(ctx, "DELETE_DOCUMENT", "soft delete document via document.Service.DeleteById").
		SetEntryObject("documents", id).
		AddSingleTrail("documents", id,
			map[string]any{"title": doc.Title, "status": doc.Status}, nil, nil, nil).
		Execute(ctx)

	return nil
}

func (s *Service) GetVersions(ctx context.Context, documentID string, viewer *user.User) ([]Version, error) {
	if _, err := s.loadReadable(ctx, documentID, viewer); err != nil {
		return nil, err
	}
	return s.repository.GetVersions(ctx, documentID)
}

func (s *Service) GetVersion(ctx context.Context, documentID string, number uint, viewer *user.User) (*Version, error) {
	// A snapshot is the document's content as it used to be, so it answers to
	// the document's own rule. This used not to load the document at all.
	if _, err := s.loadReadable(ctx, documentID, viewer); err != nil {
		return nil, err
	}
	return s.repository.GetVersion(ctx, documentID, number)
}

// Compare diffs two states of a document. Either side may be the literal
// "current" instead of a version number: with snapshots taken *before* an
// overwrite, the live content is not itself a version row, so "what changed
// since v3" would otherwise be unanswerable — and it is the comparison a
// reviewer actually wants.
func (s *Service) Compare(ctx context.Context, documentID, from, to string, viewer *user.User) ([]DiffLine, error) {
	// A diff is the visible text of two versions, which is the content twice
	// over — the same rule as reading the document.
	doc, err := s.loadReadable(ctx, documentID, viewer)
	if err != nil {
		return nil, err
	}

	fromContent, err := s.contentAt(ctx, doc, from)
	if err != nil {
		return nil, err
	}
	toContent, err := s.contentAt(ctx, doc, to)
	if err != nil {
		return nil, err
	}

	return Diff(Flatten(fromContent), Flatten(toContent)), nil
}

// CurrentVersionRef is what Compare accepts in place of a version number.
const CurrentVersionRef = "current"

func (s *Service) contentAt(ctx context.Context, doc *Document, ref string) (Content, error) {
	if ref == CurrentVersionRef {
		return doc.Content, nil
	}

	number, err := parseVersionNumber(ref)
	if err != nil {
		return nil, err
	}

	version, err := s.repository.GetVersion(ctx, doc.ID.String(), number)
	if err != nil {
		return nil, err
	}
	return version.Content, nil
}

// Rollback puts an earlier version's content back, snapshotting the content it
// replaces first — so a rollback is itself undoable and never loses work.
func (s *Service) Rollback(ctx context.Context, documentID string, number uint, request *RollbackRequest, editor *user.User) (*Document, error) {
	doc, err := s.repository.GetById(ctx, documentID)
	if err != nil {
		return nil, err
	}
	if err := s.assertCanEdit(ctx, doc, editor); err != nil {
		return nil, err
	}
	if doc.Status == StatusArchived {
		return nil, errors.InvalidStatusException("restore this document before rolling it back")
	}

	version, err := s.repository.GetVersion(ctx, documentID, number)
	if err != nil {
		return nil, err
	}

	// Rolling back to what is already live changes nothing, so it records
	// nothing — the same rule as an edit that does not touch the content.
	if SameContent(doc.Content, version.Content) {
		return doc, nil
	}

	note := request.ChangeNote
	if note == nil || *note == "" {
		generated := "rolled back to v" + formatVersion(number)
		note = &generated
	}

	snapshot := &Version{
		DocumentID: doc.ID,
		Content:    doc.Content,
		EditorID:   &editor.ID,
		ChangeNote: note,
	}

	doc.setContent(version.Content)
	doc.LastEditorID = &editor.ID

	if err := s.repository.SaveWithSnapshot(ctx, doc, snapshot); err != nil {
		return nil, err
	}

	goaudit.Init(ctx, "ROLLBACK_DOCUMENT", "roll back document via document.Service.Rollback").
		SetEntryObject("documents", doc.ID.String()).
		AddSingleTrail("documents", doc.ID.String(), nil,
			map[string]any{"rolled_back_to": number, "change_note": note}, nil, nil).
		Execute(ctx)

	return doc, nil
}

// assertCanEdit is the PRD's Contributor/Editor split, applied to the row
// rather than the route: a route gate cannot see whose document it is.
//
// A restricted document (FR-9) narrows this further — being its author is not
// enough once someone has locked it down, or the restriction would be
// trivially bypassed by whoever wrote the thing.
func (s *Service) assertCanEdit(ctx context.Context, doc *Document, actor *user.User) error {
	if err := s.assertGranted(ctx, doc, actor, AccessEdit); err != nil {
		return err
	}
	if actor.CanEditDocument(doc.AuthorID.String()) {
		return nil
	}
	return errors.UnauthorizedException("you may only edit documents you authored")
}

// assertGranted is the FR-9 half on its own: whether the grants on this
// document let this caller act on it at the level asked for, with nothing said
// about who wrote it.
//
// Separate from assertCanEdit because the two halves do not always travel
// together. A reviewer holds documents.review and may hold neither
// documents.update nor documents.update.all, so applying the authorship half
// to a review action would lock them out of every document they did not write
// — which is the entire job. The grant half still has to apply, or a
// restriction would stop at the people who cannot review.
func (s *Service) assertGranted(ctx context.Context, doc *Document, actor *user.User, required AccessLevel) error {
	grants, err := s.repository.GrantsFor(ctx, doc.ID.String())
	if err != nil {
		return err
	}
	if !allows(grants, actor, required) {
		return errors.UnauthorizedException("this document is restricted")
	}
	return nil
}

// CanEdit exposes the same rule to the resource layer, which uses it to say
// which actions the caller may actually take.
func CanEdit(doc *Document, actor *user.User) bool {
	return actor.CanEditDocument(doc.AuthorID.String())
}

// CanTransition reports whether actor may run action on doc: the move has to
// be legal from the current status, and the actor has to hold the gate.
func CanTransition(doc *Document, action Action, actor *user.User) bool {
	if _, err := Next(doc.Status, action); err != nil {
		return false
	}
	permission, gated := action.Permission()
	if !gated {
		return CanEdit(doc, actor)
	}
	return actor.HasPermission(permission)
}

func (s *Service) uniqueSlug(ctx context.Context, spaceID, title, excludeID string) (string, error) {
	return utils.UniqueSlug(utils.Slugify(title), func(candidate string) (bool, error) {
		return s.repository.SlugExists(ctx, spaceID, candidate, excludeID)
	})
}

// parseVersionNumber turns a ?from=/?to= segment into a version number. It
// refuses 0 and anything unparseable with a 400 rather than falling back to a
// default — a mistyped version silently compared against v1 would show a diff
// the caller never asked for and has no reason to doubt.
func parseVersionNumber(ref string) (uint, error) {
	number, err := strconv.ParseUint(strings.TrimPrefix(ref, "v"), 10, 32)
	if err != nil || number == 0 {
		return 0, errors.QueryParameterInvalid("version must be a positive number or \"" + CurrentVersionRef + "\"")
	}
	return uint(number), nil
}

func formatVersion(number uint) string {
	return strconv.FormatUint(uint64(number), 10)
}

func NewService(repository *Repository, tags *tag.Service, protect *protection.Service) *Service {
	return &Service{repository: repository, tags: tags, protection: protect}
}

// HiddenFrom is the set of documents in a space this caller may not see, with
// the access-manager escape hatch already applied — the same answer the list
// filters on, exposed so app/category can exclude them from its counts instead
// of restating the rule and drifting from it.
func (s *Service) HiddenFrom(ctx context.Context, spaceID string, viewer *user.User) ([]uuid.UUID, error) {
	if viewer.HasPermission(enums.PermissionDocumentsAccessManage) {
		return nil, nil
	}
	return s.repository.RestrictedIDsExcluding(ctx, spaceID, viewer)
}

// GetForEdit loads a document and asserts the caller may change it — the same
// rule Update runs, exposed so packages that hang things off a document
// (app/attachment) answer to it instead of inventing a second one.
func (s *Service) GetForEdit(ctx context.Context, id string, editor *user.User) (*Document, error) {
	doc, err := s.repository.GetById(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.assertCanEdit(ctx, doc, editor); err != nil {
		return nil, err
	}
	if doc.Status == StatusArchived {
		return nil, errors.InvalidStatusException("restore this document before changing it")
	}
	return doc, nil
}

// SpaceIDFor validates a space id for callers that need somewhere to file
// something before a document exists — an inline image uploaded while the
// document is still being written (FR-8).
//
// Parsing is the whole check: the storage path is derived from this, and a
// string that is not a uuid would put files in a folder nothing can find.
func (s *Service) SpaceIDFor(ctx context.Context, spaceID string) (uuid.UUID, error) {
	id, err := uuid.Parse(spaceID)
	if err != nil {
		return uuid.Nil, errors.EmptyDataException("space not found").WithCause(err)
	}
	return id, nil
}

// SetPinned pins or unpins a document. Answering to the edit rule rather than
// a permission of its own: pinning is curating what the space leads with,
// which is the same judgement as editing what it says.
func (s *Service) SetPinned(ctx context.Context, id string, pinned bool, actor *user.User) (*Document, error) {
	doc, err := s.repository.GetById(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.assertCanEdit(ctx, doc, actor); err != nil {
		return nil, err
	}
	if doc.IsPinned == pinned {
		return doc, nil
	}

	if err := s.repository.SetPinned(ctx, id, pinned); err != nil {
		return nil, err
	}
	doc.IsPinned = pinned

	goaudit.Init(ctx, "PIN_DOCUMENT", "pin document via document.Service.SetPinned").
		SetEntryObject("documents", id).
		AddSingleTrail("documents", id,
			map[string]any{"is_pinned": !pinned}, map[string]any{"is_pinned": pinned}, nil, nil).
		Execute(ctx)

	return doc, nil
}

// GetDeletedPaginated lists a space's bin.
//
// The restriction filter is deliberately not applied: reaching the bin needs
// documents.destroy, which is strictly more than reading, and a restricted
// document hidden from the bin could never be restored by anyone.
func (s *Service) GetDeletedPaginated(c *gin.Context, spaceID string) ([]Document, *utils.PaginationMeta, error) {
	return s.repository.GetDeletedPaginated(c, spaceID)
}

// Undelete brings a soft-deleted document back, as the draft it has to be
// re-checked as. Restoring it straight to published would put content back in
// front of readers without anyone looking at why it was deleted.
func (s *Service) Undelete(ctx context.Context, id string) (*Document, error) {
	doc, err := s.repository.GetDeletedById(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.repository.UndeleteById(ctx, id); err != nil {
		return nil, err
	}

	doc.DeletedAt = gorm.DeletedAt{}
	if doc.Status == StatusPublished {
		doc.Status = StatusDraft
		if err := s.repository.Save(ctx, doc); err != nil {
			return nil, err
		}
	}

	goaudit.Init(ctx, "UNDELETE_DOCUMENT", "restore deleted document via document.Service.Undelete").
		SetEntryObject("documents", id).
		AddSingleTrail("documents", id, nil,
			map[string]any{"title": doc.Title, "status": doc.Status}, nil, nil).
		Execute(ctx)

	return doc, nil
}
