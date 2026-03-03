package tui

import (
	"path/filepath"

	"github.com/Morfo-si/metapunk/epub"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	fieldPublisher
	fieldLanguage
	fieldDescription
	fieldSubject
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
	make := func(placeholder, value string) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.SetValue(value)
		ti.Width = 46
		ti.Prompt = ""
		return ti
	}

	inputs := [numFields]textinput.Model{
		fieldTitle:       make("Book title", m.Title),
		fieldAuthor:      make("Author name", m.Author),
		fieldPublisher:   make("Publisher", m.Publisher),
		fieldLanguage:    make("e.g. en, fr, de", m.Language),
		fieldDescription: make("Short description or blurb", m.Description),
		fieldSubject:     make("e.g. Science Fiction, Fantasy", m.Subject),
	}
	inputs[fieldTitle].Focus()

	return EditorModel{
		original: m,
		inputs:   inputs,
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
				FilePath:    e.original.FilePath,
				Title:       e.inputs[fieldTitle].Value(),
				Author:      e.inputs[fieldAuthor].Value(),
				Publisher:   e.inputs[fieldPublisher].Value(),
				Language:    e.inputs[fieldLanguage].Value(),
				Description: e.inputs[fieldDescription].Value(),
				Subject:     e.inputs[fieldSubject].Value(),
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

var fieldLabels = [numFields]string{
	fieldTitle:       "Title",
	fieldAuthor:      "Author",
	fieldPublisher:   "Publisher",
	fieldLanguage:    "Language",
	fieldDescription: "Description",
	fieldSubject:     "Subject",
}

func (e EditorModel) View() string {
	filename := filepath.Base(e.original.FilePath)
	heading := editorTitleStyle.Render("Editing: " + filename)

	rows := make([]string, numFields)
	for i := range e.inputs {
		label := labelStyle.Render(fieldLabels[i])
		box := inputBox(e.inputs[i], e.focused == i)
		rows[i] = lipgloss.JoinHorizontal(lipgloss.Center, label, box)
	}

	var statusLine string
	if e.saving {
		statusLine = helpStyle.Render("Saving…")
	} else if e.errMsg != "" {
		statusLine = statusErrStyle.Render("✗ " + e.errMsg)
	}

	help := helpStyle.Render("tab next  shift+tab prev  ctrl+s save  esc cancel")

	parts := []string{heading}
	parts = append(parts, rows[:]...)
	if statusLine != "" {
		parts = append(parts, statusLine)
	}
	parts = append(parts, help)

	return editorPanelStyle.Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
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
