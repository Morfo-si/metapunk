package tui

import "github.com/charmbracelet/lipgloss"

var (
	purple    = lipgloss.Color("#7D56F4")
	subtle    = lipgloss.Color("#383838")
	highlight = lipgloss.Color("#EE6FF8")
	green     = lipgloss.Color("#04B575")
	red       = lipgloss.Color("#FF4672")
	white     = lipgloss.Color("#FAFAFA")
	gray      = lipgloss.Color("#626262")

	titleBarStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(white).
			Background(purple).
			Padding(0, 2).
			MarginBottom(1)

	tableStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(subtle)

	selectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(highlight)

	editorPanelStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(purple).
				Padding(1, 2)

	editorTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(purple).
				MarginBottom(1)

	labelStyle = lipgloss.NewStyle().
			Foreground(gray).
			Width(13)

	focusedInputStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(purple)

	blurredInputStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(subtle)

	searchBarStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(purple).
			Padding(0, 1)

	searchLabelStyle = lipgloss.NewStyle().
				Foreground(purple).
				Bold(true)

	searchCountStyle = lipgloss.NewStyle().
				Foreground(gray).
				Italic(true)

	statusOKStyle = lipgloss.NewStyle().
			Foreground(green).
			Bold(true)

	statusErrStyle = lipgloss.NewStyle().
			Foreground(red).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(gray).
			MarginTop(1)
)
