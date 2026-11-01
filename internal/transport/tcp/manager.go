package tcp

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"servergo/internal/service"
)

type clientConn struct {
	conn    net.Conn
	writeMu sync.Mutex
}

func (c *clientConn) write(value any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if err := c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}
	defer c.conn.SetWriteDeadline(time.Time{})
	return writeLine(c.conn, value)
}

type Manager struct {
	mu      sync.RWMutex
	clients map[string]*clientConn
	nextID  atomic.Uint64
}

func NewManager() *Manager {
	return &Manager{clients: make(map[string]*clientConn)}
}

func (m *Manager) register(deviceID string, client *clientConn) {
	m.mu.Lock()
	old := m.clients[deviceID]
	m.clients[deviceID] = client
	m.mu.Unlock()

	if old != nil && old != client {
		_ = old.conn.Close()
	}
}

func (m *Manager) unregister(deviceID string, client *clientConn) {
	m.mu.Lock()
	if m.clients[deviceID] == client {
		delete(m.clients, deviceID)
	}
	m.mu.Unlock()
}

func (m *Manager) SendCommand(deviceID, command string) error {
	m.mu.RLock()
	client := m.clients[deviceID]
	m.mu.RUnlock()
	if client == nil {
		return service.ErrDeviceOffline
	}

	payload, err := json.Marshal(map[string]string{"command": command})
	if err != nil {
		return fmt.Errorf("encode command payload: %w", err)
	}
	requestID := fmt.Sprintf("cmd-%d", m.nextID.Add(1))
	message := Message{
		Version:   protocolVersion,
		Type:      "command",
		DeviceID:  deviceID,
		RequestID: requestID,
		Payload:   payload,
	}
	if err := client.write(message); err != nil {
		m.unregister(deviceID, client)
		_ = client.conn.Close()
		return fmt.Errorf("%w: write command: %v", service.ErrDeviceOffline, err)
	}
	return nil
}
