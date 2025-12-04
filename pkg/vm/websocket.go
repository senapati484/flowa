package vm

import (
	"flowa/pkg/eval"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096, // Increased from 1024 for better performance
	WriteBufferSize: 4096, // Increased from 1024 for better performance
	// Allow all origins (for development - tighten in production)
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WebSocketUpgrade upgrades an HTTP connection to WebSocket
func WebSocketUpgrade(w http.ResponseWriter, r *http.Request) (*eval.WebSocketConnection, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, fmt.Errorf("websocket upgrade failed: %v", err)
	}
	return &eval.WebSocketConnection{Conn: conn}, nil
}

// WebSocketSend sends a message over WebSocket
func WebSocketSend(conn *websocket.Conn, message string) error {
	return conn.WriteMessage(websocket.TextMessage, []byte(message))
}

// WebSocketReceive receives a message from WebSocket
func WebSocketReceive(conn *websocket.Conn) (string, error) {
	msgType, msg, err := conn.ReadMessage()
	if err != nil {
		return "", err
	}
	if msgType != websocket.TextMessage {
		return "", fmt.Errorf("unexpected message type: %d", msgType)
	}
	return string(msg), nil
}

// WebSocketClose closes a WebSocket connection
func WebSocketClose(conn *websocket.Conn) error {
	return conn.Close()
}
