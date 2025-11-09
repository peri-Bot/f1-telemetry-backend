// internal/interfaces.go
package internal

import "net/http"

// Hub is the contract for our WebSocket hub.
// It manages all client connections and broadcasts data.
type Hub interface {
	Run()
	HandleWebSocket(w http.ResponseWriter, r *http.Request)
}

// DataPoller is the contract for the service that fetches data from the sidecar.
type DataPoller interface {
	StartPolling()
}
