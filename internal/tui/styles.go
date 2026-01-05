package tui

import "github.com/charmbracelet/lipgloss"

// Color palette for consistent styling
var (
	// Primary colors
	ColorPrimary   = lipgloss.Color("#7D56F4")
	ColorSecondary = lipgloss.Color("#6B4FD8")

	// Status colors
	ColorSuccess = lipgloss.Color("#50FA7B")
	ColorError   = lipgloss.Color("#FF5F87")
	ColorWarning = lipgloss.Color("#FFB86C")
	ColorInfo    = lipgloss.Color("#8BE9FD")

	// Neutral colors
	ColorMuted   = lipgloss.Color("#6C7086")
	ColorSubtle  = lipgloss.Color("#45475A")
	ColorText    = lipgloss.Color("#CDD6F4")
	ColorTextDim = lipgloss.Color("#A6ADC8")
	ColorBg      = lipgloss.Color("#1E1E2E")
	ColorBgDim   = lipgloss.Color("#181825")

	// Spinner colors
	ColorSpinner = lipgloss.Color("#89B4FA")
)

// Reusable styles
var (
	// Title style for headers
	StyleTitle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(ColorPrimary).
			Padding(0, 1).
			Bold(true)

	// Subtitle style
	StyleSubtitle = lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Italic(true)

	// Success message style
	StyleSuccess = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	// Error message style
	StyleError = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	// Warning message style
	StyleWarning = lipgloss.NewStyle().
			Foreground(ColorWarning)

	// Info message style
	StyleInfo = lipgloss.NewStyle().
			Foreground(ColorInfo)

	// Muted/dimmed text
	StyleMuted = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Highlighted text
	StyleHighlight = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	// Task name style
	StyleTaskName = lipgloss.NewStyle().
			Foreground(ColorText).
			Width(30)

	// Status indicator styles
	StyleStatusPending = lipgloss.NewStyle().
				Foreground(ColorMuted)

	StyleStatusRunning = lipgloss.NewStyle().
				Foreground(ColorSpinner)

	StyleStatusSuccess = lipgloss.NewStyle().
				Foreground(ColorSuccess)

	StyleStatusError = lipgloss.NewStyle().
				Foreground(ColorError)

	// Box styles for sections
	StyleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSubtle).
			Padding(0, 1)

	// Progress bar styles
	StyleProgressFilled = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Background(ColorPrimary)

	StyleProgressEmpty = lipgloss.NewStyle().
				Foreground(ColorSubtle).
				Background(ColorSubtle)
)

// Icons for different states
const (
	IconPending  = "○"
	IconRunning  = "●"
	IconSuccess  = "✓"
	IconError    = "✗"
	IconWarning  = "!"
	IconInfo     = "·"
	IconArrow    = "→"
	IconBullet   = "•"
	IconSpinner1 = "⠋"
	IconSpinner2 = "⠙"
	IconSpinner3 = "⠹"
	IconSpinner4 = "⠸"
	IconSpinner5 = "⠼"
	IconSpinner6 = "⠴"
	IconSpinner7 = "⠦"
	IconSpinner8 = "⠧"
	IconSpinner9 = "⠇"
	IconSpinner0 = "⠏"
)

// SpinnerFrames for animation
var SpinnerFrames = []string{
	IconSpinner1, IconSpinner2, IconSpinner3, IconSpinner4, IconSpinner5,
	IconSpinner6, IconSpinner7, IconSpinner8, IconSpinner9, IconSpinner0,
}

// Dashboard sidebar styles
var (
	StyleSidebarContainer = lipgloss.NewStyle().
				Width(26).
				BorderRight(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(ColorSubtle).
				Padding(1, 0)

	StyleNavItem = lipgloss.NewStyle().
			Padding(0, 2)

	StyleNavItemActive = StyleNavItem.
				Background(ColorPrimary).
				Foreground(lipgloss.Color("#FFFFFF"))

	StyleNavItemHover = StyleNavItem.
				Background(ColorSubtle)

	// StyleNavItemSelected is for the active item when sidebar is not focused
	// Uses a left border accent to clearly indicate it's selected but not keyboard-focused
	StyleNavItemSelected = lipgloss.NewStyle().
				Padding(0, 2).
				BorderLeft(true).
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(ColorPrimary).
				Background(ColorBgDim)
)

// Dashboard form component styles
var (
	StyleFormLabel = lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Width(20)

	StyleFormInput = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorSubtle).
			Padding(0, 1).
			Width(40)

	StyleFormInputFocused = StyleFormInput.
				BorderForeground(ColorPrimary)

	StyleFormInputError = StyleFormInput.
				BorderForeground(ColorError)

	StyleFormHelp = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true)
)

// Dashboard section container styles
var (
	StyleSectionHeader = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true).
				MarginBottom(1)

	StyleSectionContent = lipgloss.NewStyle().
				Padding(1, 2)
)

// Dashboard status bar styles
var (
	StyleStatusBar = lipgloss.NewStyle().
			Background(ColorBgDim).
			Padding(0, 1)

	StyleStatusScope = lipgloss.NewStyle().
				Background(ColorSecondary).
				Foreground(lipgloss.Color("#FFFFFF")).
				Padding(0, 1)

	StyleStatusModified = lipgloss.NewStyle().
				Foreground(ColorWarning)

	StyleStatusSaved = lipgloss.NewStyle().
				Foreground(ColorSuccess)
)

// Dashboard icons
const (
	IconScope  = "⚙"
	IconSave   = "💾"
	IconHelp   = "?"
	IconBack   = "←"
	IconSelect = "↵"
)
