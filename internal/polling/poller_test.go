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
