package ws

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	clients map[*websocket.Conn]bool
	lock    sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]bool),
	}
}

func (h *Hub) AddClient(conn *websocket.Conn) {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.clients[conn] = true
}

func (h *Hub) RemoveClient(conn *websocket.Conn) {
	h.lock.Lock()
	defer h.lock.Unlock()
	delete(h.clients, conn)
}

func (h *Hub) Broadcast(msg []byte) {
	h.lock.Lock()
	defer h.lock.Unlock()
	for c := range h.clients {
		err := c.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Printf("WebSocket send error: %v", err)
			c.Close()
			delete(h.clients, c)
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	h.AddClient(conn)
	defer h.RemoveClient(conn)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
