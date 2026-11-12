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

type ItemStack struct {
	Identifier string   `json:"identifier"`
	Count      uint16   `json:"count"`
	Metadata   uint32   `json:"metadata"`
	UserData   []uint16 `json:"user_data,omitempty"`
}

type Event struct {
	Type            string             `json:"type"`
	ProtocolVersion uint16             `json:"protocol_version,omitempty"`
	UUID            string             `json:"uuid,omitempty"`
	Name            string             `json:"name,omitempty"`
	Dimension       int32              `json:"dimension,omitempty"`
	GameMode        uint8              `json:"game_mode,omitempty"`
	Position        Position           `json:"position,omitempty"`
	Target          *BlockTarget       `json:"target,omitempty"`
	Profile         *BlockBreakProfile `json:"profile,omitempty"`
	Tool            *MiningTool        `json:"tool,omitempty"`
	Miner           *MinerState        `json:"miner,omitempty"`
	EyePosition     [3]float32         `json:"eye_position,omitempty"`
	MaxReach        float32            `json:"max_reach,omitempty"`
	Tick            uint64             `json:"tick,omitempty"`
	SlotCount       int                `json:"slot_count,omitempty"`
	Slot            int                `json:"slot,omitempty"`
	Stack           *ItemStack         `json:"stack,omitempty"`
	Source          int                `json:"source,omitempty"`
	Destination     int                `json:"destination,omitempty"`
	Count           uint16             `json:"count,omitempty"`
}

type BlockPos struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
	Z int32 `json:"z"`
}

type BlockTarget struct {
	Position BlockPos `json:"position"`
	Face     uint8    `json:"face"`
}

type BlockBreakProfile struct {
	Hardness            float32 `json:"hardness"`
	ToolType            uint8   `json:"tool_type"`
	HarvestLevel        uint8   `json:"harvest_level"`
	RequiresCorrectTool bool    `json:"requires_correct_tool"`
}

type MiningTool struct {
	ToolType              uint8   `json:"tool_type"`
	HarvestLevel          uint8   `json:"harvest_level"`
	EfficiencyCorrect     float32 `json:"efficiency_correct"`
	EfficiencyIncorrect   float32 `json:"efficiency_incorrect"`
	EfficiencyEnchantment uint8   `json:"efficiency_enchantment"`
	SilkTouch             bool    `json:"silk_touch"`
	Fortune               uint8   `json:"fortune"`
	Unbreaking            uint8   `json:"unbreaking"`
}

type MinerState struct {
	OnGround                      bool  `json:"on_ground"`
	Flying                        bool  `json:"flying"`
	UnderwaterWithoutAquaAffinity bool  `json:"underwater_without_aqua_affinity"`
	Haste                         uint8 `json:"haste"`
	MiningFatigue                 uint8 `json:"mining_fatigue"`
}

type MiningResult struct {
	UUID       string  `json:"uuid"`
	Status     string  `json:"status"`
	Progress   float32 `json:"progress,omitempty"`
	CrackStage uint8   `json:"crack_stage,omitempty"`
	Error      string  `json:"error,omitempty"`
}

type InventoryResult struct {
	UUID      string       `json:"uuid"`
	Status    string       `json:"status"`
	Error     string       `json:"error,omitempty"`
	Slots     []*ItemStack `json:"slots,omitempty"`
	Remainder *ItemStack   `json:"remainder,omitempty"`
	Removed   *ItemStack   `json:"removed,omitempty"`
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
	ProtocolVersion uint16         `json:"protocol_version"`
	OK              bool           `json:"ok"`
	Message         string         `json:"message"`
	OnlinePlayers   int            `json:"online_players"`
	ProcessedEvents int            `json:"processed_events"`
	MiningResults   []MiningResult    `json:"mining_results,omitempty"`
	InventoryResults []InventoryResult `json:"inventory_results,omitempty"`
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
	if c == nil {
		return Response{}, errors.New("game core client is nil")
	}
	if len(events) == 0 {
		return Response{ProtocolVersion: ProtocolVersion, OK: true, Message: "empty_batch"}, nil
	}
	if len(events) > maxBatchEvents {
		return Response{}, fmt.Errorf("game core batch has %d events; maximum is %d", len(events), maxBatchEvents)
	}
	normalized := append([]Event(nil), events...)
	for i := range normalized {
		normalized[i].ProtocolVersion = ProtocolVersion
	}
	response, err := c.call(ctx, batchRequest{Type: "batch", ProtocolVersion: ProtocolVersion, Events: normalized})
	if err != nil {
		return response, err
	}
	if response.ProcessedEvents != len(normalized) {
		return response, fmt.Errorf("game core processed %d of %d events", response.ProcessedEvents, len(normalized))
	}
	return response, nil
}

func (c *Client) call(ctx context.Context, request any) (Response, error) {
	if c == nil || c.SocketPath == "" {
		return Response{}, errors.New("game core socket path is empty")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Response{}, err
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
	if err = c.conn.SetDeadline(time.Time{}); err != nil {
		c.resetLocked()
		return Response{}, fmt.Errorf("clear IPC deadline: %w", err)
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
	if c == nil {
		return nil
	}
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
