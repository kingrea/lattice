package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"lattice/internal/discovery"
	"lattice/internal/teams"
)

// AuditWizardStep identifies the current stage in the audit setup flow.
type AuditWizardStep int

const (
	AuditWizardStepMode AuditWizardStep = iota
	AuditWizardStepDiscovery
	AuditWizardStepTypes
	AuditWizardStepAgentCount
	AuditWizardStepRigor
	AuditWizardStepConfirm
	AuditWizardStepGenerating
)

// WizardMode is the generation mode selected in step 0.
type WizardMode int

const (
	WizardModeManual WizardMode = iota
	WizardModeAutoGenerate
)

// WizardRigor controls how many iteration loops run per investigator.
type WizardRigor struct {
	Label string
	Loops int
}

// AuditWizardModel provides a six-step configuration flow for launching audits.
type AuditWizardModel struct {
	styles Styles
	keyMap KeyMap

	step                  AuditWizardStep
	projectDir            string
	discover              func(projectDir string) (discovery.Result, error)
	modeCursor            int
	mode                  WizardMode
	auditTypeSelect       MultiSelectModel[teams.AuditType]
	agentCursor           int
	rigorCursor           int
	discoveryAreas        []discovery.Area
	discoveryRunning      bool
	discoveryUsedFallback bool

	spinner  spinner.Model
	launched bool

	validationErr string
}

type discoveryFinishedMsg struct {
	result discovery.Result
	err    error
}

var wizardModeOptions = []struct {
	mode        WizardMode
	label       string
	description string
}{
	{mode: WizardModeManual, label: "Manual", description: "Pick audit settings step-by-step."},
	{mode: WizardModeAutoGenerate, label: "Auto-generate", description: "Use presets with minimal input."},
}

var wizardRigorOptions = []WizardRigor{
	{Label: "Light", Loops: 1},
	{Label: "Standard", Loops: 3},
	{Label: "Go Hard", Loops: 99},
}

var wizardAgentOptions = []int{1, 2, 3}

// NewAuditWizardModel builds the initial wizard state.
func NewAuditWizardModel() AuditWizardModel {
	s := spinner.New()
	s.Spinner = spinner.Dot

	m := AuditWizardModel{
		styles:     DefaultStyles(),
		keyMap:     DefaultKeyMap(),
		step:       AuditWizardStepMode,
		projectDir: ".",
		discover:   discovery.Discover,
		mode:       WizardModeManual,
		spinner:    s,
	}

	m.auditTypeSelect = newAuditTypeSelect(nil)
	return m
}

// SetStyles overrides visual styling used by the wizard.
func (m AuditWizardModel) SetStyles(styles Styles) AuditWizardModel {
	m.styles = styles
	m.auditTypeSelect = m.auditTypeSelect.SetStyles(styles)
	return m
}

// SetKeyMap overrides shared key bindings used by the wizard.
func (m AuditWizardModel) SetKeyMap(keyMap KeyMap) AuditWizardModel {
	m.keyMap = keyMap
	return m
}

// SetProjectDir sets the target directory for discovery.
func (m AuditWizardModel) SetProjectDir(projectDir string) AuditWizardModel {
	m.projectDir = projectDir
	return m
}

// SetDiscover overrides discovery execution, primarily for tests.
func (m AuditWizardModel) SetDiscover(discoverFn func(projectDir string) (discovery.Result, error)) AuditWizardModel {
	m.discover = discoverFn
	return m
}

// Update handles key input and timer messages.
func (m AuditWizardModel) Update(msg tea.Msg) (AuditWizardModel, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(typed, m.keyMap.Back) {
			m.validationErr = ""
			if m.step > AuditWizardStepMode {
				switch m.step {
				case AuditWizardStepTypes:
					if m.mode == WizardModeAutoGenerate {
						m.step = AuditWizardStepDiscovery
					} else {
						m.step = AuditWizardStepMode
					}
				case AuditWizardStepDiscovery:
					m.step = AuditWizardStepMode
				default:
					m.step--
				}
				m.launched = false
				m.discoveryRunning = false
			}
			return m, nil
		}

		if m.step == AuditWizardStepGenerating {
			return m, nil
		}
		if m.step == AuditWizardStepDiscovery && m.discoveryRunning {
			return m, nil
		}

		switch m.step {
		case AuditWizardStepMode:
			return m.updateStepMode(typed)
		case AuditWizardStepDiscovery:
			return m, nil
		case AuditWizardStepTypes:
			return m.updateStepTypes(typed)
		case AuditWizardStepAgentCount:
			return m.updateStepAgentCount(typed)
		case AuditWizardStepRigor:
			return m.updateStepRigor(typed)
		case AuditWizardStepConfirm:
			return m.updateStepConfirm(typed)
		}

	case spinner.TickMsg:
		if (m.step == AuditWizardStepGenerating && !m.launched) || (m.step == AuditWizardStepDiscovery && m.discoveryRunning) {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	case discoveryFinishedMsg:
		if m.step == AuditWizardStepDiscovery {
			m.discoveryRunning = false
			if typed.err != nil {
				m.validationErr = typed.err.Error()
				m.step = AuditWizardStepMode
				return m, nil
			}

			m.discoveryAreas = typed.result.Areas
			m.discoveryUsedFallback = typed.result.UsedFallback
			m.validationErr = ""
			m.step = AuditWizardStepTypes
		}
		return m, nil
	case LaunchCompleteMsg:
		if m.step == AuditWizardStepGenerating {
			m.launched = true
			m.validationErr = ""
		}
		return m, nil
	case LaunchFailedMsg:
		if m.step == AuditWizardStepGenerating {
			m.launched = false
			m.step = AuditWizardStepConfirm
			m.validationErr = typed.Err.Error()
		}
		return m, nil
	}

	return m, nil
}

func (m AuditWizardModel) updateStepMode(msg tea.KeyMsg) (AuditWizardModel, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Up):
		if m.modeCursor == 0 {
			m.modeCursor = len(wizardModeOptions) - 1
		} else {
			m.modeCursor--
		}
	case key.Matches(msg, m.keyMap.Down):
		m.modeCursor = (m.modeCursor + 1) % len(wizardModeOptions)
	case key.Matches(msg, m.keyMap.Select):
		m.mode = wizardModeOptions[m.modeCursor].mode
		if m.mode == WizardModeAutoGenerate {
			m.step = AuditWizardStepDiscovery
			m.discoveryRunning = true
			m.discoveryAreas = nil
			m.discoveryUsedFallback = false
			m.validationErr = ""
			return m, m.discoveryCmd()
		}
		m.step = AuditWizardStepTypes
	}

	return m, nil
}

func (m AuditWizardModel) discoveryCmd() tea.Cmd {
	projectDir := m.projectDir
	discoverFn := m.discover
	return func() tea.Msg {
		result, err := discoverFn(projectDir)
		return discoveryFinishedMsg{result: result, err: err}
	}
}

func (m AuditWizardModel) updateStepTypes(msg tea.KeyMsg) (AuditWizardModel, tea.Cmd) {
	nextModel, cmd := m.auditTypeSelect.Update(msg)
	m.auditTypeSelect = nextModel

	if !m.auditTypeSelect.Confirmed() {
		return m, cmd
	}

	if len(m.auditTypeSelect.SelectedItems()) == 0 {
		m.validationErr = "Select at least one audit type to continue."
		m.auditTypeSelect = newAuditTypeSelect(nil)
		return m, nil
	}

	m.validationErr = ""
	m.step = AuditWizardStepAgentCount
	m.auditTypeSelect = newAuditTypeSelect(m.selectedAuditTypeIDs())
	return m, cmd
}

func (m AuditWizardModel) updateStepAgentCount(msg tea.KeyMsg) (AuditWizardModel, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Up):
		if m.agentCursor == 0 {
			m.agentCursor = len(wizardAgentOptions) - 1
		} else {
			m.agentCursor--
		}
	case key.Matches(msg, m.keyMap.Down):
		m.agentCursor = (m.agentCursor + 1) % len(wizardAgentOptions)
	case key.Matches(msg, m.keyMap.Select):
		m.step = AuditWizardStepRigor
	}

	return m, nil
}

func (m AuditWizardModel) updateStepRigor(msg tea.KeyMsg) (AuditWizardModel, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Up):
		if m.rigorCursor == 0 {
			m.rigorCursor = len(wizardRigorOptions) - 1
		} else {
			m.rigorCursor--
		}
	case key.Matches(msg, m.keyMap.Down):
		m.rigorCursor = (m.rigorCursor + 1) % len(wizardRigorOptions)
	case key.Matches(msg, m.keyMap.Select):
		m.step = AuditWizardStepConfirm
	}

	return m, nil
}

func (m AuditWizardModel) updateStepConfirm(msg tea.KeyMsg) (AuditWizardModel, tea.Cmd) {
	if !key.Matches(msg, m.keyMap.Select) {
		return m, nil
	}

	m.step = AuditWizardStepGenerating
	m.launched = false
	m.validationErr = ""

	return m, m.spinner.Tick
}

func (m AuditWizardModel) selectedAuditTypeIDs() map[string]struct{} {
	ids := make(map[string]struct{}, len(m.auditTypeSelect.SelectedItems()))
	for _, item := range m.auditTypeSelect.SelectedItems() {
		ids[item.Value.ID] = struct{}{}
	}

	return ids
}

// View renders the current wizard screen.
func (m AuditWizardModel) View() string {
	var lines []string

	lines = append(lines, m.styles.Header.Render("Audit Wizard"))
	lines = append(lines, m.styles.Subheader.Render(m.stepLabel()))
	lines = append(lines, "")

	switch m.step {
	case AuditWizardStepMode:
		lines = append(lines, m.viewModeStep()...)
	case AuditWizardStepDiscovery:
		lines = append(lines, m.viewDiscoveryStep()...)
	case AuditWizardStepTypes:
		lines = append(lines, m.viewTypesStep()...)
	case AuditWizardStepAgentCount:
		lines = append(lines, m.viewAgentStep()...)
	case AuditWizardStepRigor:
		lines = append(lines, m.viewRigorStep()...)
	case AuditWizardStepConfirm:
		lines = append(lines, m.viewConfirmStep()...)
	case AuditWizardStepGenerating:
		lines = append(lines, m.viewGeneratingStep()...)
	}

	if m.validationErr != "" {
		lines = append(lines, "")
		lines = append(lines, m.styles.Error.Render(m.validationErr))
	}

	lines = append(lines, "")
	lines = append(lines, m.styles.Help.Render(m.helpText()))

	return strings.Join(lines, "\n")
}

func (m AuditWizardModel) viewModeStep() []string {
	lines := []string{"Choose how to configure this audit run:"}
	for idx, option := range wizardModeOptions {
		prefix := " "
		if idx == m.modeCursor {
			prefix = m.styles.FocusedMark.Render(">")
		}

		line := fmt.Sprintf("%s %s", prefix, option.label)
		if idx == m.modeCursor {
			lines = append(lines, m.styles.Selected.Render(line))
		} else {
			lines = append(lines, m.styles.ListItem.Render(line))
		}
		lines = append(lines, m.styles.Muted.PaddingLeft(4).Render(option.description))
	}

	return lines
}

func (m AuditWizardModel) viewDiscoveryStep() []string {
	if m.discoveryRunning {
		return []string{m.styles.Body.Render(fmt.Sprintf("%s Discovering auditable areas...", m.spinner.View()))}
	}

	return []string{m.styles.Success.Render("Discovery complete.")}
}

func (m AuditWizardModel) viewTypesStep() []string {
	return []string{m.auditTypeSelect.View()}
}

func (m AuditWizardModel) viewAgentStep() []string {
	lines := []string{"Select investigator count:"}
	for idx, count := range wizardAgentOptions {
		prefix := " "
		if idx == m.agentCursor {
			prefix = m.styles.FocusedMark.Render(">")
		}

		line := fmt.Sprintf("%s %d", prefix, count)
		if idx == m.agentCursor {
			lines = append(lines, m.styles.Selected.Render(line))
		} else {
			lines = append(lines, m.styles.ListItem.Render(line))
		}
	}

	return lines
}

func (m AuditWizardModel) viewRigorStep() []string {
	lines := []string{"Select rigor level:"}
	for idx, rigor := range wizardRigorOptions {
		prefix := " "
		if idx == m.rigorCursor {
			prefix = m.styles.FocusedMark.Render(">")
		}

		line := fmt.Sprintf("%s %s (%d loop%s)", prefix, rigor.Label, rigor.Loops, pluralSuffix(rigor.Loops))
		if idx == m.rigorCursor {
			lines = append(lines, m.styles.Selected.Render(line))
		} else {
			lines = append(lines, m.styles.ListItem.Render(line))
		}
	}

	return lines
}

func (m AuditWizardModel) viewConfirmStep() []string {
	selectedTypes := m.SelectedAuditTypes()
	typeNames := make([]string, 0, len(selectedTypes))
	for _, auditType := range selectedTypes {
		typeNames = append(typeNames, auditType.Name)
	}

	discoveryStatus := "n/a"
	if m.mode == WizardModeAutoGenerate {
		discoveryStatus = "opencode"
		if m.discoveryUsedFallback {
			discoveryStatus = "fallback"
		}
	}

	return []string{
		"Confirm launch settings:",
		m.styles.ListItem.Render(fmt.Sprintf("Mode: %s", m.Mode().String())),
		m.styles.ListItem.Render(fmt.Sprintf("Audit types: %s", strings.Join(typeNames, ", "))),
		m.styles.ListItem.Render(fmt.Sprintf("Discovery areas: %d (%s)", len(m.discoveryAreas), discoveryStatus)),
		m.styles.ListItem.Render(fmt.Sprintf("Investigators: %d", m.AgentCount())),
		m.styles.ListItem.Render(fmt.Sprintf("Rigor: %s (%d loop%s)", m.Rigor().Label, m.Rigor().Loops, pluralSuffix(m.Rigor().Loops))),
	}
}

func (m AuditWizardModel) viewGeneratingStep() []string {
	if m.launched {
		return []string{
			m.styles.Success.Render("Generation complete."),
			m.styles.Success.Render("Audit launch request is ready."),
		}
	}

	return []string{m.styles.Body.Render(fmt.Sprintf("%s Generating team and preparing launch...", m.spinner.View()))}
}

func (m AuditWizardModel) helpText() string {
	if m.step == AuditWizardStepTypes {
		return "esc: back • ↑/k: up • ↓/j: down • space: toggle • a: select all • enter: continue"
	}
	if m.step == AuditWizardStepDiscovery {
		return "esc: back • analyzing project structure"
	}

	return "esc: back • ↑/k: up • ↓/j: down • enter: continue"
}

func (m AuditWizardModel) stepLabel() string {
	switch m.step {
	case AuditWizardStepMode:
		return "Step 0/6: Mode"
	case AuditWizardStepDiscovery:
		return "Step 1/6: Discovery"
	case AuditWizardStepTypes:
		return "Step 2/6: Audit Types"
	case AuditWizardStepAgentCount:
		return "Step 3/6: Agent Count"
	case AuditWizardStepRigor:
		return "Step 4/6: Rigor"
	case AuditWizardStepConfirm:
		return "Step 5/6: Confirm"
	case AuditWizardStepGenerating:
		return "Step 6/6: Generating"
	default:
		return "Audit Wizard"
	}
}

func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}

	return "s"
}

func newAuditTypeSelect(selectedIDs map[string]struct{}) MultiSelectModel[teams.AuditType] {
	items := make([]MultiSelectItem[teams.AuditType], 0, len(teams.AuditTypes))
	for _, auditType := range teams.AuditTypes {
		_, selected := selectedIDs[auditType.ID]
		items = append(items, MultiSelectItem[teams.AuditType]{
			Label:       auditType.Name,
			Description: auditType.Description,
			Selected:    selected,
			Value:       auditType,
		})
	}

	return NewMultiSelectModel("Select audit types", items)
}

// Mode returns the selected wizard mode.
func (m AuditWizardModel) Mode() WizardMode {
	return m.mode
}

// SelectedAuditTypes returns selected audit type definitions.
func (m AuditWizardModel) SelectedAuditTypes() []teams.AuditType {
	selectedItems := m.auditTypeSelect.SelectedItems()
	selected := make([]teams.AuditType, 0, len(selectedItems))
	for _, item := range selectedItems {
		selected = append(selected, item.Value)
	}

	return selected
}

// AgentCount returns selected number of investigators.
func (m AuditWizardModel) AgentCount() int {
	return wizardAgentOptions[m.agentCursor]
}

// Rigor returns selected rigor settings.
func (m AuditWizardModel) Rigor() WizardRigor {
	return wizardRigorOptions[m.rigorCursor]
}

// Step returns the active wizard step.
func (m AuditWizardModel) Step() AuditWizardStep {
	return m.step
}

// Launched reports whether generation completed and launch can proceed.
func (m AuditWizardModel) Launched() bool {
	return m.launched
}

// DiscoveredFocusAreas returns discovered area summaries for audit context.
func (m AuditWizardModel) DiscoveredFocusAreas() []string {
	if m.mode != WizardModeAutoGenerate || len(m.discoveryAreas) == 0 {
		return nil
	}

	focus := make([]string, 0, len(m.discoveryAreas))
	for _, area := range m.discoveryAreas {
		focus = append(focus, fmt.Sprintf("%s (%s): %s", area.Name, area.Path, area.Description))
	}

	return focus
}

// String renders wizard modes for summary output.
func (m WizardMode) String() string {
	switch m {
	case WizardModeManual:
		return "Manual"
	case WizardModeAutoGenerate:
		return "Auto-generate"
	default:
		return "Unknown"
	}
}
