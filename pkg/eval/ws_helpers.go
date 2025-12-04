package eval

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096, // Increased from 1024 for better performance
	WriteBufferSize: 4096, // Increased from 1024 for better performance
	// Allow all origins for development - configure this for production
	// TODO: Make this configurable via environment variable
	CheckOrigin: func(r *http.Request) bool {
		// For production, check against whitelist:
		// origin := r.Header.Get("Origin")
		// return isAllowedOrigin(origin)
		return true
	},
}

// WebSocketConnection wraps the gorilla websocket connection
type WebSocketConnection struct {
	Conn *websocket.Conn
	kind ObjectKind
	mu   sync.Mutex // Protects concurrent writes
}

func (ws *WebSocketConnection) Kind() ObjectKind { return KindNative }
func (ws *WebSocketConnection) Inspect() string  { return "<WebSocketConnection>" }

func upgradeToWebSocket(w http.ResponseWriter, r *http.Request) (*WebSocketConnection, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	// Set message size limit (512KB max)
	conn.SetReadLimit(512 * 1024)

	return &WebSocketConnection{Conn: conn}, nil
}
