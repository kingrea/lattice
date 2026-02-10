package teams

import (
	"strings"
	"testing"

	"lattice/internal/config"
)

func TestAllocateBeadPrefixUsesGlobalCounterAcrossTypes(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	cfg, err := config.Init(tmp)
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	prefix, err := AllocateBeadPrefix(cfg, "perf")
	if err != nil {
		t.Fatalf("AllocateBeadPrefix(perf) returned error: %v", err)
	}
	if prefix != "perf-1" {
		t.Fatalf("unexpected prefix for perf: got %q want %q", prefix, "perf-1")
	}

	prefix, err = AllocateBeadPrefix(cfg, "sec")
	if err != nil {
		t.Fatalf("AllocateBeadPrefix(sec) returned error: %v", err)
	}
	if prefix != "sec-2" {
		t.Fatalf("unexpected prefix for sec: got %q want %q", prefix, "sec-2")
	}

	prefix, err = AllocateBeadPrefix(cfg, "perf")
	if err != nil {
		t.Fatalf("AllocateBeadPrefix(perf second) returned error: %v", err)
	}
	if prefix != "perf-3" {
		t.Fatalf("unexpected prefix for perf second run: got %q want %q", prefix, "perf-3")
	}

	reloaded, err := config.Load(tmp)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if reloaded.BeadCounter != 3 {
		t.Fatalf("expected BeadCounter=3 after allocations, got %d", reloaded.BeadCounter)
	}

	prefix, err = AllocateBeadPrefix(reloaded, "mem")
	if err != nil {
		t.Fatalf("AllocateBeadPrefix(mem) on reloaded config returned error: %v", err)
	}
	if prefix != "mem-4" {
		t.Fatalf("unexpected prefix after reload: got %q want %q", prefix, "mem-4")
	}
}

func TestAllocateBeadPrefixRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	if _, err := AllocateBeadPrefix(nil, "perf"); err == nil || !strings.Contains(err.Error(), "config must not be nil") {
		t.Fatalf("expected nil config error, got: %v", err)
	}

	tmp := t.TempDir()
	cfg, err := config.Init(tmp)
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	if _, err := AllocateBeadPrefix(cfg, "   "); err == nil || !strings.Contains(err.Error(), "type prefix must not be empty") {
		t.Fatalf("expected empty prefix error, got: %v", err)
	}
}

func TestAllocateBeadPrefixRestoresCounterWhenSaveFails(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	if _, err := AllocateBeadPrefix(cfg, "perf"); err == nil || !strings.Contains(err.Error(), "persist bead counter") {
		t.Fatalf("expected save failure to be wrapped, got: %v", err)
	}

	if cfg.BeadCounter != 0 {
		t.Fatalf("expected BeadCounter to be restored on save failure, got %d", cfg.BeadCounter)
	}
}
