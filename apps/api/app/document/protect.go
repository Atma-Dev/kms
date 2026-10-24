package document

import (
	"api-kms/app/protection"
	"api-kms/app/render"
	"api-kms/app/user"
	"api-kms/errors"
	"api-kms/utils"
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	goaudit "github.com/iqbalatma/go-audit"
)

// Modul 4 lives here rather than in app/protection because every one of these
// starts with "may this person see this document" — the FR-9 grants, the
// author rule, the lifecycle — and that question only this package can answer.
// app/protection holds the rows and the rendering rules; nothing in it knows
// what a document is, which is what keeps the two from importing each other.

// watermarkTimeFormat matches the app's own display format
// (app/resources/format.go), so the line burned into a page reads the same way
// as every date in the UI the reader came from.
const watermarkTimeFormat = "02/01/2006 15:04"

// Watermark is the identity a rendered copy carries: who asked for it and
// when (FR-1, FR-2).
//
// Built on the server from the authenticated user, never from anything the
// client sends. That is the whole of the NFR "tidak boleh bisa dipalsukan dari
// sisi client" that this side can actually guarantee — see app/protection's
// package doc for what the browser-side overlay can and cannot promise.
func Watermark(viewer *user.User) string {
	return viewer.Email + " · " + time.Now().Format(watermarkTimeFormat)
}

// sharedWatermark is what a reader who arrived through a share link sees
// instead. They have no account, so the trace runs to whoever shared it — the
// person answerable for the link existing.
func sharedWatermark(creatorEmail string) string {
	return "shared by " + creatorEmail + " · " + time.Now().Format(watermarkTimeFormat)
}

// renderOptions is the one place a document becomes something app/render can
// draw, so the PDF and the page images cannot be given different titles or
// different watermarks.
func renderOptions(doc *Document, watermark string) ([]render.Block, render.Options) {
	return Blocks(doc.Content), render.Options{Title: doc.Title, Watermark: watermark}
}

// ExportPDF renders the document with the caller's own watermark (FR-2).
//
// Generated per request, never cached and never stored: two people exporting
// the same document get byte-different files, which is what makes a leaked PDF
// traceable at all. A cache keyed on the document would hand the second person
// the first person's watermark and quietly destroy the property the whole
// module exists for.
//
// The strict level exports pages of pixels instead of text, so a document
// nobody may copy out of in the viewer is not copyable out of its own export
// either.
func (s *Service) ExportPDF(c *gin.Context, id string, viewer *user.User) (*Document, []byte, error) {
	ctx := c.Request.Context()
	doc, err := s.getReadable(ctx, id, viewer)
	if err != nil {
		return nil, nil, err
	}

	blocks, opts := renderOptions(doc, Watermark(viewer))

	var data []byte
	if doc.EffectiveProtection() == protection.LevelStrict {
		data, err = render.ImagePDF(blocks, opts)
	} else {
		data, err = render.PDF(blocks, opts)
	}
	if err != nil {
		return nil, nil, errors.InternalServerError("failed to render the document as a PDF").WithCause(err)
	}

	s.recordAccess(c, doc.ID, &viewer.ID, protection.ActionExportPDF)
	return doc, data, nil
}

// PageImages renders the document as watermarked page images (FR-3).
//
// This is what the strict viewer shows in place of the document's own text.
// It is offered at every protection level, because a client that wants to show
// pages rather than prose is making a presentation choice — what strict
// actually changes is that the text stops being sent at all (see
// ContentVisibleTo), so at that level these images are the only way to read
// the document.
//
// ponytail: renders every page on every request, with no cache. One document
// is a few hundred milliseconds and the result is per-viewer — a cache would
// have to be keyed by reader as well as by document, and holding a few
// megabytes of pixels per concurrent reader is a worse failure than the
// re-render. Add a short-TTL cache keyed (document, updated_at, viewer) if
// page turns ever feel slow.
func (s *Service) PageImages(c *gin.Context, id string, viewer *user.User) (*Document, [][]byte, error) {
	doc, err := s.getReadable(c.Request.Context(), id, viewer)
	if err != nil {
		return nil, nil, err
	}

	blocks, opts := renderOptions(doc, Watermark(viewer))
	pages, err := render.Pages(blocks, opts)
	if err != nil {
		return nil, nil, errors.InternalServerError("failed to render the document").WithCause(err)
	}

	// Deliberately not logged as a second access. The reader opened the
	// document — Show already recorded that — and these images are how that
	// one view is drawn, not a separate reach for the content.
	return doc, pages, nil
}

// ContentVisibleTo decides whether the document's own text goes in the
// response at all.
//
// At the strict level a reader gets page images and no content, because
// shipping the JSON and then asking the browser not to let anyone select it
// would be a lock on a door standing next to an open window. Anyone who may
// *edit* the document still gets it: they can rewrite the text, so withholding
// it from them protects nothing and breaks the editor.
func (s *Service) ContentVisibleTo(ctx context.Context, doc *Document, viewer *user.User) bool {
	if doc.EffectiveProtection() != protection.LevelStrict {
		return true
	}
	return s.assertCanEdit(ctx, doc, viewer) == nil
}

// getReadable is GetById's access check without the view count. Rendering a
// document is not a second read of it.
func (s *Service) getReadable(ctx context.Context, id string, viewer *user.User) (*Document, error) {
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

// RecordAccess appends one row to the trail (FR-4).
//
// Exposed for the handlers that observe an access the service does not:
// opening a document, printing one, downloading a file attached to one.
func (s *Service) RecordAccess(c *gin.Context, documentID uuid.UUID, viewer *user.User, action protection.Action) {
	var userID *uuid.UUID
	if viewer != nil {
		id := viewer.ID
		userID = &id
	}
	s.recordAccess(c, documentID, userID, action)
}

func (s *Service) recordAccess(c *gin.Context, documentID uuid.UUID, userID *uuid.UUID, action protection.Action) {
	s.protection.Record(c.Request.Context(), protection.EventFrom(c, documentID, userID, action))
}

// SetProtectionLevel writes the per-document override (FR-6). A nil level
// clears it, which puts the document back on its space's default — "inherit"
// is the absence of a value, not a third value.
func (s *Service) SetProtectionLevel(ctx context.Context, id string, level *protection.Level) (*Document, error) {
	doc, err := s.repository.GetById(ctx, id)
	if err != nil {
		return nil, err
	}
	if level != nil && !level.Valid() {
		return nil, errors.InvalidStatusException("unknown protection level").
			WithErrors(map[string][]string{"protection_level": {"must be normal or strict"}})
	}

	before := doc.ProtectionLevel
	if err := s.repository.SetProtectionLevel(ctx, id, level); err != nil {
		return nil, err
	}
	doc.ProtectionLevel = level

	goaudit.Init(ctx, "SET_DOCUMENT_PROTECTION", "set document protection level via document.Service.SetProtectionLevel").
		SetEntryObject("documents", doc.ID.String()).
		AddSingleTrail("documents", doc.ID.String(),
			map[string]any{"protection_level": levelValue(before)},
			map[string]any{"protection_level": levelValue(level)}, nil, nil).
		Execute(ctx)

	return doc, nil
}

// levelValue renders a nil override as the word the API uses for it, so the
// audit trail reads "inherit → strict" rather than "<nil> → strict".
func levelValue(level *protection.Level) string {
	if level == nil {
		return "inherit"
	}
	return string(*level)
}

// CreateViewSession mints a share link for a document (FR-5).
//
// Gated on documents.share at the route, and on the caller being able to read
// this particular document here: a permission cannot see whose document it is,
// and handing out a link to a restricted document you are not named on would
// be a way to launder FR-9.
func (s *Service) CreateViewSession(ctx context.Context, id string, viewer *user.User, label *string, ttl time.Duration) (*protection.ViewSession, string, error) {
	doc, err := s.getReadable(ctx, id, viewer)
	if err != nil {
		return nil, "", err
	}
	// Only a published document can be shared outward. A draft is unfinished
	// by definition, and an archived one was deliberately taken down — neither
	// is something to hand to someone outside the team.
	if doc.Status != StatusPublished {
		return nil, "", errors.InvalidStatusException("only a published document can be shared by link")
	}

	session, token, err := s.protection.CreateSession(ctx, doc.ID, viewer.ID, label, ttl)
	if err != nil {
		return nil, "", err
	}

	goaudit.Init(ctx, "CREATE_VIEW_SESSION", "create document view session via document.Service.CreateViewSession").
		SetEntryObject("documents", doc.ID.String()).
		AddSingleTrail("view_sessions", session.ID.String(), nil, map[string]any{
			"expires_at": session.ExpiresAt, "label": label,
		}, nil, nil).
		Execute(ctx)

	return session, token, nil
}

func (s *Service) ViewSessions(ctx context.Context, id string, viewer *user.User) ([]protection.ViewSession, error) {
	if _, err := s.getReadable(ctx, id, viewer); err != nil {
		return nil, err
	}
	return s.protection.SessionsFor(ctx, id)
}

func (s *Service) RevokeViewSession(ctx context.Context, id, sessionID string, viewer *user.User) error {
	if _, err := s.getReadable(ctx, id, viewer); err != nil {
		return err
	}
	if err := s.protection.RevokeSession(ctx, id, sessionID); err != nil {
		return err
	}

	goaudit.Init(ctx, "REVOKE_VIEW_SESSION", "revoke document view session via document.Service.RevokeViewSession").
		SetEntryObject("documents", id).
		AddSingleTrail("view_sessions", sessionID, map[string]any{"revoked": false}, nil, nil, nil).
		Execute(ctx)

	return nil
}

// SharedDocument is what a share link resolves to: the document, and the
// watermark the recipient's copy carries.
type SharedDocument struct {
	Document  *Document
	Session   *protection.ViewSession
	Watermark string
	// Pages is filled for a strict document, and nil otherwise — the two
	// levels are as different for a link recipient as for a logged-in reader.
	Pages [][]byte
}

// OpenViewSession is the whole of what an unauthenticated recipient may do
// (FR-5).
//
// Everything is decided from the token: which document, for how long, and
// whose name is on the copy. Nothing the recipient sends chooses any of it,
// and no route beyond this one is reachable with a share token — a link to one
// document is not a session on the API.
//
// The FR-9 grants are deliberately not re-checked. They restrict a document to
// named *accounts*, and the recipient has none; the check that mattered ran
// when whoever created the link proved they could read it themselves.
func (s *Service) OpenViewSession(c *gin.Context, token string) (*SharedDocument, error) {
	ctx := c.Request.Context()
	session, err := s.protection.OpenSession(ctx, token)
	if err != nil {
		return nil, err
	}

	doc, err := s.repository.GetById(ctx, session.DocumentID.String())
	if err != nil {
		return nil, err
	}
	// A document unpublished or archived after the link went out stops
	// opening. The link was to something the team had published; it is not a
	// standing right to whatever that document becomes.
	if doc.Status != StatusPublished {
		return nil, errors.EmptyDataException("this document is no longer available")
	}

	shared := SharedDocument{
		Document:  doc,
		Session:   session,
		Watermark: sharedWatermark(session.CreatorEmail()),
	}

	if doc.EffectiveProtection() == protection.LevelStrict {
		blocks, opts := renderOptions(doc, shared.Watermark)
		pages, err := render.Pages(blocks, opts)
		if err != nil {
			return nil, errors.InternalServerError("failed to render the document").WithCause(err)
		}
		shared.Pages = pages
	}

	// user_id is nil: nobody was logged in. The link's creator is who the row
	// leads back to, and that is on the session it names.
	s.recordAccess(c, doc.ID, nil, protection.ActionView)
	return &shared, nil
}

// UserAccessLogs is the same trail read by person instead of by document
// (FR-4). No document is loaded first — the filter names an account, and the
// rows may span every space there is.
func (s *Service) UserAccessLogs(c *gin.Context, userID string, filter protection.AccessLogFilter) ([]protection.AccessLog, *utils.PaginationMeta, error) {
	filter.UserID = userID
	return s.protection.AccessLogs(c, filter)
}

// ReportAccess records something only the client could have seen (FR-4).
//
// The action is re-checked here rather than trusted from the request binding
// alone: the binding is one `oneof` tag away from silently accepting every
// action, and this is the one write path into the trail that a client
// controls. The document is loaded through the ordinary read check, so nobody
// can write rows about a document they cannot open.
func (s *Service) ReportAccess(c *gin.Context, id string, viewer *user.User, action protection.Action) error {
	if !action.ClientReportable() {
		return errors.UnauthorizedException("that action is recorded by the server, not reported by the client")
	}

	doc, err := s.getReadable(c.Request.Context(), id, viewer)
	if err != nil {
		return err
	}

	s.recordAccess(c, doc.ID, &viewer.ID, action)
	return nil
}

// AccessLogs is the trail for one document (FR-4), gated at the route on the
// admin permission. The document is loaded first so a log request for an id
// that does not exist answers 404 rather than an empty list.
func (s *Service) AccessLogs(c *gin.Context, id string, filter protection.AccessLogFilter) ([]protection.AccessLog, *utils.PaginationMeta, error) {
	if _, err := s.repository.GetById(c.Request.Context(), id); err != nil {
		return nil, nil, err
	}
	filter.DocumentID = id
	return s.protection.AccessLogs(c, filter)
}
