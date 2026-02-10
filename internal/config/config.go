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

// Config is persisted to .lattice/config.toml.
type Config struct {
	Session     SessionMetadata      `toml:"session"`
	BeadCounter int                  `toml:"bead_counter"`
	Teams       map[string]TeamState `toml:"teams"`

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

	if err := os.MkdirAll(filepath.Dir(c.filePath), 0o755); err != nil {
		return fmt.Errorf("create lattice directory: %w", err)
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(c); err != nil {
		return fmt.Errorf("encode config file: %w", err)
	}

	if err := os.WriteFile(c.filePath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write config file: %w", err)
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
	}
}
