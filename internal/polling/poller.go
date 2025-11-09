// internal/polling/poller.go
package polling

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// Define a type for the function that will handle the data
type DataHandler func(data []byte)

type poller struct {
	sidecarURL string
	client     *http.Client
	handler    DataHandler
}

func NewPoller(url string, handler DataHandler) *poller {
	return &poller{
		sidecarURL: url,
		client: &http.Client{
			Timeout: 2 * time.Second,
		},
		handler: handler,
	}
}

func (p *poller) StartPolling() {
	log.Printf("Starting to poll sidecar at %s", p.sidecarURL)
	// Use a ticker for regular intervals
	ticker := time.NewTicker(100 * time.Millisecond) // Poll 10 times per second
	defer ticker.Stop()

	for range ticker.C {
		resp, err := p.client.Get(p.sidecarURL)
		if err != nil {
			log.Printf("Error polling sidecar: %v", err)
			continue // Don't stop, just try again on the next tick
		}
		defer resp.Body.Close()

		// For now, we'll just pass the raw bytes.
		// A real implementation would parse into our TelemetryData model.
		var data map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			log.Printf("Error decoding sidecar response: %v", err)
			continue
		}

		// Don't broadcast if the data is empty
		if len(data) == 0 {
			continue
		}

		// Convert back to bytes to send over WebSocket
		bytes, err := json.Marshal(data)
		if err != nil {
			continue
		}

		// Call the handler function we were given
		p.handler(bytes)
	}
}
