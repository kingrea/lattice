package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

const (
	DirName        = ".lattice"
	ConfigFileName = "config.toml"
)

var now = time.Now

// SessionMetadata captures run-level data for the current lattice session.
type SessionMetadata struct {
	Name       string `toml:"name"`
	CreatedAt  string `toml:"created_at"`
	WorkingDir string `toml:"working_dir"`
}

// TeamState tracks mutable launch and runtime status for one team.
type TeamState struct {
	ID         string `toml:"id"`
	Type       string `toml:"type"`
	Prefix     string `toml:"prefix"`
	AgentCount int    `toml:"agent_count"`
	Intensity  int    `toml:"intensity"`
	Status     string `toml:"status"`
	TmuxWindow string `toml:"tmux_window"`
}

// EpicState tracks mutable launch and runtime status for one epic.
type EpicState struct {
	BeadID     string `toml:"bead_id"`
	AuditType  string `toml:"audit_type"`
	AuditName  string `toml:"audit_name"`
	AgentCount int    `toml:"agent_count"`
	Intensity  int    `toml:"intensity"`
	Status     string `toml:"status"`
}

// RoleState tracks mutable launch and runtime status for one role.
type RoleState struct {
	BeadID     string `toml:"bead_id"`
	EpicBeadID string `toml:"epic_bead_id"`
	CodeName   string `toml:"code_name"`
	Title      string `toml:"title"`
	Guidance   string `toml:"guidance"`
	BeadPrefix string `toml:"bead_prefix"`
	Order      int    `toml:"order"`
	Status     string `toml:"status"`
	TmuxWindow string `toml:"tmux_window"`
	Intensity  int    `toml:"intensity"`
}

// Config is persisted to .lattice/config.toml.
type Config struct {
	Session     SessionMetadata      `toml:"session"`
	BeadCounter int                  `toml:"bead_counter"`
	Teams       map[string]TeamState `toml:"teams"`
	Epics       map[string]EpicState `toml:"epics"`
	Roles       map[string]RoleState `toml:"roles"`

	filePath string `toml:"-"`
}

// Init ensures the .lattice directory and config file exist.
// If config.toml already exists, it is loaded and returned.
func Init(cwd string) (*Config, error) {
	dirPath := filepath.Join(cwd, DirName)
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return nil, fmt.Errorf("create lattice directory: %w", err)
	}

	configPath := filepath.Join(dirPath, ConfigFileName)
	if _, err := os.Stat(configPath); err == nil {
		return Load(cwd)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("stat config file: %w", err)
	}

	cfg := defaultConfig(cwd)
	cfg.filePath = configPath
	if err := cfg.Save(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Load reads and decodes .lattice/config.toml.
func Load(cwd string) (*Config, error) {
	configPath := filepath.Join(cwd, DirName, ConfigFileName)
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("open config file: %w", err)
	}
	defer file.Close()

	var cfg Config
	if _, err := toml.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config file: %w", err)
	}

	if cfg.Teams == nil {
		cfg.Teams = map[string]TeamState{}
	}
	if cfg.Epics == nil {
		cfg.Epics = map[string]EpicState{}
	}
	if cfg.Roles == nil {
		cfg.Roles = map[string]RoleState{}
	}

	cfg.filePath = configPath
	return &cfg, nil
}

// Save encodes and writes the config to .lattice/config.toml.
func (c *Config) Save() error {
	if c.filePath == "" {
		return errors.New("config file path is not set")
	}

	if c.Teams == nil {
		c.Teams = map[string]TeamState{}
	}
	if c.Epics == nil {
		c.Epics = map[string]EpicState{}
	}
	if c.Roles == nil {
		c.Roles = map[string]RoleState{}
	}

	if err := os.MkdirAll(filepath.Dir(c.filePath), 0o755); err != nil {
		return fmt.Errorf("create lattice directory: %w", err)
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(c); err != nil {
		return fmt.Errorf("encode config file: %w", err)
	}

	tmpPath := c.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write temp config file: %w", err)
	}

	if err := os.Rename(tmpPath, c.filePath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace config file: %w", err)
	}

	if err := os.Remove(tmpPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("cleanup temp config file: %w", err)
	}

	return nil
}

func defaultConfig(cwd string) *Config {
	ts := now().UTC()
	return &Config{
		Session: SessionMetadata{
			Name:       "lattice-" + ts.Format("20060102-150405"),
			CreatedAt:  ts.Format(time.RFC3339),
			WorkingDir: cwd,
		},
		BeadCounter: 0,
		Teams:       map[string]TeamState{},
		Epics:       map[string]EpicState{},
		Roles:       map[string]RoleState{},
	}
}
