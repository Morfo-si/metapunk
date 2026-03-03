package tui

import "github.com/charmbracelet/bubbles/key"

type listKeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Edit        key.Binding
	Reload      key.Binding
	Search      key.Binding
	SwitchFocus key.Binding
	ClearSearch key.Binding
	Quit        key.Binding
	ForceQuit   key.Binding
}

var listKeys = listKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Edit: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "edit"),
	),
	Reload: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "reload"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	SwitchFocus: key.NewBinding(
		key.WithKeys("tab", "shift+tab"),
		key.WithHelp("tab", "switch focus"),
	),
	ClearSearch: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "clear search"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit"),
	),
	ForceQuit: key.NewBinding(
		key.WithKeys("ctrl+c"),
	),
}

type editorKeyMap struct {
	NextField key.Binding
	PrevField key.Binding
	Save      key.Binding
	Cancel    key.Binding
}

var editorKeys = editorKeyMap{
	NextField: key.NewBinding(
		key.WithKeys("tab", "down"),
		key.WithHelp("tab/↓", "next field"),
	),
	PrevField: key.NewBinding(
		key.WithKeys("shift+tab", "up"),
		key.WithHelp("shift+tab/↑", "prev field"),
	),
	Save: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "save"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
}
