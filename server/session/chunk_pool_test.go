package session

import (
	"bytes"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

func TestEncodeBlockEntityPayloadBorderCount(t *testing.T) {
	got := encodeBlockEntityPayload(nil, nil, nil, true)
	if !bytes.Equal(got, []byte{0}) {
		t.Fatalf("payload = %v, want one zero border-count byte", got)
	}
}

func TestEncodeBlockEntityPayloadEmpty(t *testing.T) {
	if got := encodeBlockEntityPayload(nil, nil, nil, false); got != nil {
		t.Fatalf("payload = %v, want nil", got)
	}
	first := encodeBlockEntityPayload(map[cube.Pos]world.Block{}, nil, nil, true)
	second := encodeBlockEntityPayload(nil, nil, nil, true)
	if !bytes.Equal(first, []byte{0}) || !bytes.Equal(second, []byte{0}) {
		t.Fatalf("unexpected pooled payloads: %v and %v", first, second)
	}
}
