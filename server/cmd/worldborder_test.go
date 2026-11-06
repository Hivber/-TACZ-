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
	value, err := worldBorderDuration(Optional[float64]{}.Set(2.5))
	if err != nil || value != 2500*time.Millisecond {
		t.Fatalf("2.5 second duration = %s, %v", value, err)
	}
	if _, err := worldBorderDuration(Optional[float64]{}.Set(-1)); err == nil {
		t.Fatal("negative transition time should be rejected")
	}
}
