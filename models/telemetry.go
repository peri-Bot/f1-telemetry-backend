// models/telemetry.go
package models

import (
	telemetrypb "github.com/peri-Bot/f1-telemetry-backend/proto/gen/telemetrypb"
)

// Header contains metadata for a telemetry packet.
type Header struct {
	PlayerCarIndex int `json:"m_playerCarIndex"`
}

// CarTelemetry holds the real-time car performance data.
type CarTelemetry struct {
	Speed     int     `json:"m_speed"`
	EngineRPM int     `json:"m_engineRPM"`
	Gear      int     `json:"m_gear"`
	Throttle  float64 `json:"m_throttle"`
	Brake     float64 `json:"m_brake"`
	DRS       bool    `json:"m_drs"`
}

// Position holds the car's 3D position on track.
type Position struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Z      float64 `json:"z"`
	Status string  `json:"status"`
}

// TelemetryPacket is a single driver's telemetry snapshot.
type TelemetryPacket struct {
	Header       Header       `json:"m_header"`
	CarTelemetry CarTelemetry `json:"m_carTelemetryData"`
	Position     Position     `json:"m_position"`
}

// DriverTelemetry maps driver number (as string) to their telemetry packet.
type DriverTelemetry map[string]TelemetryPacket

// FromProto converts a protobuf TelemetryBatch to DriverTelemetry.
func FromProto(batch *telemetrypb.TelemetryBatch) DriverTelemetry {
	dt := make(DriverTelemetry, len(batch.GetDrivers()))
	for _, d := range batch.GetDrivers() {
		cd := d.GetCarData()
		pos := d.GetPosition()
		dt[d.GetDriverNumber()] = TelemetryPacket{
			Header: Header{PlayerCarIndex: 0},
			CarTelemetry: CarTelemetry{
				Speed:     int(cd.GetSpeed()),
				EngineRPM: int(cd.GetRpm()),
				Gear:      int(cd.GetGear()),
				Throttle:  cd.GetThrottle(),
				Brake:     cd.GetBrake(),
				DRS:       cd.GetDrs(),
			},
			Position: Position{
				X:      pos.GetX(),
				Y:      pos.GetY(),
				Z:      pos.GetZ(),
				Status: pos.GetStatus(),
			},
		}
	}
	return dt
}
