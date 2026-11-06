package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/df-mc/dragonfly/server/world"
)

const (
	maxFunctionDepth       = 32
	maxFunctionCommandRuns = 10000
	maxFunctionFileSize    = 1 << 20
)

func init() {
	Register(New("function", "Executes a function file from the functions directory.", nil, functionCommand{}))
}

type functionCommand struct {
	Name string
}

func (functionCommand) Allow(src Source) bool { return commandOperator(src) }

func (c functionCommand) Run(src Source, out *Output, tx *world.Tx) {
	name, _, err := resolveFunctionFile(c.Name)
	if err != nil {
		out.Error(err)
		return
	}
	if tx != nil {
		if err := executeFunction(name, src, tx, 0, &functionExecution{}); err != nil {
			out.Error(err)
			return
		}
		out.Printf("Executed function %s.", name)
		return
	}
	w := commandWorld(src, nil)
	if w == nil {
		out.Error("function requires a world-bound command source.")
		return
	}
	task := w.Do(func(tx *world.Tx) {
		if err := executeFunction(name, src, tx, 0, &functionExecution{}); err != nil {
			failure := &Output{}
			failure.Error(err)
			src.SendCommandOutput(failure)
		}
	})
	task.OnDone(func(err error) {
		if err != nil {
			failure := &Output{}
			failure.Error(err)
			src.SendCommandOutput(failure)
		}
	})
	out.Printf("Queued function %s.", name)
}

type functionExecution struct {
	commands int
	stack    []string
}

func executeFunction(name string, src Source, tx *world.Tx, depth int, state *functionExecution) error {
	if depth >= maxFunctionDepth {
		return fmt.Errorf("function nesting exceeds %d levels", maxFunctionDepth)
	}
	canonical, path, err := resolveFunctionFile(name)
	if err != nil {
		return err
	}
	for _, active := range state.stack {
		if active == canonical {
			return fmt.Errorf("recursive function call detected: %s", canonical)
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("function %s: %w", canonical, err)
	}
	if info.Size() > maxFunctionFileSize {
		return fmt.Errorf("function %s exceeds the %d-byte file limit", canonical, maxFunctionFileSize)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read function %s: %w", canonical, err)
	}

	state.stack = append(state.stack, canonical)
	defer func() { state.stack = state.stack[:len(state.stack)-1] }()

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 4096), maxFunctionFileSize)
	lineNumber := 0
	var firstNestedError error
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		state.commands++
		if state.commands > maxFunctionCommandRuns {
			return fmt.Errorf("function execution exceeds %d commands", maxFunctionCommandRuns)
		}
		line = strings.TrimPrefix(line, "/")
		commandName, args := splitFunctionCommand(line)
		if strings.EqualFold(commandName, "function") {
			nestedName := strings.TrimSpace(args)
			if nestedName == "" || strings.ContainsAny(nestedName, " 	") {
				if firstNestedError == nil {
					firstNestedError = fmt.Errorf("%s:%d: nested function expects one function name", canonical, lineNumber)
				}
				continue
			}
			if err := executeFunction(nestedName, src, tx, depth+1, state); err != nil && firstNestedError == nil {
				firstNestedError = fmt.Errorf("%s:%d: %w", canonical, lineNumber, err)
			}
			continue
		}
		command, ok := ByAlias(strings.ToLower(commandName))
		if !ok {
			failure := &Output{}
			failure.Errort(MessageUnknown, commandName)
			src.SendCommandOutput(failure)
			continue
		}
		command.Execute(args, src, tx)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read function %s: %w", canonical, err)
	}
	return firstNestedError
}

func splitFunctionCommand(line string) (name, args string) {
	if i := strings.IndexAny(line, " 	"); i >= 0 {
		return line[:i], strings.TrimSpace(line[i+1:])
	}
	return line, ""
}

func resolveFunctionFile(raw string) (canonical, path string, err error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, ".mcfunction")
	if raw == "" || len(raw) > 256 || strings.Count(raw, ":") > 1 || strings.Contains(raw, "\\") {
		return "", "", fmt.Errorf("invalid function name %q", raw)
	}
	namespace, resourcePath, hasNamespace := strings.Cut(raw, ":")
	if !hasNamespace {
		namespace, resourcePath = "minecraft", namespace
	}
	if !validFunctionNamePart(namespace, false) || resourcePath == "" {
		return "", "", fmt.Errorf("invalid function resource location %q", raw)
	}
	for _, part := range strings.Split(resourcePath, "/") {
		if part == "" || part == "." || part == ".." || !validFunctionNamePart(part, true) {
			return "", "", fmt.Errorf("invalid function resource location %q", raw)
		}
	}
	canonical = namespace + ":" + resourcePath
	path = filepath.Join("functions", namespace, filepath.FromSlash(resourcePath)+".mcfunction")
	return canonical, path, nil
}

func validFunctionNamePart(value string, allowDot bool) bool {
	for _, r := range value {
		valid := r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-'
		if allowDot && r == '.' {
			valid = true
		}
		if !valid {
			return false
		}
	}
	return value != ""
}
