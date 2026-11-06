package generator

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	helixstructure "github.com/helix-bedrock/worldgen/1.26.60/structure"
)

func TestHelixGeneratesAllVanillaDimensions(t *testing.T) {
	h, err := NewHelix(0)
	if err != nil {
		t.Fatalf("create Helix generator: %v", err)
	}
	if stats := h.BlockMappingStats(); stats.Exact+stats.Defaulted+stats.Missing != h.registry.Len() {
		t.Fatalf("incomplete block mapping statistics: %+v for %d palette states", stats, h.registry.Len())
	}

	tests := []struct {
		name string
		dim  world.Dimension
		pos  world.ChunkPos
	}{
		{name: "overworld", dim: world.Overworld, pos: world.ChunkPos{1, -1}},
		{name: "nether", dim: world.Nether, pos: world.ChunkPos{0, 0}},
		{name: "end", dim: world.End, pos: world.ChunkPos{0, 0}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dst := chunk.New(world.DefaultBlockRegistry, test.dim.Range())
			generator, ok := h.ForDimension(test.dim).(world.ChunkDataGenerator)
			if !ok {
				t.Fatal("Helix dimension generator does not expose block entity data")
			}
			_ = generator.GenerateChunkWithData(test.pos, dst)

			air := world.DefaultBlockRegistry.AirRuntimeID()
			generated := false
			r := dst.Range()
			for y := r.Min(); y <= r.Max() && !generated; y++ {
				for z := uint8(0); z < 16 && !generated; z++ {
					for x := uint8(0); x < 16; x++ {
						if dst.Block(x, int16(y), z, 0) != air {
							generated = true
							break
						}
					}
				}
			}
			if !generated {
				t.Fatal("generated chunk contains no non-air blocks")
			}
		})
	}
}

func TestHelixBlockEntitiesUseChunkLocalCoordinates(t *testing.T) {
	h, err := NewHelix(1)
	if err != nil {
		t.Fatalf("create Helix generator: %v", err)
	}

	pos := world.ChunkPos{2, -3}
	local := cube.Pos{3, 64, 5}
	dst := chunk.New(world.DefaultBlockRegistry, world.Overworld.Range())
	chest := block.NewChest()
	dst.SetBlock(uint8(local.X()), int16(local.Y()), uint8(local.Z()), 0, world.DefaultBlockRegistry.BlockRuntimeID(chest))

	source := []helixstructure.BlockEntity{{
		X:    pos[0]*16 + local.X(),
		Y:    local.Y(),
		Z:    pos[1]*16 + local.Z(),
		Data: map[string]any{"id": "Chest"},
	}}
	got := h.blockEntities(pos, dst, source)
	if len(got) != 1 {
		t.Fatalf("expected one compatible block entity, got %d", len(got))
	}
	want := cube.Pos{pos[0]*16 + local.X(), local.Y(), pos[1]*16 + local.Z()}
	if got[0].Pos != want {
		t.Fatalf("block entity position = %v, want %v", got[0].Pos, want)
	}
}
