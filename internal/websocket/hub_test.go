package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// helper: starts a test server with a Hub and returns its WebSocket URL.
func setupTestHub(t *testing.T) (*Hub, string, func()) {
	t.Helper()
	hub := NewHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(hub.HandleWebSocket))
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	return hub, wsURL, server.Close
}

// helper: dials a WebSocket to the test server.
func dialWS(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to connect to websocket: %v", err)
	}
	return conn
}

func TestHub_Broadcast(t *testing.T) {
	hub, wsURL, cleanup := setupTestHub(t)
	defer cleanup()

	// Connect two clients
	conn1 := dialWS(t, wsURL)
	defer conn1.Close()
	conn2 := dialWS(t, wsURL)
	defer conn2.Close()

	// Give time for registration
	time.Sleep(50 * time.Millisecond)

	// Broadcast a message
	message := []byte(`{"speed":321}`)
	hub.Broadcast(message)

	// Both clients should receive it
	for i, conn := range []*websocket.Conn{conn1, conn2} {
		_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("client %d failed to read: %v", i+1, err)
		}
		if string(msg) != string(message) {
			t.Errorf("client %d got %q, want %q", i+1, string(msg), string(message))
		}
	}
}

func TestHub_BroadcastAfterDisconnect(t *testing.T) {
	hub, wsURL, cleanup := setupTestHub(t)
	defer cleanup()

	// Connect two clients
	conn1 := dialWS(t, wsURL)
	conn2 := dialWS(t, wsURL)
	defer conn2.Close()

	// Give time for registration
	time.Sleep(50 * time.Millisecond)

	// Disconnect client 1
	conn1.Close()
	time.Sleep(100 * time.Millisecond)

	// Broadcast — only conn2 should receive it
	message := []byte(`{"speed":200}`)
	hub.Broadcast(message)

	_ = conn2.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, msg, err := conn2.ReadMessage()
	if err != nil {
		t.Fatalf("remaining client failed to read: %v", err)
	}
	if string(msg) != string(message) {
		t.Errorf("got %q, want %q", string(msg), string(message))
	}
}
