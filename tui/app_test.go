package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/morfo-si/metapunk/epub"
)

// newTestAppModel returns an AppModel pointed at an empty temp dir so no
// real epub scanning happens during tests.
func newTestAppModel(t *testing.T) AppModel {
	t.Helper()
	dir := t.TempDir()
	return AppModel{
		state: listView,
		list:  NewListModel(dir),
	}
}

// ── initial state ─────────────────────────────────────────────────────────────

func TestAppModel_StartsInListView(t *testing.T) {
	a := newTestAppModel(t)
	if a.state != listView {
		t.Errorf("initial state = %v, want listView", a.state)
	}
}

func TestAppModel_ViewRendersListWhenInListView(t *testing.T) {
	a := newTestAppModel(t)
	view := a.View()
	if view == "" {
		t.Error("View() returned empty string in list view")
	}
}

// ── editMsg: list → editor ────────────────────────────────────────────────────

func TestAppModel_EditMsgTransitionsToEditorView(t *testing.T) {
	a := newTestAppModel(t)
	m, _ := a.Update(editMsg{metadata: epub.Metadata{Title: "T", FilePath: "/a.epub"}})
	app := m.(AppModel)
	if app.state != editorView {
		t.Errorf("state after editMsg = %v, want editorView", app.state)
	}
}

func TestAppModel_EditMsgPopulatesEditor(t *testing.T) {
	a := newTestAppModel(t)
	meta := epub.Metadata{Title: "Dune", Author: "Frank Herbert", FilePath: "/dune.epub"}
	m, _ := a.Update(editMsg{metadata: meta})
	app := m.(AppModel)
	if app.editor.inputs[fieldTitle].Value() != "Dune" {
		t.Errorf("editor title = %q, want %q", app.editor.inputs[fieldTitle].Value(), "Dune")
	}
	if app.editor.inputs[fieldAuthor].Value() != "Frank Herbert" {
		t.Errorf("editor author = %q, want %q", app.editor.inputs[fieldAuthor].Value(), "Frank Herbert")
	}
}

// ── cancelMsg: editor → list ──────────────────────────────────────────────────

func TestAppModel_CancelMsgTransitionsBackToListView(t *testing.T) {
	a := newTestAppModel(t)
	// first go to editor
	m, _ := a.Update(editMsg{metadata: epub.Metadata{FilePath: "/a.epub"}})
	app := m.(AppModel)
	if app.state != editorView {
		t.Fatalf("expected editorView after editMsg, got %v", app.state)
	}
	// then cancel
	m, _ = app.Update(cancelMsg{})
	app = m.(AppModel)
	if app.state != listView {
		t.Errorf("state after cancelMsg = %v, want listView", app.state)
	}
}

// ── savedMsg: editor → list ───────────────────────────────────────────────────

func TestAppModel_SavedMsgTransitionsBackToListView(t *testing.T) {
	a := newTestAppModel(t)
	m, _ := a.Update(editMsg{metadata: epub.Metadata{FilePath: "/a.epub"}})
	app := m.(AppModel)

	m, _ = app.Update(savedMsg{metadata: epub.Metadata{FilePath: "/a.epub", Title: "New"}})
	app = m.(AppModel)
	if app.state != listView {
		t.Errorf("state after savedMsg = %v, want listView", app.state)
	}
}

func TestAppModel_SavedMsgSetsListStatus(t *testing.T) {
	a := newTestAppModel(t)
	m, _ := a.Update(editMsg{metadata: epub.Metadata{FilePath: "/a.epub"}})
	app := m.(AppModel)

	m, _ = app.Update(savedMsg{metadata: epub.Metadata{FilePath: "/a.epub", Title: "New"}})
	app = m.(AppModel)
	if !app.list.statusOK {
		t.Error("list.statusOK should be true after savedMsg")
	}
	if app.list.status == "" {
		t.Error("list.status should be non-empty after savedMsg")
	}
}

// ── passthrough in editor view ────────────────────────────────────────────────

func TestAppModel_OtherMsgDelegatedToEditorWhileInEditorView(t *testing.T) {
	a := newTestAppModel(t)
	m, _ := a.Update(editMsg{metadata: epub.Metadata{FilePath: "/a.epub"}})
	app := m.(AppModel)

	// A tab keypress should cycle the editor focus, not change app state
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = m.(AppModel)
	if app.state != editorView {
		t.Errorf("state after tab in editor = %v, want editorView", app.state)
	}
	if app.editor.focused != fieldAuthor {
		t.Errorf("editor.focused after tab = %d, want %d (fieldAuthor)", app.editor.focused, fieldAuthor)
	}
}

// ── View routing ──────────────────────────────────────────────────────────────

func TestAppModel_ViewRoutesToEditorWhenInEditorView(t *testing.T) {
	a := newTestAppModel(t)
	listViewStr := a.View()

	m, _ := a.Update(editMsg{metadata: epub.Metadata{Title: "T", FilePath: "/a.epub"}})
	app := m.(AppModel)
	editorViewStr := app.View()

	if editorViewStr == listViewStr {
		t.Error("editor view and list view should render differently")
	}
}
