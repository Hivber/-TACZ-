package server

import "testing"

func TestNormaliseChunkRadius(t *testing.T) {
	tests := []struct {
		configured int
		want       int
	}{
		{configured: 0, want: defaultChunkRadius},
		{configured: 8, want: 8},
		{configured: 10, want: 10},
		{configured: 32, want: maximumChunkRadius},
	}
	for _, test := range tests {
		if got := normaliseChunkRadius(test.configured, nil); got != test.want {
			t.Errorf("normaliseChunkRadius(%d) = %d, want %d", test.configured, got, test.want)
		}
	}
}

func TestNormaliseChunkLoadWorkers(t *testing.T) {
	tests := []struct {
		configured int
		want       int
	}{
		{configured: 0, want: 1},
		{configured: 1, want: 1},
		{configured: 4, want: 4},
		{configured: 32, want: maximumChunkLoadWorkers},
	}
	for _, test := range tests {
		if got := normaliseChunkLoadWorkers(test.configured); got != test.want {
			t.Errorf("normaliseChunkLoadWorkers(%d) = %d, want %d", test.configured, got, test.want)
		}
	}
}
