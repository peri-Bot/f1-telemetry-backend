// cmd/server/main.go
package main

import (
	"log"
	"net/http"
	"os"

	// Use the correct, full import path for your project
	"github.com/peri-Bot/f1-telemetry-backend/internal/polling"
	"github.com/peri-Bot/f1-telemetry-backend/internal/websocket"
)

func main() {
	hub := websocket.NewHub()
	go hub.Run()

	sidecarURL := os.Getenv("SIDECAR_API_URL")
	if sidecarURL == "" {
		sidecarURL = "http://localhost:5000/data"
	}

	// The poller's handler is the hub's Broadcast method. This wires them together.
	poller := polling.NewPoller(sidecarURL, hub.Broadcast)
	go poller.StartPolling()

	// The root endpoint is just for health checks
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Backend server is running"))
	})

	// The /ws endpoint is handled by our hub
	http.HandleFunc("/ws", hub.HandleWebSocket)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
