package notification

// Control frames share the notification socket but are not notifications:
// nothing is persisted and nothing appears in the bell. The frontend tells
// them apart by the "event" key, which a notification resource never carries.
const EventPermissionsChanged = "permissions.changed"

type controlFrame struct {
	Event string `json:"event"`
}

// BroadcastPermissionsChanged tells each listed user to re-read its
// permissions. Best effort, like SendToUser: a user with no socket open picks
// the change up on their next page load, which re-reads /profiles/me anyway.
//
// Callers hold this through a one-method interface of their own rather than
// importing this package — notification imports user, so user and rbac cannot
// import back.
func (h *Hub) BroadcastPermissionsChanged(userIDs []string) {
	frame := controlFrame{Event: EventPermissionsChanged}
	for _, userID := range userIDs {
		h.SendToUser(userID, frame)
	}
}
