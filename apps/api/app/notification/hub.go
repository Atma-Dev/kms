package notification

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	// The request already passed WSTokenAuthMiddleware before reaching here — this
	// is a same-origin SPA with no cookie-based session to forge cross-site,
	// so origin checking adds no real protection.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Hub fans a user's notifications out to every WebSocket connection they
// currently have open. In-process only — correct for the current
// single-instance deployment.
//
// ponytail: in-memory map only reaches connections on this instance; if the
// app ever runs behind multiple replicas, fan out via Redis pub/sub instead
// (redis is already a dependency) rather than this map.
type Hub struct {
	mu    sync.Mutex
	conns map[string][]*websocket.Conn
}

func NewHub() *Hub {
	return &Hub{conns: make(map[string][]*websocket.Conn)}
}

func (h *Hub) Register(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[userID] = append(h.conns[userID], conn)
}

func (h *Hub) Unregister(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	conns := h.conns[userID]
	for i, c := range conns {
		if c == conn {
			h.conns[userID] = append(conns[:i], conns[i+1:]...)
			break
		}
	}
	if len(h.conns[userID]) == 0 {
		delete(h.conns, userID)
	}
}

// SendToUser is best-effort: a write failure just drops silently, the read
// loop that owns the connection (Handler.Serve) is what unregisters it once
// it notices the socket is gone.
func (h *Hub) SendToUser(userID string, payload any) {
	h.mu.Lock()
	conns := append([]*websocket.Conn(nil), h.conns[userID]...)
	h.mu.Unlock()

	for _, conn := range conns {
		_ = conn.WriteJSON(payload)
	}
}

// Upgrade promotes an HTTP request to a WebSocket connection and registers
// it under userID until the caller's read loop exits.
func (h *Hub) Upgrade(w http.ResponseWriter, r *http.Request, userID string) (*websocket.Conn, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	h.Register(userID, conn)
	return conn, nil
}
