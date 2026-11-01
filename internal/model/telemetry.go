package model

import "time"

type Telemetry struct {
	DeviceID    string    `json:"device_id"`
	Temperature float64   `json:"temperature"`
	RecordedAt  time.Time `json:"recorded_at"`
}
