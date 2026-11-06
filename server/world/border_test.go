package world

import (
	"testing"
	"time"

	"github.com/go-gl/mathgl/mgl64"
)

func TestWorldBorderClamp(t *testing.T) {
	border := DefaultWorldBorder().WithCenter(10, -10).WithSize(20, 0)
	got := border.Clamp(mgl64.Vec3{9, 64, -9}, mgl64.Vec3{30, 70, -30})
	if got[0] != 20 || got[1] != 70 || got[2] != -20 {
		t.Fatalf("clamped position = %v, want [20 70 -20]", got)
	}
}

func TestWorldBorderAllowsMovementBackInside(t *testing.T) {
	border := DefaultWorldBorder().WithSize(100, 0)
	outside := mgl64.Vec3{60, 64, 0}
	inward := border.Clamp(outside, mgl64.Vec3{55, 64, 0})
	if inward[0] != 55 {
		t.Fatalf("inward movement was clipped to %v", inward)
	}
	outward := border.Clamp(outside, mgl64.Vec3{65, 64, 0})
	if outward[0] != 60 {
		t.Fatalf("outward movement = %v, want current position 60", outward[0])
	}
}

func TestWorldBorderSizeTransition(t *testing.T) {
	start := time.Now()
	border := DefaultWorldBorder().WithSize(100, 10*time.Second)
	middle := border.SizeAt(start.Add(5 * time.Second))
	if middle >= DefaultWorldBorderSize || middle <= 100 {
		t.Fatalf("transition midpoint size = %f", middle)
	}
	if end := border.SizeAt(start.Add(11 * time.Second)); end != 100 {
		t.Fatalf("transition end size = %f, want 100", end)
	}
}
