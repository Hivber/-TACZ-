package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player/chat"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/pelletier/go-toml"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	if err := ensureRuntimeDirectories("."); err != nil {
		panic(err)
	}
	chat.Global.Subscribe(chat.StdoutSubscriber{})
	conf, err := readConfig(slog.Default())
	if err != nil {
		panic(err)
	}

	if conf.Features.JavaCompatibilityCommands {
		cmd.RegisterJavaCompatibilityCommands()
	}
	if conf.Features.PerformanceDiagnostics {
		cmd.RegisterPerformanceDiagnostics()
	}

	srv := conf.New()
	srv.CloseOnProgramEnd()
	if conf.Features.Pprof {
		startPprofServer(slog.Default(), conf.Features.PprofAddress)
	}

	operators, err := loadOperatorStore("operators.json", conf.OwnerXUID)
	if err != nil {
		panic(err)
	}
	registerOperatorCommands(srv, operators)

	srv.Listen()
	go runConsoleCommands(srv.World())
	for p := range srv.Accept() {
		if (conf.OwnerXUID != "" && p.XUID() == conf.OwnerXUID) || operators.isOperator(p.UUID()) {
			p.SetOperator(true)
		}
	}
}

// readConfig reads the configuration from the config.toml file, or creates the
// file if it does not yet exist.
func readConfig(log *slog.Logger) (server.Config, error) {
	c := server.DefaultConfig()
	var zero server.Config
	if _, err := os.Stat("config.toml"); os.IsNotExist(err) {
		data, err := toml.Marshal(c)
		if err != nil {
			return zero, fmt.Errorf("encode default config: %v", err)
		}
		if err := os.WriteFile("config.toml", data, 0644); err != nil {
			return zero, fmt.Errorf("create default config: %v", err)
		}
		return c.Config(log)
	}
	data, err := os.ReadFile("config.toml")
	if err != nil {
		return zero, fmt.Errorf("read config: %v", err)
	}
	if err := toml.Unmarshal(data, &c); err != nil {
		return zero, fmt.Errorf("decode config: %v", err)
	}
	return c.Config(log)
}

// consoleCommandSource adapts the server's standard input to Dragonfly's
// native cmd.Source interface. A console source is not attached to a world,
// so commands are executed with a nil *world.Tx.
type consoleCommandSource struct{ world *world.World }

func (s *consoleCommandSource) World() *world.World { return s.world }

func (*consoleCommandSource) Position() mgl64.Vec3 { return mgl64.Vec3{} }

func (*consoleCommandSource) SendCommandOutput(output *cmd.Output) {
	for _, message := range output.Messages() {
		fmt.Printf("[Console] %s\n", message)
	}
	for _, err := range output.Errors() {
		fmt.Fprintf(os.Stderr, "[Console] ERROR: %v\n", err)
	}
}

// runConsoleCommands dispatches console input through the same command
// registry and parser used by in-game commands. It accepts both "op name"
// and "/op name" forms, matching the usual dedicated-server console syntax.
func runConsoleCommands(w *world.World) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	source := &consoleCommandSource{world: w}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "/"))
		if line == "" {
			continue
		}

		name := line
		args := ""
		if i := strings.IndexAny(line, " \t"); i >= 0 {
			name = line[:i]
			args = strings.TrimSpace(line[i+1:])
		}
		name = strings.ToLower(name)

		command, ok := cmd.ByAlias(name)
		if !ok {
			output := &cmd.Output{}
			output.Errort(cmd.MessageUnknown, name)
			source.SendCommandOutput(output)
			continue
		}
		command.Execute(args, source, nil)
	}
	if err := scanner.Err(); err != nil {
		slog.Error("read console input", "error", err)
	}
}
