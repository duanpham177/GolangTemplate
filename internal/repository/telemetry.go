package repository

import (
	"sync"

	"servergo/internal/model"
)

type TelemetryRepository interface {
	Save(telemetry model.Telemetry) error
	ListByDevice(deviceID string) []model.Telemetry
}

type telemetryMemory struct {
	mu      sync.RWMutex
	records []model.Telemetry
}

func NewTelemetryMemory() TelemetryRepository {
	return &telemetryMemory{}
}

func (r *telemetryMemory) Save(telemetry model.Telemetry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, telemetry)
	return nil
}

func (r *telemetryMemory) ListByDevice(deviceID string) []model.Telemetry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	records := make([]model.Telemetry, 0)
	for _, record := range r.records {
		if record.DeviceID == deviceID {
			records = append(records, record)
		}
	}
	return records
}
