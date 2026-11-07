package item

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/world"
)

func TestWindChargeGameplayProperties(t *testing.T) {
	charge := WindCharge{}
	if charge.MaxCount() != 64 {
		t.Fatalf("wind charge max count = %d, want 64", charge.MaxCount())
	}
	if charge.Cooldown() != 500*time.Millisecond {
		t.Fatalf("wind charge cooldown = %v, want 500ms", charge.Cooldown())
	}
	if name, meta := charge.EncodeItem(); name != "minecraft:wind_charge" || meta != 0 {
		t.Fatalf("wind charge encodes as %q:%d", name, meta)
	}
	got, ok := world.ItemByName("minecraft:wind_charge", 0)
	if !ok {
		t.Fatal("wind charge is not registered")
	}
	if _, ok := got.(WindCharge); !ok {
		t.Fatalf("wind charge resolved to %T, want item.WindCharge", got)
	}
}
