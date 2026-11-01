package tcp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"servergo/internal/repository"
	"servergo/internal/service"
)

func TestServerAuthenticatesAndStoresTelemetry(t *testing.T) {
	manager := NewManager()
	deviceService := service.NewDeviceService(repository.NewTelemetryMemory(), manager)
	srv := NewServer("", "test-token", "", "", deviceService, manager, slog.New(slog.NewTextHandler(io.Discard, nil)))
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	done := make(chan struct{})
	go func() {
		srv.handleConnection(serverConn)
		close(done)
	}()
	defer clientConn.Close()

	reader := bufio.NewReader(clientConn)
	writeTestMessage(t, clientConn, Message{
		Version:   protocolVersion,
		Type:      "authenticate",
		DeviceID:  "sensor-1",
		RequestID: "auth-1",
		Token:     "test-token",
	})
	authResponse := readTestResponse(t, reader)
	if authResponse.Status != "authenticated" || authResponse.RequestID != "auth-1" {
		t.Fatalf("unexpected authentication response: %+v", authResponse)
	}

	payload, err := json.Marshal(map[string]float64{"temperature": 21.75})
	if err != nil {
		t.Fatal(err)
	}
	writeTestMessage(t, clientConn, Message{
		Version:   protocolVersion,
		Type:      "telemetry",
		DeviceID:  "sensor-1",
		RequestID: "telemetry-1",
		Payload:   payload,
	})
	telemetryResponse := readTestResponse(t, reader)
	if telemetryResponse.Status != "stored" || telemetryResponse.RequestID != "telemetry-1" {
		t.Fatalf("unexpected telemetry response: %+v", telemetryResponse)
	}

	records, err := deviceService.ListTelemetry("sensor-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Temperature != 21.75 {
		t.Fatalf("telemetry was not stored: %+v", records)
	}

	_ = clientConn.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("TCP connection handler did not stop")
	}
}

func TestServerRejectsInvalidToken(t *testing.T) {
	manager := NewManager()
	deviceService := service.NewDeviceService(repository.NewTelemetryMemory(), manager)
	srv := NewServer("", "expected-token", "", "", deviceService, manager, slog.New(slog.NewTextHandler(io.Discard, nil)))
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	done := make(chan struct{})
	go func() {
		srv.handleConnection(serverConn)
		close(done)
	}()
	defer clientConn.Close()

	writeTestMessage(t, clientConn, Message{
		Version:   protocolVersion,
		Type:      "authenticate",
		DeviceID:  "sensor-1",
		RequestID: "auth-1",
		Token:     "wrong-token",
	})
	authResponse := readTestResponse(t, bufio.NewReader(clientConn))
	if authResponse.Type != "error" {
		t.Fatalf("expected authentication error, got %+v", authResponse)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("rejected TCP connection did not close")
	}
}

func TestServeStopsWhenContextIsCancelled(t *testing.T) {
	manager := NewManager()
	deviceService := service.NewDeviceService(repository.NewTelemetryMemory(), manager)
	srv := NewServer("127.0.0.1:0", "test-token", "", "", deviceService, manager, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		done <- srv.Serve(ctx)
	}()
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("TCP server did not stop after context cancellation")
	}
}

func writeTestMessage(t *testing.T, conn net.Conn, message Message) {
	t.Helper()
	if err := writeLine(conn, message); err != nil {
		t.Fatalf("write message: %v", err)
	}
}

func readTestResponse(t *testing.T, reader *bufio.Reader) response {
	t.Helper()
	line, err := reader.ReadBytes('\n')
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	var result response
	if err := json.Unmarshal(line, &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return result
}
