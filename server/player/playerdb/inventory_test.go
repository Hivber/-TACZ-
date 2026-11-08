package playerdb

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
)

func TestOffHandShieldInventoryRoundTrip(t *testing.T) {
	want := item.NewStack(item.Shield{}, 1)
	encoded := invToData(InventoryData{OffHand: want})
	got := dataToInv(encoded).OffHand

	if got.Empty() {
		t.Fatal("off-hand shield was lost during inventory serialization")
	}
	name, meta := got.Item().EncodeItem()
	if name != "minecraft:shield" || meta != 0 {
		t.Fatalf("off-hand item = %s:%d, want minecraft:shield:0", name, meta)
	}
	if got.Count() != 1 {
		t.Fatalf("off-hand shield count = %d, want 1", got.Count())
	}
}
