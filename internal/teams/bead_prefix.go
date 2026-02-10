package teams

import (
	"errors"
	"fmt"
	"strings"

	"lattice/internal/config"
)

// AllocateBeadPrefix returns a unique bead prefix and persists the global counter.
func AllocateBeadPrefix(cfg *config.Config, typePrefix string) (string, error) {
	if cfg == nil {
		return "", errors.New("config must not be nil")
	}

	normalized := strings.TrimSpace(typePrefix)
	if normalized == "" {
		return "", errors.New("type prefix must not be empty")
	}

	previous := cfg.BeadCounter
	next := previous + 1
	cfg.BeadCounter = next

	if err := cfg.Save(); err != nil {
		cfg.BeadCounter = previous
		return "", fmt.Errorf("persist bead counter: %w", err)
	}

	return fmt.Sprintf("%s-%d", normalized, next), nil
}
