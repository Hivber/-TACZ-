package item

import (
	"testing"

	"github.com/df-mc/dragonfly/server/world"
)

type shieldRepairTestItem string

func (i shieldRepairTestItem) EncodeItem() (string, int16) { return string(i), 0 }

func TestShieldGameplayProperties(t *testing.T) {
	shield := Shield{}
	if shield.MaxCount() != 1 {
		t.Fatalf("shield max count = %d, want 1", shield.MaxCount())
	}
	if !shield.OffHand() {
		t.Fatal("shield must be available in the off-hand")
	}
	if got := shield.DurabilityInfo().MaxDurability; got != 336 {
		t.Fatalf("shield durability = %d, want 336", got)
	}
	if name, meta := shield.EncodeItem(); name != "minecraft:shield" || meta != 0 {
		t.Fatalf("shield encodes as %q:%d", name, meta)
	}
	if got, ok := world.ItemByName("minecraft:shield", 0); !ok {
		t.Fatal("shield is not registered")
	} else if _, ok := got.(Shield); !ok {
		t.Fatalf("shield resolved to %T, want item.Shield", got)
	}
}

func TestShieldRepairableByPlanks(t *testing.T) {
	if !(Shield{}).RepairableBy(NewStack(shieldRepairTestItem("minecraft:oak_planks"), 1)) {
		t.Fatal("shield should be repairable with planks")
	}
	if (Shield{}).RepairableBy(Stack{}) {
		t.Fatal("empty item must not repair shield")
	}
}
