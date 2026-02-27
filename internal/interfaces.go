// internal/interfaces.go
package internal

import (
	"context"
	"net/http"
)

// Hub is the contract for our WebSocket hub.
// It manages all client connections and broadcasts data.
type Hub interface {
	Run()
	HandleWebSocket(w http.ResponseWriter, r *http.Request)
	Broadcast(message []byte)
}

// DataSource is the contract for the service that provides telemetry data.
// Implementations include gRPC stream consumers (or HTTP pollers for fallback).
type DataSource interface {
	Start(ctx context.Context) error
}
