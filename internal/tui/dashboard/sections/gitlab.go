package sections

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/gendocs/internal/tui"
	"github.com/user/gendocs/internal/tui/dashboard/components"
	"github.com/user/gendocs/internal/tui/dashboard/types"
	"github.com/user/gendocs/internal/tui/dashboard/validation"
)

type GitLabSectionModel struct {
	apiURL       components.TextFieldModel
	userName     components.TextFieldModel
	userUsername components.TextFieldModel
	userEmail    components.TextFieldModel
	oauthToken   components.MaskedInputModel

	focusIndex int
}

func NewGitLabSection() *GitLabSectionModel {
	return &GitLabSectionModel{
		apiURL: components.NewTextField("API URL",
			components.WithPlaceholder("https://gitlab.com/api/v4"),
			components.WithValidator(validation.ValidateURL()),
			components.WithHelp("GitLab API endpoint")),
		userName: components.NewTextField("User Name",
			components.WithPlaceholder("John Doe"),
			components.WithHelp("Display name for commits")),
		userUsername: components.NewTextField("Username",
			components.WithPlaceholder("johndoe"),
			components.WithHelp("GitLab username")),
		userEmail: components.NewTextField("Email",
			components.WithPlaceholder("john@example.com"),
			components.WithValidator(validation.ValidateEmail()),
			components.WithHelp("Email for commits")),
		oauthToken: components.NewMaskedInput("OAuth Token", "GitLab personal access token"),
	}
}

func (m *GitLabSectionModel) Title() string       { return "GitLab Integration" }
func (m *GitLabSectionModel) Icon() string        { return "🦊" }
func (m *GitLabSectionModel) Description() string { return "Configure GitLab API access" }

func (m *GitLabSectionModel) Init() tea.Cmd { return nil }

func (m *GitLabSectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.blurCurrent()
			m.focusIndex = (m.focusIndex + 1) % 5
			cmds = append(cmds, m.focusCurrent())
			return m, tea.Batch(cmds...)

		case "shift+tab":
			m.blurCurrent()
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = 4
			}
			cmds = append(cmds, m.focusCurrent())
			return m, tea.Batch(cmds...)
		}
	}

	switch m.focusIndex {
	case 0:
		m.apiURL, _ = m.apiURL.Update(msg)
	case 1:
		m.userName, _ = m.userName.Update(msg)
	case 2:
		m.userUsername, _ = m.userUsername.Update(msg)
	case 3:
		m.userEmail, _ = m.userEmail.Update(msg)
	case 4:
		m.oauthToken, _ = m.oauthToken.Update(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m *GitLabSectionModel) blurCurrent() {
	switch m.focusIndex {
	case 0:
		m.apiURL.Blur()
	case 1:
		m.userName.Blur()
	case 2:
		m.userUsername.Blur()
	case 3:
		m.userEmail.Blur()
	case 4:
		m.oauthToken.Blur()
	}
}

func (m *GitLabSectionModel) focusCurrent() tea.Cmd {
	switch m.focusIndex {
	case 0:
		return m.apiURL.Focus()
	case 1:
		return m.userName.Focus()
	case 2:
		return m.userUsername.Focus()
	case 3:
		return m.userEmail.Focus()
	case 4:
		return m.oauthToken.Focus()
	}
	return nil
}

func (m *GitLabSectionModel) View() string {
	header := tui.StyleSectionHeader.Render(m.Icon() + " " + m.Title())
	desc := tui.StyleMuted.Render(m.Description())

	fields := lipgloss.JoinVertical(lipgloss.Left,
		m.apiURL.View(),
		"",
		m.userName.View(),
		"",
		m.userUsername.View(),
		"",
		m.userEmail.View(),
		"",
		m.oauthToken.View(),
	)

	return lipgloss.JoinVertical(lipgloss.Left, header, desc, "", fields)
}

func (m *GitLabSectionModel) Validate() []types.ValidationError {
	return nil
}

func (m *GitLabSectionModel) IsDirty() bool {
	return m.apiURL.IsDirty() || m.userName.IsDirty() || m.userUsername.IsDirty() ||
		m.userEmail.IsDirty() || m.oauthToken.IsDirty()
}

func (m *GitLabSectionModel) GetValues() map[string]any {
	return map[string]any{
		KeyGitLabAPIURL:       m.apiURL.Value(),
		KeyGitLabUserName:     m.userName.Value(),
		KeyGitLabUserUsername: m.userUsername.Value(),
		KeyGitLabUserEmail:    m.userEmail.Value(),
		KeyGitLabOAuthToken:   m.oauthToken.Value(),
	}
}

func (m *GitLabSectionModel) SetValues(values map[string]any) error {
	if v, ok := values[KeyGitLabAPIURL].(string); ok {
		m.apiURL.SetValue(v)
	}
	if v, ok := values[KeyGitLabUserName].(string); ok {
		m.userName.SetValue(v)
	}
	if v, ok := values[KeyGitLabUserUsername].(string); ok {
		m.userUsername.SetValue(v)
	}
	if v, ok := values[KeyGitLabUserEmail].(string); ok {
		m.userEmail.SetValue(v)
	}
	if v, ok := values[KeyGitLabOAuthToken].(string); ok {
		m.oauthToken.SetValue(v)
	}
	return nil
}

func (m *GitLabSectionModel) FocusFirst() tea.Cmd {
	m.blurAll()
	m.focusIndex = 0
	return m.apiURL.Focus()
}

func (m *GitLabSectionModel) FocusLast() tea.Cmd {
	m.blurAll()
	m.focusIndex = 4
	return m.oauthToken.Focus()
}

func (m *GitLabSectionModel) blurAll() {
	m.apiURL.Blur()
	m.userName.Blur()
	m.userUsername.Blur()
	m.userEmail.Blur()
	m.oauthToken.Blur()
}
