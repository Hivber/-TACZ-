package gamecore

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func readFrame(t *testing.T, conn net.Conn) []byte {
	t.Helper()
	var header [4]byte
	if _, err := io.ReadFull(conn, header[:]); err != nil {
		t.Fatal(err)
	}
	payload := make([]byte, binary.BigEndian.Uint32(header[:]))
	if _, err := io.ReadFull(conn, payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func writeFrame(t *testing.T, conn net.Conn, response Response) {
	t.Helper()
	payload, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
	if _, err = conn.Write(header[:]); err != nil {
		t.Fatal(err)
	}
	if _, err = conn.Write(payload); err != nil {
		t.Fatal(err)
	}
}

func TestClientReusesPersistentConnection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "core.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	var accepts atomic.Int32
	serverDone := make(chan error, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			serverDone <- acceptErr
			return
		}
		defer conn.Close()
		accepts.Add(1)
		for i := 0; i < 2; i++ {
			var request batchRequest
			var header [4]byte
			if _, err := io.ReadFull(conn, header[:]); err != nil {
				serverDone <- err
				return
			}
			payload := make([]byte, binary.BigEndian.Uint32(header[:]))
			if _, err := io.ReadFull(conn, payload); err != nil {
				serverDone <- err
				return
			}
			if err := json.Unmarshal(payload, &request); err != nil {
				serverDone <- err
				return
			}
			responseBytes, err := json.Marshal(Response{
				ProtocolVersion: ProtocolVersion, OK: true,
				Message: "batch_applied", ProcessedEvents: len(request.Events),
			})
			if err != nil {
				serverDone <- err
				return
			}
			binary.BigEndian.PutUint32(header[:], uint32(len(responseBytes)))
			if _, err = conn.Write(header[:]); err == nil {
				_, err = conn.Write(responseBytes)
			}
			if err != nil {
				serverDone <- err
				return
			}
		}
		serverDone <- nil
	}()

	client := &Client{SocketPath: path, Timeout: time.Second}
	for i := 0; i < 2; i++ {
		response, callErr := client.CallBatch(context.Background(), []Event{{Type: "tick"}})
		if callErr != nil {
			t.Fatal(callErr)
		}
		if response.ProcessedEvents != 1 {
			t.Fatalf("processed events = %d, want 1", response.ProcessedEvents)
		}
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
	if got := accepts.Load(); got != 1 {
		t.Fatalf("accepted %d connections, want one persistent connection", got)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestBridgeFlushesQueuedEventsOnClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bridge.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	received := make(chan int, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		var header [4]byte
		if _, err := io.ReadFull(conn, header[:]); err != nil {
			return
		}
		payload := make([]byte, binary.BigEndian.Uint32(header[:]))
		if _, err := io.ReadFull(conn, payload); err != nil {
			return
		}
		var request batchRequest
		if json.Unmarshal(payload, &request) != nil {
			return
		}
		responseBytes, err := json.Marshal(Response{
			ProtocolVersion: ProtocolVersion, OK: true,
			Message: "batch_applied", ProcessedEvents: len(request.Events),
		})
		if err != nil {
			return
		}
		binary.BigEndian.PutUint32(header[:], uint32(len(responseBytes)))
		if _, err = conn.Write(header[:]); err != nil {
			return
		}
		if _, err = conn.Write(responseBytes); err != nil {
			return
		}
		received <- len(request.Events)
	}()

	bridge := NewBridge(path, func(error) {})
	if !bridge.Enqueue(Event{Type: "player_join", UUID: "u", Name: "p"}) ||
		!bridge.Enqueue(Event{Type: "player_move", UUID: "u"}) {
		t.Fatal("failed to enqueue events")
	}
	bridge.Close()
	select {
	case count := <-received:
		if count != 2 {
			t.Fatalf("received %d events, want 2", count)
		}
	case <-time.After(time.Second):
		t.Fatal("bridge did not flush its queue on close")
	}
}
