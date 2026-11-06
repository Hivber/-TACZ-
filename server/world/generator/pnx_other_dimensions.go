package generator

import (
	"math"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/biome"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

// PNXNether creates a multi-level Nether cavern system with a ceiling, lava
// basins and biome-dependent floor materials, following the broad stages of
// PowerNukkitX's Nether generator.
type PNXNether struct {
	seed                                                            int64
	netherrack, bedrock, lava, soulSand, gravel, basalt, blackstone uint32
	quartzOre, netherGoldOre, air                                   uint32
}

// NewPNXNether creates a deterministic PNX-style Nether generator.
func NewPNXNether(seed int64) *PNXNether {
	br := world.DefaultBlockRegistry
	id := func(b world.Block) uint32 { return br.BlockRuntimeID(b) }
	return &PNXNether{
		seed:       seed,
		netherrack: id(block.Netherrack{}), bedrock: id(block.Bedrock{}),
		lava: id(block.Lava{Still: true, Depth: 8}), soulSand: id(block.SoulSand{}),
		gravel: id(block.Gravel{}), basalt: id(block.Basalt{}), blackstone: id(block.Blackstone{}),
		quartzOre: id(block.NetherQuartzOre{}), netherGoldOre: id(block.NetherGoldOre{}), air: br.AirRuntimeID(),
	}
}

func (g *PNXNether) GenerateChunk(pos world.ChunkPos, c *chunk.Chunk) {
	baseX, baseZ := int(pos.X())<<4, int(pos.Z())<<4
	minY, maxY := c.Range().Min(), c.Range().Max()
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			wx, wz := baseX+x, baseZ+z
			floor := 31 + int(math.Round(fbm2Normalized(g.seed+301, float64(wx), float64(wz), 0.009, 4)*12+fbm2Normalized(g.seed+303, float64(wx), float64(wz), 0.035, 2)*4))
			roof := 105 + int(math.Round(fbm2Normalized(g.seed+307, float64(wx), float64(wz), 0.006, 4)*13+fbm2Normalized(g.seed+311, float64(wx), float64(wz), 0.023, 2)*5))
			if roof < floor+38 {
				roof = floor + 38
			}
			biomeNoise := fbm2Normalized(g.seed+313, float64(wx), float64(wz), 0.0027, 3)
			troughNoise := fbm2Normalized(g.seed+317, float64(wx), float64(wz), 0.011, 2)
			floorBlock := g.netherrack
			biomeID := uint32(biome.NetherWastes{}.EncodeBiome())
			switch {
			case biomeNoise > 0.47:
				biomeID = uint32(biome.WarpedForest{}.EncodeBiome())
			case biomeNoise < -0.48:
				biomeID = uint32(biome.CrimsonForest{}.EncodeBiome())
			case troughNoise > 0.58:
				biomeID = uint32(biome.BasaltDeltas{}.EncodeBiome())
				floorBlock = g.basalt
			case troughNoise < -0.58:
				biomeID = uint32(biome.SoulSandValley{}.EncodeBiome())
				floorBlock = g.soulSand
				if noise2(g.seed+319, float64(wx)*0.04, float64(wz)*0.04) > 0.18 {
					floorBlock = g.gravel
				}
			}
			for y := minY; y <= maxY; y++ {
				c.SetBiome(uint8(x), int16(y), uint8(z), biomeID)
				id := g.air
				switch {
				case y == minY || y <= minY+int(latticeHash(g.seed+331, int64(wx), 0, int64(wz))%5):
					id = g.bedrock
				case y == maxY || y >= maxY-int(latticeHash(g.seed+337, int64(wx), 0, int64(wz))%5):
					id = g.bedrock
				case y <= floor:
					id = g.netherrack
					if y == floor {
						id = floorBlock
					} else if y < floor-4 && y > 8 && troughNoise > 0.58 && noise3(g.seed+347, float64(wx)*0.08, float64(y)*0.06, float64(wz)*0.08) > 0.66 {
						id = g.blackstone
					}
				case y >= roof:
					id = g.netherrack
				case y <= 31:
					if noise3(g.seed+353, float64(wx)*0.027, float64(y)*0.036, float64(wz)*0.027) < -0.12 {
						id = g.lava
					}
				default:
					cave := noise3(g.seed+359, float64(wx)*0.031, float64(y)*0.038, float64(wz)*0.031)
					caveMask := noise3(g.seed+367, float64(wx)*0.012, float64(y)*0.016, float64(wz)*0.012)
					if cave+0.42*caveMask > 0.29 {
						id = g.netherrack
					}
				}
				if id != g.air {
					c.SetBlock(uint8(x), int16(y), uint8(z), 0, id)
				}
			}
		}
	}
	g.generateOres(pos, c, minY, maxY)
}

func (g *PNXNether) generateOres(pos world.ChunkPos, c *chunk.Chunk, minY, maxY int) {
	seed := int64(chunkSeed(g.seed+379, pos))
	for i := 0; i < 28; i++ {
		x, y, z := int(latticeHash(seed, int64(i), 1, 0)%16), 8+int(latticeHash(seed, int64(i), 2, 0)%uint64(max(1, min(112, maxY-8)))), int(latticeHash(seed, int64(i), 3, 0)%16)
		id := g.quartzOre
		if i%4 == 0 {
			id = g.netherGoldOre
		}
		for j := 0; j < 5+i%4; j++ {
			sx, sy, sz := x+(j%3)-1, y+(j%2), z+((j/2)%3)-1
			if sx < 0 || sx > 15 || sz < 0 || sz > 15 || sy < minY || sy > maxY {
				continue
			}
			if c.Block(uint8(sx), int16(sy), uint8(sz), 0) == g.netherrack {
				c.SetBlock(uint8(sx), int16(sy), uint8(sz), 0, id)
			}
		}
	}
}

func (g *PNXNether) DefaultSpawn(dim world.Dimension) cube.Pos {
	floor := 31 + int(math.Round(fbm2Normalized(g.seed+301, 0, 0, 0.009, 4)*12+fbm2Normalized(g.seed+303, 0, 0, 0.035, 2)*4))
	y := max(dim.Range().Min()+8, min(dim.Range().Max()-2, floor+2))
	return cube.Pos{0, y, 0}
}

// PNXEnd generates the central End island and deterministic remote island
// clusters. The result is intentionally void between the central and outer
// archipelagos instead of repeating a flat End Stone sheet.
type PNXEnd struct {
	seed              int64
	endStone, bedrock uint32
}

// NewPNXEnd creates a deterministic PNX-style End generator.
func NewPNXEnd(seed int64) *PNXEnd {
	br := world.DefaultBlockRegistry
	return &PNXEnd{seed: seed, endStone: br.BlockRuntimeID(block.EndStone{}), bedrock: br.BlockRuntimeID(block.Bedrock{})}
}

func (g *PNXEnd) GenerateChunk(pos world.ChunkPos, c *chunk.Chunk) {
	baseX, baseZ := int(pos.X())<<4, int(pos.Z())<<4
	minY, maxY := c.Range().Min(), c.Range().Max()
	biomeID := uint32(biome.End{}.EncodeBiome())
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			wx, wz := baseX+x, baseZ+z
			height, exists := g.islandHeight(wx, wz)
			for y := minY; y <= maxY; y++ {
				c.SetBiome(uint8(x), int16(y), uint8(z), biomeID)
				if !exists || y > height {
					continue
				}
				id := g.endStone
				if y == minY || y == minY+1 && latticeHash(g.seed+401, int64(wx), 0, int64(wz))%3 == 0 {
					id = g.bedrock
				}
				if y < height-8 && y > minY+4 && noise3(g.seed+409, float64(wx)*0.028, float64(y)*0.035, float64(wz)*0.028) > 0.64 {
					continue
				}
				c.SetBlock(uint8(x), int16(y), uint8(z), 0, id)
			}
		}
	}
}

func (g *PNXEnd) islandHeight(x, z int) (int, bool) {
	distance := math.Hypot(float64(x), float64(z))
	if distance < 180 {
		radialNoise := fbm2(g.seed+419, float64(x), float64(z), 0.018, 3)
		radius := 112 + radialNoise*20
		if distance < radius {
			heightNoise := fbm2(g.seed+421, float64(x), float64(z), 0.035, 4)
			return int(math.Round(52 + (1-distance/radius)*18 + heightNoise*12)), true
		}
		return 0, false
	}
	if distance < 620 {
		return 0, false
	}

	const cellSize = 2560
	cellX, cellZ := floorDiv(x, cellSize), floorDiv(z, cellSize)
	bestHeight, found := 0, false
	for dx := -1; dx <= 1; dx++ {
		for dz := -1; dz <= 1; dz++ {
			cx, cz := cellX+dx, cellZ+dz
			centerX := cx*cellSize + int(latticeHash(g.seed+431, int64(cx), 0, int64(cz))%1200) - 600
			centerZ := cz*cellSize + int(latticeHash(g.seed+433, int64(cx), 0, int64(cz))%1200) - 600
			centerDistance := math.Hypot(float64(x-centerX), float64(z-centerZ))
			radius := 48 + int(latticeHash(g.seed+439, int64(cx), 1, int64(cz))%76)
			shapeNoise := fbm2(g.seed+443, float64(x), float64(z), 0.012, 3)
			if centerDistance < float64(radius)+shapeNoise*18 {
				height := int(math.Round(38 + shapeNoise*13 + (1-centerDistance/float64(radius))*5))
				if !found || height > bestHeight {
					bestHeight, found = height, true
				}
			}
		}
	}
	return bestHeight, found
}

func (g *PNXEnd) DefaultSpawn(dim world.Dimension) cube.Pos {
	height, _ := g.islandHeight(0, 0)
	y := min(dim.Range().Max()-2, max(dim.Range().Min()+2, height+1))
	return cube.Pos{0, y, 0}
}

func floorDiv(value, divisor int) int {
	q, r := value/divisor, value%divisor
	if r != 0 && (r < 0) != (divisor < 0) {
		q--
	}
	return q
}
