package ws

import (
	"encoding/json"
	"log"
)

type EventType string

const (
	EventChatMessage EventType = "chat.message"
	EventUserStatus  EventType = "user.status"
	EventJoinRoom    EventType = "user.joinRoom"
	EventCreateRoom  EventType = "user.createRoom"
	EventRoomsList   EventType = "user.roomsList"
	EventFetchRooms  EventType = "user.fetchRooms"
)

type ChatMessage struct {
	Author  string `json:"author"`
	Content string `json:"content"`
	Room    string `json:"room"`
	// Timestamp string `json:"timestamp"`
}

type UserStatus struct {
	User  string `json:"user"`
	State string `json:"state"`
}

type CreateRoom struct {
	User string `json:"user"`
	Room string `json:"room"`
}

type JoinRoom struct {
	User string `json:"user"`
	Room string `json:"room"`
}

type RoomsList struct {
	Rooms map[string]*Room
}

type FetchRooms struct {
	User string `json:"user"`
}

type Event struct {
	Type    EventType       `json:"type"`
	Payload json.RawMessage `json:"data"`
}

type Room struct {
	ID      string
	Clients map[string]*Client
}

type RoomInfo struct {
	ID string `json:"id"`
}

type RoomsListResponse struct {
	Rooms []RoomInfo `json:"rooms"`
}

type Hub struct {
	// Зарегистрированные клиенты
	clients map[string]*Client
	// Входящие сообщения от клиентов
	broadcast chan Event
	// Канал для регистрации
	register chan *Client
	// Канал для отмены регистрации
	unregister chan *Client

	rooms map[string]*Room
}

func (h *Hub) Register(c *Client) {
	h.register <- c
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan Event),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[string]*Client),
		rooms:      make(map[string]*Room),
	}
}

func (h *Hub) JoinRoom(roomID string, client *Client) {
	room, ok := h.rooms[roomID]
	if !ok {
		room = &Room{
			ID:      roomID,
			Clients: make(map[string]*Client),
		}
		h.rooms[roomID] = room
	}

	room.Clients[client.ID] = client
	client.Rooms[roomID] = true
}

func (h *Hub) RemoveClientFromRooms(client *Client) {
	for roomID := range client.Rooms {
		if room, ok := h.rooms[roomID]; ok {
			delete(room.Clients, client.ID)
			delete(client.Rooms, roomID)

			if len(room.Clients) == 0 {
				delete(h.rooms, roomID)
			}
		}
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client.ID] = client
		case client := <-h.unregister:
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				h.RemoveClientFromRooms(client)
				close(client.send)
			}
		case e := <-h.broadcast:
			jsonMsg, _ := json.Marshal(e)

			switch e.Type {
			case EventChatMessage:
				var p ChatMessage
				if err := json.Unmarshal(e.Payload, &p); err != nil {
					log.Printf("Error unmarshaling: %v", err)
					continue
				}

				switch p.Room {
				case "":
					for _, client := range h.clients {
						client.SendMessage([]byte(jsonMsg))
					}
				default:
					room, ok := h.rooms[p.Room]
					if !ok {
						continue
					}

					if _, isMember := room.Clients[p.Author]; !isMember {
						log.Println(p.Author + " - недостаточно прав")
						continue
					}

					for _, client := range room.Clients {
						client.SendMessage([]byte(jsonMsg))
					}
				}

			case EventUserStatus:
				for _, client := range h.clients {
					client.SendMessage([]byte(jsonMsg))
				}

			case EventJoinRoom:
				var p JoinRoom

				if err := json.Unmarshal(e.Payload, &p); err != nil {
					log.Printf("Error unmarshaling: %v", err)
				}

				if client, ok := h.clients[p.User]; ok {
					h.JoinRoom(p.Room, client)

					room, ok := h.rooms[p.Room]
					if !ok {
						continue
					}

					for _, client := range room.Clients {
						client.SendMessage([]byte(jsonMsg))
					}
				}
			case EventFetchRooms:
				var p FetchRooms

				if err := json.Unmarshal(e.Payload, &p); err != nil {
					log.Printf("Error unmarshaling: %v", err)
				}

				if client, ok := h.clients[p.User]; ok {
					// формируем ответ
					rooms := make([]RoomInfo, 0, len(h.rooms))
					for id := range h.rooms {
						rooms = append(rooms, RoomInfo{ID: id})
					}
					a := RoomsListResponse{Rooms: rooms}

					aJson, _ := json.Marshal(a)

					evt := Event{
						Type:    EventRoomsList,
						Payload: aJson,
					}

					evtJson, _ := json.Marshal(evt)

					log.Println(string(evtJson))
					client.SendMessage([]byte(evtJson))
				}
			}
		}
	}
}
