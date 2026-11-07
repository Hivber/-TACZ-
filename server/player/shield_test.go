package player

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl64"
)

func TestShieldCoversOnlyFrontHemisphere(t *testing.T) {
	tests := []struct {
		name     string
		rotation cube.Rotation
		source   mgl64.Vec3
		want     bool
	}{
		{name: "front", rotation: cube.Rotation{0, 0}, source: mgl64.Vec3{0, 0, 4}, want: true},
		{name: "behind", rotation: cube.Rotation{0, 0}, source: mgl64.Vec3{0, 0, -4}, want: false},
		{name: "side", rotation: cube.Rotation{0, 0}, source: mgl64.Vec3{4, 0, 0}, want: false},
		{name: "rotated front", rotation: cube.Rotation{90, 0}, source: mgl64.Vec3{-4, 0, 0}, want: true},
		{name: "rotated behind", rotation: cube.Rotation{90, 0}, source: mgl64.Vec3{4, 0, 0}, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shieldCovers(test.rotation, mgl64.Vec3{}, test.source); got != test.want {
				t.Fatalf("shieldCovers() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestShieldRaiseDelayAndDurabilityThreshold(t *testing.T) {
	if shieldRaiseDelay != 250*time.Millisecond {
		t.Fatalf("shield raise delay = %v, want 250ms", shieldRaiseDelay)
	}
	for _, test := range []struct {
		attack float64
		want   int
	}{
		{attack: 2, want: 0},
		{attack: 2.99, want: 0},
		{attack: 3, want: 3},
		{attack: 3.1, want: 4},
		{attack: 7.5, want: 8},
	} {
		if got := shieldDurabilityDamage(test.attack); got != test.want {
			t.Errorf("shieldDurabilityDamage(%v) = %d, want %d", test.attack, got, test.want)
		}
	}
}
