package grpcclient

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	telemetrypb "github.com/peri-Bot/f1-telemetry-backend/proto/gen/telemetrypb"
	"google.golang.org/grpc"
)

// mockTelemetryServer implements the TelemetryService server for testing.
type mockTelemetryServer struct {
	telemetrypb.UnimplementedTelemetryServiceServer
	batches []*telemetrypb.TelemetryBatch
}

func (s *mockTelemetryServer) StreamTelemetry(_ *telemetrypb.StreamRequest, stream grpc.ServerStreamingServer[telemetrypb.TelemetryBatch]) error {
	for _, batch := range s.batches {
		if err := stream.Send(batch); err != nil {
			return err
		}
	}
	return nil // stream ends after sending all batches
}

// startMockServer starts a gRPC server with the mock telemetry service.
func startMockServer(t *testing.T, batches []*telemetrypb.TelemetryBatch) (string, func()) {
	t.Helper()
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	server := grpc.NewServer()
	telemetrypb.RegisterTelemetryServiceServer(server, &mockTelemetryServer{batches: batches})
	go func() {
		_ = server.Serve(lis)
	}()

	return lis.Addr().String(), server.Stop
}

func TestClient_ReceivesBatch(t *testing.T) {
	// Arrange: a mock server that sends one batch
	batch := &telemetrypb.TelemetryBatch{
		Timestamp: "2024-07-28T14:00:00.000Z",
		Drivers: []*telemetrypb.DriverUpdate{
			{
				DriverNumber: "44",
				CarData: &telemetrypb.CarData{
					Speed:    321,
					Rpm:      11877,
					Gear:     8,
					Throttle: 1.0,
					Brake:    0.0,
					Drs:      true,
				},
				Position: &telemetrypb.Position{
					X: 100.5, Y: -200.3, Z: 10.0, Status: "OnTrack",
				},
			},
			{
				DriverNumber: "1",
				CarData: &telemetrypb.CarData{
					Speed: 290, Rpm: 10730, Gear: 8, Throttle: 0.9, Brake: 0.0, Drs: true,
				},
				Position: &telemetrypb.Position{
					X: 150.0, Y: -180.0, Z: 11.0, Status: "OnTrack",
				},
			},
		},
	}

	addr, cleanup := startMockServer(t, []*telemetrypb.TelemetryBatch{batch})
	defer cleanup()

	// Act: client connects and receives
	received := make(chan []byte, 1)
	handler := func(data []byte) {
		received <- data
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	client := New(addr, handler)
	go func() {
		_ = client.Start(ctx)
	}()

	// Assert
	select {
	case data := <-received:
		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		// Should contain both drivers
		if _, ok := result["44"]; !ok {
			t.Error("expected driver '44' in output")
		}
		if _, ok := result["1"]; !ok {
			t.Error("expected driver '1' in output")
		}

		// Check speed for driver 44
		driver44 := result["44"].(map[string]interface{})
		carData := driver44["m_carTelemetryData"].(map[string]interface{})
		if speed := carData["m_speed"].(float64); speed != 321 {
			t.Errorf("speed: got %v, want 321", speed)
		}

	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for data")
	}
}
