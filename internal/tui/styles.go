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
