package cmd

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

const (
	maxForceloadChunksPerCommand = 256
	maxForceloadChunksPerWorld   = 2048
	maxForceloadBlockCoordinate  = 30000000
)

var forcedChunkLoaders = struct {
	sync.Mutex
	worlds map[*world.World]map[world.ChunkPos]*world.Loader
}{
	worlds: make(map[*world.World]map[world.ChunkPos]*world.Loader),
}

func init() {
	Register(New("forceload", "Keeps selected chunks loaded while the server is running.", nil,
		forceloadAddCommand{}, forceloadRemoveCommand{}, forceloadRemoveAllCommand{}, forceloadQueryCommand{}))
}

type forceloadAddCommand struct {
	Add   SubCommand
	FromX int
	FromZ int
	ToX   Optional[int]
	ToZ   Optional[int]
}

func (forceloadAddCommand) Allow(src Source) bool { return commandOperator(src) }

func (c forceloadAddCommand) Run(_ Source, out *Output, tx *world.Tx) {
	if tx == nil {
		out.Error("forceload requires a world transaction.")
		return
	}
	toX, hasX := c.ToX.Load()
	toZ, hasZ := c.ToZ.Load()
	if hasX != hasZ {
		out.Error("The optional end position must include both X and Z.")
		return
	}
	if !hasX {
		toX, toZ = c.FromX, c.FromZ
	}
	positions, err := forceloadChunkRange(c.FromX, c.FromZ, toX, toZ)
	if err != nil {
		out.Error(err)
		return
	}
	added, err := addForcedChunks(tx, positions)
	if err != nil {
		out.Error(err)
		return
	}
	out.Printf("Force-loaded %d new chunk(s).", added)
}

type forceloadRemoveCommand struct {
	Remove SubCommand
	FromX  int
	FromZ  int
	ToX    Optional[int]
	ToZ    Optional[int]
}

func (forceloadRemoveCommand) Allow(src Source) bool { return commandOperator(src) }

func (c forceloadRemoveCommand) Run(_ Source, out *Output, tx *world.Tx) {
	if tx == nil {
		out.Error("forceload requires a world transaction.")
		return
	}
	toX, hasX := c.ToX.Load()
	toZ, hasZ := c.ToZ.Load()
	if hasX != hasZ {
		out.Error("The optional end position must include both X and Z.")
		return
	}
	if !hasX {
		toX, toZ = c.FromX, c.FromZ
	}
	positions, err := forceloadChunkRange(c.FromX, c.FromZ, toX, toZ)
	if err != nil {
		out.Error(err)
		return
	}
	removed := removeForcedChunks(tx, positions)
	out.Printf("Removed %d force-loaded chunk(s).", removed)
}

type forceloadRemoveAllCommand struct {
	Remove SubCommand
	All    SubCommand
}

func (forceloadRemoveAllCommand) Allow(src Source) bool { return commandOperator(src) }

func (forceloadRemoveAllCommand) Run(_ Source, out *Output, tx *world.Tx) {
	if tx == nil {
		out.Error("forceload requires a world transaction.")
		return
	}
	removed := removeAllForcedChunks(tx)
	out.Printf("Removed %d force-loaded chunk(s).", removed)
}

type forceloadQueryCommand struct {
	Query SubCommand
	X     Optional[int]
	Z     Optional[int]
}

func (forceloadQueryCommand) Allow(src Source) bool { return commandOperator(src) }

func (c forceloadQueryCommand) Run(src Source, out *Output, tx *world.Tx) {
	if tx == nil {
		out.Error("forceload requires a world transaction.")
		return
	}
	x, hasX := c.X.Load()
	z, hasZ := c.Z.Load()
	if hasX != hasZ {
		out.Error("A query position must include both X and Z.")
		return
	}
	if !hasX {
		position := src.Position()
		x, z = int(math.Floor(position[0])), int(math.Floor(position[2]))
	}
	pos := world.ChunkPos{blockToChunkCoordinate(x), blockToChunkCoordinate(z)}
	forcedChunkLoaders.Lock()
	_, found := forcedChunkLoaders.worlds[tx.World()][pos]
	total := len(forcedChunkLoaders.worlds[tx.World()])
	forcedChunkLoaders.Unlock()
	out.Printf("Chunk %d %d is force-loaded: %t (%d chunk(s) currently forced).", pos[0], pos[1], found, total)
}

func forceloadChunkRange(fromX, fromZ, toX, toZ int) ([]world.ChunkPos, error) {
	for _, coordinate := range []int{fromX, fromZ, toX, toZ} {
		if coordinate < -maxForceloadBlockCoordinate || coordinate > maxForceloadBlockCoordinate {
			return nil, fmt.Errorf("forceload coordinates must be within +/- %d blocks", maxForceloadBlockCoordinate)
		}
	}
	minX, maxX := blockToChunkCoordinate(fromX), blockToChunkCoordinate(toX)
	minZ, maxZ := blockToChunkCoordinate(fromZ), blockToChunkCoordinate(toZ)
	if minX > maxX {
		minX, maxX = maxX, minX
	}
	if minZ > maxZ {
		minZ, maxZ = maxZ, minZ
	}
	width, depth := int(maxX-minX+1), int(maxZ-minZ+1)
	if int64(width)*int64(depth) > maxForceloadChunksPerCommand {
		return nil, fmt.Errorf("a single forceload operation may affect at most %d chunks", maxForceloadChunksPerCommand)
	}
	positions := make([]world.ChunkPos, 0, width*depth)
	for x := minX; x <= maxX; x++ {
		for z := minZ; z <= maxZ; z++ {
			positions = append(positions, world.ChunkPos{x, z})
		}
	}
	return positions, nil
}

func blockToChunkCoordinate(blockCoordinate int) int32 {
	chunkCoordinate := blockCoordinate / 16
	if blockCoordinate%16 < 0 {
		chunkCoordinate--
	}
	return int32(chunkCoordinate)
}

func addForcedChunks(tx *world.Tx, positions []world.ChunkPos) (int, error) {
	w := tx.World()
	forcedChunkLoaders.Lock()
	byChunk := forcedChunkLoaders.worlds[w]
	if byChunk == nil {
		byChunk = make(map[world.ChunkPos]*world.Loader)
		forcedChunkLoaders.worlds[w] = byChunk
	}
	var added []world.ChunkPos
	for _, pos := range positions {
		if _, exists := byChunk[pos]; !exists {
			added = append(added, pos)
		}
	}
	if len(byChunk)+len(added) > maxForceloadChunksPerWorld {
		forcedChunkLoaders.Unlock()
		return 0, fmt.Errorf("this world may have at most %d force-loaded chunks", maxForceloadChunksPerWorld)
	}
	for _, pos := range added {
		loader := world.NewLoader(0, w, world.NopViewer{})
		loader.Move(tx, mgl64.Vec3{float64(pos[0])*16 + 8, 0, float64(pos[1])*16 + 8})
		loader.Load(tx, 1)
		byChunk[pos] = loader
		retryForcedChunkLoad(w, loader, pos, 3)
	}
	forcedChunkLoaders.Unlock()
	return len(added), nil
}

func retryForcedChunkLoad(w *world.World, loader *world.Loader, pos world.ChunkPos, attempts int) {
	if attempts <= 0 {
		return
	}
	w.DoAfter(100*time.Millisecond, func(tx *world.Tx) {
		if _, loaded := loader.Chunk(pos); !loaded {
			loader.Load(tx, 1)
			retryForcedChunkLoad(w, loader, pos, attempts-1)
		}
	})
}

func removeForcedChunks(tx *world.Tx, positions []world.ChunkPos) int {
	w := tx.World()
	forcedChunkLoaders.Lock()
	byChunk := forcedChunkLoaders.worlds[w]
	var loaders []*world.Loader
	for _, pos := range positions {
		if loader, ok := byChunk[pos]; ok {
			loaders = append(loaders, loader)
			delete(byChunk, pos)
		}
	}
	if len(byChunk) == 0 {
		delete(forcedChunkLoaders.worlds, w)
	}
	forcedChunkLoaders.Unlock()
	for _, loader := range loaders {
		loader.Close(tx)
	}
	return len(loaders)
}

func removeAllForcedChunks(tx *world.Tx) int {
	w := tx.World()
	forcedChunkLoaders.Lock()
	byChunk := forcedChunkLoaders.worlds[w]
	loaders := make([]*world.Loader, 0, len(byChunk))
	for _, loader := range byChunk {
		loaders = append(loaders, loader)
	}
	delete(forcedChunkLoaders.worlds, w)
	forcedChunkLoaders.Unlock()
	for _, loader := range loaders {
		loader.Close(tx)
	}
	return len(loaders)
}
