package cmd

import (
	"errors"
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/df-mc/dragonfly/server/world"
)

type randomMode string

func (randomMode) Type() string            { return "RandomMode" }
func (randomMode) Options(Source) []string { return []string{"value", "roll"} }

type randomCommand struct {
	Mode  randomMode
	Range string
}

// Run implements the Java-style /random value|roll <range> forms. Named random
// sequences and /random reset are intentionally not advertised until the
// server has a world-persistent random-sequence store.
func (c randomCommand) Run(_ Source, out *Output, tx *world.Tx) {
	minimum, maximum, err := parseCommandRandomRange(c.Range)
	if err != nil {
		out.Error(err)
		return
	}

	result := minimum
	if maximum > minimum {
		result += rand.Int64N(maximum - minimum + 1)
	}
	if c.Mode == "roll" && tx != nil {
		for p := range tx.Players() {
			if receiver, ok := p.(interface{ Message(...any) }); ok {
				receiver.Message("Rolled", result, "from range", c.Range)
			}
		}
	}
	out.Printf("Random value: %d", result)
}

// parseCommandRandomRange parses Java's inclusive int_range notation. A lone
// integer represents that exact value; open-ended ranges use signed 32-bit
// limits just like the Java command argument.
func parseCommandRandomRange(raw string) (int64, int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, 0, errors.New("random range cannot be empty")
	}
	if !strings.Contains(raw, "..") {
		value, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return 0, 0, errors.New("random range must contain signed 32-bit integers")
		}
		return value, value, nil
	}
	if strings.Count(raw, "..") != 1 {
		return 0, 0, errors.New("random range must use the form min..max")
	}
	left, right, _ := strings.Cut(raw, "..")
	minimum, maximum := int64(-1<<31), int64(1<<31-1)
	if left != "" {
		value, err := strconv.ParseInt(left, 10, 32)
		if err != nil {
			return 0, 0, errors.New("random range minimum must be a signed 32-bit integer")
		}
		minimum = value
	}
	if right != "" {
		value, err := strconv.ParseInt(right, 10, 32)
		if err != nil {
			return 0, 0, errors.New("random range maximum must be a signed 32-bit integer")
		}
		maximum = value
	}
	if minimum > maximum {
		return 0, 0, errors.New("random range minimum cannot exceed its maximum")
	}
	// The full signed 32-bit interval has a width of 2^32, which still
	// fits in int64 and is accepted by rand.Int64N. Equal endpoints are valid.
	return minimum, maximum, nil
}
