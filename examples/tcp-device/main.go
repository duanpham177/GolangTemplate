package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"time"
)

type message struct {
	Version   int             `json:"version"`
	Type      string          `json:"type"`
	DeviceID  string          `json:"device_id,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
	Token     string          `json:"token,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type response struct {
	Version   int             `json:"version"`
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Status    string          `json:"status,omitempty"`
	Error     string          `json:"error,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	token := os.Getenv("TCP_TOKEN")
	if token == "" {
		return errors.New("TCP_TOKEN must be set")
	}
	addr := os.Getenv("TCP_ADDR")
	if addr == "" {
		addr = "127.0.0.1:9000"
	}
	deviceID := os.Getenv("DEVICE_ID")
	if deviceID == "" {
		deviceID = "sensor-001"
	}

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("connect to TCP server: %w", err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(24 * time.Hour)); err != nil {
		return fmt.Errorf("set connection deadline: %w", err)
	}

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(bufio.NewReader(conn))
	if err := encoder.Encode(message{
		Version:   1,
		Type:      "authenticate",
		DeviceID:  deviceID,
		RequestID: "auth-1",
		Token:     token,
	}); err != nil {
		return fmt.Errorf("send authentication: %w", err)
	}
	var auth response
	if err := decoder.Decode(&auth); err != nil {
		return fmt.Errorf("read authentication response: %w", err)
	}
	if auth.Type != "ack" || auth.Status != "authenticated" {
		return fmt.Errorf("device authentication failed: %s", auth.Error)
	}

	payload, err := json.Marshal(map[string]float64{"temperature": 23.5})
	if err != nil {
		return fmt.Errorf("encode telemetry: %w", err)
	}
	if err := encoder.Encode(message{
		Version:   1,
		Type:      "telemetry",
		DeviceID:  deviceID,
		RequestID: "telemetry-1",
		Payload:   payload,
	}); err != nil {
		return fmt.Errorf("send telemetry: %w", err)
	}
	for {
		var incoming response
		if err := decoder.Decode(&incoming); err != nil {
			return fmt.Errorf("read telemetry response: %w", err)
		}
		if incoming.Type == "ack" && incoming.RequestID == "telemetry-1" && incoming.Status == "stored" {
			break
		}
		if incoming.Type == "error" {
			return fmt.Errorf("server rejected telemetry: %s", incoming.Error)
		}
		if err := handleServerMessage(encoder, deviceID, incoming); err != nil {
			return err
		}
	}
	fmt.Println("Telemetry stored. Waiting for commands; press Ctrl+C to stop.")

	type readResult struct {
		message response
		err     error
	}
	incomingMessages := make(chan readResult)
	go func() {
		for {
			var incoming response
			err := decoder.Decode(&incoming)
			incomingMessages <- readResult{message: incoming, err: err}
			if err != nil {
				return
			}
		}
	}()

	heartbeat := time.NewTicker(time.Minute)
	defer heartbeat.Stop()
	pingID := 0
	for {
		select {
		case <-heartbeat.C:
			pingID++
			if err := encoder.Encode(message{
				Version:   1,
				Type:      "ping",
				DeviceID:  deviceID,
				RequestID: fmt.Sprintf("ping-%d", pingID),
			}); err != nil {
				return fmt.Errorf("send heartbeat: %w", err)
			}
		case result := <-incomingMessages:
			if result.err != nil {
				return fmt.Errorf("read server message: %w", result.err)
			}
			if err := handleServerMessage(encoder, deviceID, result.message); err != nil {
				return err
			}
		}
	}
}

func handleServerMessage(encoder *json.Encoder, deviceID string, incoming response) error {
	if incoming.Type != "command" {
		fmt.Printf("Server message: type=%s status=%s error=%s\n", incoming.Type, incoming.Status, incoming.Error)
		return nil
	}

	var command struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(incoming.Payload, &command); err != nil {
		return fmt.Errorf("decode command: %w", err)
	}
	fmt.Printf("Received command: %s\n", command.Command)
	if err := encoder.Encode(message{
		Version:   1,
		Type:      "ack",
		DeviceID:  deviceID,
		RequestID: incoming.RequestID,
	}); err != nil {
		return fmt.Errorf("acknowledge command: %w", err)
	}
	return nil
}
