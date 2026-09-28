package protection

import (
	"api-kms/errors"
	"api-kms/utils"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Service struct {
	repository *Repository
	logger     *logrus.Logger
	// maxSessionTTL caps how far ahead a share link may be set to expire. A
	// link with no ceiling is a permanent one someone meant to be temporary.
	maxSessionTTL time.Duration
}

func NewService(repository *Repository, logger *logrus.Logger, maxSessionTTL time.Duration) *Service {
	return &Service{repository: repository, logger: logger, maxSessionTTL: maxSessionTTL}
}

// Event is one thing that happened to a document, with the request details the
// log needs (FR-4).
//
// Built by EventFrom at the edge so nothing below the handler has to hold a
// *gin.Context to record an access.
type Event struct {
	DocumentID uuid.UUID
	// UserID is nil for a reader who came through a share link: they have no
	// account, and the link's creator is who is answerable for them.
	UserID    *uuid.UUID
	Action    Action
	IPAddress string
	UserAgent string
}

// EventFrom reads the request half of a log row. c.ClientIP() honours the
// proxy headers Gin is configured to trust, which is the address worth
// recording when the app runs behind one.
func EventFrom(c *gin.Context, documentID uuid.UUID, userID *uuid.UUID, action Action) Event {
	event := Event{DocumentID: documentID, UserID: userID, Action: action}
	if c != nil {
		event.IPAddress = c.ClientIP()
		if c.Request != nil {
			event.UserAgent = c.Request.UserAgent()
		}
	}
	return event
}

// Record appends the row and reports nothing back.
//
// Best effort, deliberately: the log lives in the same database as the
// document that was just read, so a write that fails here almost certainly
// means the read failed too and the caller already has an error. Failing a
// reader's page because a statistic did not save is the same trade
// countView (Modul 3) already makes — and the failure is logged rather than
// swallowed, so a trail that has started dropping rows is visible in go.log
// instead of only in the gap it leaves.
func (s *Service) Record(ctx context.Context, event Event) {
	log := AccessLog{
		DocumentID: event.DocumentID,
		UserID:     event.UserID,
		Action:     event.Action,
		IPAddress:  nilIfEmpty(event.IPAddress),
		UserAgent:  nilIfEmpty(truncate(event.UserAgent, userAgentMaxLength)),
	}
	if err := s.repository.RecordAccess(ctx, &log); err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"document_id": event.DocumentID.String(),
			"action":      string(event.Action),
		}).Error("failed to record document access")
	}
}

// userAgentMaxLength matches the column. A user agent longer than this is a
// client padding the header, and storing the first 512 characters of one is
// as identifying as storing all of it.
const userAgentMaxLength = 512

// nilIfEmpty keeps "the client sent no user agent" as NULL rather than as an
// empty string. An investigation reading the row can tell the two apart.
func nilIfEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func (s *Service) AccessLogs(c *gin.Context, filter AccessLogFilter) ([]AccessLog, *utils.PaginationMeta, error) {
	return s.repository.AccessLogsPaginated(c, filter)
}

// SessionTokenBytes is the size of a share token before hex encoding. 32 bytes
// is the same order as a session cookie: the token is the only thing standing
// between a URL and a document, so it has to be unguessable rather than short.
const SessionTokenBytes = 32

// CreateSession mints a share link (FR-5) and returns the plaintext token
// beside the row.
//
// The plaintext exists only in this return value — the row holds its SHA-256 —
// so the link can be shown once, at creation, and never again. Anyone who
// loses it revokes and makes another, which is also what should happen: a link
// somebody has mislaid is a link they can no longer account for.
//
// The caller is responsible for deciding whether this user may share this
// document. This package does not know what a document is.
func (s *Service) CreateSession(ctx context.Context, documentID, creatorID uuid.UUID, label *string, ttl time.Duration) (*ViewSession, string, error) {
	if ttl <= 0 {
		return nil, "", errors.InvalidStatusException("a view link must expire in the future").
			WithErrors(map[string][]string{"expires_in_hours": {"must be at least 1"}})
	}
	if ttl > s.maxSessionTTL {
		return nil, "", errors.InvalidStatusException("a view link cannot last that long").
			WithErrors(map[string][]string{"expires_in_hours": {
				"must be at most " + formatHours(s.maxSessionTTL),
			}})
	}

	raw := make([]byte, SessionTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		// crypto/rand failing is the system being unable to produce a secret.
		// A weaker token is not an acceptable fallback, so this is the one
		// path here that refuses rather than degrades.
		return nil, "", errors.InternalServerError("failed to generate a view link").WithCause(err)
	}
	token := hex.EncodeToString(raw)

	session := ViewSession{
		DocumentID: documentID,
		CreatedBy:  creatorID,
		Label:      label,
		TokenHash:  HashToken(token),
		ExpiresAt:  time.Now().Add(ttl),
	}
	if err := s.repository.CreateSession(ctx, &session); err != nil {
		return nil, "", err
	}
	return &session, token, nil
}

// HashToken is how a share token becomes the value stored beside it.
//
// SHA-256 rather than bcrypt, which is what utils.MakeHash does for passwords:
// a password is low entropy and needs the work factor to survive guessing,
// while a 256-bit random token has nothing to guess. Bcrypt here would only
// mean every link open pays a deliberate ~60ms, and would rule out the unique
// index that makes the lookup one query.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Service) SessionsFor(ctx context.Context, documentID string) ([]ViewSession, error) {
	return s.repository.SessionsFor(ctx, documentID)
}

// OpenSession resolves a share token to the document it opens.
//
// An expired link answers 403 rather than 404: the recipient was given
// something real, and "this expired" tells them to ask for a new one, while
// "not found" tells them the link was wrong. A token that never existed still
// answers 404, so nothing here confirms a guess.
func (s *Service) OpenSession(ctx context.Context, token string) (*ViewSession, error) {
	if token == "" {
		return nil, errors.EmptyDataException("view link not found")
	}
	session, err := s.repository.SessionByTokenHash(ctx, HashToken(token))
	if err != nil {
		return nil, err
	}
	if session.Expired(time.Now()) {
		return nil, errors.UnauthorizedException("this view link has expired")
	}
	return session, nil
}

// RevokeSession kills a link before it expires.
//
// Keyed on the session's id, not on the token the PRD's DELETE
// /view-session/{token} names. Storing only the hash is what forces it: the
// plaintext was shown once at creation and nothing can look a row up from a
// token nobody kept. The id is what the person revoking actually has — they
// are reading the list of live links — and scoping it to the document means a
// mistyped id cannot revoke a link on a document the caller never opened.
func (s *Service) RevokeSession(ctx context.Context, documentID, sessionID string) error {
	return s.repository.DeleteSession(ctx, documentID, sessionID)
}

func formatHours(d time.Duration) string {
	return strconv.Itoa(int(d.Hours())) + " hours"
}
