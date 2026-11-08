package world

import (
	"strconv"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
)

func TestRedstoneDirtyIndexMatchesLinearNearest(t *testing.T) {
	for _, size := range []int{redstoneDirtyIndexThreshold - 1, redstoneDirtyIndexThreshold, 127, 512, 2048} {
		dirty := make(map[cube.Pos]redstoneDirty, size)
		for i := 0; i < size; i++ {
			pos := cube.Pos{(i*37)%211 - 105, (i*19)%97 - 48, (i*53)%257 - 128}
			dirty[pos] = redstoneDirty{
				changed: pos, hasChanged: true, source: pos, hasSource: true,
				cause: RedstoneUpdateCauseBlockUpdate,
			}
		}
		index := newRedstoneDirtyIndex(dirty)
		for i := 0; i < 200; i++ {
			pos := cube.Pos{i*17%401 - 200, i*31%181 - 90, i*43%503 - 251}
			want := redstoneDirtyContext(dirty, pos)
			var got redstoneDirty
			if index == nil {
				got = redstoneDirtyContext(dirty, pos)
			} else {
				got, _ = index.nearest(pos)
			}
			if got != want {
				t.Fatalf("size %d, query %v: indexed context %+v, linear context %+v", size, pos, got, want)
			}
		}
	}
}

func TestRedstoneDirtyIndexTieBreaksByBlockPosition(t *testing.T) {
	left := cube.Pos{-1, 64, 0}
	right := cube.Pos{1, 64, 0}
	dirty := map[cube.Pos]redstoneDirty{
		left:  {changed: left, hasChanged: true},
		right: {changed: right, hasChanged: true},
	}
	index := newRedstoneDirtyIndex(expandDirtyForIndexTest(dirty))
	got, _ := index.nearest(cube.Pos{0, 64, 0})
	want := redstoneDirtyContext(expandDirtyForIndexTest(dirty), cube.Pos{0, 64, 0})
	if got != want || got.changed != left {
		t.Fatalf("tie context = %+v, want lexicographically first %+v", got, want)
	}
}

func expandDirtyForIndexTest(dirty map[cube.Pos]redstoneDirty) map[cube.Pos]redstoneDirty {
	for i := 0; len(dirty) < redstoneDirtyIndexThreshold; i++ {
		pos := cube.Pos{1000 + i*3, 1000, -1000 - i*5}
		dirty[pos] = redstoneDirty{changed: pos, hasChanged: true}
	}
	return dirty
}

func BenchmarkRedstoneDirtyContext(b *testing.B) {
	for _, size := range []int{64, 256, 1024, 4096} {
		dirty := make(map[cube.Pos]redstoneDirty, size)
		for i := 0; i < size; i++ {
			pos := cube.Pos{i * 3, (i * 7) % 128, i * 5}
			dirty[pos] = redstoneDirty{changed: pos, hasChanged: true}
		}
		queries := make([]cube.Pos, 128)
		for i := range queries {
			queries[i] = cube.Pos{i*29 + 1, (i * 11) % 128, i*31 + 2}
		}
		b.Run("linear/"+strconv.Itoa(size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = redstoneDirtyContext(dirty, queries[i%len(queries)])
			}
		})
		index := newRedstoneDirtyIndex(dirty)
		b.Run("indexed/"+strconv.Itoa(size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = index.nearest(queries[i%len(queries)])
			}
		})
	}
}
