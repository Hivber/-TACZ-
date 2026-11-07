package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type dataMergeBlockCommand struct {
	Merge SubCommand `cmd:"merge"`
	Block SubCommand `cmd:"block"`
	X     int
	Y     int
	Z     int
	Data  Varargs
}

func (dataMergeBlockCommand) Allow(src Source) bool { return commandOperator(src) }

func (c dataMergeBlockCommand) Run(_ Source, out *Output, tx *world.Tx) {
	if tx == nil {
		out.Error("/data merge block requires a world transaction.")
		return
	}
	pos := cube.Pos{c.X, c.Y, c.Z}
	current := tx.Block(pos)
	nbtBlock, ok := current.(world.NBTer)
	if !ok {
		out.Errorf("block at %v does not expose writable block-entity NBT", pos)
		return
	}
	value, err := parseCommandSNBT(string(c.Data))
	if err != nil {
		out.Error(err)
		return
	}
	compound, ok := value.(map[string]any)
	if !ok {
		out.Error("/data merge block expects an SNBT compound such as {Lock:\"vault\"}.")
		return
	}
	for _, reserved := range []string{"id", "x", "y", "z"} {
		if _, exists := compound[reserved]; exists {
			out.Errorf("block entity field %q cannot be changed", reserved)
			return
		}
	}
	data := nbtBlock.EncodeNBT()
	mergeCommandNBT(data, compound)
	if err := writeCommandBlockNBT(tx, pos, nbtBlock, data); err != nil {
		out.Error(err)
		return
	}
	out.Printf("Merged NBT data into block at %v.", pos)
}

type dataModifyBlockCommand struct {
	Modify SubCommand `cmd:"modify"`
	Block  SubCommand `cmd:"block"`
	X      int
	Y      int
	Z      int
	Path   string
	Set    SubCommand `cmd:"set"`
	Value  SubCommand `cmd:"value"`
	Data   Varargs
}

func (dataModifyBlockCommand) Allow(src Source) bool { return commandOperator(src) }

func (c dataModifyBlockCommand) Run(_ Source, out *Output, tx *world.Tx) {
	if tx == nil {
		out.Error("/data modify block requires a world transaction.")
		return
	}
	pos := cube.Pos{c.X, c.Y, c.Z}
	nbtBlock, ok := tx.Block(pos).(world.NBTer)
	if !ok {
		out.Errorf("block at %v does not expose writable block-entity NBT", pos)
		return
	}
	value, err := parseCommandSNBT(string(c.Data))
	if err != nil {
		out.Error(err)
		return
	}
	data := nbtBlock.EncodeNBT()
	if err := setCommandNBTPath(data, c.Path, value); err != nil {
		out.Error(err)
		return
	}
	if err := writeCommandBlockNBT(tx, pos, nbtBlock, data); err != nil {
		out.Error(err)
		return
	}
	out.Printf("Modified NBT path %s on block at %v.", c.Path, pos)
}

type dataRemoveBlockCommand struct {
	Remove SubCommand `cmd:"remove"`
	Block  SubCommand `cmd:"block"`
	X      int
	Y      int
	Z      int
	Path   string
}

func (dataRemoveBlockCommand) Allow(src Source) bool { return commandOperator(src) }

func (c dataRemoveBlockCommand) Run(_ Source, out *Output, tx *world.Tx) {
	if tx == nil {
		out.Error("/data remove block requires a world transaction.")
		return
	}
	pos := cube.Pos{c.X, c.Y, c.Z}
	nbtBlock, ok := tx.Block(pos).(world.NBTer)
	if !ok {
		out.Errorf("block at %v does not expose writable block-entity NBT", pos)
		return
	}
	data := nbtBlock.EncodeNBT()
	if err := removeCommandNBTPath(data, c.Path); err != nil {
		out.Error(err)
		return
	}
	if err := writeCommandBlockNBT(tx, pos, nbtBlock, data); err != nil {
		out.Error(err)
		return
	}
	out.Printf("Removed NBT path %s from block at %v.", c.Path, pos)
}

func writeCommandBlockNBT(tx *world.Tx, pos cube.Pos, original world.NBTer, data map[string]any) error {
	decoded, ok := original.DecodeNBT(data).(world.Block)
	if !ok {
		return fmt.Errorf("decoding block entity NBT at %v did not return a block", pos)
	}
	tx.SetBlockEntity(pos, decoded)
	return nil
}

func mergeCommandNBT(destination, source map[string]any) {
	for key, value := range source {
		if sourceCompound, ok := value.(map[string]any); ok {
			if destinationCompound, exists := destination[key].(map[string]any); exists {
				mergeCommandNBT(destinationCompound, sourceCompound)
				continue
			}
		}
		destination[key] = value
	}
}

type commandNBTPathPart struct {
	key   string
	index int
	array bool
}

func formatCommandNBTPath(parts []commandNBTPathPart) string {
	var out strings.Builder
	for _, part := range parts {
		if part.array {
			out.WriteString("[")
			out.WriteString(strconv.Itoa(part.index))
			out.WriteString("]")
			continue
		}
		if out.Len() > 0 {
			out.WriteString(".")
		}
		out.WriteString(part.key)
	}
	return out.String()
}

func parseCommandNBTPath(raw string) ([]commandNBTPathPart, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("NBT path cannot be empty")
	}
	parts := make([]commandNBTPathPart, 0, 4)
	for i := 0; i < len(raw); {
		if raw[i] == '.' {
			return nil, fmt.Errorf("invalid NBT path %q", raw)
		}
		if raw[i] == '[' {
			end := strings.IndexByte(raw[i+1:], ']')
			if end < 0 {
				return nil, fmt.Errorf("invalid list index in NBT path %q", raw)
			}
			end += i + 1
			index, err := strconv.Atoi(raw[i+1 : end])
			if err != nil || index < 0 {
				return nil, fmt.Errorf("invalid list index in NBT path %q", raw)
			}
			parts = append(parts, commandNBTPathPart{index: index, array: true})
			i = end + 1
		} else {
			start := i
			for i < len(raw) && raw[i] != '.' && raw[i] != '[' {
				i++
			}
			if start == i {
				return nil, fmt.Errorf("invalid NBT path %q", raw)
			}
			parts = append(parts, commandNBTPathPart{key: raw[start:i]})
		}
		if i < len(raw) && raw[i] == '.' {
			i++
			if i == len(raw) {
				return nil, fmt.Errorf("invalid NBT path %q", raw)
			}
		} else if i < len(raw) && raw[i] != '[' {
			return nil, fmt.Errorf("invalid NBT path %q", raw)
		}
	}
	return parts, nil
}

func commandNBTPathParent(root map[string]any, raw string) (any, commandNBTPathPart, error) {
	parts, err := parseCommandNBTPath(raw)
	if err != nil {
		return nil, commandNBTPathPart{}, err
	}
	current := any(root)
	for _, part := range parts[:len(parts)-1] {
		switch {
		case part.array:
			list, ok := current.([]any)
			if !ok || part.index >= len(list) {
				return nil, commandNBTPathPart{}, fmt.Errorf("NBT path %q has an invalid list index", raw)
			}
			current = list[part.index]
		default:
			compound, ok := current.(map[string]any)
			if !ok {
				return nil, commandNBTPathPart{}, fmt.Errorf("NBT path %q expected a compound", raw)
			}
			next, exists := compound[part.key]
			if !exists {
				return nil, commandNBTPathPart{}, fmt.Errorf("NBT path %q was not found", raw)
			}
			current = next
		}
	}
	return current, parts[len(parts)-1], nil
}

func setCommandNBTPath(root map[string]any, raw string, value any) error {
	if strings.TrimSpace(raw) == "id" || strings.TrimSpace(raw) == "x" || strings.TrimSpace(raw) == "y" || strings.TrimSpace(raw) == "z" {
		return fmt.Errorf("block entity identity and position fields cannot be modified")
	}
	parent, last, err := commandNBTPathParent(root, raw)
	if err != nil {
		return err
	}
	if last.array {
		list, ok := parent.([]any)
		if !ok || last.index >= len(list) {
			return fmt.Errorf("NBT path %q has an invalid list index", raw)
		}
		list[last.index] = value
		return nil
	}
	compound, ok := parent.(map[string]any)
	if !ok {
		return fmt.Errorf("NBT path %q expected a compound", raw)
	}
	compound[last.key] = value
	return nil
}

func removeCommandNBTPath(root map[string]any, raw string) error {
	if strings.TrimSpace(raw) == "id" || strings.TrimSpace(raw) == "x" || strings.TrimSpace(raw) == "y" || strings.TrimSpace(raw) == "z" {
		return fmt.Errorf("block entity identity and position fields cannot be removed")
	}
	parts, err := parseCommandNBTPath(raw)
	if err != nil {
		return err
	}
	parent, last, err := commandNBTPathParent(root, raw)
	if err != nil {
		return err
	}
	if last.array {
		list, ok := parent.([]any)
		if !ok || last.index >= len(list) {
			return fmt.Errorf("NBT path %q has an invalid list index", raw)
		}
		if len(parts) == 1 {
			return fmt.Errorf("cannot remove an element from the root NBT value")
		}
		prefix := formatCommandNBTPath(parts[:len(parts)-1])
		updated := append(append([]any(nil), list[:last.index]...), list[last.index+1:]...)
		return setCommandNBTPath(root, prefix, updated)
	}
	compound, ok := parent.(map[string]any)
	if !ok {
		return fmt.Errorf("NBT path %q expected a compound", raw)
	}
	if _, exists := compound[last.key]; !exists {
		return fmt.Errorf("NBT path %q was not found", raw)
	}
	if last.key == "id" {
		return fmt.Errorf("the block entity id cannot be removed")
	}
	delete(compound, last.key)
	return nil
}

// parseCommandSNBT supports standard compounds, lists, strings and scalar NBT
// values. Typed arrays and the full Java SNBT grammar are intentionally excluded.
func parseCommandSNBT(raw string) (any, error) {
	parser := commandSNBTParser{runes: []rune(strings.TrimSpace(raw))}
	value, err := parser.value()
	if err != nil {
		return nil, err
	}
	parser.space()
	if parser.pos != len(parser.runes) {
		return nil, fmt.Errorf("unexpected trailing data in SNBT value")
	}
	return value, nil
}

type commandSNBTParser struct {
	runes []rune
	pos   int
}

func (p *commandSNBTParser) space() {
	for p.pos < len(p.runes) && unicode.IsSpace(p.runes[p.pos]) {
		p.pos++
	}
}

func (p *commandSNBTParser) value() (any, error) {
	p.space()
	if p.pos >= len(p.runes) {
		return nil, fmt.Errorf("SNBT value is empty")
	}
	switch p.runes[p.pos] {
	case '{':
		return p.compound()
	case '[':
		return p.list()
	case '"', '\'':
		return p.quoted()
	default:
		return p.scalar()
	}
}

func (p *commandSNBTParser) compound() (map[string]any, error) {
	p.pos++
	out := map[string]any{}
	p.space()
	if p.take('}') {
		return out, nil
	}
	for {
		p.space()
		key, err := p.key()
		if err != nil {
			return nil, err
		}
		p.space()
		if !p.take(':') {
			return nil, fmt.Errorf("expected ':' after SNBT compound key %q", key)
		}
		value, err := p.value()
		if err != nil {
			return nil, err
		}
		out[key] = value
		p.space()
		if p.take('}') {
			return out, nil
		}
		if !p.take(',') {
			return nil, fmt.Errorf("expected ',' or '}' in SNBT compound")
		}
	}
}

func (p *commandSNBTParser) key() (string, error) {
	if p.pos >= len(p.runes) {
		return "", fmt.Errorf("expected SNBT compound key")
	}
	if p.runes[p.pos] == '"' || p.runes[p.pos] == '\'' {
		return p.quotedString()
	}
	start := p.pos
	for p.pos < len(p.runes) && p.runes[p.pos] != ':' && !unicode.IsSpace(p.runes[p.pos]) {
		if p.runes[p.pos] == ',' || p.runes[p.pos] == '}' {
			break
		}
		p.pos++
	}
	if start == p.pos {
		return "", fmt.Errorf("expected SNBT compound key")
	}
	return string(p.runes[start:p.pos]), nil
}

func (p *commandSNBTParser) list() ([]any, error) {
	p.pos++
	out := make([]any, 0)
	p.space()
	if p.take(']') {
		return out, nil
	}
	for {
		value, err := p.value()
		if err != nil {
			return nil, err
		}
		out = append(out, value)
		p.space()
		if p.take(']') {
			return out, nil
		}
		if !p.take(',') {
			return nil, fmt.Errorf("expected ',' or ']' in SNBT list")
		}
	}
}

func (p *commandSNBTParser) quoted() (any, error) {
	value, err := p.quotedString()
	return value, err
}

func (p *commandSNBTParser) quotedString() (string, error) {
	quote := p.runes[p.pos]
	p.pos++
	var out strings.Builder
	for p.pos < len(p.runes) {
		char := p.runes[p.pos]
		p.pos++
		if char == quote {
			return out.String(), nil
		}
		if char == '\\' {
			if p.pos >= len(p.runes) {
				break
			}
			escaped := p.runes[p.pos]
			p.pos++
			switch escaped {
			case 'n':
				out.WriteRune('\n')
			case 'r':
				out.WriteRune('\r')
			case 't':
				out.WriteRune('\t')
			default:
				out.WriteRune(escaped)
			}
			continue
		}
		out.WriteRune(char)
	}
	return "", fmt.Errorf("unterminated quoted SNBT string")
}

func (p *commandSNBTParser) scalar() (any, error) {
	start := p.pos
	for p.pos < len(p.runes) {
		char := p.runes[p.pos]
		if char == ',' || char == '}' || char == ']' || unicode.IsSpace(char) {
			break
		}
		p.pos++
	}
	if start == p.pos {
		return nil, fmt.Errorf("expected SNBT scalar value")
	}
	raw := string(p.runes[start:p.pos])
	lower := strings.ToLower(raw)
	if lower == "true" {
		return true, nil
	}
	if lower == "false" {
		return false, nil
	}
	if len(raw) > 1 {
		suffix := raw[len(raw)-1]
		number := raw[:len(raw)-1]
		switch suffix {
		case 'b', 'B':
			if value, err := strconv.ParseInt(number, 10, 8); err == nil {
				return int8(value), nil
			}
		case 's', 'S':
			if value, err := strconv.ParseInt(number, 10, 16); err == nil {
				return int16(value), nil
			}
		case 'l', 'L':
			if value, err := strconv.ParseInt(number, 10, 64); err == nil {
				return value, nil
			}
		case 'f', 'F':
			if value, err := strconv.ParseFloat(number, 32); err == nil {
				return float32(value), nil
			}
		case 'd', 'D':
			if value, err := strconv.ParseFloat(number, 64); err == nil {
				return value, nil
			}
		}
	}
	if value, err := strconv.ParseInt(raw, 10, 32); err == nil {
		return int32(value), nil
	}
	if strings.ContainsAny(raw, ".eE") {
		if value, err := strconv.ParseFloat(raw, 64); err == nil {
			return value, nil
		}
	}
	return raw, nil
}

func (p *commandSNBTParser) take(expected rune) bool {
	if p.pos >= len(p.runes) || p.runes[p.pos] != expected {
		return false
	}
	p.pos++
	return true
}
