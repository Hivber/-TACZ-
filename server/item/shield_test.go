package item

import (
	"testing"

	"github.com/df-mc/dragonfly/server/world"
)

type shieldRepairTestItem string

func (i shieldRepairTestItem) EncodeItem() (string, int16) { return string(i), 0 }

func TestShieldRegisteredWithDedicatedImplementation(t *testing.T) {
	got, ok := world.ItemByName("minecraft:shield", 0)
	if !ok {
		t.Fatal("shield is not registered")
	}
	if _, ok := got.(Shield); !ok {
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
