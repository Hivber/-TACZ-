package generator

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	helixbiome "github.com/helix-bedrock/worldgen/1.26.60/biome"
	helixblocks "github.com/helix-bedrock/worldgen/1.26.60/blocks"
	helixend "github.com/helix-bedrock/worldgen/1.26.60/end"
	helixnether "github.com/helix-bedrock/worldgen/1.26.60/nether"
	helixoverworld "github.com/helix-bedrock/worldgen/1.26.60/overworld"
	helixstructure "github.com/helix-bedrock/worldgen/1.26.60/structure"
)

//go:embed helixdata/block_palette.nbt helixdata/block_types.json helixdata/biomes.json helixdata/block_light.json helixdata/block_translucency.json
var helixData embed.FS

// Helix contains the Bedrock 1.26.60 world generators and the immutable block
// and biome mappings used to translate their output into Dragonfly chunks.
type Helix struct {
	registry       *helixblocks.Registry
	overworld      *helixoverworld.Overworld
	overworldCache *helixoverworld.TerrainCache
	nether         *helixnether.Nether
	end            *helixend.End

	blockRuntimeIDs []uint32
	biomeRuntimeIDs map[uint32]uint32
	airRuntimeID    uint32
	waterRuntimeID  uint32
	plainsBiomeID   uint32

	exactBlockStates     int
	defaultedBlockStates int
	missingBlockStates   int
	droppedBlockEntities atomic.Uint64
}

// BlockMappingStats reports how many states from the Helix palette were
// matched exactly, mapped to the Dragonfly default state, or unavailable.
type BlockMappingStats struct {
	Exact     int
	Defaulted int
	Missing   int
}

// NewHelix loads the upstream Bedrock 1.26.60 generator data and builds the
// three vanilla dimension generators for a world seed.
func NewHelix(seed int64) (*Helix, error) {
	read := func(name string) ([]byte, error) {
		return helixData.ReadFile("helixdata/" + name)
	}
	palette, err := read("block_palette.nbt")
	if err != nil {
		return nil, fmt.Errorf("helix: read block palette: %w", err)
	}
	types, err := read("block_types.json")
	if err != nil {
		return nil, fmt.Errorf("helix: read block types: %w", err)
	}
	reg, err := helixblocks.Load(bytes.NewReader(palette), bytes.NewReader(types))
	if err != nil {
		return nil, fmt.Errorf("helix: load block palette: %w", err)
	}
	light, err := read("block_light.json")
	if err != nil {
		return nil, fmt.Errorf("helix: read block light table: %w", err)
	}
	if err := reg.LoadLight(bytes.NewReader(light)); err != nil {
		return nil, fmt.Errorf("helix: load block light table: %w", err)
	}
	translucency, err := read("block_translucency.json")
	if err != nil {
		return nil, fmt.Errorf("helix: read block translucency table: %w", err)
	}
	if err := reg.LoadTranslucency(bytes.NewReader(translucency)); err != nil {
		return nil, fmt.Errorf("helix: load block translucency table: %w", err)
	}
	biomeData, err := read("biomes.json")
	if err != nil {
		return nil, fmt.Errorf("helix: read biome IDs: %w", err)
	}
	biomeIDs, err := helixbiome.LoadIDs(bytes.NewReader(biomeData))
	if err != nil {
		return nil, fmt.Errorf("helix: load biome IDs: %w", err)
	}

	// The server finalises the default registry when Config.New is called.
	// Helix is constructed while parsing config, so finalise it here first.
	world.DefaultBlockRegistry.Finalize()
	h := &Helix{
		registry:        reg,
		airRuntimeID:    world.DefaultBlockRegistry.AirRuntimeID(),
		blockRuntimeIDs: make([]uint32, reg.Len()),
		biomeRuntimeIDs: make(map[uint32]uint32, len(biomeIDs)),
	}
	if err := h.mapBlocks(); err != nil {
		return nil, err
	}
	if err := h.mapBiomes(biomeIDs); err != nil {
		return nil, err
	}

	waterProperties, ok := world.DefaultBlockRegistry.DefaultProperties("minecraft:water")
	if !ok {
		return nil, fmt.Errorf("helix: Dragonfly has no default water block state")
	}
	h.waterRuntimeID, ok = world.DefaultBlockRegistry.StateToRuntimeID("minecraft:water", waterProperties)
	if !ok {
		return nil, fmt.Errorf("helix: Dragonfly could not resolve its default water block state")
	}

	h.overworld, err = helixoverworld.NewOverworld(seed, reg, biomeIDs, helixoverworld.WithConfig(helixoverworld.Vanilla()))
	if err != nil {
		return nil, fmt.Errorf("helix: create overworld generator: %w", err)
	}
	h.overworldCache = helixoverworld.NewTerrainCache(h.overworld, 1024)
	h.nether, err = helixnether.New(seed, reg, biomeIDs, helixnether.WithConfig(helixnether.Vanilla()))
	if err != nil {
		return nil, fmt.Errorf("helix: create nether generator: %w", err)
	}
	h.end, err = helixend.New(seed, reg, biomeIDs, helixend.WithConfig(helixend.Vanilla()))
	if err != nil {
		return nil, fmt.Errorf("helix: create end generator: %w", err)
	}
	return h, nil
}

func (h *Helix) mapBlocks() error {
	for i := 0; i < h.registry.Len(); i++ {
		state, ok := h.registry.State(uint32(i))
		if !ok {
			return fmt.Errorf("helix: block palette state %d is missing", i)
		}
		if rid, found := world.DefaultBlockRegistry.StateToRuntimeID(state.Name, state.Props); found {
			h.blockRuntimeIDs[i] = rid
			h.exactBlockStates++
			continue
		}
		if properties, found := world.DefaultBlockRegistry.DefaultProperties(state.Name); found {
			if rid, ok := world.DefaultBlockRegistry.StateToRuntimeID(state.Name, properties); ok {
				h.blockRuntimeIDs[i] = rid
				h.defaultedBlockStates++
				continue
			}
		}
		h.blockRuntimeIDs[i] = h.airRuntimeID
		h.missingBlockStates++
	}
	return nil
}

var helixBiomeAliases = map[string]string{
	"birch_forest_hills_mutated":       "tall_birch_hills",
	"birch_forest_mutated":             "old_growth_birch_forest",
	"cold_beach":                       "snowy_beach",
	"cold_taiga":                       "snowy_taiga",
	"cold_taiga_hills":                 "snowy_taiga_hills",
	"cold_taiga_mutated":               "snowy_taiga_mountains",
	"desert_mutated":                   "desert_lakes",
	"extreme_hills":                    "windswept_hills",
	"extreme_hills_edge":               "mountain_edge",
	"extreme_hills_mutated":            "gravelly_mountains_plus",
	"extreme_hills_plus_trees":         "windswept_forest",
	"extreme_hills_plus_trees_mutated": "gravelly_mountains_plus",
	"forest_hills":                     "wooded_hills",
	"hell":                             "nether_wastes",
	"ice_mountains":                    "snowy_mountains",
	"ice_plains":                       "snowy_plains",
	"ice_plains_spikes":                "ice_spikes",
	"ice_caves":                        "frozen_peaks",
	"jungle_edge_mutated":              "modified_jungle_edge",
	"jungle_mutated":                   "modified_jungle",
	"mega_taiga":                       "old_growth_spruce_taiga",
	"mega_taiga_hills":                 "giant_tree_taiga_hills",
	"mesa":                             "badlands",
	"mesa_bryce":                       "eroded_badlands",
	"mesa_plateau":                     "badlands_plateau",
	"mesa_plateau_mutated":             "modified_badlands_plateau",
	"mesa_plateau_stone":               "wooded_badlands_plateau",
	"mesa_plateau_stone_mutated":       "modified_wooded_badlands_plateau",
	"mushroom_island":                  "mushroom_fields",
	"mushroom_island_shore":            "mushroom_field_shore",
	"redwood_taiga_hills_mutated":      "giant_spruce_taiga_hills",
	"redwood_taiga_mutated":            "old_growth_pine_taiga",
	"roofed_forest":                    "dark_forest",
	"roofed_forest_mutated":            "dark_forest_hills",
	"savanna_mutated":                  "shattered_savanna_plateau",
	"savanna_plateau_mutated":          "windswept_savanna",
	"soulsand_valley":                  "soul_sand_valley",
	"stone_beach":                      "stony_shore",
	"swampland":                        "swamp",
	"swampland_mutated":                "swamp_hills",
	"taiga_mutated":                    "taiga_mountains",
}

func (h *Helix) mapBiomes(ids map[string]uint32) error {
	plains, ok := world.BiomeByName("plains")
	if !ok {
		return fmt.Errorf("helix: Dragonfly plains biome is unavailable")
	}
	h.plainsBiomeID = uint32(plains.EncodeBiome())
	for name, id := range ids {
		name = strings.TrimPrefix(name, "minecraft:")
		// Prefer a direct Dragonfly biome name. The Helix biome data contains
		// both legacy Bedrock names and newer aliases, while several Dragonfly
		// biome String values intentionally retain the legacy spelling.
		b, ok := world.BiomeByName(name)
		if !ok {
			if alias, exists := helixBiomeAliases[name]; exists {
				b, ok = world.BiomeByName(alias)
			}
		}
		if !ok {
			return fmt.Errorf("helix: no Dragonfly biome mapping for Bedrock biome %q", name)
		}
		h.biomeRuntimeIDs[id] = uint32(b.EncodeBiome())
	}
	return nil
}

// BlockMappingStats reports the compatibility of the Helix block palette with
// Dragonfly's runtime block registry.
func (h *Helix) BlockMappingStats() BlockMappingStats {
	return BlockMappingStats{Exact: h.exactBlockStates, Defaulted: h.defaultedBlockStates, Missing: h.missingBlockStates}
}

// ForDimension returns the Helix generator for one vanilla dimension.
func (h *Helix) ForDimension(dim world.Dimension) world.Generator {
	return &helixDimensionGenerator{helix: h, dim: dim}
}

type helixDimensionGenerator struct {
	helix *Helix
	dim   world.Dimension
}

func (g *helixDimensionGenerator) GenerateChunk(pos world.ChunkPos, dst *chunk.Chunk) {
	g.GenerateChunkWithData(pos, dst)
}

// GenerateChunkWithData generates a chunk and returns block entity data for
// structures that Dragonfly has a compatible NBT block implementation for.
func (g *helixDimensionGenerator) GenerateChunkWithData(pos world.ChunkPos, dst *chunk.Chunk) []chunk.BlockEntity {
	switch g.dim {
	case world.Overworld:
		var src helixoverworld.Chunk
		g.helix.overworldCache.Generate(pos[0], pos[1], &src)
		g.copyOverworld(dst, &src)
		return g.helix.blockEntities(pos, dst, src.BlockEntities)
	case world.Nether:
		var src helixnether.Chunk
		g.helix.nether.GenerateChunk(pos[0], pos[1], &src)
		g.copyNether(dst, &src)
		return g.helix.blockEntities(pos, dst, src.BlockEntities)
	case world.End:
		var src helixend.Chunk
		g.helix.end.GenerateChunk(pos[0], pos[1], &src)
		g.copyEnd(dst, &src)
		return g.helix.blockEntities(pos, dst, src.BlockEntities)
	default:
		return nil
	}
}

func (g *helixDimensionGenerator) DefaultSpawn(dim world.Dimension) cube.Pos {
	switch g.dim {
	case world.Overworld:
		var src helixoverworld.Chunk
		g.helix.overworld.GenerateChunk(0, 0, &src)
		top := helixoverworld.MinY
		for y := helixoverworld.MaxY - 1; y >= helixoverworld.MinY; y-- {
			state, _ := g.helix.registry.State(src.Block(8, y, 8))
			if state.Name != "minecraft:air" && state.Name != "minecraft:water" && state.Name != "minecraft:flowing_water" {
				top = y
				break
			}
		}
		return cube.Pos{8, top + 1, 8}
	case world.Nether:
		return cube.Pos{8, 64, 8}
	case world.End:
		return cube.Pos{0, 65, 0}
	default:
		return cube.Pos{}
	}
}

func (g *helixDimensionGenerator) copyOverworld(dst *chunk.Chunk, src *helixoverworld.Chunk) {
	for y := helixoverworld.MinY; y < helixoverworld.MaxY; y++ {
		for z := 0; z < 16; z++ {
			for x := 0; x < 16; x++ {
				dst.SetBlock(uint8(x), int16(y), uint8(z), 0, g.helix.blockRuntimeID(src.Block(x, y, z)))
				if src.Waterlogged(x, y, z) {
					dst.SetBlock(uint8(x), int16(y), uint8(z), 1, g.helix.waterRuntimeID)
				}
			}
		}
	}
	for y := helixoverworld.MinY; y < helixoverworld.MaxY; y += 4 {
		for z := 0; z < 16; z += 4 {
			for x := 0; x < 16; x += 4 {
				dst.SetBiome(uint8(x), int16(y), uint8(z), g.helix.biomeID(src.Biome(x, y, z)))
			}
		}
	}
}

func (g *helixDimensionGenerator) copyNether(dst *chunk.Chunk, src *helixnether.Chunk) {
	for y := 0; y < helixnether.Height; y++ {
		for z := 0; z < 16; z++ {
			for x := 0; x < 16; x++ {
				dst.SetBlock(uint8(x), int16(y), uint8(z), 0, g.helix.blockRuntimeID(src.Block(x, y, z)))
			}
		}
	}
	for y := 0; y < helixnether.Height; y += 4 {
		for z := 0; z < 16; z += 4 {
			for x := 0; x < 16; x += 4 {
				dst.SetBiome(uint8(x), int16(y), uint8(z), g.helix.biomeID(src.Biome(x, z)))
			}
		}
	}
}

func (g *helixDimensionGenerator) copyEnd(dst *chunk.Chunk, src *helixend.Chunk) {
	for y := 0; y < helixend.Height; y++ {
		for z := 0; z < 16; z++ {
			for x := 0; x < 16; x++ {
				dst.SetBlock(uint8(x), int16(y), uint8(z), 0, g.helix.blockRuntimeID(src.Block(x, y, z)))
			}
		}
	}
	for y := 0; y < helixend.Height; y += 4 {
		for z := 0; z < 16; z += 4 {
			for x := 0; x < 16; x += 4 {
				dst.SetBiome(uint8(x), int16(y), uint8(z), g.helix.biomeID(src.Biome(x, z)))
			}
		}
	}
}

func (h *Helix) blockRuntimeID(id uint32) uint32 {
	if int(id) >= len(h.blockRuntimeIDs) {
		return h.airRuntimeID
	}
	return h.blockRuntimeIDs[id]
}

func (h *Helix) biomeID(id uint32) uint32 {
	if mapped, ok := h.biomeRuntimeIDs[id]; ok {
		return mapped
	}
	return h.plainsBiomeID
}

func (h *Helix) blockEntities(pos world.ChunkPos, c *chunk.Chunk, source []helixstructure.BlockEntity) []chunk.BlockEntity {
	out := make([]chunk.BlockEntity, 0, len(source))
	for _, be := range source {
		if be.X>>4 != pos[0] || be.Z>>4 != pos[1] {
			continue
		}
		// The Helix generator reports block entity coordinates in world space,
		// while chunk.Chunk indexes blocks using 0..15 local X/Z coordinates.
		localX := int(be.X) - int(pos[0])*16
		localZ := int(be.Z) - int(pos[1])*16
		y := int(be.Y)
		r := c.Range()
		if localX < 0 || localX >= 16 || localZ < 0 || localZ >= 16 || y < r.Min() || y > r.Max() {
			h.droppedBlockEntities.Add(1)
			continue
		}
		rid := c.Block(uint8(localX), int16(y), uint8(localZ), 0)
		b, ok := world.DefaultBlockRegistry.BlockByRuntimeID(rid)
		if !ok {
			h.droppedBlockEntities.Add(1)
			continue
		}
		if _, ok := b.(world.NBTer); !ok {
			h.droppedBlockEntities.Add(1)
			continue
		}
		out = append(out, chunk.BlockEntity{
			Pos:  cube.Pos{int(be.X), y, int(be.Z)},
			Data: be.Data,
		})
	}
	return out
}

// DroppedBlockEntityCount returns the number of generated structure block
// entities that could not be represented by Dragonfly's NBT block registry.
func (h *Helix) DroppedBlockEntityCount() uint64 {
	return h.droppedBlockEntities.Load()
}
