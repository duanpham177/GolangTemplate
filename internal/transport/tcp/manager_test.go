package tcp

import (
	"bufio"
	"encoding/json"
	"net"
	"testing"
)

func TestManagerSendsCommandToConnectedDevice(t *testing.T) {
	serverConn, deviceConn := net.Pipe()
	defer serverConn.Close()
	defer deviceConn.Close()

	manager := NewManager()
	manager.register("sensor-1", &clientConn{conn: serverConn})

	writeDone := make(chan error, 1)
	go func() {
		writeDone <- manager.SendCommand("sensor-1", "restart")
	}()

	var message Message
	if err := json.NewDecoder(bufio.NewReader(deviceConn)).Decode(&message); err != nil {
		t.Fatalf("read command: %v", err)
	}
	if message.Type != "command" || message.DeviceID != "sensor-1" || message.RequestID == "" {
		t.Fatalf("unexpected command: %+v", message)
	}
	var payload map[string]string
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		t.Fatalf("decode command payload: %v", err)
	}
	if payload["command"] != "restart" {
		t.Fatalf("unexpected command payload: %+v", payload)
	}
	if err := <-writeDone; err != nil {
		t.Fatalf("SendCommand() error = %v", err)
	}
}
