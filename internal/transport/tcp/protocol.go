package tcp

import (
	"encoding/json"
	"errors"
	"io"
)

const (
	protocolVersion = 1
	maxMessageSize  = 64 * 1024
)

type Message struct {
	Version   int             `json:"version"`
	Type      string          `json:"type"`
	DeviceID  string          `json:"device_id,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
	Token     string          `json:"token,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type response struct {
	Version   int    `json:"version"`
	Type      string `json:"type"`
	RequestID string `json:"request_id,omitempty"`
	Status    string `json:"status,omitempty"`
	Error     string `json:"error,omitempty"`
}

func writeLine(w io.Writer, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	for len(data) > 0 {
		n, err := w.Write(data)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
}

func decodeMessage(data []byte) (Message, error) {
	var message Message
	if err := json.Unmarshal(data, &message); err != nil {
		return Message{}, err
	}
	if message.Version != protocolVersion {
		return Message{}, errors.New("unsupported protocol version")
	}
	return message, nil
}
