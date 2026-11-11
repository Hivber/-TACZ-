// Package gamecore implements Dragonfly's semantic-event bridge to the Rust gameplay core.
// It deliberately has no Bedrock packet, RakNet, login, or encryption dependencies.
package gamecore

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

const (
	ProtocolVersion uint16 = 1
	maxFrameSize           = 1 << 20
	maxBatchEvents         = 4096
)

type Position struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Z     float64 `json:"z"`
	Yaw   float32 `json:"yaw"`
	Pitch float32 `json:"pitch"`
}

type Event struct {
	Type            string   `json:"type"`
	ProtocolVersion uint16   `json:"protocol_version,omitempty"`
	UUID            string   `json:"uuid,omitempty"`
	Name            string   `json:"name,omitempty"`
	Dimension       int32    `json:"dimension,omitempty"`
	GameMode        uint8    `json:"game_mode,omitempty"`
	Position        Position `json:"position,omitempty"`
}

type batchRequest struct {
	Type            string  `json:"type"`
	ProtocolVersion uint16  `json:"protocol_version"`
	Events          []Event `json:"events"`
}

type pingRequest struct {
	Type            string `json:"type"`
	ProtocolVersion uint16 `json:"protocol_version"`
}

type Response struct {
	ProtocolVersion uint16 `json:"protocol_version"`
	OK              bool   `json:"ok"`
	Message         string `json:"message"`
	OnlinePlayers   int    `json:"online_players"`
	ProcessedEvents int    `json:"processed_events"`
}

// Client serializes access to one persistent local IPC connection. A failed
// exchange invalidates the stream; the next call reconnects automatically.
type Client struct {
	SocketPath string
	Timeout    time.Duration

	mu   sync.Mutex
	conn net.Conn
}

func (c *Client) Call(ctx context.Context, event Event) (Response, error) {
	if event.Type == "ping" {
		return c.call(ctx, pingRequest{Type: "ping", ProtocolVersion: ProtocolVersion})
	}
	return c.CallBatch(ctx, []Event{event})
}

func (c *Client) CallBatch(ctx context.Context, events []Event) (Response, error) {
	if len(events) == 0 {
		return Response{ProtocolVersion: ProtocolVersion, OK: true, Message: "empty_batch"}, nil
	}
	if len(events) > maxBatchEvents {
		return Response{}, fmt.Errorf("game core batch has %d events; maximum is %d", len(events), maxBatchEvents)
	}
	for i := range events {
		events[i].ProtocolVersion = ProtocolVersion
	}
	return c.call(ctx, batchRequest{Type: "batch", ProtocolVersion: ProtocolVersion, Events: events})
}

func (c *Client) call(ctx context.Context, request any) (Response, error) {
	if c.SocketPath == "" {
		return Response{}, errors.New("game core socket path is empty")
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return Response{}, fmt.Errorf("marshal game core request: %w", err)
	}
	if len(payload) == 0 || len(payload) > maxFrameSize {
		return Response{}, fmt.Errorf("game core request frame size %d is invalid", len(payload))
	}

	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		dialer := net.Dialer{Timeout: timeout}
		c.conn, err = dialer.DialContext(ctx, "unix", c.SocketPath)
		if err != nil {
			return Response{}, fmt.Errorf("connect to Rust gameplay core: %w", err)
		}
	}
	deadline := time.Now().Add(timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	if err = c.conn.SetDeadline(deadline); err != nil {
		c.resetLocked()
		return Response{}, fmt.Errorf("set IPC deadline: %w", err)
	}

	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
	if err = writeAll(c.conn, header[:]); err == nil {
		err = writeAll(c.conn, payload)
	}
	if err != nil {
		c.resetLocked()
		return Response{}, fmt.Errorf("write game core IPC request: %w", err)
	}
	if _, err = io.ReadFull(c.conn, header[:]); err != nil {
		c.resetLocked()
		return Response{}, fmt.Errorf("read game core frame header: %w", err)
	}
	size := binary.BigEndian.Uint32(header[:])
	if size == 0 || size > maxFrameSize {
		c.resetLocked()
		return Response{}, fmt.Errorf("invalid response frame size %d", size)
	}
	responseBytes := make([]byte, size)
	if _, err = io.ReadFull(c.conn, responseBytes); err != nil {
		c.resetLocked()
		return Response{}, fmt.Errorf("read game core response: %w", err)
	}
	var response Response
	if err = json.Unmarshal(responseBytes, &response); err != nil {
		c.resetLocked()
		return Response{}, fmt.Errorf("decode game core response: %w", err)
	}
	if response.ProtocolVersion != ProtocolVersion {
		c.resetLocked()
		return Response{}, fmt.Errorf("game core protocol mismatch: Go=%d Rust=%d", ProtocolVersion, response.ProtocolVersion)
	}
	if !response.OK {
		return response, fmt.Errorf("rust gameplay core rejected request: %s", response.Message)
	}
	return response, nil
}

func writeAll(writer io.Writer, payload []byte) error {
	for len(payload) > 0 {
		n, err := writer.Write(payload)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		payload = payload[n:]
	}
	return nil
}

func (c *Client) resetLocked() {
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.Call(ctx, Event{Type: "ping"})
	return err
}
