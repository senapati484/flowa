package main

import (
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)
		}
	}()

	// Send a message
	err = c.WriteMessage(websocket.TextMessage, []byte("Hello Flowa WS!"))
	if err != nil {
		log.Println("write:", err)
		return
	}

	// Wait for echo
	time.Sleep(1 * time.Second)

	// Close
	err = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Println("write close:", err)
		return
	}
	select {
	case <-done:
	case <-time.After(time.Second):
	}
}
