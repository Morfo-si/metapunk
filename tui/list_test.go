package tui

import (
	"testing"

	"github.com/Morfo-si/metapunk/epub"
	tea "github.com/charmbracelet/bubbletea"
)

// ── truncate ──────────────────────────────────────────────────────────────────

func TestTruncate(t *testing.T) {
	cases := []struct {
		input string
		max   int
		want  string
	}{
		{"short", 10, "short"},
		{"exactly10c", 10, "exactly10c"},
		{"this is a very long string", 10, "this is a…"},
		{"", 5, ""},
		{"  spaces  ", 10, "spaces"}, // TrimSpace applied first
		{"abcde", 5, "abcde"},        // exactly at limit
		{"abcdef", 5, "abcd…"},       // one over
		{"日本語テスト", 4, "日本語…"},        // multibyte runes
	}
	for _, tc := range cases {
		got := truncate(tc.input, tc.max)
		if got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.max, got, tc.want)
		}
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func makeBooks() []epub.Metadata {
	return []epub.Metadata{
		{Title: "Dune", Author: "Frank Herbert", FilePath: "/dune.epub"},
		{Title: "Neuromancer", Author: "William Gibson", FilePath: "/neuromancer.epub"},
		{Title: "Foundation", Author: "Isaac Asimov", FilePath: "/foundation.epub"},
	}
}

func newListWithBooks(t *testing.T, books []epub.Metadata) ListModel {
	t.Helper()
	m := NewListModel(t.TempDir())
	m.books = books
	m.applyFilter("")
	return m
}

func runeKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func escKey() tea.KeyMsg   { return tea.KeyMsg{Type: tea.KeyEsc} }
func enterKey() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyEnter} }

// ── applyFilter ───────────────────────────────────────────────────────────────

func TestApplyFilter_EmptyQueryShowsAll(t *testing.T) {
	m := newListWithBooks(t, makeBooks())
	m.applyFilter("")
	if len(m.filtered) != 3 {
		t.Errorf("filtered len = %d, want 3", len(m.filtered))
	}
	if len(m.table.Rows()) != 3 {
		t.Errorf("table rows = %d, want 3", len(m.table.Rows()))
	}
}

func TestApplyFilter_MatchesTitle(t *testing.T) {
	m := newListWithBooks(t, makeBooks())
	m.applyFilter("dune")
	if len(m.filtered) != 1 {
		t.Fatalf("filtered len = %d, want 1", len(m.filtered))
	}
	if m.filtered[0].Title != "Dune" {
		t.Errorf("filtered[0].Title = %q, want %q", m.filtered[0].Title, "Dune")
	}
}

func TestApplyFilter_MatchesAuthor(t *testing.T) {
	m := newListWithBooks(t, makeBooks())
	m.applyFilter("gibson")
	if len(m.filtered) != 1 {
		t.Fatalf("filtered len = %d, want 1", len(m.filtered))
	}
	if m.filtered[0].Author != "William Gibson" {
		t.Errorf("filtered[0].Author = %q, want %q", m.filtered[0].Author, "William Gibson")
	}
}

func TestApplyFilter_CaseInsensitive(t *testing.T) {
	m := newListWithBooks(t, makeBooks())
	m.applyFilter("DUNE")
	if len(m.filtered) != 1 {
		t.Errorf("filtered len = %d, want 1 (case-insensitive)", len(m.filtered))
	}
}

func TestApplyFilter_PartialMatch(t *testing.T) {
	m := newListWithBooks(t, makeBooks())
	m.applyFilter("foun") // matches "Foundation"
	if len(m.filtered) != 1 {
		t.Errorf("filtered len = %d, want 1", len(m.filtered))
	}
}

func TestApplyFilter_NoMatch(t *testing.T) {
	m := newListWithBooks(t, makeBooks())
	m.applyFilter("tolkien")
	if len(m.filtered) != 0 {
		t.Errorf("filtered len = %d, want 0", len(m.filtered))
	}
	if len(m.table.Rows()) != 0 {
		t.Errorf("table rows = %d, want 0", len(m.table.Rows()))
	}
}

func TestApplyFilter_WhitespaceOnlyTreatedAsEmpty(t *testing.T) {
	m := newListWithBooks(t, makeBooks())
	m.applyFilter("   ")
	if len(m.filtered) != 3 {
		t.Errorf("filtered len = %d, want 3 (whitespace = no filter)", len(m.filtered))
	}
}

func TestApplyFilter_MultipleMatches(t *testing.T) {
	books := []epub.Metadata{
		{Title: "Dune", Author: "Frank Herbert", FilePath: "/dune.epub"},
		{Title: "Dune Messiah", Author: "Frank Herbert", FilePath: "/dune2.epub"},
		{Title: "Foundation", Author: "Isaac Asimov", FilePath: "/foundation.epub"},
	}
	m := newListWithBooks(t, books)
	m.applyFilter("dune")
	if len(m.filtered) != 2 {
		t.Errorf("filtered len = %d, want 2", len(m.filtered))
	}
}

// ── search mode via Update ────────────────────────────────────────────────────

func TestListUpdate_SlashActivatesSearch(t *testing.T) {
	m := newListWithBooks(t, makeBooks())
	m, _ = m.Update(runeKey('/'))
	if !m.searching {
		t.Error("searching should be true after '/' key")
	}
}

func TestListUpdate_SlashClearsStatus(t *testing.T) {
	m := newListWithBooks(t, makeBooks())
	m.status = "some status"
	m, _ = m.Update(runeKey('/'))
	if m.status != "" {
		t.Errorf("status should be cleared on search, got %q", m.status)
	}
}

func TestListUpdate_EscClearsSearchAndShowsAll(t *testing.T) {
	m := newListWithBooks(t, makeBooks())
	// activate search, apply a filter manually
	m, _ = m.Update(runeKey('/'))
	m.search.SetValue("dune")
	m.applyFilter("dune")
	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 result before esc, got %d", len(m.filtered))
	}
	// clear with esc
	m, _ = m.Update(escKey())
	if m.searching {
		t.Error("searching should be false after esc")
	}
	if m.search.Value() != "" {
		t.Errorf("search value should be empty after esc, got %q", m.search.Value())
	}
	if len(m.filtered) != 3 {
		t.Errorf("filtered len = %d, want 3 after esc", len(m.filtered))
	}
}

func TestListUpdate_SearchEnterWithNoResultsIsNoop(t *testing.T) {
	m := newListWithBooks(t, makeBooks())
	m.searching = true
	m.applyFilter("nomatch")
	_, cmd := m.Update(enterKey())
	if cmd != nil {
		t.Error("enter with no filtered results should return nil cmd")
	}
}

func TestListUpdate_SearchEnterWithResultsEmitsEditMsg(t *testing.T) {
	m := newListWithBooks(t, makeBooks())
	m.searching = true
	m.applyFilter("dune")
	_, cmd := m.Update(enterKey())
	if cmd == nil {
		t.Fatal("expected non-nil cmd when enter pressed with results")
	}
	msg := cmd()
	edit, ok := msg.(editMsg)
	if !ok {
		t.Fatalf("cmd() returned %T, want editMsg", msg)
	}
	if edit.metadata.Title != "Dune" {
		t.Errorf("editMsg title = %q, want %q", edit.metadata.Title, "Dune")
	}
}

func TestListUpdate_NormalEnterWithNoFilesIsNoop(t *testing.T) {
	m := NewListModel(t.TempDir()) // empty dir
	_, cmd := m.Update(enterKey())
	if cmd != nil {
		t.Error("enter on empty list should return nil cmd")
	}
}

// ── search focus switching ─────────────────────────────────────────────────────

func TestListUpdate_TabSwitchesFromInputToTable(t *testing.T) {
	m := newListWithBooks(t, makeBooks())
	// activate search — input is focused
	m, _ = m.Update(runeKey('/'))
	if !m.search.Focused() {
		t.Fatal("search input should be focused after '/'")
	}
	// Tab moves focus to table
	m, _ = m.Update(tabKey())
	if m.search.Focused() {
		t.Error("search input should be blurred after Tab")
	}
	if !m.searching {
		t.Error("searching should still be true after Tab")
	}
}

func TestListUpdate_TabSwitchesFromTableToInput(t *testing.T) {
	m := newListWithBooks(t, makeBooks())
	m, _ = m.Update(runeKey('/'))
	m, _ = m.Update(tabKey()) // input → table
	if m.search.Focused() {
		t.Fatal("expected table focus after first Tab")
	}
	m, _ = m.Update(tabKey()) // table → input
	if !m.search.Focused() {
		t.Error("search input should be focused after second Tab")
	}
}

func TestListUpdate_ShiftTabAlsoSwitchesFocus(t *testing.T) {
	m := newListWithBooks(t, makeBooks())
	m, _ = m.Update(runeKey('/'))
	// shift+tab from input should also move to table
	m, _ = m.Update(shiftTabKey())
	if m.search.Focused() {
		t.Error("search input should be blurred after Shift+Tab")
	}
}

func TestListUpdate_EscClearsSearchWhenTableFocused(t *testing.T) {
	m := newListWithBooks(t, makeBooks())
	m, _ = m.Update(runeKey('/'))
	m.search.SetValue("dune")
	m.applyFilter("dune")
	m, _ = m.Update(tabKey()) // move to table
	// Esc should still clear search
	m, _ = m.Update(escKey())
	if m.searching {
		t.Error("searching should be false after Esc even when table was focused")
	}
	if len(m.filtered) != 3 {
		t.Errorf("filtered len = %d, want 3 after Esc", len(m.filtered))
	}
}

func TestListUpdate_EnterSelectsWhenTableFocused(t *testing.T) {
	m := newListWithBooks(t, makeBooks())
	m, _ = m.Update(runeKey('/'))
	m.search.SetValue("dune")
	m.applyFilter("dune")
	m, _ = m.Update(tabKey()) // move to table
	_, cmd := m.Update(enterKey())
	if cmd == nil {
		t.Fatal("expected non-nil cmd when Enter pressed from table focus")
	}
	if _, ok := cmd().(editMsg); !ok {
		t.Error("expected editMsg from Enter when table is focused")
	}
}

func tabKey() tea.KeyMsg      { return tea.KeyMsg{Type: tea.KeyTab} }
func shiftTabKey() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyShiftTab} }
