package generator

import (
	"math"
	"testing"
)

func TestPNXNoiseIsDeterministicAndBounded(t *testing.T) {
	for _, seed := range []int64{0, 1, -7, 123456789} {
		for i := -64; i <= 64; i++ {
			x := float64(i) * 0.173
			y := float64(i) * 0.091
			z := float64(-i) * 0.217
			a, b := noise3(seed, x, y, z), noise3(seed, x, y, z)
			c := noise2(seed, x, z)
			if a != b || math.IsNaN(a) || math.IsInf(a, 0) || a < -1 || a > 1 || c < -1 || c > 1 {
				t.Fatalf("invalid noise sample for seed %d and index %d: 3D=%v, 2D=%v", seed, i, a, c)
			}
		}
	}
}

func TestPNXOverworldSamplingIsNotFlat(t *testing.T) {
	g := &PNXOverworld{seed: 0}
	minimum, maximum := math.MaxInt, math.MinInt
	biomes := make(map[pnxBiome]struct{})
	for x := -2048; x <= 2048; x += 64 {
		for z := -2048; z <= 2048; z += 64 {
			column := g.sampleColumn(x, z)
			if column.height < minimum {
				minimum = column.height
			}
			if column.height > maximum {
				maximum = column.height
			}
			biomes[column.biome] = struct{}{}
		}
	}
	if maximum-minimum < 70 {
		t.Fatalf("terrain height range is unexpectedly small: min=%d max=%d", minimum, maximum)
	}
	if len(biomes) < 5 {
		t.Fatalf("terrain sampler selected too few biomes: got %d", len(biomes))
	}
}
