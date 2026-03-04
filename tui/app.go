package tui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type viewState int

const (
	listView viewState = iota
	editorView
	readerView
)

// AppModel is the root model that owns all child models and routes messages.
type AppModel struct {
	state  viewState
	list   ListModel
	editor EditorModel
	reader ReaderModel
	width  int
	height int
}

// NewAppModel creates the root model, scanning the current working directory.
func NewAppModel() AppModel {
	dir, _ := os.Getwd()
	return AppModel{
		state: listView,
		list:  NewListModel(dir),
	}
}

func (a AppModel) Init() tea.Cmd {
	return a.list.Init()
}

func (a AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Track terminal size for initialising the reader at the right dimensions.
	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		a.width = sz.Width
		a.height = sz.Height
	}

	switch a.state {
	case listView:
		switch msg := msg.(type) {
		case editMsg:
			a.editor = NewEditorModel(msg.metadata)
			a.state = editorView
			return a, a.editor.Init()
		case openReaderMsg:
			a.reader = NewReaderModel(msg.metadata, a.width, a.height)
			a.state = readerView
			return a, a.reader.Init()
		default:
			var cmd tea.Cmd
			a.list, cmd = a.list.Update(msg)
			return a, cmd
		}

	case editorView:
		switch msg := msg.(type) {
		case cancelMsg:
			a.state = listView
			return a, nil
		case savedMsg:
			a.state = listView
			// Forward savedMsg to list so it can update its table.
			var cmd tea.Cmd
			a.list, cmd = a.list.Update(msg)
			return a, cmd
		default:
			var cmd tea.Cmd
			a.editor, cmd = a.editor.Update(msg)
			return a, cmd
		}

	case readerView:
		switch msg.(type) {
		case backToListMsg:
			a.state = listView
			return a, nil
		default:
			var cmd tea.Cmd
			a.reader, cmd = a.reader.Update(msg)
			return a, cmd
		}
	}

	return a, nil
}

func (a AppModel) View() string {
	switch a.state {
	case editorView:
		return a.editor.View()
	case readerView:
		return a.reader.View()
	default:
		return a.list.View()
	}
}
