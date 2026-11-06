package cmd

import (
	"testing"
	"time"
)

func TestWorldBorderDuration(t *testing.T) {
	zero, err := worldBorderDuration(Optional[float64]{})
	if err != nil || zero != 0 {
		t.Fatalf("empty duration = %s, %v", zero, err)
	}
	value, err := worldBorderDuration(Optional[float64]{val: 2.5, set: true})
	if err != nil || value != 2500*time.Millisecond {
		t.Fatalf("2.5 second duration = %s, %v", value, err)
	}
	if _, err := worldBorderDuration(Optional[float64]{val: -1, set: true}); err == nil {
		t.Fatal("negative transition time should be rejected")
	}
}
