package websocket

import (
	"context"
	"log"
	"sync"
)

type Broadcast struct {
	RoomID  string
	Data    []byte
	Exclude *Client
}

// Hub orchestrates websocket clients and room broadcasts.
type Hub struct {
	register   chan *Client
	unregister chan *Client
	broadcast  chan Broadcast

	mu      sync.RWMutex
	clients map[*Client]struct{}
	rooms   map[string]map[*Client]struct{}
}

// NewHub constructs an empty Hub instance.
func NewHub() *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Broadcast),
		clients:    make(map[*Client]struct{}),
		rooms:      make(map[string]map[*Client]struct{}),
	}
}

// Run starts the hub event loop.
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case client := <-h.register:
			h.addClient(client)
		case client := <-h.unregister:
			h.removeClient(client)
		case msg := <-h.broadcast:
			h.sendToRoom(msg)
		case <-ctx.Done():
			h.shutdown()
			return
		}
	}
}

// Register queues a client for inclusion in the hub.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister queue a client for removal from the hub.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// BroadcastToRoom enqueues a message for a specific room.
func (h *Hub) BroadcastToRoom(roomID string, data []byte) {
	h.broadcast <- Broadcast{RoomID: roomID, Data: data}
}

// BroadcastToRoomExcept enqueues a room message while omitting a specific client.
func (h *Hub) BroadcastToRoomExcept(roomID string, data []byte, exclude *Client) {
	h.broadcast <- Broadcast{RoomID: roomID, Data: data, Exclude: exclude}
}

func (h *Hub) addClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client] = struct{}{}
	for _, roomID := range client.Rooms {
		if h.rooms[roomID] == nil {
			h.rooms[roomID] = make(map[*Client]struct{})
		}
		h.rooms[roomID][client] = struct{}{}
	}
}

func (h *Hub) removeClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.clients, client)
	for _, roomID := range client.Rooms {
		if members, ok := h.rooms[roomID]; ok {
			delete(members, client)
			if len(members) == 0 {
				delete(h.rooms, roomID)
			}
		}
	}

	close(client.Send)
}

func (h *Hub) sendToRoom(msg Broadcast) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	targets := h.rooms[msg.RoomID]
	for client := range targets {
		if msg.Exclude != nil && client == msg.Exclude {
			continue
		}
		select {
		case client.Send <- msg.Data:
		default:
			//Slow consumer; drop and disconnect
			go h.Unregister(client)
		}
	}
}

func (h *Hub) shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.clients {
		close(client.Send)
	}
	h.clients = make(map[*Client]struct{})
	h.rooms = make(map[string]map[*Client]struct{})
	log.Println("websocket hub shut down")
}
