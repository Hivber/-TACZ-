package entity

import (
	"testing"

	"github.com/go-gl/mathgl/mgl64"
)

func TestWindChargeBurstConstants(t *testing.T) {
	if windChargeBurstRadius != 2 {
		t.Fatalf("wind charge burst radius = %v, want 2", windChargeBurstRadius)
	}
}

func TestWindChargeImpulse(t *testing.T) {
	tests := []struct {
		name     string
		velocity mgl64.Vec3
		offset   mgl64.Vec3
		want     mgl64.Vec3
	}{
		{
			name:     "centre adds upward force and damps velocity",
			velocity: mgl64.Vec3{0.4, 0.2, -0.6},
			offset:   mgl64.Vec3{},
			want:     mgl64.Vec3{0.2, 0.7, -0.3},
		},
		{
			name:     "radial force",
			velocity: mgl64.Vec3{},
			offset:   mgl64.Vec3{1, 0, -2},
			want:     mgl64.Vec3{0.2, 0.6, -0.4},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := windChargeImpulse(test.velocity, test.offset); got != test.want {
				t.Fatalf("windChargeImpulse() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestWindChargeBurstRange(t *testing.T) {
	if (mgl64.Vec3{windChargeBurstRadius, 0, 0}).Len() < windChargeBurstRadius {
		t.Fatal("wind charge burst radius check is inconsistent")
	}
}
