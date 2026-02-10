package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"lattice/internal/teams"
)

// AuditWizardStep identifies the current stage in the audit setup flow.
type AuditWizardStep int

const (
	AuditWizardStepMode AuditWizardStep = iota
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

type auditWizardLaunchMsg struct{}

// AuditWizardModel provides a six-step configuration flow for launching audits.
type AuditWizardModel struct {
	styles Styles
	keyMap KeyMap

	step            AuditWizardStep
	modeCursor      int
	mode            WizardMode
	auditTypeSelect MultiSelectModel[teams.AuditType]
	agentCursor     int
	rigorCursor     int

	spinner     spinner.Model
	launchDelay time.Duration
	launched    bool

	validationErr string
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
		styles:      DefaultStyles(),
		keyMap:      DefaultKeyMap(),
		step:        AuditWizardStepMode,
		mode:        WizardModeManual,
		spinner:     s,
		launchDelay: 900 * time.Millisecond,
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

// SetLaunchDelay overrides the spinner wait before reporting launch completion.
func (m AuditWizardModel) SetLaunchDelay(delay time.Duration) AuditWizardModel {
	m.launchDelay = delay
	return m
}

// Update handles key input and timer messages.
func (m AuditWizardModel) Update(msg tea.Msg) (AuditWizardModel, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(typed, m.keyMap.Back) {
			m.validationErr = ""
			if m.step > AuditWizardStepMode {
				m.step--
				m.launched = false
			}
			return m, nil
		}

		if m.step == AuditWizardStepGenerating {
			return m, nil
		}

		switch m.step {
		case AuditWizardStepMode:
			return m.updateStepMode(typed)
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
		if m.step == AuditWizardStepGenerating && !m.launched {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	case auditWizardLaunchMsg:
		if m.step == AuditWizardStepGenerating {
			m.launched = true
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
		m.step = AuditWizardStepTypes
	}

	return m, nil
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

	return m, tea.Batch(
		m.spinner.Tick,
		tea.Tick(m.launchDelay, func(time.Time) tea.Msg {
			return auditWizardLaunchMsg{}
		}),
	)
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

	return []string{
		"Confirm launch settings:",
		m.styles.ListItem.Render(fmt.Sprintf("Mode: %s", m.Mode().String())),
		m.styles.ListItem.Render(fmt.Sprintf("Audit types: %s", strings.Join(typeNames, ", "))),
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

	return "esc: back • ↑/k: up • ↓/j: down • enter: continue"
}

func (m AuditWizardModel) stepLabel() string {
	switch m.step {
	case AuditWizardStepMode:
		return "Step 0/5: Mode"
	case AuditWizardStepTypes:
		return "Step 1/5: Audit Types"
	case AuditWizardStepAgentCount:
		return "Step 2/5: Agent Count"
	case AuditWizardStepRigor:
		return "Step 3/5: Rigor"
	case AuditWizardStepConfirm:
		return "Step 4/5: Confirm"
	case AuditWizardStepGenerating:
		return "Step 5/5: Generating"
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
