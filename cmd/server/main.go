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

	"github.com/peri-Bot/f1-telemetry-backend/internal/grpcclient"
	"github.com/peri-Bot/f1-telemetry-backend/internal/websocket"
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	hub := websocket.NewHub()
	go hub.Run()

	// gRPC sidecar address
	sidecarAddr := os.Getenv("SIDECAR_GRPC_ADDR")
	if sidecarAddr == "" {
		sidecarAddr = "localhost:50051"
	}

	// Create a context that is cancelled on shutdown signal
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Wire the gRPC client: stream → hub.Broadcast
	grpcClient := grpcclient.New(sidecarAddr, hub.Broadcast)
	go func() {
		if err := grpcClient.Start(ctx); err != nil && ctx.Err() == nil {
			logger.Error("gRPC client error", "error", err)
		}
	}()

	// Serve static files from frontend directory
	fs := http.FileServer(http.Dir("./frontend"))
	http.Handle("/", fs)

	// The /ws endpoint is handled by our hub
	http.HandleFunc("/ws", hub.HandleWebSocket)

	// Health check endpoint for Kubernetes
	http.HandleFunc("/health", healthHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr: ":" + port,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting server", "port", port, "sidecar_grpc_addr", sidecarAddr)
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

	// Cancel gRPC stream context
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	logger.Info("Server exiting")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
