package grpcclient

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	telemetrypb "github.com/peri-Bot/f1-telemetry-backend/proto/gen/telemetrypb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// DataHandler processes encoded telemetry data for broadcasting.
type DataHandler func(data []byte)

// Client connects to the sidecar's gRPC TelemetryService and streams data.
type Client struct {
	addr    string
	handler DataHandler
	logger  *slog.Logger
}

// New creates a gRPC telemetry client targeting the given sidecar address.
func New(addr string, handler DataHandler) *Client {
	return &Client{
		addr:    addr,
		handler: handler,
		logger:  slog.Default(),
	}
}

// Start connects to the gRPC server and streams TelemetryBatch messages.
// It reconnects with exponential backoff on errors.
// Blocks until the context is cancelled.
func (c *Client) Start(ctx context.Context) error {
	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := c.stream(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}

		c.logger.Warn("gRPC stream disconnected, reconnecting...", "error", err, "backoff", backoff)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		// Exponential backoff
		backoff = backoff * 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// stream opens a single gRPC connection and reads TelemetryBatch messages.
func (c *Client) stream(ctx context.Context) error {
	c.logger.Info("Connecting to sidecar gRPC server", "addr", c.addr)

	conn, err := grpc.NewClient(
		c.addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := telemetrypb.NewTelemetryServiceClient(conn)
	stream, err := client.StreamTelemetry(ctx, &telemetrypb.StreamRequest{})
	if err != nil {
		return err
	}

	c.logger.Info("Connected — receiving telemetry stream")
	var packetCount uint64

	// Reset backoff on successful connection
	for {
		batch, err := stream.Recv()
		if err != nil {
			return err
		}

		packetCount++
		if packetCount%100 == 0 {
			c.logger.Info("Telemetry stream heartbeat",
				"packets_received", packetCount,
				"drivers_in_last_batch", len(batch.GetDrivers()))
		}

		data := batchToJSON(batch)
		if data != nil {
			c.handler(data)
		}
	}
}

// batchToJSON converts a TelemetryBatch proto into a JSON map suitable
// for broadcasting over WebSocket. The output shape matches the existing
// DriverTelemetry JSON format for frontend compatibility.
func batchToJSON(batch *telemetrypb.TelemetryBatch) []byte {
	result := make(map[string]interface{})
	for _, d := range batch.GetDrivers() {
		cd := d.GetCarData()
		pos := d.GetPosition()
		result[d.GetDriverNumber()] = map[string]interface{}{
			"m_header": map[string]interface{}{
				"m_playerCarIndex": 0, // placeholder
			},
			"m_carTelemetryData": map[string]interface{}{
				"m_speed":     cd.GetSpeed(),
				"m_engineRPM": cd.GetRpm(),
				"m_gear":      cd.GetGear(),
				"m_throttle":  cd.GetThrottle(),
				"m_brake":     cd.GetBrake(),
				"m_drs":       cd.GetDrs(),
			},
			"m_position": map[string]interface{}{
				"x":      pos.GetX(),
				"y":      pos.GetY(),
				"z":      pos.GetZ(),
				"status": pos.GetStatus(),
			},
		}
	}

	bytes, err := json.Marshal(result)
	if err != nil {
		return nil
	}
	return bytes
}

// SetSession sends a unary RPC to the sidecar to manually change the historic replay session
func (c *Client) SetSession(ctx context.Context, year int32, meeting string, session string) (*telemetrypb.SessionResponse, error) {
	conn, err := grpc.NewClient(
		c.addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := telemetrypb.NewTelemetryServiceClient(conn)
	req := &telemetrypb.SessionRequest{
		Year:    year,
		Meeting: meeting,
		Session: session,
	}
	return client.SetSession(ctx, req)
}
