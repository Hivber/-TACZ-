package cmd

import "testing"

func TestParseCommandRandomRange(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		min     int64
		max     int64
		wantErr bool
	}{
		{name: "dice", raw: "1..6", min: 1, max: 6},
		{name: "exact value", raw: "0", min: 0, max: 0},
		{name: "open lower", raw: "..0", min: -1 << 31, max: 0},
		{name: "open upper", raw: "0..", min: 0, max: 1<<31 - 1},
		{name: "reversed", raw: "6..1", wantErr: true},
		{name: "too narrow", raw: "1..1", wantErr: true},
		{name: "bad integer", raw: "a..6", wantErr: true},
		{name: "too wide", raw: "..2147483647", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			minimum, maximum, err := parseCommandRandomRange(test.raw)
			if (err != nil) != test.wantErr {
				t.Fatalf("parseCommandRandomRange(%q) error = %v, wantErr %v", test.raw, err, test.wantErr)
			}
			if err == nil && (minimum != test.min || maximum != test.max) {
				t.Fatalf("parseCommandRandomRange(%q) = %d..%d, want %d..%d", test.raw, minimum, maximum, test.min, test.max)
			}
		})
	}
}
