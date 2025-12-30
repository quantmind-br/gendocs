package sections

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/gendocs/internal/tui"
	"github.com/user/gendocs/internal/tui/dashboard/components"
	"github.com/user/gendocs/internal/tui/dashboard/types"
)

type GeminiSectionModel struct {
	useVertexAI components.ToggleModel
	projectID   components.TextFieldModel
	location    components.TextFieldModel

	focusIndex int
}

func NewGeminiSection() *GeminiSectionModel {
	return &GeminiSectionModel{
		useVertexAI: components.NewToggle("Use Vertex AI", "Use Google Cloud Vertex AI instead of direct Gemini API"),
		projectID: components.NewTextField("Project ID",
			components.WithPlaceholder("my-gcp-project"),
			components.WithHelp("Required when using Vertex AI")),
		location: components.NewTextField("Location",
			components.WithPlaceholder("us-central1"),
			components.WithHelp("GCP region for Vertex AI")),
	}
}

func (m *GeminiSectionModel) Title() string       { return "Gemini / Vertex AI" }
func (m *GeminiSectionModel) Icon() string        { return "☁️" }
func (m *GeminiSectionModel) Description() string { return "Configure Google Cloud Gemini settings" }

func (m *GeminiSectionModel) Init() tea.Cmd { return nil }

func (m *GeminiSectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.blurCurrent()
			m.focusIndex = (m.focusIndex + 1) % 3
			cmds = append(cmds, m.focusCurrent())
			return m, tea.Batch(cmds...)

		case "shift+tab":
			m.blurCurrent()
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = 2
			}
			cmds = append(cmds, m.focusCurrent())
			return m, tea.Batch(cmds...)
		}
	}

	switch m.focusIndex {
	case 0:
		m.useVertexAI, _ = m.useVertexAI.Update(msg)
	case 1:
		m.projectID, _ = m.projectID.Update(msg)
	case 2:
		m.location, _ = m.location.Update(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m *GeminiSectionModel) blurCurrent() {
	switch m.focusIndex {
	case 0:
		m.useVertexAI.Blur()
	case 1:
		m.projectID.Blur()
	case 2:
		m.location.Blur()
	}
}

func (m *GeminiSectionModel) focusCurrent() tea.Cmd {
	switch m.focusIndex {
	case 0:
		return m.useVertexAI.Focus()
	case 1:
		return m.projectID.Focus()
	case 2:
		return m.location.Focus()
	}
	return nil
}

func (m *GeminiSectionModel) View() string {
	header := tui.StyleSectionHeader.Render(m.Icon() + " " + m.Title())
	desc := tui.StyleMuted.Render(m.Description())

	var vertexNote string
	if m.useVertexAI.Value() {
		vertexNote = tui.StyleInfo.Render("Vertex AI mode: Project ID and Location are required")
	}

	fields := lipgloss.JoinVertical(lipgloss.Left,
		m.useVertexAI.View(),
		vertexNote,
		"",
		m.projectID.View(),
		"",
		m.location.View(),
	)

	return lipgloss.JoinVertical(lipgloss.Left, header, desc, "", fields)
}

func (m *GeminiSectionModel) Validate() []types.ValidationError {
	var errors []types.ValidationError

	if m.useVertexAI.Value() {
		if m.projectID.Value() == "" {
			errors = append(errors, types.ValidationError{
				Field:    "Project ID",
				Message:  "Project ID is required when using Vertex AI",
				Severity: types.SeverityError,
			})
		}
		if m.location.Value() == "" {
			errors = append(errors, types.ValidationError{
				Field:    "Location",
				Message:  "Location is required when using Vertex AI",
				Severity: types.SeverityError,
			})
		}
	}

	return errors
}

func (m *GeminiSectionModel) IsDirty() bool {
	return m.useVertexAI.IsDirty() || m.projectID.IsDirty() || m.location.IsDirty()
}

func (m *GeminiSectionModel) GetValues() map[string]any {
	return map[string]any{
		"use_vertex_ai": m.useVertexAI.Value(),
		"project_id":    m.projectID.Value(),
		"location":      m.location.Value(),
	}
}

func (m *GeminiSectionModel) SetValues(values map[string]any) error {
	if v, ok := values["use_vertex_ai"].(bool); ok {
		m.useVertexAI.SetValue(v)
	}
	if v, ok := values["project_id"].(string); ok {
		m.projectID.SetValue(v)
	}
	if v, ok := values["location"].(string); ok {
		m.location.SetValue(v)
	}
	return nil
}

func (m *GeminiSectionModel) FocusFirst() tea.Cmd {
	m.blurAll()
	m.focusIndex = 0
	return m.useVertexAI.Focus()
}

func (m *GeminiSectionModel) FocusLast() tea.Cmd {
	m.blurAll()
	m.focusIndex = 2
	return m.location.Focus()
}

func (m *GeminiSectionModel) blurAll() {
	m.useVertexAI.Blur()
	m.projectID.Blur()
	m.location.Blur()
}
