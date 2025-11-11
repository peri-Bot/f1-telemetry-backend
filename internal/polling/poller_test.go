// internal/polling/poller_test.go
package polling

import (
	"encoding/json" // <-- Import the json package
	"net/http"
	"net/http/httptest"
	"reflect" // <-- Import the reflect package
	"testing"
	"time"
)

func TestPoller(t *testing.T) {
	// 1. ARRANGE
	const fakeJSON = `{"speed":321,"gear":8}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fakeJSON))
	}))
	defer server.Close()

	dataChan := make(chan []byte, 1)
	mockHandler := func(data []byte) {
		dataChan <- data
	}

	// 2. ACT
	p := NewPoller(server.URL, mockHandler)
	go p.StartPolling()

	// 3. ASSERT
	select {
	case receivedData := <-dataChan:
		// --- THE CORRECT WAY TO COMPARE JSON ---

		// a. Unmarshal the expected JSON into a map.
		var want map[string]interface{}
		if err := json.Unmarshal([]byte(fakeJSON), &want); err != nil {
			t.Fatalf("failed to unmarshal expected JSON: %v", err)
		}

		// b. Unmarshal the received JSON into another map.
		var got map[string]interface{}
		if err := json.Unmarshal(receivedData, &got); err != nil {
			t.Fatalf("failed to unmarshal received JSON: %v", err)
		}

		// c. Compare the two maps using reflect.DeepEqual.
		if !reflect.DeepEqual(got, want) {
			t.Errorf("handler received wrong data:\n got: %v\nwant: %v", got, want)
		}

	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for handler to be called")
	}
}


// internal/polling/poller_test.go
package polling

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPoller(t *testing.T) {
	// 1. ARRANGE: Set up the test conditions.

	// Define the fake JSON data our mock server will return.
	const fakeJSON = `{"speed":321,"gear":8}`

	// Create a mock HTTP server to simulate the Python sidecar.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// When the poller calls this server, it will get our fake JSON.
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fakeJSON))
	}))
	// Ensure the server is closed when the test finishes.
	defer server.Close()

	// Create a channel to receive the data. This is our mock handler.
	// It simulates the WebSocket hub by letting us "catch" the data the poller sends.
	dataChan := make(chan []byte, 1) // Buffered channel of size 1
	mockHandler := func(data []byte) {
		dataChan <- data
	}

	// 2. ACT: Execute the code we are testing.

	// Create a new poller instance, pointing it to our mock server and mock handler.
	p := NewPoller(server.URL, mockHandler)

	// Run the poller in a separate goroutine because it's an infinite loop.
	go p.StartPolling()

	// 3. ASSERT: Check if the results are correct.

	// Wait for the data to arrive on our channel, but with a timeout.
	// This prevents the test from hanging forever if the poller is broken.
	select {
	case receivedData := <-dataChan:
		// If we receive data, check if it matches what we expect.
		if string(receivedData) != fakeJSON {
			t.Errorf("handler received wrong data: got %s want %s", string(receivedData), fakeJSON)
		}
	case <-time.After(1 * time.Second):
		// If a second passes and we haven't received anything, fail the test.
		t.Fatal("timed out waiting for handler to be called")
	}
}
