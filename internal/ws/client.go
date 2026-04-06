package ws

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Client struct {
	ID    string
	hub   *Hub
	conn  *websocket.Conn
	send  chan []byte // Буферизированный канал для сообщений
	Rooms map[string]bool
}

func Online() {

}

func (c *Client) SendMessage(msg []byte) {
	select {
	case c.send <- msg:
	default:
	}
}

func NewClient(hub *Hub, conn *websocket.Conn, id string) *Client {
	return &Client{
		ID:    id,
		hub:   hub,
		conn:  conn,
		send:  make(chan []byte, 256),
		Rooms: make(map[string]bool),
	}
}

// readPump: Читает сообщения из сокета и отправляет их в Hub
func (c *Client) ReadPump() {
	data := UserStatus{
		User:  c.ID,
		State: "online",
	}

	payload, _ := json.Marshal(data)
	evt := Event{
		Type:    EventUserStatus,
		Payload: json.RawMessage(payload),
	}

	c.hub.broadcast <- evt
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
		data := UserStatus{
			User:  c.ID,
			State: "offline",
		}

		payload, _ := json.Marshal(data)
		evt := Event{
			Type:    EventUserStatus,
			Payload: json.RawMessage(payload),
		}

		c.hub.broadcast <- evt
	}()
	for {
		_, rawData, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("error: %v", err)
			break
		}
		var evt Event

		if err := json.Unmarshal(rawData, &evt); err != nil {
			log.Printf("error decoding json: %v", err)
			continue
		}

		switch evt.Type {
		case EventChatMessage:
			var p ChatMessage

			json.Unmarshal(evt.Payload, &p)
			p.Author = c.ID
			newPayload, _ := json.Marshal(p)
			evt.Payload = json.RawMessage(newPayload)
		case EventUserStatus:
			// var p UserStatus

			// json.Unmarshal(evt.Payload, &p)

		case EventJoinRoom:
			var p JoinRoom

			json.Unmarshal(evt.Payload, &p)
			p.User = c.ID
			newPayload, _ := json.Marshal(p)
			evt.Payload = json.RawMessage(newPayload)

		case EventFetchRooms:
			var p FetchRooms

			json.Unmarshal(evt.Payload, &p)
			p.User = c.ID
			newPayload, _ := json.Marshal(p)
			evt.Payload = json.RawMessage(newPayload)

		default:
			log.Println("Unknown message type:" + evt.Type)
		}

		c.hub.broadcast <- evt
	}
}

// writePump: Берет сообщения из канала c.send и пишет их в сокет
func (c *Client) WritePump() {
	defer c.conn.Close()
	for {
		message, ok := <-c.send

		if !ok {
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
		c.conn.WriteMessage(websocket.TextMessage, message)
	}
}
