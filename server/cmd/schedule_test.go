package cmd

import (
	"testing"
	"time"
)

func TestParseScheduleDelay(t *testing.T) {
	tests := []struct {
		value string
		want  time.Duration
	}{
		{value: "10", want: 500 * time.Millisecond},
		{value: "20t", want: time.Second},
		{value: "2s", want: 2 * time.Second},
		{value: "1d", want: 24 * time.Hour},
	}
	for _, test := range tests {
		got, err := parseScheduleDelay(test.value)
		if err != nil {
			t.Errorf("parseScheduleDelay(%q): %v", test.value, err)
			continue
		}
		if got != test.want {
			t.Errorf("parseScheduleDelay(%q) = %s, want %s", test.value, got, test.want)
		}
	}
	for _, value := range []string{"", "0", "-1t", "1w", "31d", "1.5s"} {
		if _, err := parseScheduleDelay(value); err == nil {
			t.Errorf("parseScheduleDelay(%q) should fail", value)
		}
	}
}
