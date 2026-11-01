package service

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"servergo/internal/model"
	"servergo/internal/repository"
)

var (
	ErrInvalidDeviceID  = errors.New("device id is required")
	ErrInvalidTelemetry = errors.New("temperature must be a finite number")
	ErrInvalidCommand   = errors.New("command is required")
	ErrDeviceOffline    = errors.New("device is not connected")
)

type CommandSender interface {
	SendCommand(deviceID, command string) error
}

type DeviceService struct {
	telemetry repository.TelemetryRepository
	commands  CommandSender
}

func NewDeviceService(telemetry repository.TelemetryRepository, commands CommandSender) *DeviceService {
	return &DeviceService{telemetry: telemetry, commands: commands}
}

func (s *DeviceService) RecordTelemetry(deviceID string, temperature float64) (model.Telemetry, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return model.Telemetry{}, ErrInvalidDeviceID
	}
	if math.IsNaN(temperature) || math.IsInf(temperature, 0) {
		return model.Telemetry{}, ErrInvalidTelemetry
	}

	record := model.Telemetry{
		DeviceID:    deviceID,
		Temperature: temperature,
		RecordedAt:  time.Now().UTC(),
	}
	if err := s.telemetry.Save(record); err != nil {
		return model.Telemetry{}, fmt.Errorf("save telemetry: %w", err)
	}
	return record, nil
}

func (s *DeviceService) ListTelemetry(deviceID string) ([]model.Telemetry, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil, ErrInvalidDeviceID
	}
	return s.telemetry.ListByDevice(deviceID), nil
}

func (s *DeviceService) SendCommand(deviceID, command string) error {
	deviceID = strings.TrimSpace(deviceID)
	command = strings.TrimSpace(command)
	if deviceID == "" {
		return ErrInvalidDeviceID
	}
	if command == "" {
		return ErrInvalidCommand
	}
	if err := s.commands.SendCommand(deviceID, command); err != nil {
		if errors.Is(err, ErrDeviceOffline) {
			return ErrDeviceOffline
		}
		return err
	}
	return nil
}
