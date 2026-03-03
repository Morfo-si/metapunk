package tui

import (
	"path/filepath"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/morfo-si/metapunk/epub"
)

// savedMsg is sent when a save completes successfully.
type savedMsg struct {
	metadata epub.Metadata
}

// cancelMsg is sent when the user cancels the editor.
type cancelMsg struct{}

// saveErrMsg is sent when a save fails.
type saveErrMsg struct {
	err error
}

const (
	fieldTitle = iota
	fieldAuthor
	numFields
)

// EditorModel is the metadata editing form.
type EditorModel struct {
	original epub.Metadata
	inputs   [numFields]textinput.Model
	focused  int
	saving   bool
	errMsg   string
}

func NewEditorModel(m epub.Metadata) EditorModel {
	titleInput := textinput.New()
	titleInput.Placeholder = "Book title"
	titleInput.SetValue(m.Title)
	titleInput.Focus()
	titleInput.Width = 50
	titleInput.Prompt = ""

	authorInput := textinput.New()
	authorInput.Placeholder = "Author name"
	authorInput.SetValue(m.Author)
	authorInput.Width = 50
	authorInput.Prompt = ""

	return EditorModel{
		original: m,
		inputs:   [numFields]textinput.Model{titleInput, authorInput},
		focused:  fieldTitle,
	}
}

func (e EditorModel) Init() tea.Cmd {
	return textinput.Blink
}

func (e EditorModel) Update(msg tea.Msg) (EditorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return e, func() tea.Msg { return cancelMsg{} }

		case "ctrl+s":
			if e.saving {
				return e, nil
			}
			e.saving = true
			e.errMsg = ""
			updated := epub.Metadata{
				FilePath: e.original.FilePath,
				Title:    e.inputs[fieldTitle].Value(),
				Author:   e.inputs[fieldAuthor].Value(),
			}
			return e, saveCmd(updated)

		case "tab", "down":
			e.focused = (e.focused + 1) % numFields
			return e, e.syncFocus()

		case "shift+tab", "up":
			e.focused = (e.focused - 1 + numFields) % numFields
			return e, e.syncFocus()
		}

	case saveErrMsg:
		e.saving = false
		e.errMsg = msg.err.Error()
		return e, nil
	}

	// Forward remaining key events to the focused input
	var cmd tea.Cmd
	e.inputs[e.focused], cmd = e.inputs[e.focused].Update(msg)
	return e, cmd
}

func (e EditorModel) View() string {
	filename := filepath.Base(e.original.FilePath)

	heading := editorTitleStyle.Render("Editing: " + filename)

	titleLabel := labelStyle.Render("Title")
	authorLabel := labelStyle.Render("Author")

	titleBox := inputBox(e.inputs[fieldTitle], e.focused == fieldTitle)
	authorBox := inputBox(e.inputs[fieldAuthor], e.focused == fieldAuthor)

	titleRow := lipgloss.JoinHorizontal(lipgloss.Center, titleLabel, titleBox)
	authorRow := lipgloss.JoinHorizontal(lipgloss.Center, authorLabel, authorBox)

	var statusLine string
	if e.saving {
		statusLine = helpStyle.Render("Saving…")
	} else if e.errMsg != "" {
		statusLine = statusErrStyle.Render("✗ " + e.errMsg)
	}

	help := helpStyle.Render("tab next field  shift+tab prev  ctrl+s save  esc cancel")

	inner := lipgloss.JoinVertical(lipgloss.Left,
		heading,
		titleRow,
		authorRow,
	)
	if statusLine != "" {
		inner = lipgloss.JoinVertical(lipgloss.Left, inner, statusLine)
	}
	inner = lipgloss.JoinVertical(lipgloss.Left, inner, help)

	return editorPanelStyle.Render(inner)
}

// syncFocus updates focus state across all inputs.
func (e *EditorModel) syncFocus() tea.Cmd {
	var cmds []tea.Cmd
	for i := range e.inputs {
		if i == e.focused {
			cmds = append(cmds, e.inputs[i].Focus())
		} else {
			e.inputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

// inputBox renders a textinput with the appropriate border style.
func inputBox(ti textinput.Model, focused bool) string {
	style := blurredInputStyle
	if focused {
		style = focusedInputStyle
	}
	return style.Render(ti.View())
}

// saveCmd is a tea.Cmd that writes EPUB metadata and returns a result message.
func saveCmd(m epub.Metadata) tea.Cmd {
	return func() tea.Msg {
		if err := epub.WriteMetadata(m); err != nil {
			return saveErrMsg{err: err}
		}
		return savedMsg{metadata: m}
	}
}
