package generator

import (
	"math"
	"math/rand"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/biome"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

const (
	pnxSeaLevel = 63
)

type pnxBiome uint8

const (
	pnxPlains pnxBiome = iota
	pnxForest
	pnxBirchForest
	pnxTaiga
	pnxJungle
	pnxDesert
	pnxSavanna
	pnxSwamp
	pnxSnowyPlains
	pnxMeadow
	pnxBeach
	pnxSnowyBeach
	pnxBadlands
	pnxOcean
	pnxDeepOcean
	pnxWarmOcean
	pnxColdOcean
	pnxFrozenOcean
	pnxDeepFrozenOcean
	pnxDeepWarmOcean
	pnxRiver
	pnxFrozenRiver
	pnxStonyPeaks
	pnxJaggedPeaks
	pnxFrozenPeaks
)

type pnxColumn struct {
	height      int
	biome       pnxBiome
	temperature float64
	humidity    float64
	continental float64
	isRiver     bool
}

// pnxCaveGrid samples the four cave density fields on a 4x8x4 lattice and
// interpolates them for individual blocks. This mirrors PNX's cell-based
// density generation and avoids evaluating expensive 3D noise at every voxel.
type pnxCaveGrid struct {
	minY, yCells int
	cheese       []float64
	mask         []float64
	spaghettiA   []float64
	spaghettiB   []float64
}

func newPNXCaveGrid(seed int64, baseX, baseZ, minY, maxY int) *pnxCaveGrid {
	cellMinY := floorDiv(minY, 8) * 8
	cellMaxY := -floorDiv(-maxY, 8) * 8
	yCells := (cellMaxY - cellMinY) / 8
	size := 5 * (yCells + 1) * 5
	grid := &pnxCaveGrid{
		minY: cellMinY, yCells: yCells,
		cheese: make([]float64, size), mask: make([]float64, size),
		spaghettiA: make([]float64, size), spaghettiB: make([]float64, size),
	}
	for x := 0; x <= 4; x++ {
		for y := 0; y <= yCells; y++ {
			for z := 0; z <= 4; z++ {
				wx, wy, wz := float64(baseX+x*4), float64(cellMinY+y*8), float64(baseZ+z*4)
				i := grid.index(x, y, z)
				grid.cheese[i] = noise3(seed+211, wx*0.017, wy*0.021, wz*0.017)
				grid.mask[i] = noise3(seed+223, wx*0.008, wy*0.012, wz*0.008)
				grid.spaghettiA[i] = noise3(seed+227, wx*0.031, wy*0.043, wz*0.031)
				grid.spaghettiB[i] = noise3(seed+229, (wx+391)*0.031, (wy-83)*0.043, (wz+127)*0.031)
			}
		}
	}
	return grid
}

func (grid *pnxCaveGrid) index(x, y, z int) int {
	return (x*(grid.yCells+1)+y)*5 + z
}

func (grid *pnxCaveGrid) interpolate(values []float64, x, y, z int) float64 {
	cx, cz := x/4, z/4
	cy := (y - grid.minY) / 8
	fx, fy, fz := float64(x%4)/4, float64((y-grid.minY)%8)/8, float64(z%4)/4
	if cy < 0 {
		cy = 0
		fy = 0
	}
	if cy >= grid.yCells {
		cy = grid.yCells - 1
		fy = 1
	}
	if cx > 3 {
		cx = 3
		fx = 1
	}
	if cz > 3 {
		cz = 3
		fz = 1
	}
	get := func(ix, iy, iz int) float64 { return values[grid.index(ix, iy, iz)] }
	v000 := get(cx, cy, cz)
	v100 := get(cx+1, cy, cz)
	v010 := get(cx, cy+1, cz)
	v110 := get(cx+1, cy+1, cz)
	v001 := get(cx, cy, cz+1)
	v101 := get(cx+1, cy, cz+1)
	v011 := get(cx, cy+1, cz+1)
	v111 := get(cx+1, cy+1, cz+1)
	x00, x10 := lerp(v000, v100, fx), lerp(v010, v110, fx)
	x01, x11 := lerp(v001, v101, fx), lerp(v011, v111, fx)
	return lerp(lerp(x00, x10, fy), lerp(x01, x11, fy), fz)
}

type pnxOverworldBlocks struct {
	stone, deepslate, bedrock, grass, dirt, podzol   uint32
	sand, redSand, sandstone, redSandstone, gravel   uint32
	water, lava, ice, snow, air                      uint32
	coalOre, ironOre, copperOre, goldOre             uint32
	redstoneOre, lapisOre, diamondOre, emeraldOre    uint32
	oakLog, oakLeaves, birchLog, birchLeaves         uint32
	spruceLog, spruceLeaves, jungleLog, jungleLeaves uint32
	acaciaLog, acaciaLeaves                          uint32
	shortGrass, fern, dandelion, poppy, blueOrchid   uint32
}

// PNXOverworld is a PNX-inspired overworld generator for Dragonfly. It ports the
// multi-signal terrain approach used by PowerNukkitX's Normal generator:
// continentalness, erosion, ridges, climate sampling, 3D density/cave sampling,
// surface replacement and deterministic feature placement. The implementation
// is native Go and writes directly into Dragonfly chunks.
type PNXOverworld struct {
	seed   int64
	blocks pnxOverworldBlocks
}

// NewPNXOverworld creates a deterministic PNX-style overworld generator.
func NewPNXOverworld(seed int64) *PNXOverworld {
	br := world.DefaultBlockRegistry
	id := func(b world.Block) uint32 { return br.BlockRuntimeID(b) }
	return &PNXOverworld{
		seed: seed,
		blocks: pnxOverworldBlocks{
			stone: id(block.Stone{}), deepslate: id(block.Deepslate{}), bedrock: id(block.Bedrock{}),
			grass: id(block.Grass{}), dirt: id(block.Dirt{}), podzol: id(block.Podzol{}),
			sand: id(block.Sand{}), redSand: id(block.Sand{Red: true}),
			sandstone: id(block.Sandstone{}), redSandstone: id(block.Sandstone{Red: true}), gravel: id(block.Gravel{}),
			water: id(block.Water{Still: true, Depth: 8}), lava: id(block.Lava{Still: true, Depth: 8}), ice: id(block.Ice{}), snow: id(block.Snow{}), air: br.AirRuntimeID(),
			coalOre: id(block.CoalOre{Type: block.StoneOre()}), ironOre: id(block.IronOre{Type: block.StoneOre()}),
			copperOre: id(block.CopperOre{Type: block.StoneOre()}), goldOre: id(block.GoldOre{Type: block.StoneOre()}),
			redstoneOre: id(block.RedstoneOre{Type: block.StoneOre()}), lapisOre: id(block.LapisOre{Type: block.StoneOre()}),
			diamondOre: id(block.DiamondOre{Type: block.StoneOre()}), emeraldOre: id(block.EmeraldOre{Type: block.StoneOre()}),
			oakLog:       id(block.Log{Wood: block.OakWood(), Axis: cube.Y}),
			oakLeaves:    id(block.Leaves{Type: block.OakLeaves(), Persistent: true}),
			birchLog:     id(block.Log{Wood: block.BirchWood(), Axis: cube.Y}),
			birchLeaves:  id(block.Leaves{Type: block.BirchLeaves(), Persistent: true}),
			spruceLog:    id(block.Log{Wood: block.SpruceWood(), Axis: cube.Y}),
			spruceLeaves: id(block.Leaves{Type: block.SpruceLeaves(), Persistent: true}),
			jungleLog:    id(block.Log{Wood: block.JungleWood(), Axis: cube.Y}),
			jungleLeaves: id(block.Leaves{Type: block.JungleLeaves(), Persistent: true}),
			acaciaLog:    id(block.Log{Wood: block.AcaciaWood(), Axis: cube.Y}),
			acaciaLeaves: id(block.Leaves{Type: block.AcaciaLeaves(), Persistent: true}),
			shortGrass:   id(block.ShortGrass{}), fern: id(block.Fern{}),
			dandelion: id(block.Flower{Type: block.Dandelion()}), poppy: id(block.Flower{Type: block.Poppy()}),
			blueOrchid: id(block.Flower{Type: block.BlueOrchid()}),
		},
	}
}

// GenerateChunk generates a complete overworld chunk. Each chunk is generated
// solely from the configured seed and world coordinates, so generation order
// and worker scheduling cannot change the result.
func (g *PNXOverworld) GenerateChunk(pos world.ChunkPos, c *chunk.Chunk) {
	minY, maxY := c.Range().Min(), c.Range().Max()
	baseX, baseZ := int(pos.X())<<4, int(pos.Z())<<4
	var columns [16][16]pnxColumn

	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			columns[x][z] = g.sampleColumn(baseX+x, baseZ+z)
		}
	}
	caveGrid := newPNXCaveGrid(g.seed, baseX, baseZ, minY, maxY)
	g.writeBiomeSections(c, columns, caveGrid, minY, maxY)

	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			col := columns[x][z]
			worldX, worldZ := baseX+x, baseZ+z
			bedrockDepth := int(latticeHash(g.seed+71, int64(worldX), 0, int64(worldZ)) % 5)
			surfaceDepth := 3 + int(latticeHash(g.seed+73, int64(worldX), 0, int64(worldZ))%3)

			for y := minY; y <= maxY; y++ {
				runtimeID := g.blocks.air
				switch {
				case y == minY || y <= minY+bedrockDepth:
					runtimeID = g.blocks.bedrock
				case y <= col.height:
					if g.isCave(caveGrid, x, y, z, col.height, minY) {
						if y <= -54 {
							runtimeID = g.blocks.lava
						} else if y <= -28 && noise3(g.seed+233, float64(worldX)*0.02, float64(y)*0.02, float64(worldZ)*0.02) > 0.42 {
							runtimeID = g.blocks.water
						}
					} else {
						runtimeID = g.solidBlock(worldX, y, worldZ, col, surfaceDepth)
					}
				case y <= pnxSeaLevel:
					if y == pnxSeaLevel && (col.biome == pnxFrozenOcean || col.biome == pnxDeepFrozenOcean || col.biome == pnxSnowyBeach) {
						runtimeID = g.blocks.ice
					} else {
						runtimeID = g.blocks.water
					}
				}
				if runtimeID != g.blocks.air {
					c.SetBlock(uint8(x), int16(y), uint8(z), 0, runtimeID)
				}
			}
		}
	}

	rng := rand.New(rand.NewSource(int64(chunkSeed(g.seed, pos))))
	g.generateOres(c, rng, minY, maxY)
	g.generateVegetation(c, rng, columns, minY, maxY)
}

func (g *PNXOverworld) DefaultSpawn(dim world.Dimension) cube.Pos {
	bestX, bestZ, bestHeight := 0, 0, pnxSeaLevel+4
	bestScore := math.MaxFloat64
	for radius := 0; radius <= 256; radius += 8 {
		for x := -radius; x <= radius; x += 8 {
			for _, z := range []int{-radius, radius} {
				if radius == 0 && z != 0 {
					continue
				}
				col := g.sampleColumn(x, z)
				if col.height <= pnxSeaLevel+1 || col.biome == pnxOcean || col.biome == pnxDeepOcean || col.biome == pnxFrozenOcean || col.biome == pnxDeepFrozenOcean || col.biome == pnxWarmOcean || col.biome == pnxColdOcean || col.biome == pnxDeepWarmOcean {
					continue
				}
				score := float64(x*x+z*z) + math.Abs(float64(col.height-72))*8
				if score < bestScore {
					bestX, bestZ, bestHeight, bestScore = x, z, col.height, score
				}
			}
			if radius > 0 {
				for _, z := range []int{-radius + 8, radius - 8} {
					if x == -radius || x == radius {
						continue
					}
					col := g.sampleColumn(x, z)
					if col.height <= pnxSeaLevel+1 || col.biome == pnxOcean || col.biome == pnxDeepOcean || col.biome == pnxFrozenOcean || col.biome == pnxDeepFrozenOcean || col.biome == pnxWarmOcean || col.biome == pnxColdOcean || col.biome == pnxDeepWarmOcean {
						continue
					}
					score := float64(x*x+z*z) + math.Abs(float64(col.height-72))*8
					if score < bestScore {
						bestX, bestZ, bestHeight, bestScore = x, z, col.height, score
					}
				}
			}
		}
		if bestScore < math.MaxFloat64 {
			break
		}
	}
	return cube.Pos{bestX, max(dim.Range().Min()+2, min(dim.Range().Max()-2, bestHeight+1)), bestZ}
}

func (g *PNXOverworld) sampleColumn(x, z int) pnxColumn {
	// Low-frequency climate and continental signals are sampled separately,
	// as in PNX's biome picker and density-function graph.
	warpX := fbm2(g.seed+11, float64(x), float64(z), 0.00055, 2) * 220
	warpZ := fbm2(g.seed+13, float64(x+913), float64(z-127), 0.00055, 2) * 220
	sx, sz := float64(x)+warpX, float64(z)+warpZ
	continental := fbm2Normalized(g.seed+101, sx, sz, 0.00038, 5)
	erosion := fbm2Normalized(g.seed+103, sx, sz, 0.00115, 4)
	weirdness := fbm2Normalized(g.seed+107, sx, sz, 0.00082, 4)
	temperature := fbm2Normalized(g.seed+109, sx, sz, 0.00048, 4)
	humidity := fbm2Normalized(g.seed+113, sx, sz, 0.00052, 4)
	detail := fbm2(g.seed+127, sx, sz, 0.0062, 4)*9 + fbm2(g.seed+131, sx, sz, 0.021, 2)*2.5

	var height float64
	if continental < -0.18 {
		height = 30 + (continental+1)*35 + detail*0.7
	} else {
		height = 67 + continental*38 + erosion*8 + detail
	}

	ridgeSignal := 1 - math.Abs(fbm2Normalized(g.seed+137, sx, sz, 0.00092, 4))
	ridgeStrength := smoothstep(0.56, 0.88, ridgeSignal)
	mountainMask := smoothstep(-0.02, 0.48, continental)
	peakBias := 0.40 + 0.60*math.Max(0, weirdness)
	height += mountainMask * ridgeStrength * peakBias * 118
	height += smoothstep(0.0, 0.5, continental) * math.Max(0, 0.35-erosion) * 20

	// Rivers use a separate, narrow density valley, rather than being painted
	// onto an otherwise flat terrain surface.
	riverNoise := math.Abs(noise2(g.seed+149, sx*0.00135, sz*0.00135))
	isRiver := continental > -0.12 && riverNoise < 0.035 && height < 98
	if isRiver {
		height = math.Min(height, 61+riverNoise*28)
	}
	height = clamp(height, -48, 245)

	col := pnxColumn{
		height: int(math.Round(height)), temperature: temperature,
		humidity: humidity, continental: continental, isRiver: isRiver,
	}
	col.biome = pickPNXBiome(col)
	return col
}

func pickPNXBiome(col pnxColumn) pnxBiome {
	if col.continental < -0.62 {
		if col.temperature < -0.32 {
			return pnxDeepFrozenOcean
		}
		if col.temperature > 0.48 {
			return pnxDeepWarmOcean
		}
		return pnxDeepOcean
	}
	if col.continental < -0.27 {
		if col.temperature < -0.32 {
			return pnxFrozenOcean
		}
		if col.temperature > 0.42 {
			return pnxWarmOcean
		}
		if col.temperature < -0.08 {
			return pnxColdOcean
		}
		return pnxOcean
	}
	if col.isRiver {
		if col.temperature < -0.4 {
			return pnxFrozenRiver
		}
		return pnxRiver
	}
	if col.height >= pnxSeaLevel-4 && col.height <= pnxSeaLevel+3 {
		if col.temperature < -0.4 {
			return pnxSnowyBeach
		}
		return pnxBeach
	}
	if col.height > 145 {
		if col.temperature < -0.28 {
			return pnxFrozenPeaks
		}
		if col.temperature < 0.1 {
			return pnxJaggedPeaks
		}
		return pnxStonyPeaks
	}
	if col.temperature > 0.32 && col.humidity < -0.10 {
		if col.height > 105 {
			return pnxBadlands
		}
		return pnxDesert
	}
	if col.temperature > 0.24 && col.humidity > 0.20 {
		return pnxJungle
	}
	if col.temperature > 0.12 && col.humidity < 0.05 {
		return pnxSavanna
	}
	if col.humidity > 0.48 && col.height < 82 {
		return pnxSwamp
	}
	if col.temperature < -0.46 {
		return pnxSnowyPlains
	}
	if col.temperature < -0.12 {
		return pnxTaiga
	}
	if col.height > 112 && col.humidity > 0.05 {
		return pnxMeadow
	}
	if col.humidity > 0.08 && col.temperature > -0.12 && col.temperature < 0.42 {
		return pnxBirchForest
	}
	if col.humidity > -0.25 {
		return pnxForest
	}
	return pnxPlains
}

func (g *PNXOverworld) writeBiomeSections(c *chunk.Chunk, columns [16][16]pnxColumn, caves *pnxCaveGrid, minY, maxY int) {
	for sectionY := minY; sectionY <= maxY; sectionY += 16 {
		var section [4][4][4]uint32
		for qx := 0; qx < 4; qx++ {
			for qz := 0; qz < 4; qz++ {
				col := columns[qx*4+2][qz*4+2]
				surfaceBiomeID := biomeID(col.biome)
				for qy := 0; qy < 4; qy++ {
					id := surfaceBiomeID
					centerY := sectionY + qy*4 + 2
					if centerY <= -40 && g.isCave(caves, qx*4+2, centerY, qz*4+2, col.height, minY) {
						switch {
						case col.humidity > 0.26:
							id = uint32(biome.LushCaves{}.EncodeBiome())
						case col.humidity < -0.24:
							id = uint32(biome.DeepDark{}.EncodeBiome())
						default:
							id = uint32(biome.DripstoneCaves{}.EncodeBiome())
						}
					}
					section[qx][qy][qz] = id
				}
			}
		}
		c.SetBiomeSection(int16(sectionY), section)
	}
}

func biomeID(kind pnxBiome) uint32 {
	switch kind {
	case pnxForest:
		return uint32(biome.Forest{}.EncodeBiome())
	case pnxBirchForest:
		return uint32(biome.BirchForest{}.EncodeBiome())
	case pnxTaiga:
		return uint32(biome.Taiga{}.EncodeBiome())
	case pnxJungle:
		return uint32(biome.Jungle{}.EncodeBiome())
	case pnxDesert:
		return uint32(biome.Desert{}.EncodeBiome())
	case pnxSavanna:
		return uint32(biome.Savanna{}.EncodeBiome())
	case pnxSwamp:
		return uint32(biome.Swamp{}.EncodeBiome())
	case pnxSnowyPlains:
		return uint32(biome.SnowyPlains{}.EncodeBiome())
	case pnxMeadow:
		return uint32(biome.Meadow{}.EncodeBiome())
	case pnxBeach:
		return uint32(biome.Beach{}.EncodeBiome())
	case pnxSnowyBeach:
		return uint32(biome.SnowyBeach{}.EncodeBiome())
	case pnxBadlands:
		return uint32(biome.Badlands{}.EncodeBiome())
	case pnxOcean:
		return uint32(biome.Ocean{}.EncodeBiome())
	case pnxDeepOcean:
		return uint32(biome.DeepOcean{}.EncodeBiome())
	case pnxWarmOcean:
		return uint32(biome.WarmOcean{}.EncodeBiome())
	case pnxColdOcean:
		return uint32(biome.ColdOcean{}.EncodeBiome())
	case pnxFrozenOcean:
		return uint32(biome.FrozenOcean{}.EncodeBiome())
	case pnxDeepFrozenOcean:
		return uint32(biome.DeepFrozenOcean{}.EncodeBiome())
	case pnxDeepWarmOcean:
		return uint32(biome.DeepWarmOcean{}.EncodeBiome())
	case pnxRiver:
		return uint32(biome.River{}.EncodeBiome())
	case pnxFrozenRiver:
		return uint32(biome.FrozenRiver{}.EncodeBiome())
	case pnxStonyPeaks:
		return uint32(biome.StonyPeaks{}.EncodeBiome())
	case pnxJaggedPeaks:
		return uint32(biome.JaggedPeaks{}.EncodeBiome())
	case pnxFrozenPeaks:
		return uint32(biome.FrozenPeaks{}.EncodeBiome())
	default:
		return uint32(biome.Plains{}.EncodeBiome())
	}
}

func (g *PNXOverworld) solidBlock(x, y, z int, col pnxColumn, surfaceDepth int) uint32 {
	if y < 0 {
		if y >= col.height-surfaceDepth-1 {
			return g.blocks.deepslate
		}
		return g.blocks.deepslate
	}
	if y == col.height {
		switch col.biome {
		case pnxDesert, pnxBeach, pnxWarmOcean, pnxOcean, pnxDeepOcean, pnxColdOcean, pnxFrozenOcean, pnxDeepFrozenOcean, pnxDeepWarmOcean:
			return g.blocks.sand
		case pnxSnowyBeach:
			return g.blocks.gravel
		case pnxBadlands:
			return g.blocks.redSand
		case pnxStonyPeaks, pnxJaggedPeaks:
			return g.blocks.stone
		case pnxFrozenPeaks:
			return g.blocks.snow
		case pnxTaiga:
			return g.blocks.podzol
		default:
			return g.blocks.grass
		}
	}
	if y >= col.height-surfaceDepth {
		switch col.biome {
		case pnxDesert, pnxBeach, pnxWarmOcean, pnxOcean, pnxDeepOcean, pnxColdOcean, pnxFrozenOcean, pnxDeepFrozenOcean, pnxDeepWarmOcean:
			if y <= col.height-5 {
				return g.blocks.sandstone
			}
			return g.blocks.sand
		case pnxBadlands:
			if y <= col.height-5 {
				return g.blocks.redSandstone
			}
			return g.blocks.redSand
		default:
			return g.blocks.dirt
		}
	}
	if col.biome == pnxBadlands && y > 40 && noise2(g.seed+157, float64(x)*0.12, float64(z)*0.12) > 0.40 {
		return g.blocks.redSandstone
	}
	if y < 0 {
		return g.blocks.deepslate
	}
	return g.blocks.stone
}

func (g *PNXOverworld) isCave(grid *pnxCaveGrid, localX, y, localZ, surface, minY int) bool {
	if y >= surface-7 || y <= minY+5 {
		return false
	}
	cheese := grid.interpolate(grid.cheese, localX, y, localZ)
	mask := grid.interpolate(grid.mask, localX, y, localZ)
	spaghettiA := grid.interpolate(grid.spaghettiA, localX, y, localZ)
	spaghettiB := grid.interpolate(grid.spaghettiB, localX, y, localZ)
	room := cheese > 0.36 && mask > -0.1 && y < surface-18
	tunnel := math.Abs(spaghettiA) < 0.045 && math.Abs(spaghettiB) < 0.12 && y < surface-10
	return room || tunnel
}

func (g *PNXOverworld) generateOres(c *chunk.Chunk, rng *rand.Rand, minY, maxY int) {
	type vein struct {
		id         uint32
		count      int
		minY, maxY int
		size       int
	}
	veins := []vein{
		{g.blocks.coalOre, 24, max(minY+8, -48), min(maxY-12, 192), 7},
		{g.blocks.ironOre, 20, max(minY+8, -52), 112, 6},
		{g.blocks.copperOre, 18, -16, 112, 7},
		{g.blocks.goldOre, 7, -48, 40, 5},
		{g.blocks.lapisOre, 5, -48, 48, 5},
		{g.blocks.redstoneOre, 8, minY + 8, 24, 6},
		{g.blocks.diamondOre, 5, minY + 8, 16, 4},
		{g.blocks.emeraldOre, 2, 8, 220, 3},
	}
	for _, v := range veins {
		low, high := max(minY+1, v.minY), min(maxY-1, v.maxY)
		if low > high {
			continue
		}
		for attempt := 0; attempt < v.count; attempt++ {
			x, y, z := rng.Intn(16), low+rng.Intn(high-low+1), rng.Intn(16)
			for i := 0; i < v.size; i++ {
				sx := x + rng.Intn(3) - 1
				sy := y + rng.Intn(3) - 1
				sz := z + rng.Intn(3) - 1
				if sx < 0 || sx > 15 || sz < 0 || sz > 15 || sy < minY || sy > maxY {
					continue
				}
				old := c.Block(uint8(sx), int16(sy), uint8(sz), 0)
				if old != g.blocks.stone && old != g.blocks.deepslate {
					continue
				}
				oreID := v.id
				if sy < 0 {
					oreID = g.deepslateOre(v.id)
				}
				c.SetBlock(uint8(sx), int16(sy), uint8(sz), 0, oreID)
			}
		}
	}
}

func (g *PNXOverworld) deepslateOre(stoneOreID uint32) uint32 {
	switch stoneOreID {
	case g.blocks.coalOre:
		return world.DefaultBlockRegistry.BlockRuntimeID(block.CoalOre{Type: block.DeepslateOre()})
	case g.blocks.ironOre:
		return world.DefaultBlockRegistry.BlockRuntimeID(block.IronOre{Type: block.DeepslateOre()})
	case g.blocks.copperOre:
		return world.DefaultBlockRegistry.BlockRuntimeID(block.CopperOre{Type: block.DeepslateOre()})
	case g.blocks.goldOre:
		return world.DefaultBlockRegistry.BlockRuntimeID(block.GoldOre{Type: block.DeepslateOre()})
	case g.blocks.lapisOre:
		return world.DefaultBlockRegistry.BlockRuntimeID(block.LapisOre{Type: block.DeepslateOre()})
	case g.blocks.redstoneOre:
		return world.DefaultBlockRegistry.BlockRuntimeID(block.RedstoneOre{Type: block.DeepslateOre()})
	case g.blocks.diamondOre:
		return world.DefaultBlockRegistry.BlockRuntimeID(block.DiamondOre{Type: block.DeepslateOre()})
	case g.blocks.emeraldOre:
		return world.DefaultBlockRegistry.BlockRuntimeID(block.EmeraldOre{Type: block.DeepslateOre()})
	default:
		return stoneOreID
	}
}

func (g *PNXOverworld) generateVegetation(c *chunk.Chunk, rng *rand.Rand, columns [16][16]pnxColumn, minY, maxY int) {
	for attempt := 0; attempt < 9; attempt++ {
		x, z := 3+rng.Intn(10), 3+rng.Intn(10)
		col := columns[x][z]
		if col.height <= pnxSeaLevel+1 || col.height+8 >= maxY || rng.Float64() > treeChance(col.biome) {
			continue
		}
		logID, leavesID := g.treeBlocks(col.biome)
		if logID == 0 || leavesID == 0 {
			continue
		}
		baseY := col.height + 1
		height := 4 + rng.Intn(3)
		for dy := 0; dy < height; dy++ {
			c.SetBlock(uint8(x), int16(baseY+dy), uint8(z), 0, logID)
		}
		for dy := -2; dy <= 1; dy++ {
			radius := 2
			if dy == 1 {
				radius = 1
			}
			for dx := -radius; dx <= radius; dx++ {
				for dz := -radius; dz <= radius; dz++ {
					if absInt(dx)+absInt(dz) > radius+1 || (dx == 0 && dz == 0 && dy <= 0) {
						continue
					}
					lx, lz, ly := x+dx, z+dz, baseY+height+dy
					if lx < 0 || lx > 15 || lz < 0 || lz > 15 || ly < minY || ly > maxY {
						continue
					}
					if c.Block(uint8(lx), int16(ly), uint8(lz), 0) == g.blocks.air {
						c.SetBlock(uint8(lx), int16(ly), uint8(lz), 0, leavesID)
					}
				}
			}
		}
	}

	for attempt := 0; attempt < 34; attempt++ {
		x, z := rng.Intn(16), rng.Intn(16)
		col := columns[x][z]
		if col.height <= pnxSeaLevel+1 || col.height+1 > maxY || col.biome == pnxDesert || col.biome == pnxBadlands || col.biome == pnxBeach || col.biome == pnxSnowyBeach || col.biome == pnxFrozenPeaks {
			continue
		}
		if c.Block(uint8(x), int16(col.height+1), uint8(z), 0) != g.blocks.air {
			continue
		}
		id := g.blocks.shortGrass
		switch {
		case col.biome == pnxTaiga && rng.Intn(2) == 0:
			id = g.blocks.fern
		case col.biome == pnxSwamp && rng.Intn(3) == 0:
			id = g.blocks.blueOrchid
		case rng.Intn(7) == 0:
			if rng.Intn(2) == 0 {
				id = g.blocks.dandelion
			} else {
				id = g.blocks.poppy
			}
		}
		c.SetBlock(uint8(x), int16(col.height+1), uint8(z), 0, id)
	}
}

func treeChance(b pnxBiome) float64 {
	switch b {
	case pnxForest, pnxBirchForest, pnxTaiga, pnxJungle:
		return 0.82
	case pnxSwamp, pnxMeadow:
		return 0.38
	case pnxSavanna:
		return 0.22
	default:
		return 0.015
	}
}

func (g *PNXOverworld) treeBlocks(b pnxBiome) (uint32, uint32) {
	switch b {
	case pnxBirchForest:
		return g.blocks.birchLog, g.blocks.birchLeaves
	case pnxTaiga, pnxSnowyPlains:
		return g.blocks.spruceLog, g.blocks.spruceLeaves
	case pnxJungle:
		return g.blocks.jungleLog, g.blocks.jungleLeaves
	case pnxSavanna:
		return g.blocks.acaciaLog, g.blocks.acaciaLeaves
	case pnxForest, pnxSwamp, pnxMeadow:
		return g.blocks.oakLog, g.blocks.oakLeaves
	default:
		return 0, 0
	}
}

func chunkSeed(seed int64, pos world.ChunkPos) uint64 {
	return latticeHash(seed, int64(pos.X()), 0x51ed, int64(pos.Z()))
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
