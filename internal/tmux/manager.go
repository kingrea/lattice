package tmux

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	pathpkg "path"
	"strconv"
	"strings"
)

type runCommand func(ctx context.Context, args ...string) (string, error)
type runInteractiveCommand func(ctx context.Context, args ...string) error

var errEmptyName = errors.New("name must not be empty")

// WindowInfo describes a tmux window in one session.
type WindowInfo struct {
	Index  int
	Name   string
	Active bool
}

// Manager wraps WSL-based tmux operations.
type Manager struct {
	runCommand            runCommand
	runInteractiveCommand runInteractiveCommand
}

// NewManager creates a manager and verifies tmux is available in WSL.
func NewManager() (*Manager, error) {
	m := &Manager{
		runCommand:            defaultRunCommand,
		runInteractiveCommand: defaultRunInteractiveCommand,
	}

	if err := m.ensureAvailable(context.Background()); err != nil {
		return nil, err
	}

	return m, nil
}

func newManagerWithRunners(run runCommand, runInteractive runInteractiveCommand) *Manager {
	return &Manager{runCommand: run, runInteractiveCommand: runInteractive}
}

// CreateSession creates a detached tmux session with a dashboard window.
func (m *Manager) CreateSession(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errEmptyName
	}

	if _, err := m.runCommand(context.Background(), "new-session", "-d", "-s", name, "-n", "dashboard"); err != nil {
		return fmt.Errorf("create tmux session %q: %w", name, err)
	}

	return nil
}

// AttachSession attaches terminal IO to a running tmux session.
func (m *Manager) AttachSession(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errEmptyName
	}

	if err := m.runInteractiveCommand(context.Background(), "attach-session", "-t", name); err != nil {
		return fmt.Errorf("attach tmux session %q: %w", name, err)
	}

	return nil
}

// KillSession terminates a tmux session.
func (m *Manager) KillSession(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errEmptyName
	}

	if _, err := m.runCommand(context.Background(), "kill-session", "-t", name); err != nil {
		return fmt.Errorf("kill tmux session %q: %w", name, err)
	}

	return nil
}

// CreateWindow creates a new window in the target session.
func (m *Manager) CreateWindow(session, name string) error {
	session = strings.TrimSpace(session)
	name = strings.TrimSpace(name)
	if session == "" || name == "" {
		return errEmptyName
	}

	if _, err := m.runCommand(context.Background(), "new-window", "-t", session, "-n", name); err != nil {
		return fmt.Errorf("create tmux window %q in session %q: %w", name, session, err)
	}

	return nil
}

// RenameWindow renames a window in a session.
func (m *Manager) RenameWindow(session, currentName, newName string) error {
	session = strings.TrimSpace(session)
	currentName = strings.TrimSpace(currentName)
	newName = strings.TrimSpace(newName)
	if session == "" || currentName == "" || newName == "" {
		return errEmptyName
	}

	target := fmt.Sprintf("%s:%s", session, currentName)
	if _, err := m.runCommand(context.Background(), "rename-window", "-t", target, newName); err != nil {
		return fmt.Errorf("rename tmux window %q to %q in session %q: %w", currentName, newName, session, err)
	}

	return nil
}

// SendKeys sends a command then Enter to a tmux window.
func (m *Manager) SendKeys(session, window, command string) error {
	session = strings.TrimSpace(session)
	window = strings.TrimSpace(window)
	if session == "" || window == "" || command == "" {
		return errEmptyName
	}

	target := fmt.Sprintf("%s:%s", session, window)
	if _, err := m.runCommand(context.Background(), "send-keys", "-t", target, command, "C-m"); err != nil {
		return fmt.Errorf("send keys to tmux window %q in session %q: %w", window, session, err)
	}

	return nil
}

// ListWindows returns indexed window metadata for one session.
func (m *Manager) ListWindows(session string) ([]WindowInfo, error) {
	session = strings.TrimSpace(session)
	if session == "" {
		return nil, errEmptyName
	}

	out, err := m.runCommand(context.Background(), "list-windows", "-t", session, "-F", "#{window_index}\t#{window_name}\t#{window_active}")
	if err != nil {
		return nil, fmt.Errorf("list windows for session %q: %w", session, err)
	}

	if strings.TrimSpace(out) == "" {
		return []WindowInfo{}, nil
	}

	lines := strings.Split(out, "\n")
	result := make([]WindowInfo, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) != 3 {
			return nil, fmt.Errorf("parse list-windows output: unexpected line %q", line)
		}

		idx, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("parse list-windows output: invalid index %q", parts[0])
		}

		result = append(result, WindowInfo{
			Index:  idx,
			Name:   parts[1],
			Active: parts[2] == "1",
		})
	}

	return result, nil
}

func (m *Manager) ensureAvailable(ctx context.Context) error {
	if _, err := m.runCommand(ctx, "-V"); err != nil {
		return fmt.Errorf("tmux is not available in WSL: %w", err)
	}

	return nil
}

func defaultRunCommand(ctx context.Context, args ...string) (string, error) {
	tmuxArgs := append([]string{"tmux"}, args...)
	cmd := exec.CommandContext(ctx, "wsl", tmuxArgs...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", wrapExecError(err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

func defaultRunInteractiveCommand(ctx context.Context, args ...string) error {
	tmuxArgs := append([]string{"tmux"}, args...)
	cmd := exec.CommandContext(ctx, "wsl", tmuxArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run wsl tmux %s: %w", strings.Join(args, " "), err)
	}

	return nil
}

func wrapExecError(err error, stderr string) error {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return err
	}

	return fmt.Errorf("%w: %s", err, stderr)
}

// TranslateToWSLPath converts Windows style paths for WSL tmux commands.
func TranslateToWSLPath(windowsPath string) (string, error) {
	trimmed := strings.TrimSpace(windowsPath)
	if trimmed == "" {
		return "", errors.New("path must not be empty")
	}

	normalized := strings.ReplaceAll(trimmed, "\\", "/")
	if strings.HasPrefix(normalized, "/") {
		return pathpkg.Clean(normalized), nil
	}

	if len(normalized) >= 3 && normalized[1] == ':' && normalized[2] == '/' {
		drive := normalized[0]
		if (drive < 'A' || drive > 'Z') && (drive < 'a' || drive > 'z') {
			return "", fmt.Errorf("invalid drive letter in path %q", windowsPath)
		}

		rest := strings.TrimPrefix(normalized[3:], "/")
		mount := fmt.Sprintf("/mnt/%s/%s", strings.ToLower(string(drive)), rest)
		return pathpkg.Clean(mount), nil
	}

	if len(normalized) >= 2 && normalized[1] == ':' {
		return "", fmt.Errorf("unsupported Windows path format %q", windowsPath)
	}

	return pathpkg.Clean(normalized), nil
}
