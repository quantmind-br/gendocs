package dashboard

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/tui"
	"github.com/user/gendocs/internal/tui/dashboard/components"
	"github.com/user/gendocs/internal/tui/dashboard/sections"
)

type FocusPane int

const (
	FocusSidebar FocusPane = iota
	FocusContent
)

type DashboardModel struct {
	sidebar   SidebarModel
	statusbar StatusBarModel
	sections  map[string]SectionModel
	modal     components.ModalModel

	cfg    *config.GlobalConfig
	loader *config.Loader
	saver  *config.Saver

	focusPane   FocusPane
	width       int
	height      int
	quitting    bool
	helpVisible bool
	err         error
}

func NewDashboard() DashboardModel {
	sidebar := NewSidebar(DefaultNavItems)
	statusbar := NewStatusBar()
	modal := components.NewConfirmModal(
		"Unsaved Changes",
		"You have unsaved changes. What would you like to do?",
	)

	return DashboardModel{
		sidebar:   sidebar,
		statusbar: statusbar,
		sections:  make(map[string]SectionModel),
		modal:     modal,
		loader:    config.NewLoader(),
		saver:     config.NewSaver(),
		focusPane: FocusSidebar,
	}
}

func (m DashboardModel) Init() tea.Cmd {
	return tea.Batch(
		m.loadConfig(),
		tea.EnterAltScreen,
	)
}

func (m DashboardModel) loadConfig() tea.Cmd {
	return func() tea.Msg {
		cfg, err := m.loader.LoadGlobalConfig()
		if err != nil {
			return ConfigLoadedMsg{Config: &config.GlobalConfig{}, Err: nil}
		}
		return ConfigLoadedMsg{Config: cfg, Err: nil}
	}
}

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.sidebar, _ = m.sidebar.Update(msg)
		m.statusbar, _ = m.statusbar.Update(msg)
		m.modal.SetSize(msg.Width, msg.Height)

		if section := m.activeSection(); section != nil {
			activeID := m.sidebar.ActiveItem().ID
			updated, _ := section.Update(msg)
			m.sections[activeID] = updated.(SectionModel)
		}

	case tea.KeyMsg:
		if m.modal.Visible() {
			var modalCmd tea.Cmd
			m.modal, modalCmd = m.modal.Update(msg)
			if modalCmd != nil {
				cmds = append(cmds, modalCmd)
			}
			return m, tea.Batch(cmds...)
		}

		if m.helpVisible {
			if msg.String() == "?" || msg.String() == "esc" || msg.String() == "q" {
				m.helpVisible = false
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "q":
			if m.hasUnsavedChanges() {
				m.modal.Show()
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit

		case "tab":
			if m.focusPane == FocusSidebar {
				m.focusPane = FocusContent
				m.sidebar.SetFocused(false)
				if section := m.activeSection(); section != nil {
					cmds = append(cmds, section.FocusFirst())
				}
				return m, tea.Batch(cmds...)
			}
			// When in content, let section handle tab for field cycling

		case "shift+tab":
			// Let section handle shift+tab for field cycling backward
			// User can press Esc to go back to sidebar

		case "esc":
			if m.focusPane == FocusContent {
				m.focusPane = FocusSidebar
				m.sidebar.SetFocused(true)
				return m, nil
			}

		case "ctrl+s":
			return m, m.saveConfig()

		case "ctrl+t":
			newScope := ScopeProject
			if m.statusbar.Scope() == ScopeProject {
				newScope = ScopeGlobal
			}
			m.statusbar.SetScope(newScope)
			cmds = append(cmds, func() tea.Msg {
				return SetScopeMsg(newScope)
			})

		case "?":
			m.helpVisible = true
		}

	case components.ModalResultMsg:
		switch msg.Action {
		case components.ModalActionSave:
			cmds = append(cmds, m.saveConfig())
			cmds = append(cmds, func() tea.Msg {
				return quitAfterSaveMsg{}
			})
		case components.ModalActionDiscard:
			m.quitting = true
			return m, tea.Quit
		case components.ModalActionCancel:
		}

	case quitAfterSaveMsg:
		m.quitting = true
		return m, tea.Quit

	case ConfigLoadedMsg:
		if msg.Err != nil {
			m.err = msg.Err
		}
		m.cfg = msg.Config
		m.populateSections()

	case ConfigSavedMsg:
		cmds = append(cmds, func() tea.Msg {
			return ShowMessageMsg{Text: "Configuration saved!", Type: MessageSuccess}
		})
		cmds = append(cmds, func() tea.Msg {
			return SetModifiedMsg(false)
		})

	case ConfigSaveErrorMsg:
		cmds = append(cmds, func() tea.Msg {
			return ShowMessageMsg{Text: "Save failed: " + msg.Err.Error(), Type: MessageError}
		})

	case sections.TestConnectionResultMsg:
		if msg.Success {
			cmds = append(cmds, func() tea.Msg {
				return ShowMessageMsg{Text: msg.Message, Type: MessageSuccess}
			})
		} else {
			cmds = append(cmds, func() tea.Msg {
				return ShowMessageMsg{Text: msg.Message, Type: MessageError}
			})
		}
	}

	var sidebarCmd tea.Cmd
	m.sidebar, sidebarCmd = m.sidebar.Update(msg)
	cmds = append(cmds, sidebarCmd)

	var statusCmd tea.Cmd
	m.statusbar, statusCmd = m.statusbar.Update(msg)
	cmds = append(cmds, statusCmd)

	if m.focusPane == FocusContent {
		if section := m.activeSection(); section != nil {
			activeID := m.sidebar.ActiveItem().ID
			updated, sectionCmd := section.Update(msg)
			m.sections[activeID] = updated.(SectionModel)
			cmds = append(cmds, sectionCmd)

			if updated.(SectionModel).IsDirty() {
				cmds = append(cmds, func() tea.Msg {
					return SetModifiedMsg(true)
				})
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m DashboardModel) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	if m.modal.Visible() {
		return m.modal.View()
	}

	if m.helpVisible {
		return m.renderHelp()
	}

	const minContentWidth = 20
	const minContentHeight = 10
	const sidebarWidth = 28
	const statusbarHeight = 3

	contentWidth := max(m.width-sidebarWidth, minContentWidth)
	contentHeight := max(m.height-statusbarHeight, minContentHeight)

	sidebarView := m.sidebar.View()

	var contentView string
	if section := m.activeSection(); section != nil {
		contentView = tui.StyleSectionContent.
			Width(contentWidth).
			Height(contentHeight).
			Render(section.View())
	} else {
		contentView = m.renderPlaceholder()
	}

	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, contentView)
	statusView := m.statusbar.View()

	return lipgloss.JoinVertical(lipgloss.Left, mainArea, statusView)
}

func (m DashboardModel) renderPlaceholder() string {
	activeItem := m.sidebar.ActiveItem()
	header := tui.StyleSectionHeader.Render(activeItem.Icon + " " + activeItem.Title)
	desc := tui.StyleMuted.Render(activeItem.Description)
	placeholder := tui.StyleMuted.Render("\n\nSection not yet implemented.\nPress Tab to enter content area.")
	return lipgloss.JoinVertical(lipgloss.Left, header, desc, placeholder)
}

func (m DashboardModel) renderHelp() string {
	help := `
╭──────────────────────────────────────────────────────────────╮
│                    Gendocs Configuration                     │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Navigation                                                  │
│  ──────────                                                  │
│  Tab                Enter content / Next field               │
│  Shift+Tab          Previous field                           │
│  Esc                Return to sidebar                        │
│  ↑/↓ or j/k         Navigate sidebar items                   │
│  Enter              Select / Confirm                         │
│                                                              │
│  Actions                                                     │
│  ───────                                                     │
│  Ctrl+S             Save configuration                       │
│  Ctrl+T             Toggle scope (Global/Project)            │
│  Ctrl+U             Toggle password visibility               │
│  ?                  Toggle this help                         │
│  q                  Quit (prompts if unsaved changes)        │
│                                                              │
╰──────────────────────────────────────────────────────────────╯

                     Press ? or Esc to close help
`
	return tui.StyleBox.Render(help)
}

func (m DashboardModel) activeSection() SectionModel {
	activeID := m.sidebar.ActiveItem().ID
	if section, ok := m.sections[activeID]; ok {
		return section
	}
	return nil
}

func (m DashboardModel) hasUnsavedChanges() bool {
	for _, section := range m.sections {
		if section.IsDirty() {
			return true
		}
	}
	return false
}

func (m DashboardModel) saveConfig() tea.Cmd {
	return func() tea.Msg {
		for _, section := range m.sections {
			values := section.GetValues()
			m.applyValuesToConfig(values)
		}

		var err error
		if m.statusbar.Scope() == ScopeGlobal {
			err = m.saver.SaveGlobalConfig(m.cfg)
		} else {
			err = m.saver.SaveProjectConfig(".", m.cfg)
		}

		if err != nil {
			return ConfigSaveErrorMsg{Err: err}
		}
		return ConfigSavedMsg{}
	}
}

func (m *DashboardModel) populateSections() {
	if m.cfg == nil {
		return
	}

	if section, ok := m.sections["llm"]; ok {
		_ = section.SetValues(map[string]any{
			"provider":    m.cfg.Analyzer.LLM.Provider,
			"model":       m.cfg.Analyzer.LLM.Model,
			"api_key":     m.cfg.Analyzer.LLM.APIKey,
			"base_url":    m.cfg.Analyzer.LLM.BaseURL,
			"temperature": m.cfg.Analyzer.LLM.Temperature,
			"max_tokens":  m.cfg.Analyzer.LLM.MaxTokens,
			"timeout":     m.cfg.Analyzer.LLM.Timeout,
			"retries":     m.cfg.Analyzer.LLM.Retries,
		})
	}

	if section, ok := m.sections["documenter_llm"]; ok {
		_ = section.SetValues(map[string]any{
			"documenter_provider":    m.cfg.Documenter.LLM.Provider,
			"documenter_model":       m.cfg.Documenter.LLM.Model,
			"documenter_api_key":     m.cfg.Documenter.LLM.APIKey,
			"documenter_base_url":    m.cfg.Documenter.LLM.BaseURL,
			"documenter_temperature": m.cfg.Documenter.LLM.Temperature,
			"documenter_max_tokens":  m.cfg.Documenter.LLM.MaxTokens,
			"documenter_timeout":     m.cfg.Documenter.LLM.Timeout,
			"documenter_retries":     m.cfg.Documenter.LLM.Retries,
		})
	}

	if section, ok := m.sections["ai_rules_llm"]; ok {
		_ = section.SetValues(map[string]any{
			"ai_rules_provider":    m.cfg.AIRules.LLM.Provider,
			"ai_rules_model":       m.cfg.AIRules.LLM.Model,
			"ai_rules_api_key":     m.cfg.AIRules.LLM.APIKey,
			"ai_rules_base_url":    m.cfg.AIRules.LLM.BaseURL,
			"ai_rules_temperature": m.cfg.AIRules.LLM.Temperature,
			"ai_rules_max_tokens":  m.cfg.AIRules.LLM.MaxTokens,
			"ai_rules_timeout":     m.cfg.AIRules.LLM.Timeout,
			"ai_rules_retries":     m.cfg.AIRules.LLM.Retries,
		})
	}

	if section, ok := m.sections["cache"]; ok {
		section.SetValues(map[string]any{
			"cache_enabled":  m.cfg.Analyzer.LLM.Cache.Enabled,
			"cache_max_size": m.cfg.Analyzer.LLM.Cache.MaxSize,
			"cache_ttl":      m.cfg.Analyzer.LLM.Cache.TTL,
			"cache_path":     m.cfg.Analyzer.LLM.Cache.CachePath,
		})
	}

	if section, ok := m.sections["analysis"]; ok {
		section.SetValues(map[string]any{
			"exclude_code_structure": m.cfg.Analyzer.ExcludeStructure,
			"exclude_data_flow":      m.cfg.Analyzer.ExcludeDataFlow,
			"exclude_dependencies":   m.cfg.Analyzer.ExcludeDeps,
			"exclude_request_flow":   m.cfg.Analyzer.ExcludeReqFlow,
			"exclude_api_analysis":   m.cfg.Analyzer.ExcludeAPI,
			"max_workers":            m.cfg.Analyzer.MaxWorkers,
			"max_hash_workers":       m.cfg.Analyzer.MaxHashWorkers,
			"force":                  m.cfg.Analyzer.Force,
			"incremental":            m.cfg.Analyzer.Incremental,
		})
	}

	if section, ok := m.sections["retry"]; ok {
		section.SetValues(map[string]any{
			"max_attempts":         m.cfg.Analyzer.RetryConfig.MaxAttempts,
			"multiplier":           m.cfg.Analyzer.RetryConfig.Multiplier,
			"max_wait_per_attempt": m.cfg.Analyzer.RetryConfig.MaxWaitPerAttempt,
			"max_total_wait":       m.cfg.Analyzer.RetryConfig.MaxTotalWait,
		})
	}

	if section, ok := m.sections["gemini"]; ok {
		section.SetValues(map[string]any{
			"use_vertex_ai": m.cfg.Gemini.UseVertexAI,
			"project_id":    m.cfg.Gemini.ProjectID,
			"location":      m.cfg.Gemini.Location,
		})
	}

	if section, ok := m.sections["gitlab"]; ok {
		section.SetValues(map[string]any{
			"gitlab_api_url":       m.cfg.GitLab.APIURL,
			"gitlab_user_name":     m.cfg.GitLab.UserName,
			"gitlab_user_username": m.cfg.GitLab.UserUsername,
			"gitlab_user_email":    m.cfg.GitLab.UserEmail,
			"gitlab_oauth_token":   m.cfg.GitLab.OAuthToken,
		})
	}

	if section, ok := m.sections["cronjob"]; ok {
		section.SetValues(map[string]any{
			"max_days_since_last_commit": m.cfg.Cronjob.MaxDaysSinceLastCommit,
			"working_path":               m.cfg.Cronjob.WorkingPath,
			"group_project_id":           m.cfg.Cronjob.GroupProjectID,
		})
	}

	if section, ok := m.sections["logging"]; ok {
		section.SetValues(map[string]any{
			"log_dir":       m.cfg.Logging.LogDir,
			"file_level":    m.cfg.Logging.FileLevel,
			"console_level": m.cfg.Logging.ConsoleLevel,
		})
	}
}

func (m *DashboardModel) applyValuesToConfig(values map[string]any) {
	if m.cfg == nil {
		return
	}

	if v, ok := values["provider"].(string); ok && v != "" {
		m.cfg.Analyzer.LLM.Provider = v
	}
	if v, ok := values["model"].(string); ok && v != "" {
		m.cfg.Analyzer.LLM.Model = v
	}
	if v, ok := values["api_key"].(string); ok && v != "" {
		m.cfg.Analyzer.LLM.APIKey = v
	}
	if v, ok := values["base_url"].(string); ok {
		m.cfg.Analyzer.LLM.BaseURL = v
	}
	if v, ok := values["temperature"].(float64); ok {
		m.cfg.Analyzer.LLM.Temperature = v
	}
	if v, ok := values["max_tokens"].(int); ok && v > 0 {
		m.cfg.Analyzer.LLM.MaxTokens = v
	}
	if v, ok := values["timeout"].(int); ok && v > 0 {
		m.cfg.Analyzer.LLM.Timeout = v
	}
	if v, ok := values["retries"].(int); ok {
		m.cfg.Analyzer.LLM.Retries = v
	}

	if v, ok := values["documenter_provider"].(string); ok && v != "" {
		m.cfg.Documenter.LLM.Provider = v
	}
	if v, ok := values["documenter_model"].(string); ok && v != "" {
		m.cfg.Documenter.LLM.Model = v
	}
	if v, ok := values["documenter_api_key"].(string); ok && v != "" {
		m.cfg.Documenter.LLM.APIKey = v
	}
	if v, ok := values["documenter_base_url"].(string); ok {
		m.cfg.Documenter.LLM.BaseURL = v
	}
	if v, ok := values["documenter_temperature"].(float64); ok {
		m.cfg.Documenter.LLM.Temperature = v
	}
	if v, ok := values["documenter_max_tokens"].(int); ok && v > 0 {
		m.cfg.Documenter.LLM.MaxTokens = v
	}
	if v, ok := values["documenter_timeout"].(int); ok && v > 0 {
		m.cfg.Documenter.LLM.Timeout = v
	}
	if v, ok := values["documenter_retries"].(int); ok {
		m.cfg.Documenter.LLM.Retries = v
	}

	if v, ok := values["ai_rules_provider"].(string); ok && v != "" {
		m.cfg.AIRules.LLM.Provider = v
	}
	if v, ok := values["ai_rules_model"].(string); ok && v != "" {
		m.cfg.AIRules.LLM.Model = v
	}
	if v, ok := values["ai_rules_api_key"].(string); ok && v != "" {
		m.cfg.AIRules.LLM.APIKey = v
	}
	if v, ok := values["ai_rules_base_url"].(string); ok {
		m.cfg.AIRules.LLM.BaseURL = v
	}
	if v, ok := values["ai_rules_temperature"].(float64); ok {
		m.cfg.AIRules.LLM.Temperature = v
	}
	if v, ok := values["ai_rules_max_tokens"].(int); ok && v > 0 {
		m.cfg.AIRules.LLM.MaxTokens = v
	}
	if v, ok := values["ai_rules_timeout"].(int); ok && v > 0 {
		m.cfg.AIRules.LLM.Timeout = v
	}
	if v, ok := values["ai_rules_retries"].(int); ok {
		m.cfg.AIRules.LLM.Retries = v
	}

	if v, ok := values["cache_enabled"].(bool); ok {
		m.cfg.Analyzer.LLM.Cache.Enabled = v
	}
	if v, ok := values["cache_max_size"].(int); ok && v > 0 {
		m.cfg.Analyzer.LLM.Cache.MaxSize = v
	}
	if v, ok := values["cache_ttl"].(int); ok && v > 0 {
		m.cfg.Analyzer.LLM.Cache.TTL = v
	}
	if v, ok := values["cache_path"].(string); ok {
		m.cfg.Analyzer.LLM.Cache.CachePath = v
	}

	if v, ok := values["exclude_code_structure"].(bool); ok {
		m.cfg.Analyzer.ExcludeStructure = v
	}
	if v, ok := values["exclude_data_flow"].(bool); ok {
		m.cfg.Analyzer.ExcludeDataFlow = v
	}
	if v, ok := values["exclude_dependencies"].(bool); ok {
		m.cfg.Analyzer.ExcludeDeps = v
	}
	if v, ok := values["exclude_request_flow"].(bool); ok {
		m.cfg.Analyzer.ExcludeReqFlow = v
	}
	if v, ok := values["exclude_api_analysis"].(bool); ok {
		m.cfg.Analyzer.ExcludeAPI = v
	}
	if v, ok := values["max_workers"].(int); ok {
		m.cfg.Analyzer.MaxWorkers = v
	}
	if v, ok := values["max_hash_workers"].(int); ok {
		m.cfg.Analyzer.MaxHashWorkers = v
	}
	if v, ok := values["force"].(bool); ok {
		m.cfg.Analyzer.Force = v
	}
	if v, ok := values["incremental"].(bool); ok {
		m.cfg.Analyzer.Incremental = v
	}

	if v, ok := values["max_attempts"].(int); ok && v > 0 {
		m.cfg.Analyzer.RetryConfig.MaxAttempts = v
	}
	if v, ok := values["multiplier"].(int); ok && v > 0 {
		m.cfg.Analyzer.RetryConfig.Multiplier = v
	}
	if v, ok := values["max_wait_per_attempt"].(int); ok && v > 0 {
		m.cfg.Analyzer.RetryConfig.MaxWaitPerAttempt = v
	}
	if v, ok := values["max_total_wait"].(int); ok && v > 0 {
		m.cfg.Analyzer.RetryConfig.MaxTotalWait = v
	}

	if v, ok := values["use_vertex_ai"].(bool); ok {
		m.cfg.Gemini.UseVertexAI = v
	}
	if v, ok := values["project_id"].(string); ok {
		m.cfg.Gemini.ProjectID = v
	}
	if v, ok := values["location"].(string); ok {
		m.cfg.Gemini.Location = v
	}

	if v, ok := values["gitlab_api_url"].(string); ok {
		m.cfg.GitLab.APIURL = v
	}
	if v, ok := values["gitlab_user_name"].(string); ok {
		m.cfg.GitLab.UserName = v
	}
	if v, ok := values["gitlab_user_username"].(string); ok {
		m.cfg.GitLab.UserUsername = v
	}
	if v, ok := values["gitlab_user_email"].(string); ok {
		m.cfg.GitLab.UserEmail = v
	}
	if v, ok := values["gitlab_oauth_token"].(string); ok {
		m.cfg.GitLab.OAuthToken = v
	}

	if v, ok := values["max_days_since_last_commit"].(int); ok && v > 0 {
		m.cfg.Cronjob.MaxDaysSinceLastCommit = v
	}
	if v, ok := values["working_path"].(string); ok {
		m.cfg.Cronjob.WorkingPath = v
	}
	if v, ok := values["group_project_id"].(int); ok {
		m.cfg.Cronjob.GroupProjectID = v
	}

	if v, ok := values["log_dir"].(string); ok {
		m.cfg.Logging.LogDir = v
	}
	if v, ok := values["file_level"].(string); ok {
		m.cfg.Logging.FileLevel = v
	}
	if v, ok := values["console_level"].(string); ok {
		m.cfg.Logging.ConsoleLevel = v
	}
}

func (m DashboardModel) HasError() bool {
	return m.err != nil
}

func (m DashboardModel) Error() error {
	return m.err
}

func (m *DashboardModel) RegisterSection(id string, section SectionModel) {
	m.sections[id] = section
}

type ConfigLoadedMsg struct {
	Config *config.GlobalConfig
	Err    error
}

type ConfigSavedMsg struct{}

type ConfigSaveErrorMsg struct {
	Err error
}

type quitAfterSaveMsg struct{}

func ShowError(text string) tea.Cmd {
	return func() tea.Msg {
		return ShowMessageMsg{Text: text, Type: MessageError}
	}
}

func ShowSuccess(text string) tea.Cmd {
	return func() tea.Msg {
		return ShowMessageMsg{Text: text, Type: MessageSuccess}
	}
}

func ShowInfo(text string) tea.Cmd {
	return func() tea.Msg {
		return ShowMessageMsg{Text: text, Type: MessageInfo}
	}
}
