// Package gamecore implements Dragonfly's local semantic-event bridge to the Rust gameplay core.
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
	"time"
)

const ProtocolVersion uint16 = 1
const maxFrameSize = 1 << 20

type Position struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Z     float64 `json:"z"`
	Yaw   float32 `json:"yaw"`
	Pitch float32 `json:"pitch"`
}

type Event struct {
	Type            string   `json:"type"`
	ProtocolVersion uint16   `json:"protocol_version"`
	UUID            string   `json:"uuid,omitempty"`
	Name            string   `json:"name,omitempty"`
	Dimension       int32    `json:"dimension,omitempty"`
	GameMode        uint8    `json:"game_mode,omitempty"`
	Position        Position `json:"position,omitempty"`
}

type Response struct {
	ProtocolVersion uint16 `json:"protocol_version"`
	OK               bool   `json:"ok"`
	Message          string `json:"message"`
	OnlinePlayers    int    `json:"online_players"`
}

type Client struct {
	SocketPath string
	Timeout    time.Duration
}

func (c Client) Call(ctx context.Context, event Event) (Response, error) {
	if c.SocketPath == "" {
		return Response{}, errors.New("game core socket path is empty")
	}
	if event.ProtocolVersion == 0 {
		event.ProtocolVersion = ProtocolVersion
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return Response{}, fmt.Errorf("marshal game core event: %w", err)
	}
	if len(payload) == 0 || len(payload) > maxFrameSize {
		return Response{}, fmt.Errorf("game core event frame size %d is invalid", len(payload))
	}

	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "unix", c.SocketPath)
	if err != nil {
		return Response{}, fmt.Errorf("connect to Rust gameplay core: %w", err)
	}
	defer conn.Close()
	deadline := time.Now().Add(timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	_ = conn.SetDeadline(deadline)

	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
	if _, err = conn.Write(header[:]); err != nil {
		return Response{}, fmt.Errorf("write game core frame header: %w", err)
	}
	if _, err = conn.Write(payload); err != nil {
		return Response{}, fmt.Errorf("write game core event: %w", err)
	}
	if _, err = io.ReadFull(conn, header[:]); err != nil {
		return Response{}, fmt.Errorf("read game core frame header: %w", err)
	}
	size := binary.BigEndian.Uint32(header[:])
	if size == 0 || size > maxFrameSize {
		return Response{}, fmt.Errorf("invalid game core response frame size %d", size)
	}
	responseBytes := make([]byte, size)
	if _, err = io.ReadFull(conn, responseBytes); err != nil {
		return Response{}, fmt.Errorf("read game core response: %w", err)
	}
	var response Response
	if err = json.Unmarshal(responseBytes, &response); err != nil {
		return Response{}, fmt.Errorf("decode game core response: %w", err)
	}
	if response.ProtocolVersion != ProtocolVersion {
		return Response{}, fmt.Errorf("game core protocol mismatch: Go=%d Rust=%d", ProtocolVersion, response.ProtocolVersion)
	}
	if !response.OK {
		return response, fmt.Errorf("Rust gameplay core rejected event: %s", response.Message)
	}
	return response, nil
}

func (c Client) Ping(ctx context.Context) error {
	_, err := c.Call(ctx, Event{Type: "ping"})
	return err
}
