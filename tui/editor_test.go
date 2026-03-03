package tui

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Morfo-si/metapunk/epub"
	tea "github.com/charmbracelet/bubbletea"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func buildEditorEPUB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.epub")

	const containerXML = `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`
	const opfXML = `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
    <dc:title>Original</dc:title>
    <dc:creator opf:role="aut">Original Author</dc:creator>
  </metadata>
</package>`

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	add := func(name, content string) {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		f.Write([]byte(content))
	}
	add("mimetype", "application/epub+zip")
	add("META-INF/container.xml", containerXML)
	add("OEBPS/content.opf", opfXML)
	w.Close()
	os.WriteFile(path, buf.Bytes(), 0644)
	return path
}

func keyMsg(key tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: key}
}

// ── NewEditorModel ────────────────────────────────────────────────────────────

func TestNewEditorModel_PopulatesInputs(t *testing.T) {
	m := epub.Metadata{
		Title:       "My Title",
		Author:      "My Author",
		Publisher:   "My Publisher",
		Language:    "en",
		Description: "My Description",
		Subject:     "My Subject",
		FilePath:    "/some/book.epub",
	}
	e := NewEditorModel(m)

	cases := []struct {
		field int
		want  string
	}{
		{fieldTitle, "My Title"},
		{fieldAuthor, "My Author"},
		{fieldPublisher, "My Publisher"},
		{fieldLanguage, "en"},
		{fieldDescription, "My Description"},
		{fieldSubject, "My Subject"},
	}
	for _, tc := range cases {
		if got := e.inputs[tc.field].Value(); got != tc.want {
			t.Errorf("inputs[%d].Value() = %q, want %q", tc.field, got, tc.want)
		}
	}
}

func TestNewEditorModel_TitleFocusedInitially(t *testing.T) {
	e := NewEditorModel(epub.Metadata{})
	if e.focused != fieldTitle {
		t.Errorf("focused = %d, want %d (fieldTitle)", e.focused, fieldTitle)
	}
}

func TestNewEditorModel_EmptyMetadata(t *testing.T) {
	e := NewEditorModel(epub.Metadata{})
	for i := range e.inputs {
		if got := e.inputs[i].Value(); got != "" {
			t.Errorf("inputs[%d].Value() = %q, want empty string", i, got)
		}
	}
}

// ── EditorModel.Update ────────────────────────────────────────────────────────

func TestEditorUpdate_EscEmitsCancelMsg(t *testing.T) {
	e := NewEditorModel(epub.Metadata{})
	_, cmd := e.Update(keyMsg(tea.KeyEsc))
	if cmd == nil {
		t.Fatal("expected non-nil cmd after esc")
	}
	msg := cmd()
	if _, ok := msg.(cancelMsg); !ok {
		t.Errorf("cmd() returned %T, want cancelMsg", msg)
	}
}

func TestEditorUpdate_TabAdvancesFocus(t *testing.T) {
	e := NewEditorModel(epub.Metadata{})
	if e.focused != fieldTitle {
		t.Fatalf("initial focus = %d, want 0", e.focused)
	}
	e, _ = e.Update(keyMsg(tea.KeyTab))
	if e.focused != fieldAuthor {
		t.Errorf("after tab focused = %d, want %d (fieldAuthor)", e.focused, fieldAuthor)
	}
}

func TestEditorUpdate_TabWrapsAround(t *testing.T) {
	e := NewEditorModel(epub.Metadata{})
	// Tab through all fields to wrap back to 0
	for i := 0; i < numFields; i++ {
		e, _ = e.Update(keyMsg(tea.KeyTab))
	}
	if e.focused != fieldTitle {
		t.Errorf("focused after full wrap = %d, want %d (fieldTitle)", e.focused, fieldTitle)
	}
}

func TestEditorUpdate_ShiftTabGoesBackward(t *testing.T) {
	e := NewEditorModel(epub.Metadata{})
	// shift+tab from 0 should wrap to last field
	e, _ = e.Update(keyMsg(tea.KeyShiftTab))
	if e.focused != numFields-1 {
		t.Errorf("focused after shift+tab from 0 = %d, want %d", e.focused, numFields-1)
	}
}

func TestEditorUpdate_CtrlSSetssSavingTrue(t *testing.T) {
	e := NewEditorModel(epub.Metadata{FilePath: "/nonexistent.epub"})
	e, cmd := e.Update(keyMsg(tea.KeyCtrlS))
	if !e.saving {
		t.Error("saving should be true after ctrl+s")
	}
	if cmd == nil {
		t.Error("expected non-nil saveCmd after ctrl+s")
	}
}

func TestEditorUpdate_CtrlSWhileSavingIsNoop(t *testing.T) {
	e := NewEditorModel(epub.Metadata{FilePath: "/nonexistent.epub"})
	e.saving = true
	_, cmd := e.Update(keyMsg(tea.KeyCtrlS))
	if cmd != nil {
		t.Error("expected nil cmd when ctrl+s pressed while already saving")
	}
}

func TestEditorUpdate_SaveErrMsgClearsSavingAndSetsError(t *testing.T) {
	e := NewEditorModel(epub.Metadata{})
	e.saving = true
	e, _ = e.Update(saveErrMsg{err: errFixed("disk full")})
	if e.saving {
		t.Error("saving should be false after saveErrMsg")
	}
	if e.errMsg != "disk full" {
		t.Errorf("errMsg = %q, want %q", e.errMsg, "disk full")
	}
}

// ── saveCmd ───────────────────────────────────────────────────────────────────

func TestSaveCmd_Success(t *testing.T) {
	path := buildEditorEPUB(t)
	m := epub.Metadata{
		FilePath: path,
		Title:    "Saved Title",
		Author:   "Saved Author",
	}
	cmd := saveCmd(m)
	msg := cmd()

	saved, ok := msg.(savedMsg)
	if !ok {
		t.Fatalf("cmd() returned %T, want savedMsg", msg)
	}
	if saved.metadata.Title != "Saved Title" {
		t.Errorf("saved title = %q, want %q", saved.metadata.Title, "Saved Title")
	}
}

func TestSaveCmd_Failure(t *testing.T) {
	m := epub.Metadata{FilePath: "/nonexistent/path/book.epub"}
	cmd := saveCmd(m)
	msg := cmd()

	if _, ok := msg.(saveErrMsg); !ok {
		t.Errorf("cmd() returned %T, want saveErrMsg", msg)
	}
}

// errFixed is a minimal error implementation for tests.
type errFixed string

func (e errFixed) Error() string { return string(e) }
