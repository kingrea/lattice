package discovery

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const discoveryPrompt = `Analyze this codebase and identify 3-10 auditable areas for engineering review.
Return ONLY valid JSON (no markdown) using this exact schema:
[
  {
    "name": "<short area name>",
    "path": "<project-relative path>",
    "description": "<one sentence describing risk/opportunity>"
  }
]

Rules:
- Include between 3 and 10 areas.
- Prefer high-impact areas that are realistically auditable.
- Use project-relative paths.`

// Area is one auditable area discovered in the target project.
type Area struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

// Result captures discovered areas and whether fallback logic was used.
type Result struct {
	Areas        []Area
	UsedFallback bool
	RawOutput    string
}

type runOpencodeFunc func(projectDir, prompt string) (string, error)

// Discover runs opencode against projectDir and extracts auditable areas.
func Discover(projectDir string) (Result, error) {
	return discoverWithRunner(projectDir, runOpencode)
}

func discoverWithRunner(projectDir string, runner runOpencodeFunc) (Result, error) {
	if strings.TrimSpace(projectDir) == "" {
		return Result{}, fmt.Errorf("project directory must not be empty")
	}

	rawOutput, runErr := runner(projectDir, discoveryPrompt)
	trimmedOutput := strings.TrimSpace(rawOutput)
	if runErr != nil {
		return Result{
			Areas:        manualAreas(projectDir),
			UsedFallback: true,
			RawOutput:    trimmedOutput,
		}, nil
	}

	areas, parseErr := parseAreasFromOutput(trimmedOutput)
	if parseErr != nil {
		return Result{
			Areas:        manualAreas(projectDir),
			UsedFallback: true,
			RawOutput:    trimmedOutput,
		}, nil
	}

	return Result{Areas: areas, RawOutput: trimmedOutput}, nil
}

func runOpencode(projectDir, prompt string) (string, error) {
	cmd := exec.Command("opencode", "run", prompt)
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("run opencode discovery: %w", err)
	}

	return string(output), nil
}

func parseAreasFromOutput(output string) ([]Area, error) {
	if strings.TrimSpace(output) == "" {
		return nil, fmt.Errorf("empty discovery output")
	}

	candidates := extractJSONCandidates(output)
	for _, candidate := range candidates {
		areas, err := parseAreasJSON(candidate)
		if err == nil {
			return areas, nil
		}
	}

	return nil, fmt.Errorf("could not extract discovery JSON")
}

func parseAreasJSON(blob string) ([]Area, error) {
	var areaList []Area
	if err := json.Unmarshal([]byte(blob), &areaList); err == nil {
		return normalizeAreas(areaList)
	}

	var wrapped struct {
		Areas []Area `json:"areas"`
	}
	if err := json.Unmarshal([]byte(blob), &wrapped); err == nil {
		return normalizeAreas(wrapped.Areas)
	}

	return nil, fmt.Errorf("blob is not valid discovery JSON")
}

func normalizeAreas(areas []Area) ([]Area, error) {
	normalized := make([]Area, 0, len(areas))
	for _, area := range areas {
		name := strings.TrimSpace(area.Name)
		path := strings.TrimSpace(area.Path)
		description := strings.TrimSpace(area.Description)
		if name == "" || path == "" || description == "" {
			continue
		}

		normalized = append(normalized, Area{
			Name:        name,
			Path:        filepath.Clean(path),
			Description: description,
		})
	}

	if len(normalized) < 3 {
		return nil, fmt.Errorf("need at least 3 discovery areas")
	}
	if len(normalized) > 10 {
		normalized = normalized[:10]
	}

	return normalized, nil
}

func extractJSONCandidates(output string) []string {
	candidates := make([]string, 0, 4)
	trimmed := strings.TrimSpace(output)
	if trimmed != "" {
		candidates = append(candidates, trimmed)
	}

	for _, block := range extractFencedCodeBlocks(output) {
		if block != "" {
			candidates = append(candidates, block)
		}
	}

	for _, block := range extractBalancedJSON(output) {
		if block != "" {
			candidates = append(candidates, block)
		}
	}

	return uniqueStrings(candidates)
}

func extractFencedCodeBlocks(output string) []string {
	segments := strings.Split(output, "```")
	blocks := make([]string, 0, len(segments)/2)
	for i := 1; i < len(segments); i += 2 {
		block := strings.TrimSpace(segments[i])
		if block == "" {
			continue
		}

		if newline := strings.IndexByte(block, '\n'); newline >= 0 {
			lang := strings.TrimSpace(strings.ToLower(block[:newline]))
			if lang == "json" || lang == "javascript" || lang == "js" || lang == "" {
				block = strings.TrimSpace(block[newline+1:])
			}
		}

		blocks = append(blocks, strings.TrimSpace(block))
	}

	return blocks
}

func extractBalancedJSON(output string) []string {
	blocks := make([]string, 0, 4)
	for idx := 0; idx < len(output); idx++ {
		ch := output[idx]
		if ch != '{' && ch != '[' {
			continue
		}

		if block, ok := consumeBalancedJSON(output, idx); ok {
			blocks = append(blocks, block)
			idx += len(block) - 1
		}
	}

	return blocks
}

func consumeBalancedJSON(text string, start int) (string, bool) {
	if start < 0 || start >= len(text) {
		return "", false
	}

	stack := []byte{text[start]}
	inString := false
	escaped := false

	for i := start + 1; i < len(text); i++ {
		ch := text[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{', '[':
			stack = append(stack, ch)
		case '}':
			if len(stack) == 0 || stack[len(stack)-1] != '{' {
				return "", false
			}
			stack = stack[:len(stack)-1]
		case ']':
			if len(stack) == 0 || stack[len(stack)-1] != '[' {
				return "", false
			}
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			return strings.TrimSpace(text[start : i+1]), true
		}
	}

	return "", false
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	uniq := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		uniq = append(uniq, value)
	}

	return uniq
}

func manualAreas(projectDir string) []Area {
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return defaultManualAreas()
	}

	type candidate struct {
		name  string
		path  string
		score int
	}

	candidates := make([]candidate, 0, len(entries))
	for _, entry := range entries {
		name := strings.TrimSpace(entry.Name())
		if name == "" || strings.HasPrefix(name, ".") {
			continue
		}
		if name == "node_modules" || name == "vendor" {
			continue
		}

		score := 0
		if entry.IsDir() {
			score = 2
		} else {
			score = 1
		}
		candidates = append(candidates, candidate{name: name, path: name, score: score})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].name < candidates[j].name
	})

	areas := make([]Area, 0, 10)
	for _, c := range candidates {
		areas = append(areas, Area{
			Name:        titleize(c.name),
			Path:        c.path,
			Description: "Manual fallback area based on top-level project structure.",
		})
		if len(areas) == 10 {
			break
		}
	}

	if len(areas) >= 3 {
		return areas
	}

	defaults := defaultManualAreas()
	for _, area := range defaults {
		if len(areas) >= 3 {
			break
		}
		areas = append(areas, area)
	}

	return areas
}

func defaultManualAreas() []Area {
	return []Area{
		{Name: "Application Entrypoints", Path: ".", Description: "Review startup paths and top-level execution flow manually."},
		{Name: "Core Domain Logic", Path: "internal", Description: "Inspect core business logic modules and cross-module coupling manually."},
		{Name: "Build And Tooling", Path: "go.mod", Description: "Assess dependency, build, and tooling configuration manually."},
	}
}

func titleize(value string) string {
	value = strings.ReplaceAll(value, "-", " ")
	value = strings.ReplaceAll(value, "_", " ")
	parts := strings.Fields(value)
	if len(parts) == 0 {
		return "Area"
	}

	for i, part := range parts {
		runes := []rune(strings.ToLower(part))
		if len(runes) == 0 {
			continue
		}
		runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
		parts[i] = string(runes)
	}

	return strings.Join(parts, " ")
}
