package configwebsocket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait       = 10 * time.Second
	pongWait        = 60 * time.Second
	pingPeriod      = (pongWait * 9) / 10
	maxMessageSize  = 512
	websocketBuffer = 1024
)

type subscription struct {
	conn *connection
	room string
}

type connection struct {
	ws  *websocket.Conn
	send chan []byte
}

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

func (s *subscription) readPump() {
	c := s.conn
	defer func() {
		H.unregister <- *s
		c.ws.Close()
	}()
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error { 
		c.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, msg, err := c.ws.ReadMessage()
		fmt.Println("test msg", msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("read error: %v", err)
			}
			break
		}
		fmt.Println("Received message from client:", string(msg))

		msg = bytes.TrimSpace(bytes.Replace(msg, newline, space, -1))
		m := message{s.room, msg}
		H.broadcast <- m
	}
}

func (s *subscription) writePump() {
	c := s.conn
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()
	fmt.Println("send message")
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				return
			}
			// messageWithTime := fmt.Sprintf("room [%s] [%s] %s", s.room, time.Now().Format(time.RFC3339), message)
			m := map[string]interface{}{
				"time": time.Now().Format(time.RFC3339),
				"room": s.room,
				"message": []byte(message),
			}
			jsonData, err := json.Marshal(m)
			if err != nil {
				log.Fatal(err)
			}
			if err := c.write(websocket.TextMessage, jsonData); err != nil {
				fmt.Println(websocket.TextMessage)
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// write payload of the message
func (c *connection) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	fmt.Println(string(payload[:]))
	return c.ws.WriteMessage(mt, payload)
}

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal("this error", err)
	}
	// Get room's id from client...
	queryValues := r.URL.Query()
	roomId := queryValues.Get("roomId")
	if err != nil {
		log.Println(err)
		return
	}
	c := &connection{ws, make(chan []byte, 256)}
	s := subscription{c, roomId}
	log.Println("test", roomId)
	hub.register <- s
	go s.writePump()
	go s.readPump()

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			message := "This is an automated message."
			s.conn.send <- []byte(message)
		}
	}()
}