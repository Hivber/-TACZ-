package block

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

var (
	_ world.RedstonePowerSource   = DaylightDetector{}
	_ world.ScheduledTicker       = DaylightDetector{}
	_ world.NeighbourUpdateTicker = DaylightDetector{}
)

// DaylightDetector emits a redstone signal based on the internal sky light in the Overworld.
// Inverted changes the detector to output the inverse of the internal sky light instead.
type DaylightDetector struct {
	empty
	transparent
	flowingWaterDisplacer

	// Inverted reports whether the detector is in inverted (night sensor) mode.
	Inverted bool
	// Power is the current redstone signal strength, in the range 0-15.
	Power int
}

// RedstonePower returns the detector's signal in every direction.
func (d DaylightDetector) RedstonePower(cube.Pos, *world.Tx, cube.Face) int {
	return world.ClampRedstonePower(d.Power)
}

// ScheduledTick recalculates the signal once every 20 world ticks while the detector's chunk is ticking.
func (d DaylightDetector) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	current, ok := tx.Block(pos).(DaylightDetector)
	if !ok {
		return
	}
	current.Power = daylightDetectorSignal(pos, tx, current.Inverted)
	if current.Power != d.Power {
		tx.SetBlock(pos, current, nil)
	}
	if tx.World().Dimension() == world.Overworld {
		tx.ScheduleBlockUpdate(pos, current, daylightDetectorUpdateDelay(tx.CurrentTick()))
	}
}

// NeighbourUpdateTick removes the detector if its support is removed and schedules a refresh for sky exposure changes.
func (d DaylightDetector) NeighbourUpdateTick(pos, changedNeighbour cube.Pos, tx *world.Tx) {
	support := pos.Side(cube.FaceDown)
	if changedNeighbour == support && !tx.Block(support).Model().FaceSolid(support, cube.FaceUp, tx) {
		breakBlock(d, pos, tx)
		return
	}
	tx.ScheduleBlockUpdate(pos, d, daylightDetectorUpdateDelay(tx.CurrentTick()))
}

// UseOnBlock places the detector on the upper face of a supporting block.
func (d DaylightDetector) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	pos, face, used := firstReplaceable(tx, pos, face, d)
	if !used || face != cube.FaceUp {
		return false
	}
	support := pos.Side(cube.FaceDown)
	if !tx.Block(support).Model().FaceSolid(support, cube.FaceUp, tx) {
		return false
	}
	d.Inverted = false
	d.Power = daylightDetectorSignal(pos, tx, false)
	place(tx, pos, d, user, ctx)
	if !placed(ctx) {
		return false
	}
	tx.ScheduleBlockUpdate(pos, d, daylightDetectorUpdateDelay(tx.CurrentTick()))
	return true
}

// Activate switches between daylight and inverted mode and immediately updates the signal.
func (DaylightDetector) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, _ item.User, _ *item.UseContext) bool {
	current, ok := tx.Block(pos).(DaylightDetector)
	if !ok {
		return false
	}
	current.Inverted = !current.Inverted
	current.Power = daylightDetectorSignal(pos, tx, current.Inverted)
	tx.SetBlock(pos, current, nil)
	if tx.World().Dimension() == world.Overworld {
		tx.ScheduleBlockUpdate(pos, current, daylightDetectorUpdateDelay(tx.CurrentTick()))
	}
	return true
}

// SideClosed reports that the detector does not seal any of its faces.
func (DaylightDetector) SideClosed(cube.Pos, cube.Pos, *world.Tx) bool {
	return false
}

// BreakInfo returns the mining properties of the detector. Both modes drop the regular daylight detector item.
func (DaylightDetector) BreakInfo() BreakInfo {
	return newBreakInfo(0.2, alwaysHarvestable, axeEffective, oneOf(DaylightDetector{}))
}

// EncodeItem returns the item identifier of the regular daylight detector.
func (DaylightDetector) EncodeItem() (string, int16) {
	return "minecraft:daylight_detector", 0
}

// EncodeBlock returns the Bedrock block identifier and current redstone signal state.
func (d DaylightDetector) EncodeBlock() (string, map[string]any) {
	name := "minecraft:daylight_detector"
	if d.Inverted {
		name = "minecraft:daylight_detector_inverted"
	}
	return name, map[string]any{"redstone_signal": int32(world.ClampRedstonePower(d.Power))}
}

// Model returns the 3/8-block-high collision model of the detector.
func (DaylightDetector) Model() world.BlockModel {
	return model.DaylightDetector{}
}

// allDaylightDetectors returns every regular and inverted detector state.
func allDaylightDetectors() (all []world.Block) {
	for inverted := 0; inverted <= 1; inverted++ {
		for power := 0; power <= 15; power++ {
			all = append(all, DaylightDetector{Inverted: inverted == 1, Power: power})
		}
	}
	return
}

// daylightDetectorSignal computes the Bedrock daylight detector output. Block light is intentionally ignored.
func daylightDetectorSignal(pos cube.Pos, tx *world.Tx, inverted bool) int {
	return daylightDetectorSignalForDimension(
		tx.SkyLight(pos),
		int64(tx.World().Time()),
		tx.Raining(),
		tx.Thundering(),
		inverted,
		tx.World().Dimension(),
	)
}

// daylightDetectorSignalForDimension returns zero outside the Overworld, including for inverted detectors.
func daylightDetectorSignalForDimension(skyLight uint8, worldTime int64, raining, thundering, inverted bool, dimension world.Dimension) int {
	if dimension != world.Overworld {
		return 0
	}
	return daylightDetectorSignalFromSkyLight(skyLight, worldTime, raining, thundering, inverted)
}

// daylightDetectorSignalFromSkyLight computes the detector output from the raw sky light and world conditions.
// The calculation follows the Bedrock behaviour model used by PowerNukkitX, adapted to Dragonfly's boolean
// weather state. The resulting sky darkening is applied before the inverted or daylight response.
func daylightDetectorSignalFromSkyLight(skyLight uint8, worldTime int64, raining, thundering, inverted bool) int {
	angle := daylightDetectorCelestialAngle(worldTime, 1)
	rainFactor, thunderFactor := 1.0, 1.0
	if raining {
		rainFactor -= 5.0 / 16.0
	}
	if thundering {
		thunderFactor -= 5.0 / 16.0
	}

	sunFactor := 0.5 + 2*clampDaylightDetector(math.Cos(angle*2*math.Pi), -0.25, 0.25)
	skyDarkening := int((1 - sunFactor*rainFactor*thunderFactor) * 11)
	internalSkyLight := int(skyLight) - skyDarkening

	if inverted {
		return clampDaylightDetectorInt(15-internalSkyLight, 0, 15)
	}
	if internalSkyLight <= 0 {
		return 0
	}

	sunAngle := angle * 2 * math.Pi
	targetAngle := 2 * math.Pi
	if sunAngle < math.Pi {
		targetAngle = 0
	}
	sunAngle += (targetAngle - sunAngle) * 0.2
	// Floor(x+0.5) matches Java-style rounding, including negative values near sunrise and sunset.
	power := int(math.Floor(float64(internalSkyLight)*math.Cos(sunAngle) + 0.5))
	return clampDaylightDetectorInt(power, 0, 15)
}

// daylightDetectorCelestialAngle calculates the smoothed solar angle for the current day time.
func daylightDetectorCelestialAngle(worldTime int64, tickDelta float64) float64 {
	dayTime := worldTime % 24000
	if dayTime < 0 {
		dayTime += 24000
	}
	angle := (float64(dayTime)+tickDelta)/24000 - 0.25
	if angle < 0 {
		angle++
	}
	if angle > 1 {
		angle--
	}
	smooth := 1 - (math.Cos(angle*math.Pi)+1)/2
	return angle + (smooth-angle)/3
}

func clampDaylightDetector(value, minValue, maxValue float64) float64 {
	return math.Max(minValue, math.Min(maxValue, value))
}

func clampDaylightDetectorInt(value, minValue, maxValue int) int {
	return max(minValue, min(maxValue, value))
}

func daylightDetectorUpdateDelay(currentTick int64) time.Duration {
	ticks := 20 - currentTick%20
	return time.Duration(ticks) * time.Second / 20
}
