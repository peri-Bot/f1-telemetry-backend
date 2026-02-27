// internal/websocket/hub.go
package websocket

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

// allowedOrigins returns the CORS-aware origin checker.
// Reads from ALLOWED_ORIGINS env var (comma-separated). Defaults to "*" (all).
func newUpgrader() websocket.Upgrader {
	allowed := os.Getenv("ALLOWED_ORIGINS")

	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if allowed == "" || allowed == "*" {
				return true
			}
			origin := r.Header.Get("Origin")
			for _, o := range strings.Split(allowed, ",") {
				if strings.TrimSpace(o) == origin {
					return true
				}
			}
			return false
		},
	}
}

// Hub manages WebSocket client connections and broadcasts data.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	upgrader   websocket.Upgrader
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		upgrader:   newUpgrader(),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Println("Client registered")
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Println("Client unregistered")
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

// Broadcast sends data to all connected WebSocket clients.
func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}

// HandleWebSocket upgrades an HTTP connection to WebSocket and registers the client.
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{hub: h, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}
