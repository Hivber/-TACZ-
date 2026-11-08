package session

import (
	"bytes"
	"sync"

	"github.com/cespare/xxhash/v2"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// subChunkRequests is set to true to enable the sub-chunk request system. This can (likely) cause unexpected issues,
// but also solves issues with block entities such as item frames and lecterns as of v1.19.10.
const subChunkRequests = true

const encodedChunkPartCacheCapacity = 128

type encodedChunkPartKey struct {
	chunk    *chunk.Chunk
	index    int16 // -1 identifies the biome payload.
	revision uint64
}

func (s *Session) cachedChunkPart(c *chunk.Chunk, index int16, revision uint64, encode func() []byte) []byte {
	key := encodedChunkPartKey{chunk: c, index: index, revision: revision}
	s.blobMu.Lock()
	if data, ok := s.encodedChunkParts[key]; ok {
		s.encodedChunkPartCacheHits.Add(1)
		s.blobMu.Unlock()
		return data
	}
	s.encodedChunkPartCacheMisses.Add(1)
	s.blobMu.Unlock()

	data := encode()
	s.blobMu.Lock()
	if existing, ok := s.encodedChunkParts[key]; ok {
		data = existing
	} else {
		if len(s.encodedChunkPartOrder) >= encodedChunkPartCacheCapacity {
			oldest := s.encodedChunkPartOrder[0]
			delete(s.encodedChunkParts, oldest)
			copy(s.encodedChunkPartOrder, s.encodedChunkPartOrder[1:])
			s.encodedChunkPartOrder = s.encodedChunkPartOrder[:len(s.encodedChunkPartOrder)-1]
		}
		s.encodedChunkParts[key] = data
		s.encodedChunkPartOrder = append(s.encodedChunkPartOrder, key)
	}
	s.blobMu.Unlock()
	return data
}

func (s *Session) encodeSubChunk(c *chunk.Chunk, index int) []byte {
	return s.cachedChunkPart(c, int16(index), c.SubChunkRevision(index), func() []byte {
		return chunk.EncodeSubChunk(c, chunk.NetworkEncoding, index)
	})
}

func (s *Session) encodeBiomes(c *chunk.Chunk) []byte {
	return s.cachedChunkPart(c, -1, c.BiomeRevision(), func() []byte {
		return chunk.EncodeBiomes(c, chunk.NetworkEncoding)
	})
}

func (s *Session) encodeChunk(c *chunk.Chunk) chunk.SerialisedData {
	data := chunk.SerialisedData{SubChunks: make([][]byte, len(c.Sub()))}
	for i := range data.SubChunks {
		data.SubChunks[i] = s.encodeSubChunk(c, i)
	}
	data.Biomes = s.encodeBiomes(c)
	return data
}

var blockEntityBufferPool = sync.Pool{
	New: func() any { return bytes.NewBuffer(make([]byte, 0, 256)) },
}

// encodeBlockEntityPayload returns an owned copy, never a slice backed by pooled storage.
func encodeBlockEntityPayload(entities map[cube.Pos]world.Block, subChunk *int16, col *world.Column, includeBorderCount bool) []byte {
	if len(entities) == 0 && !includeBorderCount {
		return nil
	}
	buf := blockEntityBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	if includeBorderCount {
		buf.WriteByte(0)
	}
	enc := nbt.NewEncoderWithEncoding(buf, nbt.NetworkLittleEndian)
	for pos, b := range entities {
		if subChunk != nil && col.SubIndex(int16(pos.Y())) != *subChunk {
			continue
		}
		if n, ok := b.(world.NBTer); ok {
			data := n.EncodeNBT()
			data["x"], data["y"], data["z"] = int32(pos[0]), int32(pos[1]), int32(pos[2])
			_ = enc.Encode(data)
		}
	}
	payload := bytes.Clone(buf.Bytes())
	buf.Reset()
	if buf.Cap() <= 64*1024 {
		blockEntityBufferPool.Put(buf)
	}
	return payload
}

// ViewChunk ...
func (s *Session) ViewChunk(pos world.ChunkPos, dim world.Dimension, blockEntities map[cube.Pos]world.Block, c *chunk.Chunk) {
	if !s.conn.ClientCacheEnabled() {
		s.sendNetworkChunk(pos, dim, c, blockEntities)
		return
	}
	s.sendBlobHashes(pos, dim, c, blockEntities)
}

// ViewSubChunks ...
func (s *Session) ViewSubChunks(centre world.SubChunkPos, offsets []protocol.SubChunkOffset, tx *world.Tx) {
	r := tx.Range()

	entries := make([]protocol.SubChunkEntry, 0, len(offsets))
	transaction := make(map[uint64]struct{})
	for _, offset := range offsets {
		ind := int16(centre.Y()) + int16(offset[1]) - int16(r[0])>>4
		if ind < 0 || ind > int16(r.Height()>>4) {
			entries = append(entries, protocol.SubChunkEntry{Result: protocol.SubChunkResultIndexOutOfBounds, Offset: offset})
			continue
		}
		col, ok := s.chunkLoader.Chunk(world.ChunkPos{
			centre.X() + int32(offset[0]),
			centre.Z() + int32(offset[2]),
		})
		if !ok {
			entries = append(entries, protocol.SubChunkEntry{Result: protocol.SubChunkResultChunkNotFound, Offset: offset})
			continue
		}
		entries = append(entries, s.subChunkEntry(offset, ind, col, transaction))
	}
	if s.conn.ClientCacheEnabled() && len(transaction) > 0 {
		s.blobMu.Lock()
		s.openChunkTransactions = append(s.openChunkTransactions, transaction)
		s.blobMu.Unlock()
	}
	dim, _ := world.DimensionID(tx.World().Dimension())
	s.writePacket(&packet.SubChunk{
		Dimension:       int32(dim),
		Position:        protocol.SubChunkPos(centre),
		CacheEnabled:    s.conn.ClientCacheEnabled(),
		SubChunkEntries: entries,
	})
}

func (s *Session) subChunkEntry(offset protocol.SubChunkOffset, ind int16, col *world.Column, transaction map[uint64]struct{}) protocol.SubChunkEntry {
	chunkMap := col.HeightMap()
	subMapType, subMap := byte(protocol.HeightMapDataHasData), protocol.HeightMap{}
	higher, lower := true, true
	for z := uint8(0); z < 16; z++ {
		for x := uint8(0); x < 16; x++ {
			y := chunkMap.At(x, z)
			otherInd := col.SubIndex(y)
			switch {
			case otherInd > ind:
				subMap[z][x], lower = 16, false
			case otherInd < ind:
				subMap[z][x], higher = -1, false
			default:
				subMap[z][x], lower, higher = int8(y-col.SubY(otherInd)), false, false
			}
		}
	}
	var subMapData protocol.Optional[protocol.HeightMap]
	switch {
	case higher:
		subMapType = protocol.HeightMapDataTooHigh
	case lower:
		subMapType = protocol.HeightMapDataTooLow
	default:
		subMapData = protocol.Option(subMap)
	}

	sub := col.Sub()[ind]
	if sub.Empty() {
		return protocol.SubChunkEntry{
			Result:              protocol.SubChunkResultSuccessAllAir,
			HeightMapType:       subMapType,
			HeightMapData:       subMapData,
			RenderHeightMapType: subMapType,
			RenderHeightMapData: subMapData,
			Offset:              offset,
		}
	}

	serialisedSubChunk := s.encodeSubChunk(col.Chunk, int(ind))

	blockEntityData := encodeBlockEntityPayload(col.BlockEntities, &ind, col, false)

	rawPayload := make([]byte, 0, len(serialisedSubChunk)+len(blockEntityData))
	rawPayload = append(rawPayload, serialisedSubChunk...)
	rawPayload = append(rawPayload, blockEntityData...)
	entry := protocol.SubChunkEntry{
		Result:              protocol.SubChunkResultSuccess,
		RawPayload:          protocol.Option(rawPayload),
		HeightMapType:       subMapType,
		HeightMapData:       subMapData,
		RenderHeightMapType: subMapType,
		RenderHeightMapData: subMapData,
		Offset:              offset,
	}
	if s.conn.ClientCacheEnabled() {
		if hash := xxhash.Sum64(serialisedSubChunk); s.trackBlob(hash, serialisedSubChunk) {
			transaction[hash] = struct{}{}

			entry.BlobHash = protocol.Option(hash)
			entry.RawPayload = protocol.Option(blockEntityData)
		}
	}
	return entry
}

// dimensionID returns the dimension ID of the world that the session is in.
func (s *Session) dimensionID(dim world.Dimension) int32 {
	d, _ := world.DimensionID(dim)
	return int32(d)
}

// sendBlobHashes sends chunk blob hashes of the data of the chunk and stores the data in a map of blobs. Only
// data that the client doesn't yet have will be sent over the network.
func (s *Session) sendBlobHashes(pos world.ChunkPos, dim world.Dimension, c *chunk.Chunk, blockEntities map[cube.Pos]world.Block) {
	if subChunkRequests {
		biomes := s.encodeBiomes(c)
		if hash := xxhash.Sum64(biomes); s.trackBlob(hash, biomes) {
			s.writePacket(&packet.LevelChunk{
				Dimension:     s.dimensionID(dim),
				SubChunkCount: 0,
				Position:      protocol.ChunkPos(pos),
				SubChunkLimit: protocol.Option(int32(c.HighestFilledSubChunk())),
				BlobHashes:    []uint64{hash},
				RawPayload:    []byte{0},
				CacheEnabled:  true,
			})
			return
		}
	}

	var (
		data   = s.encodeChunk(c)
		count  = uint32(len(data.SubChunks))
		blobs  = append(data.SubChunks, data.Biomes)
		hashes = make([]uint64, len(blobs))
		m      = make(map[uint64]struct{}, len(blobs))
	)
	for i, blob := range blobs {
		h := xxhash.Sum64(blob)
		hashes[i], m[h] = h, struct{}{}
	}

	s.blobMu.Lock()
	s.openChunkTransactions = append(s.openChunkTransactions, m)
	if l := len(s.blobs); l > 4096 {
		s.blobMu.Unlock()
		s.conf.Log.Error("too many blobs pending", "n", l)
		return
	}
	for i := range hashes {
		s.blobs[hashes[i]] = blobs[i]
	}
	s.blobMu.Unlock()

	rawPayload := encodeBlockEntityPayload(blockEntities, nil, nil, true)

	s.writePacket(&packet.LevelChunk{
		Dimension:     s.dimensionID(dim),
		Position:      protocol.ChunkPos{pos.X(), pos.Z()},
		SubChunkCount: count,
		CacheEnabled:  true,
		BlobHashes:    hashes,
		RawPayload:    rawPayload,
	})
}

// sendNetworkChunk sends a network encoded chunk to the client.
func (s *Session) sendNetworkChunk(pos world.ChunkPos, dim world.Dimension, c *chunk.Chunk, blockEntities map[cube.Pos]world.Block) {
	if subChunkRequests {
		s.writePacket(&packet.LevelChunk{
			Dimension:     s.dimensionID(dim),
			SubChunkCount: 0,
			Position:      protocol.ChunkPos(pos),
			SubChunkLimit: protocol.Option(int32(c.HighestFilledSubChunk())),
			RawPayload:    append(s.encodeBiomes(c), 0),
		})
		return
	}

	data := s.encodeChunk(c)
	chunkBuf := bytes.NewBuffer(nil)
	for _, s := range data.SubChunks {
		_, _ = chunkBuf.Write(s)
	}
	_, _ = chunkBuf.Write(data.Biomes)

	_, _ = chunkBuf.Write(encodeBlockEntityPayload(blockEntities, nil, nil, true))

	s.writePacket(&packet.LevelChunk{
		Dimension:     s.dimensionID(dim),
		Position:      protocol.ChunkPos{pos.X(), pos.Z()},
		SubChunkCount: uint32(len(data.SubChunks)),
		RawPayload:    append([]byte(nil), chunkBuf.Bytes()...),
	})
}

// trackBlob attempts to track the given blob. If the player has too many pending blobs, it returns false and closes the
// connection.
func (s *Session) trackBlob(hash uint64, blob []byte) bool {
	s.blobMu.Lock()
	if l := len(s.blobs); l > 4096 {
		s.blobMu.Unlock()
		s.conf.Log.Error("too many blobs pending", "n", l)
		return false
	}
	s.blobs[hash] = blob
	s.blobMu.Unlock()
	return true
}
