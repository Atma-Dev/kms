package notification

import (
	"context"
	"encoding/json"
	"api-kms/errors"
)

// Notifier is the single entry point other domains use to send a
// notification: persist it, then push it live to any open socket for that
// user. Persist-then-push keeps the stored feed and the live event always
// consistent — nothing is pushed that isn't also in the list.
type Notifier struct {
	repo *Repository
	hub  *Hub
}

func NewNotifier(repo *Repository, hub *Hub) *Notifier {
	return &Notifier{repo: repo, hub: hub}
}

func (n *Notifier) Notify(ctx context.Context, userID, notifType, title, body string, data map[string]any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return errors.InternalServerError("failed to encode notification data").WithCause(err)
	}

	notif := &Notification{
		UserID: userID,
		Type:   notifType,
		Title:  title,
		Body:   body,
		Data:   raw,
	}
	if err := n.repo.Create(ctx, notif); err != nil {
		return err
	}

	n.hub.SendToUser(userID, NewResource(notif))
	return nil
}
