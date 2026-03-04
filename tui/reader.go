package tui

import (
	"fmt"
	"strings"

	"github.com/Morfo-si/metapunk/epub"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// openReaderMsg is sent by the list to open a book in the reader.
type openReaderMsg struct {
	metadata epub.Metadata
}

// backToListMsg is sent by the reader when the user presses q/esc.
type backToListMsg struct{}

// chaptersLoadedMsg carries the result of the async chapter load.
type chaptersLoadedMsg struct {
	chapters []epub.Chapter
	err      error
}

// bookmarkSavedMsg is returned after a background bookmark save.
type bookmarkSavedMsg struct{}

// ReaderModel is the epub reading view.
type ReaderModel struct {
	meta       epub.Metadata
	chapters   []epub.Chapter
	chapterIdx int
	lines      []string // current chapter's plain-text lines
	viewport   viewport.Model
	bookmarks  epub.BookmarkState
	loading    bool
	loadErr    string

	// search
	searchInput textinput.Model
	searching   bool
	searchQuery string
	matchLines  []int // line indices that contain the query
	matchCursor int   // index into matchLines pointing to current match

	// bookmark browser overlay
	showBookmarks bool
	bmCursor      int

	width, height int
}

func NewReaderModel(meta epub.Metadata, width, height int) ReaderModel {
	si := textinput.New()
	si.Placeholder = "search…"
	si.Prompt = "/"
	si.Width = 40

	vp := viewport.New(width, viewportH(height, false))

	bm := epub.LoadBookmarks(meta.FilePath)

	return ReaderModel{
		meta:        meta,
		viewport:    vp,
		bookmarks:   bm,
		loading:     true,
		width:       width,
		height:      height,
		searchInput: si,
	}
}

func (m ReaderModel) Init() tea.Cmd {
	return loadChaptersCmd(m.meta.FilePath)
}

func loadChaptersCmd(path string) tea.Cmd {
	return func() tea.Msg {
		chapters, err := epub.ReadChapters(path)
		return chaptersLoadedMsg{chapters: chapters, err: err}
	}
}

func saveBookmarksCmd(path string, state epub.BookmarkState) tea.Cmd {
	return func() tea.Msg {
		_ = epub.SaveBookmarks(path, state)
		return bookmarkSavedMsg{}
	}
}

// viewportH computes the viewport height given the total terminal height and
// whether the search bar is currently visible.
func viewportH(totalH int, searchVisible bool) int {
	// header(1) + chapter status(1) + viewport + search(0-1) + help(1)
	overhead := 4
	if searchVisible {
		overhead++
	}
	h := totalH - overhead
	if h < 3 {
		h = 3
	}
	return h
}

func (m ReaderModel) Update(msg tea.Msg) (ReaderModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = viewportH(msg.Height, m.searching)
		m.renderChapter()
		return m, nil

	case chaptersLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.loadErr = msg.err.Error()
			return m, nil
		}
		m.chapters = msg.chapters
		m.chapterIdx = m.bookmarks.LastChapter
		if m.chapterIdx >= len(m.chapters) {
			m.chapterIdx = 0
		}
		m.renderChapter()
		m.viewport.YOffset = m.bookmarks.LastOffset
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m ReaderModel) handleKey(msg tea.KeyMsg) (ReaderModel, tea.Cmd) {
	if key.Matches(msg, readerKeys.ForceQuit) {
		return m, tea.Quit
	}

	// ── Bookmark browser overlay ──────────────────────────────────────────
	if m.showBookmarks {
		return m.handleBookmarkBrowserKey(msg)
	}

	// ── Search input active ───────────────────────────────────────────────
	if m.searching && m.searchInput.Focused() {
		switch {
		case key.Matches(msg, readerKeys.ClearSearch):
			m.searching = false
			m.searchInput.Blur()
			m.searchInput.SetValue("")
			m.searchQuery = ""
			m.matchLines = nil
			m.matchCursor = 0
			m.viewport.Height = viewportH(m.height, false)
			m.renderChapter()
			return m, nil

		case key.Matches(msg, readerKeys.ConfirmSearch):
			m.searchQuery = m.searchInput.Value()
			m.searchInput.Blur()
			m.findMatches()
			m.jumpToMatch(0)
			m.renderChapter()
			return m, nil

		default:
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			return m, cmd
		}
	}

	// ── Normal navigation ─────────────────────────────────────────────────
	switch {
	case key.Matches(msg, readerKeys.Back):
		if m.searching {
			// Clear search without leaving the reader.
			m.searching = false
			m.searchInput.Blur()
			m.searchInput.SetValue("")
			m.searchQuery = ""
			m.matchLines = nil
			m.viewport.Height = viewportH(m.height, false)
			m.renderChapter()
			return m, nil
		}
		m.savePosition()
		return m, func() tea.Msg { return backToListMsg{} }

	case key.Matches(msg, readerKeys.ScrollDown):
		m.viewport.HalfViewDown()

	case key.Matches(msg, readerKeys.ScrollUp):
		m.viewport.HalfViewUp()

	case key.Matches(msg, readerKeys.LineDown):
		m.viewport.LineDown(1)

	case key.Matches(msg, readerKeys.LineUp):
		m.viewport.LineUp(1)

	case key.Matches(msg, readerKeys.NextChapter):
		if m.chapterIdx < len(m.chapters)-1 {
			m.savePosition()
			m.chapterIdx++
			m.clearSearch()
			m.renderChapter()
			m.viewport.GotoTop()
		}

	case key.Matches(msg, readerKeys.PrevChapter):
		if m.chapterIdx > 0 {
			m.savePosition()
			m.chapterIdx--
			m.clearSearch()
			m.renderChapter()
			m.viewport.GotoTop()
		}

	case key.Matches(msg, readerKeys.Search):
		m.searching = true
		m.viewport.Height = viewportH(m.height, true)
		return m, m.searchInput.Focus()

	case key.Matches(msg, readerKeys.NextMatch):
		if len(m.matchLines) > 0 {
			m.matchCursor = (m.matchCursor + 1) % len(m.matchLines)
			m.jumpToMatch(m.matchCursor)
			m.renderChapter()
		}

	case key.Matches(msg, readerKeys.PrevMatch):
		if len(m.matchLines) > 0 {
			m.matchCursor = (m.matchCursor - 1 + len(m.matchLines)) % len(m.matchLines)
			m.jumpToMatch(m.matchCursor)
			m.renderChapter()
		}

	case key.Matches(msg, readerKeys.AddBookmark):
		bm := epub.Bookmark{
			Chapter: m.chapterIdx,
			Offset:  m.viewport.YOffset,
			Note:    fmt.Sprintf("Ch.%d", m.chapterIdx+1),
		}
		m.bookmarks.Marks = append(m.bookmarks.Marks, bm)
		return m, saveBookmarksCmd(m.meta.FilePath, m.bookmarks)

	case key.Matches(msg, readerKeys.ShowBookmarks):
		m.showBookmarks = true
		m.bmCursor = 0
		return m, nil
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m ReaderModel) handleBookmarkBrowserKey(msg tea.KeyMsg) (ReaderModel, tea.Cmd) {
	marks := m.bookmarks.Marks
	switch {
	case key.Matches(msg, readerKeys.ClearSearch), key.Matches(msg, readerKeys.Back):
		m.showBookmarks = false

	case key.Matches(msg, readerKeys.LineDown):
		if m.bmCursor < len(marks)-1 {
			m.bmCursor++
		}

	case key.Matches(msg, readerKeys.LineUp):
		if m.bmCursor > 0 {
			m.bmCursor--
		}

	case key.Matches(msg, readerKeys.ConfirmSearch): // enter → jump to bookmark
		if len(marks) > 0 && m.bmCursor < len(marks) {
			bm := marks[m.bmCursor]
			m.chapterIdx = bm.Chapter
			m.showBookmarks = false
			m.renderChapter()
			m.viewport.YOffset = bm.Offset
		}

	case key.Matches(msg, readerKeys.DeleteBookmark):
		if len(marks) > 0 && m.bmCursor < len(marks) {
			m.bookmarks.Marks = append(marks[:m.bmCursor], marks[m.bmCursor+1:]...)
			if m.bmCursor >= len(m.bookmarks.Marks) && m.bmCursor > 0 {
				m.bmCursor--
			}
			return m, saveBookmarksCmd(m.meta.FilePath, m.bookmarks)
		}
	}
	return m, nil
}

// renderChapter rebuilds the viewport content for the current chapter,
// applying search highlights if a query is active.
func (m *ReaderModel) renderChapter() {
	if len(m.chapters) == 0 {
		m.viewport.SetContent("No chapters found in this epub.")
		m.lines = nil
		return
	}
	ch := m.chapters[m.chapterIdx]
	m.lines = strings.Split(ch.Text, "\n")

	var sb strings.Builder
	for i, line := range m.lines {
		rendered := line
		if m.searchQuery != "" {
			isCurrent := len(m.matchLines) > 0 &&
				m.matchCursor < len(m.matchLines) &&
				m.matchLines[m.matchCursor] == i
			rendered = highlightMatches(line, m.searchQuery, isCurrent)
		}
		sb.WriteString(rendered)
		if i < len(m.lines)-1 {
			sb.WriteByte('\n')
		}
	}
	m.viewport.SetContent(sb.String())
}

// findMatches populates matchLines with the indices of lines that contain the
// current searchQuery (case-insensitive).
func (m *ReaderModel) findMatches() {
	m.matchLines = nil
	m.matchCursor = 0
	if m.searchQuery == "" {
		return
	}
	lower := strings.ToLower(m.searchQuery)
	for i, line := range m.lines {
		if strings.Contains(strings.ToLower(line), lower) {
			m.matchLines = append(m.matchLines, i)
		}
	}
}

// jumpToMatch moves the viewport to show the match at index idx.
func (m *ReaderModel) jumpToMatch(idx int) {
	if idx < 0 || idx >= len(m.matchLines) {
		return
	}
	m.matchCursor = idx
	m.viewport.YOffset = m.matchLines[idx]
}

// savePosition persists the current chapter and scroll offset.
func (m *ReaderModel) savePosition() {
	m.bookmarks.LastChapter = m.chapterIdx
	m.bookmarks.LastOffset = m.viewport.YOffset
	_ = epub.SaveBookmarks(m.meta.FilePath, m.bookmarks)
}

// clearSearch resets search state without resizing the viewport.
func (m *ReaderModel) clearSearch() {
	m.searching = false
	m.searchInput.Blur()
	m.searchInput.SetValue("")
	m.searchQuery = ""
	m.matchLines = nil
	m.matchCursor = 0
}

func (m ReaderModel) View() string {
	if m.loading {
		return titleBarStyle.Render("metapunk — Loading…")
	}
	if m.loadErr != "" {
		return lipgloss.JoinVertical(lipgloss.Left,
			titleBarStyle.Render("metapunk — Error"),
			statusErrStyle.Render("✗ "+m.loadErr),
			helpStyle.Render("q back"),
		)
	}

	// ── Header bar ────────────────────────────────────────────────────────
	title := m.meta.Title
	if title == "" {
		title = "(unknown title)"
	}
	if m.meta.Author != "" {
		title += " · " + m.meta.Author
	}
	header := titleBarStyle.Render("metapunk — " + truncate(title, m.width-16))

	// ── Chapter / progress status ─────────────────────────────────────────
	statusParts := []string{}
	if len(m.chapters) > 0 {
		ch := m.chapters[m.chapterIdx]
		pct := 0
		total := m.viewport.TotalLineCount()
		if total > 0 {
			pct = (m.viewport.YOffset * 100) / total
		}
		chapterLine := readerChapterStyle.Render(
			fmt.Sprintf("%s  [%d / %d]  %d%%",
				ch.Title, m.chapterIdx+1, len(m.chapters), pct),
		)
		statusParts = append(statusParts, chapterLine)
	}
	if m.searchQuery != "" {
		var matchInfo string
		if len(m.matchLines) == 0 {
			matchInfo = statusErrStyle.Render("no matches")
		} else {
			matchInfo = statusOKStyle.Render(
				fmt.Sprintf("match %d / %d", m.matchCursor+1, len(m.matchLines)),
			)
		}
		statusParts = append(statusParts, matchInfo)
	}
	chapterStatus := lipgloss.JoinHorizontal(lipgloss.Center, statusParts...)

	// ── Viewport ──────────────────────────────────────────────────────────
	vpView := m.viewport.View()

	// ── Search bar ────────────────────────────────────────────────────────
	var searchBar string
	if m.searching {
		searchBar = lipgloss.JoinHorizontal(lipgloss.Center,
			searchLabelStyle.Render("/"),
			searchBarStyle.Render(m.searchInput.View()),
			helpStyle.Render("  enter confirm  esc cancel"),
		)
	}

	// ── Help bar ──────────────────────────────────────────────────────────
	var help string
	switch {
	case m.showBookmarks:
		help = helpStyle.Render("↑/↓ navigate  enter jump  d delete  esc/q close")
	case m.searching && !m.searchInput.Focused():
		help = helpStyle.Render("n next  N prev  esc clear search  q back")
	default:
		help = helpStyle.Render("space/b scroll  →/← chapter  / search  n/N match  m bookmark  M list  q back")
	}

	parts := []string{header, chapterStatus, vpView}
	if searchBar != "" {
		parts = append(parts, searchBar)
	}
	parts = append(parts, help)
	base := lipgloss.JoinVertical(lipgloss.Left, parts...)

	// ── Bookmark overlay ──────────────────────────────────────────────────
	if m.showBookmarks {
		base = renderBookmarkOverlay(base, m.bookmarks.Marks, m.bmCursor, m.width, m.height)
	}
	return base
}

// highlightMatches wraps every occurrence of query in line with the
// appropriate highlight style. isCurrent indicates the "active" match.
func highlightMatches(line, query string, isCurrent bool) string {
	if query == "" || line == "" {
		return line
	}
	style := searchMatchStyle
	if isCurrent {
		style = searchCurrentMatchStyle
	}
	lowerLine := strings.ToLower(line)
	lowerQuery := strings.ToLower(query)

	var sb strings.Builder
	for {
		idx := strings.Index(lowerLine, lowerQuery)
		if idx == -1 {
			sb.WriteString(line)
			break
		}
		sb.WriteString(line[:idx])
		sb.WriteString(style.Render(line[idx : idx+len(query)]))
		line = line[idx+len(query):]
		lowerLine = lowerLine[idx+len(query):]
	}
	return sb.String()
}

// renderBookmarkOverlay places a floating bookmark browser centred over base.
func renderBookmarkOverlay(base string, marks []epub.Bookmark, cursor, width, height int) string {
	boxWidth := width - 8
	if boxWidth < 40 {
		boxWidth = 40
	}

	heading := lipgloss.NewStyle().Bold(true).Foreground(purple).Render("Bookmarks")

	var rows []string
	if len(marks) == 0 {
		rows = append(rows,
			lipgloss.NewStyle().Foreground(gray).Italic(true).Render(
				"No bookmarks yet. Press 'm' to add one.",
			),
		)
	} else {
		for i, bm := range marks {
			label := fmt.Sprintf("Ch.%d  line %d", bm.Chapter+1, bm.Offset)
			if bm.Note != "" {
				label += "  — " + bm.Note
			}
			label = truncate(label, boxWidth-4)
			if i == cursor {
				rows = append(rows,
					lipgloss.NewStyle().Foreground(white).Background(purple).Bold(true).Render("> "+label),
				)
			} else {
				rows = append(rows,
					lipgloss.NewStyle().Foreground(gray).Render("  "+label),
				)
			}
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)
	box := bookmarkOverlayStyle.Width(boxWidth).Render(
		lipgloss.JoinVertical(lipgloss.Left, heading, "", content),
	)

	return lipgloss.Place(width, height,
		lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(subtle),
	)
}
