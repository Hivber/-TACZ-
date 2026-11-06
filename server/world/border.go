package world

import (
	"math"
	"time"

	"github.com/go-gl/mathgl/mgl64"
)

// DefaultWorldBorderSize is the default vanilla world border diameter in blocks.
const DefaultWorldBorderSize = 59999968.0

// WorldBorder describes a square horizontal movement boundary. Size is the
// diameter of the border, not its radius. The border is stored on a World and
// changed through a world transaction.
type WorldBorder struct {
	centerX, centerZ float64
	startSize        float64
	targetSize       float64
	lerpStarted      time.Time
	lerpDuration     time.Duration
}

// DefaultWorldBorder returns the default vanilla world border centred at the
// origin.
func DefaultWorldBorder() WorldBorder {
	return WorldBorder{startSize: DefaultWorldBorderSize, targetSize: DefaultWorldBorderSize}
}

// Center returns the X/Z coordinates of the world border centre.
func (b WorldBorder) Center() (float64, float64) {
	b = b.normalised()
	return b.centerX, b.centerZ
}

// Size returns the current world border diameter in blocks.
func (b WorldBorder) Size() float64 {
	return b.SizeAt(time.Now())
}

// TargetSize returns the final diameter after any active size transition.
func (b WorldBorder) TargetSize() float64 {
	b = b.normalised()
	return b.targetSize
}

// WithCenter returns a copy of the border with a new centre.
func (b WorldBorder) WithCenter(x, z float64) WorldBorder {
	b = b.normalised()
	b.centerX, b.centerZ = x, z
	return b
}

// WithSize returns a copy of the border transitioning to size over duration.
// A non-positive duration applies the size immediately.
func (b WorldBorder) WithSize(size float64, duration time.Duration) WorldBorder {
	b = b.normalised()
	now := time.Now()
	b.startSize = b.SizeAt(now)
	b.targetSize = size
	b.lerpStarted = now
	b.lerpDuration = duration
	if duration <= 0 {
		b.startSize = size
		b.lerpStarted = time.Time{}
		b.lerpDuration = 0
	}
	return b
}

// SizeAt returns the border diameter at the time passed.
func (b WorldBorder) SizeAt(now time.Time) float64 {
	b = b.normalised()
	if b.lerpDuration <= 0 {
		return b.targetSize
	}
	elapsed := now.Sub(b.lerpStarted)
	if elapsed <= 0 {
		return b.startSize
	}
	if elapsed >= b.lerpDuration {
		return b.targetSize
	}
	progress := float64(elapsed) / float64(b.lerpDuration)
	return b.startSize + (b.targetSize-b.startSize)*progress
}

// Clamp constrains a proposed player position to the border. Players already
// outside the border may move back towards the inside, but cannot move farther
// out. Teleports are intentionally not clipped.
func (b WorldBorder) Clamp(current, proposed mgl64.Vec3) mgl64.Vec3 {
	b = b.normalised()
	half := b.Size() / 2
	minX, maxX := b.centerX-half, b.centerX+half
	minZ, maxZ := b.centerZ-half, b.centerZ+half
	proposed[0] = clampBorderAxis(current[0], proposed[0], minX, maxX)
	proposed[2] = clampBorderAxis(current[2], proposed[2], minZ, maxZ)
	return proposed
}

func clampBorderAxis(current, proposed, minimum, maximum float64) float64 {
	switch {
	case current < minimum:
		if proposed <= current {
			return current
		}
		return math.Min(proposed, minimum)
	case current > maximum:
		if proposed >= current {
			return current
		}
		return math.Max(proposed, maximum)
	default:
		return math.Max(minimum, math.Min(maximum, proposed))
	}
}

func (b WorldBorder) normalised() WorldBorder {
	if b.targetSize <= 0 || math.IsNaN(b.targetSize) || math.IsInf(b.targetSize, 0) {
		return DefaultWorldBorder()
	}
	return b
}
