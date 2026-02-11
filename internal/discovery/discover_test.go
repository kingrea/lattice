package discovery

import (
	"errors"
	"strings"
	"testing"
)

func TestParseAreasFromOutputWithPreambleAndCodeFence(t *testing.T) {
	t.Parallel()

	output := "I analyzed the repository and found these areas:\n\n```json\n[\n" +
		"  {\"name\":\"API Routing\",\"path\":\"internal/tui\",\"description\":\"Audit navigation and event handling flow.\"},\n" +
		"  {\"name\":\"Team Generation\",\"path\":\"internal/teams\",\"description\":\"Validate template generation and role assignment logic.\"},\n" +
		"  {\"name\":\"Templates\",\"path\":\"templates/audit\",\"description\":\"Review prompts and guardrails for audit quality.\"}\n" +
		"]\n```\n"

	areas, err := parseAreasFromOutput(output)
	if err != nil {
		t.Fatalf("parseAreasFromOutput() returned error: %v", err)
	}
	if len(areas) != 3 {
		t.Fatalf("expected 3 areas, got %d", len(areas))
	}
	if areas[0].Path != "internal/tui" {
		t.Fatalf("unexpected first path: %q", areas[0].Path)
	}
}

func TestParseAreasFromOutputWithWrappedJSONObject(t *testing.T) {
	t.Parallel()

	output := `{"areas":[{"name":"A","path":"a","description":"aa"},{"name":"B","path":"b","description":"bb"},{"name":"C","path":"c","description":"cc"}]}`

	areas, err := parseAreasFromOutput(output)
	if err != nil {
		t.Fatalf("parseAreasFromOutput() returned error: %v", err)
	}
	if len(areas) != 3 {
		t.Fatalf("expected 3 areas, got %d", len(areas))
	}
}

func TestDiscoverFallsBackWhenCommandFails(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	result, err := discoverWithRunner(projectDir, func(string, string) (string, error) {
		return "command failed", errors.New("boom")
	})
	if err != nil {
		t.Fatalf("discoverWithRunner() returned error: %v", err)
	}
	if !result.UsedFallback {
		t.Fatal("expected fallback to be used")
	}
	if len(result.Areas) < 3 || len(result.Areas) > 10 {
		t.Fatalf("expected fallback areas between 3 and 10, got %d", len(result.Areas))
	}
}

func TestDiscoverFallsBackWhenJSONCannotBeParsed(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	result, err := discoverWithRunner(projectDir, func(string, string) (string, error) {
		return "not json at all", nil
	})
	if err != nil {
		t.Fatalf("discoverWithRunner() returned error: %v", err)
	}
	if !result.UsedFallback {
		t.Fatal("expected fallback to be used")
	}
	if !strings.Contains(result.RawOutput, "not json") {
		t.Fatalf("expected raw output to include source text, got %q", result.RawOutput)
	}
}
