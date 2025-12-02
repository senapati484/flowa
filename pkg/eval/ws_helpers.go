package eval

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins for now
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WebSocketConnection wraps the gorilla websocket connection
type WebSocketConnection struct {
	Conn *websocket.Conn
	kind ObjectKind
}

func (ws *WebSocketConnection) Kind() ObjectKind { return KindNative }
func (ws *WebSocketConnection) Inspect() string  { return "<WebSocketConnection>" }

func upgradeToWebSocket(w http.ResponseWriter, r *http.Request) (*WebSocketConnection, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return &WebSocketConnection{Conn: conn}, nil
}
