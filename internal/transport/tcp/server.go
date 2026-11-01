package tcp

import (
	"bufio"
	"context"
	"crypto/subtle"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"servergo/internal/model"
	"servergo/internal/service"
)

const (
	authTimeout   = 10 * time.Second
	idleTimeout   = 2 * time.Minute
	maxIdentifier = 128
)

type TelemetryRecorder interface {
	RecordTelemetry(deviceID string, temperature float64) (model.Telemetry, error)
}

type Server struct {
	addr        string
	token       string
	certFile    string
	keyFile     string
	telemetry   TelemetryRecorder
	manager     *Manager
	logger      *slog.Logger
	connections map[net.Conn]struct{}
	mu          sync.Mutex
	wg          sync.WaitGroup
}

func NewServer(addr, token, certFile, keyFile string, telemetry TelemetryRecorder, manager *Manager, logger *slog.Logger) *Server {
	return &Server{
		addr:        addr,
		token:       token,
		certFile:    certFile,
		keyFile:     keyFile,
		telemetry:   telemetry,
		manager:     manager,
		logger:      logger,
		connections: make(map[net.Conn]struct{}),
	}
}

func (s *Server) Serve(ctx context.Context) error {
	if s.token == "" {
		return errors.New("TCP token must not be empty")
	}
	listener, err := s.listen()
	if err != nil {
		return fmt.Errorf("start TCP listener: %w", err)
	}

	watchDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = listener.Close()
			s.closeConnections()
		case <-watchDone:
		}
	}()

	var serveErr error
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() == nil {
				serveErr = fmt.Errorf("accept TCP connection: %w", err)
			}
			break
		}

		s.track(conn)
		if ctx.Err() != nil {
			s.untrack(conn)
			break
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer s.untrack(conn)
			s.handleConnection(conn)
		}()
	}

	close(watchDone)
	_ = listener.Close()
	s.closeConnections()
	s.wg.Wait()
	return serveErr
}

func (s *Server) listen() (net.Listener, error) {
	if s.certFile == "" && s.keyFile == "" {
		return net.Listen("tcp", s.addr)
	}
	cert, err := tls.LoadX509KeyPair(s.certFile, s.keyFile)
	if err != nil {
		return nil, fmt.Errorf("load TCP TLS certificate: %w", err)
	}
	return tls.Listen("tcp", s.addr, &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	})
}

func (s *Server) handleConnection(conn net.Conn) {
	client := &clientConn{conn: conn}
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 4096), maxMessageSize)

	if err := conn.SetReadDeadline(time.Now().Add(authTimeout)); err != nil {
		s.logger.Debug("set TCP authentication deadline failed", "err", err)
		return
	}
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			s.logger.Debug("TCP authentication read failed", "err", err)
		}
		return
	}

	auth, err := decodeMessage(scanner.Bytes())
	if err != nil || auth.Type != "authenticate" || !validIdentifier(auth.DeviceID) ||
		auth.RequestID == "" || !validIdentifier(auth.RequestID) || !validToken(auth.Token, s.token) {
		s.writeResponse(client, response{
			Version: protocolVersion,
			Type:    "error",
			Error:   "authentication failed",
		})
		return
	}
	deviceID := strings.TrimSpace(auth.DeviceID)

	if !s.writeResponse(client, response{
		Version:   protocolVersion,
		Type:      "ack",
		RequestID: auth.RequestID,
		Status:    "authenticated",
	}) {
		return
	}

	s.manager.register(deviceID, client)
	defer s.manager.unregister(deviceID, client)
	s.logger.Info("TCP device connected", "device_id", deviceID, "remote_addr", conn.RemoteAddr())

	defer s.logger.Info("TCP device disconnected", "device_id", deviceID)
	for {
		if err := conn.SetReadDeadline(time.Now().Add(idleTimeout)); err != nil {
			s.logger.Debug("set TCP device read deadline failed", "device_id", deviceID, "err", err)
			return
		}
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				s.logger.Debug("TCP device read failed", "device_id", deviceID, "err", err)
			}
			return
		}

		message, err := decodeMessage(scanner.Bytes())
		if err != nil {
			s.writeResponse(client, response{Version: protocolVersion, Type: "error", Error: "invalid message"})
			return
		}
		if message.DeviceID != deviceID || !validIdentifier(message.RequestID) {
			if !s.writeResponse(client, response{
				Version:   protocolVersion,
				Type:      "error",
				RequestID: message.RequestID,
				Error:     "invalid device_id or request_id",
			}) {
				return
			}
			continue
		}

		switch message.Type {
		case "telemetry":
			if !s.handleTelemetry(client, deviceID, message) {
				return
			}
		case "ping":
			if !s.writeResponse(client, response{
				Version:   protocolVersion,
				Type:      "pong",
				RequestID: message.RequestID,
				Status:    "ok",
			}) {
				return
			}
		case "ack":
			s.logger.Debug("TCP command acknowledged", "device_id", deviceID, "request_id", message.RequestID)
		default:
			if !s.writeResponse(client, response{
				Version:   protocolVersion,
				Type:      "error",
				RequestID: message.RequestID,
				Error:     "unsupported message type",
			}) {
				return
			}
		}
	}
}

func (s *Server) handleTelemetry(client *clientConn, deviceID string, message Message) bool {
	var payload struct {
		Temperature *float64 `json:"temperature"`
	}
	if len(message.Payload) == 0 || json.Unmarshal(message.Payload, &payload) != nil || payload.Temperature == nil {
		return s.writeResponse(client, response{
			Version:   protocolVersion,
			Type:      "error",
			RequestID: message.RequestID,
			Error:     "payload must contain temperature",
		})
	}

	if _, err := s.telemetry.RecordTelemetry(deviceID, *payload.Temperature); err != nil {
		if !errors.Is(err, service.ErrInvalidTelemetry) && !errors.Is(err, service.ErrInvalidDeviceID) {
			s.logger.Error("store TCP telemetry failed", "device_id", deviceID, "err", err)
			return s.writeResponse(client, response{
				Version:   protocolVersion,
				Type:      "error",
				RequestID: message.RequestID,
				Error:     "could not store telemetry",
			})
		}
		return s.writeResponse(client, response{
			Version:   protocolVersion,
			Type:      "error",
			RequestID: message.RequestID,
			Error:     "invalid telemetry",
		})
	}
	return s.writeResponse(client, response{
		Version:   protocolVersion,
		Type:      "ack",
		RequestID: message.RequestID,
		Status:    "stored",
	})
}

func (s *Server) writeResponse(client *clientConn, value response) bool {
	if err := client.write(value); err != nil {
		s.logger.Debug("TCP response write failed", "remote_addr", client.conn.RemoteAddr(), "err", err)
		return false
	}
	return true
}

func (s *Server) track(conn net.Conn) {
	s.mu.Lock()
	s.connections[conn] = struct{}{}
	s.mu.Unlock()
}

func (s *Server) untrack(conn net.Conn) {
	s.mu.Lock()
	delete(s.connections, conn)
	s.mu.Unlock()
	_ = conn.Close()
}

func (s *Server) closeConnections() {
	s.mu.Lock()
	connections := make([]net.Conn, 0, len(s.connections))
	for conn := range s.connections {
		connections = append(connections, conn)
	}
	s.mu.Unlock()
	for _, conn := range connections {
		_ = conn.Close()
	}
}

func validIdentifier(value string) bool {
	return strings.TrimSpace(value) != "" && len(value) <= maxIdentifier
}

func validToken(got, expected string) bool {
	return len(got) == len(expected) && subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}
