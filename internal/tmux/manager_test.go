package tmux

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestCreateSessionUsesWSLCommand(t *testing.T) {
	t.Parallel()

	var calls [][]string
	m := newManagerWithRunners(func(_ context.Context, args ...string) (string, error) {
		calls = append(calls, append([]string{}, args...))
		return "", nil
	}, func(context.Context, ...string) error {
		return nil
	})

	if err := m.CreateSession("audit-1"); err != nil {
		t.Fatalf("CreateSession() returned error: %v", err)
	}

	want := []string{"new-session", "-d", "-s", "audit-1", "-n", "dashboard"}
	if len(calls) != 1 || !reflect.DeepEqual(calls[0], want) {
		t.Fatalf("unexpected command args: got %#v want %#v", calls, [][]string{want})
	}
}

func TestAttachSessionUsesInteractiveCommand(t *testing.T) {
	t.Parallel()

	var call []string
	m := newManagerWithRunners(func(context.Context, ...string) (string, error) {
		return "", nil
	}, func(_ context.Context, args ...string) error {
		call = append([]string{}, args...)
		return nil
	})

	if err := m.AttachSession("audit-1"); err != nil {
		t.Fatalf("AttachSession() returned error: %v", err)
	}

	want := []string{"attach-session", "-t", "audit-1"}
	if !reflect.DeepEqual(call, want) {
		t.Fatalf("unexpected interactive args: got %#v want %#v", call, want)
	}
}

func TestListWindowsParsesOutput(t *testing.T) {
	t.Parallel()

	m := newManagerWithRunners(func(_ context.Context, args ...string) (string, error) {
		want := []string{"list-windows", "-t", "audit-1", "-F", "#{window_index}\t#{window_name}\t#{window_active}"}
		if !reflect.DeepEqual(args, want) {
			t.Fatalf("unexpected args: got %#v want %#v", args, want)
		}
		return "0\tdashboard\t1\n1\talpha\t0", nil
	}, func(context.Context, ...string) error {
		return nil
	})

	got, err := m.ListWindows("audit-1")
	if err != nil {
		t.Fatalf("ListWindows() returned error: %v", err)
	}

	want := []WindowInfo{
		{Index: 0, Name: "dashboard", Active: true},
		{Index: 1, Name: "alpha", Active: false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected windows: got %#v want %#v", got, want)
	}
}

func TestEnsureAvailableReturnsFriendlyError(t *testing.T) {
	t.Parallel()

	m := newManagerWithRunners(func(context.Context, ...string) (string, error) {
		return "", errors.New("executable file not found")
	}, func(context.Context, ...string) error {
		return nil
	})

	err := m.ensureAvailable(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "tmux is not available") {
		t.Fatalf("expected friendly availability message, got: %v", err)
	}
}

func TestTranslateToWSLPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{name: "windows drive path", input: `G:\Node-Websites\ai`, want: "/mnt/g/Node-Websites/ai"},
		{name: "windows lowercase drive path", input: `c:\Users\me\repo`, want: "/mnt/c/Users/me/repo"},
		{name: "already unix path", input: "/mnt/g/Node-Websites/ai", want: "/mnt/g/Node-Websites/ai"},
		{name: "relative path", input: "team-audit\\context", want: "team-audit/context"},
		{name: "drive relative unsupported", input: `Z:folder`, wantErr: "unsupported Windows path format"},
		{name: "empty path", input: "   ", wantErr: "path must not be empty"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := TranslateToWSLPath(tc.input)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("TranslateToWSLPath() returned error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("TranslateToWSLPath() mismatch: got %q want %q", got, tc.want)
			}
		})
	}
}
