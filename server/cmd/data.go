package cmd

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

func init() {
	Register(New("data", "Reads NBT data from supported blocks.", nil, dataGetBlockCommand{}))
}

type dataGetBlockCommand struct {
	Get   SubCommand
	Block SubCommand
	X     int
	Y     int
	Z     int
	Path  Optional[string]
	Scale Optional[float64]
}

func (dataGetBlockCommand) Allow(src Source) bool { return commandOperator(src) }

func (c dataGetBlockCommand) Run(_ Source, out *Output, tx *world.Tx) {
	pos := cube.Pos{c.X, c.Y, c.Z}
	block := tx.Block(pos)
	nbtBlock, ok := block.(world.NBTer)
	if !ok {
		out.Errorf("block at %v does not expose block-entity NBT", pos)
		return
	}
	var value any = nbtBlock.EncodeNBT()
	if path, supplied := c.Path.Load(); supplied && path != "" {
		var err error
		value, err = lookupNBTPath(value, path)
		if err != nil {
			out.Error(err)
			return
		}
	}
	if scale, supplied := c.Scale.Load(); supplied {
		scaled, err := scaleNBTValue(value, scale)
		if err != nil {
			out.Error(err)
			return
		}
		value = scaled
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		out.Errorf("could not format NBT data: %v", err)
		return
	}
	out.Printf("Block data at %v: %s", pos, encoded)
}

func lookupNBTPath(root any, path string) (any, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return root, nil
	}
	current := root
	for _, part := range strings.Split(path, ".") {
		if part == "" {
			return nil, fmt.Errorf("invalid NBT path %q", path)
		}
		name := part
		if bracket := strings.IndexByte(part, '['); bracket >= 0 {
			name = part[:bracket]
		}
		if name != "" {
			object, ok := current.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("NBT path %q expected a compound before %q", path, name)
			}
			var exists bool
			current, exists = object[name]
			if !exists {
				return nil, fmt.Errorf("NBT path %q was not found", path)
			}
		}
		rest := part[len(name):]
		for rest != "" {
			if rest[0] != '[' {
				return nil, fmt.Errorf("invalid NBT path %q", path)
			}
			end := strings.IndexByte(rest, ']')
			if end < 0 {
				return nil, fmt.Errorf("invalid NBT list index in path %q", path)
			}
			index, err := strconv.Atoi(rest[1:end])
			if err != nil || index < 0 {
				return nil, fmt.Errorf("invalid NBT list index in path %q", path)
			}
			list := reflect.ValueOf(current)
			if !list.IsValid() || (list.Kind() != reflect.Slice && list.Kind() != reflect.Array) || index >= list.Len() {
				return nil, fmt.Errorf("NBT list index %d is out of range in path %q", index, path)
			}
			current = list.Index(index).Interface()
			rest = rest[end+1:]
		}
	}
	return current, nil
}

func scaleNBTValue(value any, scale float64) (any, error) {
	switch number := value.(type) {
	case int:
		return float64(number) * scale, nil
	case int8:
		return float64(number) * scale, nil
	case int16:
		return float64(number) * scale, nil
	case int32:
		return float64(number) * scale, nil
	case int64:
		return float64(number) * scale, nil
	case uint8:
		return float64(number) * scale, nil
	case uint16:
		return float64(number) * scale, nil
	case uint32:
		return float64(number) * scale, nil
	case uint64:
		return float64(number) * scale, nil
	case float32:
		return float64(number) * scale, nil
	case float64:
		return number * scale, nil
	default:
		return nil, fmt.Errorf("NBT scale can only be applied to a numeric value")
	}
}
