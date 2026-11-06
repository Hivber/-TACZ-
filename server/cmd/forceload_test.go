package cmd

import "testing"

func TestBlockToChunkCoordinateUsesFloorForNegativeValues(t *testing.T) {
	tests := []struct {
		block int
		want  int32
	}{
		{block: -17, want: -2},
		{block: -16, want: -1},
		{block: -1, want: -1},
		{block: 0, want: 0},
		{block: 15, want: 0},
		{block: 16, want: 1},
	}
	for _, test := range tests {
		if got := blockToChunkCoordinate(test.block); got != test.want {
			t.Errorf("blockToChunkCoordinate(%d) = %d, want %d", test.block, got, test.want)
		}
	}
}

func TestForceloadChunkRange(t *testing.T) {
	positions, err := forceloadChunkRange(0, 0, 32, 32)
	if err != nil {
		t.Fatal(err)
	}
	if len(positions) != 9 {
		t.Fatalf("got %d chunks, want 9", len(positions))
	}
	if _, err := forceloadChunkRange(0, 0, 4096, 0); err == nil {
		t.Fatal("expected large range to be rejected")
	}
	if _, err := forceloadChunkRange(-30000001, 0, 0, 0); err == nil {
		t.Fatal("expected out-of-world coordinate to be rejected")
	}
}
