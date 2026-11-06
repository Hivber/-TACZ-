package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
)

type operatorStore struct {
	mu        sync.RWMutex
	path      string
	ownerXUID string
	operators map[uuid.UUID]struct{}
}

type operatorFile struct {
	Operators []string `json:"operators"`
}

func loadOperatorStore(path, ownerXUID string) (*operatorStore, error) {
	s := &operatorStore{path: path, ownerXUID: strings.TrimSpace(ownerXUID), operators: map[uuid.UUID]struct{}{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("read operator list: %w", err)
	}
	var f operatorFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("decode operator list: %w", err)
	}
	for _, raw := range f.Operators {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid UUID %q in operator list: %w", raw, err)
		}
		s.operators[id] = struct{}{}
	}
	return s, nil
}

func (s *operatorStore) isAllowed(src cmd.Source) bool {
	// The local server console is a trusted command source, like the server
	// console in dedicated Bedrock servers. Player sources still need to be
	// the configured owner or an existing operator.
	if _, ok := src.(*consoleCommandSource); ok {
		return true
	}
	if p, ok := src.(*player.Player); ok {
		if s.ownerXUID != "" && p.XUID() == s.ownerXUID {
			return true
		}
		return p.Operator() || s.isOperator(p.UUID())
	}
	return false
}

func (s *operatorStore) isOperator(id uuid.UUID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.operators[id]
	return ok
}

func (s *operatorStore) add(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.operators[id] = struct{}{}
	return s.saveLocked()
}

func (s *operatorStore) remove(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.operators, id)
	return s.saveLocked()
}

func (s *operatorStore) list() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.operators))
	for id := range s.operators {
		out = append(out, id.String())
	}
	sort.Strings(out)
	return out
}

func (s *operatorStore) saveLocked() error {
	f := operatorFile{Operators: make([]string, 0, len(s.operators))}
	for id := range s.operators {
		f.Operators = append(f.Operators, id.String())
	}
	sort.Strings(f.Operators)
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("encode operator list: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("create operator-list directory: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0600); err != nil {
		return fmt.Errorf("write operator list: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace operator list: %w", err)
	}
	return nil
}

type xuidCommand struct{}

func (xuidCommand) Run(src cmd.Source, out *cmd.Output, _ *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok {
		out.Error("This command can only be used by a player.")
		return
	}
	if p.XUID() == "" {
		out.Error("No Xbox Live XUID is available. Sign in with an authenticated Xbox Live account.")
		return
	}
	out.Printf("Your XUID: %s", p.XUID())
}

type opCommand struct {
	Target string
	store  *operatorStore
	srv    *server.Server
}

func (c opCommand) Allow(src cmd.Source) bool { return c.store.isAllowed(src) }

func (c opCommand) Run(src cmd.Source, out *cmd.Output, _ *world.Tx) {
	h, ok := c.srv.PlayerByName(c.Target)
	if !ok {
		out.Errorf("Player %q is not online. This command currently supports online players only.", c.Target)
		return
	}
	id := h.UUID()
	if err := c.store.add(id); err != nil {
		out.Error(err)
		return
	}
	player.Do(h, func(_ *world.Tx, target *player.Player) { target.SetOperator(true) })
	out.Printf("Operator permissions granted to %s (%s).", c.Target, id)
}

type deopCommand struct {
	Target string
	store  *operatorStore
	srv    *server.Server
}

func (c deopCommand) Allow(src cmd.Source) bool { return c.store.isAllowed(src) }

func (c deopCommand) Run(src cmd.Source, out *cmd.Output, _ *world.Tx) {
	h, ok := c.srv.PlayerByName(c.Target)
	if !ok {
		out.Errorf("Player %q is not online. This command currently supports online players only.", c.Target)
		return
	}
	id := h.UUID()
	if err := c.store.remove(id); err != nil {
		out.Error(err)
		return
	}
	ownerXUID := c.store.ownerXUID
	player.Do(h, func(_ *world.Tx, target *player.Player) {
		if ownerXUID != "" && target.XUID() == ownerXUID {
			return
		}
		target.SetOperator(false)
	})
	out.Printf("Operator permissions revoked from %s (%s), unless this account is the configured owner.", c.Target, id)
}

type oplistCommand struct {
	store *operatorStore
}

func (c oplistCommand) Allow(src cmd.Source) bool { return c.store.isAllowed(src) }

func (c oplistCommand) Run(_ cmd.Source, out *cmd.Output, _ *world.Tx) {
	list := c.store.list()
	if len(list) == 0 {
		out.Print("No additional operators are configured.")
		return
	}
	out.Printf("Saved operator UUIDs: %s", strings.Join(list, ", "))
}

func registerOperatorCommands(srv *server.Server, store *operatorStore) {
	cmd.Register(cmd.New("xuid", "Show your authenticated Xbox Live XUID", nil, xuidCommand{}))
	cmd.Register(cmd.New("op", "Grant operator permissions to an online player", nil, opCommand{store: store, srv: srv}))
	cmd.Register(cmd.New("deop", "Revoke operator permissions from an online player", nil, deopCommand{store: store, srv: srv}))
	cmd.Register(cmd.New("oplist", "List saved operator UUIDs", nil, oplistCommand{store: store}))
}
