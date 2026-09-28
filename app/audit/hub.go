package audit

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	// The request already passed WSTokenAuthMiddleware + RequirePermission
	// before reaching here — this is a same-origin SPA with no cookie-based
	// session to forge cross-site, so origin checking adds no real protection.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Hub fans a newly persisted audit out to every WebSocket connection
// currently open on the audits page. In-process only — see app/activitylog's
// Hub for the note on why (single-instance deployment; fan out via Redis
// pub/sub instead if that ever changes).
type Hub struct {
	mu    sync.Mutex
	conns map[*websocket.Conn]struct{}
}

func NewHub() *Hub {
	return &Hub{conns: make(map[*websocket.Conn]struct{})}
}

func (h *Hub) Register(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[conn] = struct{}{}
}

func (h *Hub) Unregister(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns, conn)
}

// Broadcast is best-effort: a write failure just drops silently, the read
// loop that owns the connection (Handler.Serve) is what unregisters it once
// it notices the socket is gone.
func (h *Hub) Broadcast(payload any) {
	h.mu.Lock()
	conns := make([]*websocket.Conn, 0, len(h.conns))
	for conn := range h.conns {
		conns = append(conns, conn)
	}
	h.mu.Unlock()

	for _, conn := range conns {
		_ = conn.WriteJSON(payload)
	}
}

// Upgrade promotes an HTTP request to a WebSocket connection and registers
// it until the caller's read loop exits.
func (h *Hub) Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	h.Register(conn)
	return conn, nil
}
