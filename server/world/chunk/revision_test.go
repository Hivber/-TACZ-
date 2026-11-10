package chunk

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
)

type revisionTestRegistry struct{}

func (revisionTestRegistry) BlockCount() int      { return 4 }
func (revisionTestRegistry) AirRuntimeID() uint32 { return 0 }
func (revisionTestRegistry) RuntimeIDToState(id uint32) (string, map[string]any, bool) {
	return "test:block", map[string]any{"id": int32(id)}, true
}
func (revisionTestRegistry) StateToRuntimeID(string, map[string]any) (uint32, bool) { return 0, false }
func (revisionTestRegistry) DefaultProperties(string) (map[string]any, bool)        { return nil, false }
func (revisionTestRegistry) FilteringBlock(uint32) uint8                            { return 0 }
func (revisionTestRegistry) LightBlock(uint32) uint8                                { return 0 }
func (revisionTestRegistry) RandomTickBlock(uint32) bool                            { return false }
func (revisionTestRegistry) NBTBlock(uint32) bool                                   { return false }
func (revisionTestRegistry) LiquidDisplacingBlock(uint32) bool                      { return false }
func (revisionTestRegistry) LiquidBlock(uint32) bool                                { return false }
func (revisionTestRegistry) HashToRuntimeID(uint32) (uint32, bool)                  { return 0, false }
func (revisionTestRegistry) RuntimeIDToHash(uint32) (uint32, bool)                  { return 0, false }

func TestSubChunkRevisionTracksAllStorageWrites(t *testing.T) {
	c := New(revisionTestRegistry{}, cube.Range{0, 255})
	initial := c.SubChunkRevision(0)
	c.SetBlock(1, 0, 1, 0, 1)
	afterChunkSet := c.SubChunkRevision(0)
	if afterChunkSet == initial {
		t.Fatal("Chunk.SetBlock did not advance the sub-chunk revision")
	}
	c.SubChunk(0).Layer(0).Set(2, 0, 2, 2)
	if c.SubChunkRevision(0) == afterChunkSet {
		t.Fatal("direct PalettedStorage.Set did not advance the sub-chunk revision")
	}
}

func TestBiomeRevisionTracksBiomeWrites(t *testing.T) {
	c := New(revisionTestRegistry{}, cube.Range{0, 255})
	initial := c.BiomeRevision()
	c.SetBiome(0, 0, 0, 1)
	afterSingle := c.BiomeRevision()
	if afterSingle == initial {
		t.Fatal("SetBiome did not advance biome revision")
	}
	var section [4][4][4]uint32
	section[0][0][0] = 2
	c.SetBiomeSection(0, section)
	if c.BiomeRevision() == afterSingle {
		t.Fatal("SetBiomeSection did not advance biome revision")
	}
}

func TestCloneKeepsIndependentNetworkRevisions(t *testing.T) {
	original := New(revisionTestRegistry{}, cube.Range{0, 255})
	original.SetBlock(1, 0, 1, 0, 1)
	original.SetBiome(1, 0, 1, 1)
	clone := original.Clone()
	if clone.SubChunkRevision(0) != original.SubChunkRevision(0) || clone.BiomeRevision() != original.BiomeRevision() {
		t.Fatal("clone did not preserve initial network revisions")
	}
	originalSubRevision, originalBiomeRevision := original.SubChunkRevision(0), original.BiomeRevision()
	clone.SetBlock(2, 0, 2, 0, 2)
	clone.SetBiome(2, 0, 2, 2)
	if original.SubChunkRevision(0) != originalSubRevision || original.BiomeRevision() != originalBiomeRevision {
		t.Fatal("mutating clone changed original chunk revisions")
	}
	if clone.SubChunkRevision(0) == originalSubRevision || clone.BiomeRevision() == originalBiomeRevision {
		t.Fatal("mutating clone did not advance clone revisions")
	}
}

func TestUnchangedWritesDoNotAdvanceNetworkRevisions(t *testing.T) {
	c := New(revisionTestRegistry{}, cube.Range{0, 255})

	c.SetBlock(1, 0, 1, 0, 1)
	blockRevision := c.SubChunkRevision(0)
	c.SetBlock(1, 0, 1, 0, 1)
	if c.SubChunkRevision(0) != blockRevision {
		t.Fatal("writing the existing block value advanced the sub-chunk revision")
	}

	c.SetBiome(1, 0, 1, 2)
	biomeRevision := c.BiomeRevision()
	c.SetBiome(1, 0, 1, 2)
	if c.BiomeRevision() != biomeRevision {
		t.Fatal("writing the existing biome value advanced the biome revision")
	}

	// Writing air to an unallocated layer must remain a no-op.
	c.SetBlock(2, 0, 2, 1, 0)
	if c.SubChunkRevision(0) != blockRevision {
		t.Fatal("writing air to an unallocated layer advanced the sub-chunk revision")
	}
}
