// cmd/server/main.go
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Use the correct, full import path for your project
	"github.com/peri-Bot/f1-telemetry-backend/internal/polling"
	"github.com/peri-Bot/f1-telemetry-backend/internal/websocket"
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	hub := websocket.NewHub()
	go hub.Run()

	sidecarURL := os.Getenv("SIDECAR_API_URL")
	if sidecarURL == "" {
		sidecarURL = "http://localhost:5000/data"
	}

	// The poller's handler is the hub's Broadcast method. This wires them together.
	poller := polling.NewPoller(sidecarURL, hub.Broadcast)
	go poller.StartPolling()

	// Serve static files from frontend directory
	fs := http.FileServer(http.Dir("./frontend"))
	http.Handle("/", fs)

	// The /ws endpoint is handled by our hub
	http.HandleFunc("/ws", hub.HandleWebSocket)

	// Health check endpoint for Kubernetes
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr: ":" + port,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting server", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	logger.Info("Server exiting")
}
