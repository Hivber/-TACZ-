package entity

import (
	"testing"

	"github.com/go-gl/mathgl/mgl64"
)

func TestWindChargeImpulse(t *testing.T) {
	if got := windChargeImpulse(mgl64.Vec3{}); got != (mgl64.Vec3{0, windChargeVertical, 0}) {
		t.Fatalf("centre impulse = %v, want vertical impulse", got)
	}
	if got := windChargeImpulse(mgl64.Vec3{windChargeBurstRadius, 0, 0}); got.LenSqr() != 0 {
		t.Fatalf("impulse at burst edge = %v, want zero", got)
	}
	got := windChargeImpulse(mgl64.Vec3{1, 0, 0})
	if got[0] <= 0 || got[1] <= 0 || got[2] != 0 {
		t.Fatalf("unexpected horizontal burst impulse: %v", got)
	}
}
