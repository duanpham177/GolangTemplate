package service

import (
	"errors"
	"math"
	"testing"

	"servergo/internal/repository"
)

func TestRecordTelemetry(t *testing.T) {
	svc := NewDeviceService(repository.NewTelemetryMemory(), nil)

	record, err := svc.RecordTelemetry("sensor-1", 23.5)
	if err != nil {
		t.Fatalf("RecordTelemetry() error = %v", err)
	}
	if record.DeviceID != "sensor-1" || record.Temperature != 23.5 || record.RecordedAt.IsZero() {
		t.Fatalf("unexpected telemetry record: %+v", record)
	}

	records, err := svc.ListTelemetry("sensor-1")
	if err != nil {
		t.Fatalf("ListTelemetry() error = %v", err)
	}
	if len(records) != 1 || records[0] != record {
		t.Fatalf("unexpected telemetry list: %+v", records)
	}
}

func TestRecordTelemetryRejectsInvalidInput(t *testing.T) {
	svc := NewDeviceService(repository.NewTelemetryMemory(), nil)

	if _, err := svc.RecordTelemetry(" ", 23.5); !errors.Is(err, ErrInvalidDeviceID) {
		t.Fatalf("expected ErrInvalidDeviceID, got %v", err)
	}
	if _, err := svc.RecordTelemetry("sensor-1", math.NaN()); !errors.Is(err, ErrInvalidTelemetry) {
		t.Fatalf("expected ErrInvalidTelemetry, got %v", err)
	}
}
