package tui

import "github.com/charmbracelet/bubbles/key"

type listKeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Edit        key.Binding
	Open        key.Binding
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
	Open: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "read"),
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

type readerKeyMap struct {
	ScrollDown     key.Binding
	ScrollUp       key.Binding
	LineDown       key.Binding
	LineUp         key.Binding
	NextChapter    key.Binding
	PrevChapter    key.Binding
	Search         key.Binding
	ConfirmSearch  key.Binding
	NextMatch      key.Binding
	PrevMatch      key.Binding
	ClearSearch    key.Binding
	AddBookmark    key.Binding
	ShowBookmarks  key.Binding
	DeleteBookmark key.Binding
	Back           key.Binding
	ForceQuit      key.Binding
}

var readerKeys = readerKeyMap{
	ScrollDown:     key.NewBinding(key.WithKeys("space", "pgdown"), key.WithHelp("space/pgdn", "scroll down")),
	ScrollUp:       key.NewBinding(key.WithKeys("b", "pgup"), key.WithHelp("b/pgup", "scroll up")),
	LineDown:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "line down")),
	LineUp:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "line up")),
	NextChapter:    key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "next chapter")),
	PrevChapter:    key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "prev chapter")),
	Search:         key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
	ConfirmSearch:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
	NextMatch:      key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next match")),
	PrevMatch:      key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "prev match")),
	ClearSearch:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear/back")),
	AddBookmark:    key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "add bookmark")),
	ShowBookmarks:  key.NewBinding(key.WithKeys("M"), key.WithHelp("M", "bookmarks")),
	DeleteBookmark: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
	Back:           key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "back")),
	ForceQuit:      key.NewBinding(key.WithKeys("ctrl+c")),
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
