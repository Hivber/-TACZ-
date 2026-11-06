package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

const (
	maxScheduledFunctionsPerName = 256
	maxScheduleDelay             = 30 * 24 * time.Hour
)

type scheduleMode string

func (scheduleMode) Type() string { return "ScheduleMode" }
func (scheduleMode) Options(Source) []string {
	return []string{"append", "replace"}
}

type scheduledFunctionSource struct {
	world    *world.World
	position mgl64.Vec3
}

func (s *scheduledFunctionSource) Position() mgl64.Vec3 { return s.position }
func (s *scheduledFunctionSource) World() *world.World  { return s.world }
func (*scheduledFunctionSource) Operator() bool         { return true }

func (s *scheduledFunctionSource) SendCommandOutput(output *Output) {
	for _, err := range output.Errors() {
		slog.Warn("scheduled function command failed", "error", err)
	}
}

var scheduledFunctionTasks = struct {
	sync.Mutex
	worlds map[*world.World]map[string][]*world.Task
}{
	worlds: make(map[*world.World]map[string][]*world.Task),
}

func init() {
	Register(New("schedule", "Schedules or cancels execution of a function.", nil, scheduleFunctionCommand{}, scheduleClearCommand{}))
}

type scheduleFunctionCommand struct {
	Function SubCommand
	Name     string
	Delay    string
	Mode     Optional[scheduleMode]
}

func (scheduleFunctionCommand) Allow(src Source) bool { return commandOperator(src) }

func (c scheduleFunctionCommand) Run(src Source, out *Output, tx *world.Tx) {
	w := commandWorld(src, tx)
	if w == nil {
		out.Error("schedule requires a world-bound command source.")
		return
	}
	name, path, err := resolveFunctionFile(c.Name)
	if err != nil {
		out.Error(err)
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		out.Errorf("function %s: %v", name, err)
		return
	}
	if info.Size() > maxFunctionFileSize {
		out.Errorf("function %s exceeds the %d-byte file limit", name, maxFunctionFileSize)
		return
	}
	delay, err := parseScheduleDelay(c.Delay)
	if err != nil {
		out.Error(err)
		return
	}
	mode := c.Mode.LoadOr(scheduleMode("append"))
	if err := scheduleFunction(w, src.Position(), name, delay, mode); err != nil {
		out.Error(err)
		return
	}
	out.Printf("Scheduled function %s in %s (%s).", name, delay, mode)
}

type scheduleClearCommand struct {
	Clear SubCommand
	Name  Optional[string]
}

func (scheduleClearCommand) Allow(src Source) bool { return commandOperator(src) }

func (c scheduleClearCommand) Run(src Source, out *Output, tx *world.Tx) {
	w := commandWorld(src, tx)
	if w == nil {
		out.Error("schedule requires a world-bound command source.")
		return
	}
	name := "*"
	if supplied, ok := c.Name.Load(); ok {
		if supplied != "*" {
			canonical, _, err := resolveFunctionFile(supplied)
			if err != nil {
				out.Error(err)
				return
			}
			name = canonical
		}
	}
	cancelled := clearScheduledFunctions(w, name)
	out.Printf("Cancelled %d scheduled function task(s).", cancelled)
}

func parseScheduleDelay(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return 0, fmt.Errorf("schedule delay is empty")
	}
	unit := byte('t')
	last := raw[len(raw)-1]
	if last < '0' || last > '9' {
		unit = last
		raw = raw[:len(raw)-1]
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 1 {
		return 0, fmt.Errorf("invalid schedule delay %q", raw)
	}
	multiplier := time.Millisecond * 50
	switch unit {
	case 't':
	case 's':
		multiplier = time.Second
	case 'd':
		multiplier = 24 * time.Hour
	default:
		return 0, fmt.Errorf("unsupported schedule time unit %q; use t, s or d", string(unit))
	}
	if value > int64(maxScheduleDelay/multiplier) {
		return 0, fmt.Errorf("schedule delay may not exceed %s", maxScheduleDelay)
	}
	return time.Duration(value) * multiplier, nil
}

func scheduleFunction(w *world.World, position mgl64.Vec3, name string, delay time.Duration, mode scheduleMode) error {
	_, path, err := resolveFunctionFile(name)
	if err != nil {
		return err
	}
	if delay <= 0 || delay > maxScheduleDelay {
		return fmt.Errorf("schedule delay must be between one tick and %s", maxScheduleDelay)
	}
	if mode != "append" && mode != "replace" {
		return fmt.Errorf("invalid schedule mode %q", mode)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("function %s: %w", name, err)
	}
	if info.Size() > maxFunctionFileSize {
		return fmt.Errorf("function %s exceeds the %d-byte file limit", name, maxFunctionFileSize)
	}

	var cancel []*world.Task
	scheduledFunctionTasks.Lock()
	byName := scheduledFunctionTasks.worlds[w]
	if byName == nil {
		byName = make(map[string][]*world.Task)
		scheduledFunctionTasks.worlds[w] = byName
	}
	pending := byName[name]
	if mode == "replace" {
		cancel = append(cancel, pending...)
		delete(byName, name)
		pending = nil
	}
	if len(pending) >= maxScheduledFunctionsPerName {
		scheduledFunctionTasks.Unlock()
		return fmt.Errorf("function %s already has %d pending executions", name, maxScheduledFunctionsPerName)
	}

	source := &scheduledFunctionSource{world: w, position: position}
	var task *world.Task
	task = w.DoAfter(delay, func(tx *world.Tx) {
		defer removeScheduledFunctionTask(w, name, task)
		if err := executeFunction(name, source, tx, 0, &functionExecution{}); err != nil {
			slog.Error("scheduled function failed", "function", name, "error", err)
		}
	})
	byName[name] = append(pending, task)
	scheduledFunctionTasks.Unlock()

	for _, previous := range cancel {
		previous.Cancel()
	}
	task.OnDone(func(error) {
		removeScheduledFunctionTask(w, name, task)
	})
	return nil
}

func removeScheduledFunctionTask(w *world.World, name string, target *world.Task) {
	scheduledFunctionTasks.Lock()
	defer scheduledFunctionTasks.Unlock()
	byName := scheduledFunctionTasks.worlds[w]
	if byName == nil {
		return
	}
	tasks := byName[name]
	for i, task := range tasks {
		if task == target {
			tasks = append(tasks[:i], tasks[i+1:]...)
			break
		}
	}
	if len(tasks) == 0 {
		delete(byName, name)
	} else {
		byName[name] = tasks
	}
	if len(byName) == 0 {
		delete(scheduledFunctionTasks.worlds, w)
	}
}

func clearScheduledFunctions(w *world.World, name string) int {
	scheduledFunctionTasks.Lock()
	byName := scheduledFunctionTasks.worlds[w]
	var tasks []*world.Task
	if name == "*" {
		for _, pending := range byName {
			tasks = append(tasks, pending...)
		}
		delete(scheduledFunctionTasks.worlds, w)
	} else if byName != nil {
		tasks = append(tasks, byName[name]...)
		delete(byName, name)
		if len(byName) == 0 {
			delete(scheduledFunctionTasks.worlds, w)
		}
	}
	scheduledFunctionTasks.Unlock()

	cancelled := 0
	for _, task := range tasks {
		if task.Cancel() {
			cancelled++
		}
	}
	return cancelled
}
