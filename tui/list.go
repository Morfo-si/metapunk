package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Morfo-si/metapunk/epub"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// editMsg is sent when the user selects a file to edit.
type editMsg struct {
	metadata epub.Metadata
}

// ListModel is the file-browser view.
type ListModel struct {
	table     table.Model
	books     []epub.Metadata // full unfiltered list
	filtered  []epub.Metadata // currently visible subset
	search    textinput.Model
	searching bool
	dir       string
	status    string
	statusOK  bool
	width     int
	height    int
}

func NewListModel(dir string) ListModel {
	// Columns and height start at sensible defaults; they are updated on the
	// first tea.WindowSizeMsg before anything is rendered.
	cols := []table.Column{
		{Title: "File", Width: 24},
		{Title: "Title", Width: 34},
		{Title: "Author", Width: 26},
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(12),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(subtle).
		BorderBottom(true).
		Bold(true).
		Foreground(purple)
	s.Selected = s.Selected.
		Foreground(white).
		Background(purple).
		Bold(true)
	t.SetStyles(s)

	si := textinput.New()
	si.Placeholder = "Filter by title or author…"
	si.Prompt = ""
	si.Width = 50

	m := ListModel{
		table:  t,
		search: si,
		dir:    dir,
	}
	m.load()
	return m
}

// resize adapts the table height and column widths to the current terminal
// dimensions. It should be called whenever a tea.WindowSizeMsg is received.
//
// Fixed vertical overhead (lines consumed by non-table UI elements):
//
//	header(1) + header-margin(1) + subheader-margin(1) + subheader(1) +
//	count(1) + table-top-border(1) + table-header(1) + table-header-underline(1) +
//	table-bottom-border(1) + help-margin(1) + help(1) = 11 lines
//
// One extra line is reserved to absorb the optional status line without
// causing the layout to overflow.
func (m *ListModel) resize(w, h int) {
	m.width = w
	m.height = h

	tableH := h - 12
	if tableH < 3 {
		tableH = 3
	}
	m.table.SetHeight(tableH)

	// Distribute horizontal space between the three columns.
	// The tableStyle rounded border consumes 2 chars (left + right).
	inner := w - 2
	if inner < 50 {
		inner = 50
	}
	fileW := 24
	if fileW > inner/4 {
		fileW = inner / 4
	}
	rest := inner - fileW
	titleW := rest * 55 / 100
	authorW := rest - titleW
	m.table.SetColumns([]table.Column{
		{Title: "File", Width: fileW},
		{Title: "Title", Width: titleW},
		{Title: "Author", Width: authorW},
	})

	// Keep the search input roughly in proportion to the window width.
	m.search.Width = w - 22
	if m.search.Width < 20 {
		m.search.Width = 20
	}

	// Re-render rows so truncation matches the new column widths.
	m.applyFilter(m.search.Value())
}

func (m *ListModel) load() {
	books, err := epub.ScanDir(m.dir)
	if err != nil {
		m.status = "Error scanning directory: " + err.Error()
		m.statusOK = false
		return
	}
	m.books = books
	m.applyFilter(m.search.Value())
}

// applyFilter updates m.filtered and rebuilds the table rows to match query.
func (m *ListModel) applyFilter(query string) {
	query = strings.TrimSpace(strings.ToLower(query))

	if query == "" {
		m.filtered = m.books
	} else {
		m.filtered = nil
		for _, b := range m.books {
			if strings.Contains(strings.ToLower(b.Title), query) ||
				strings.Contains(strings.ToLower(b.Author), query) {
				m.filtered = append(m.filtered, b)
			}
		}
	}

	cols := m.table.Columns()
	fileW, titleW, authorW := 22, 32, 24
	if len(cols) == 3 {
		fileW = cols[0].Width - 2
		titleW = cols[1].Width - 2
		authorW = cols[2].Width - 2
	}

	rows := make([]table.Row, len(m.filtered))
	for i, b := range m.filtered {
		title := b.Title
		if title == "" {
			title = "(unknown)"
		}
		author := b.Author
		if author == "" {
			author = "(unknown)"
		}
		rows[i] = table.Row{
			truncate(filepath.Base(b.FilePath), fileW),
			truncate(title, titleW),
			truncate(author, authorW),
		}
	}
	m.table.SetRows(rows)
	// SetRows only clamps the cursor downward; reset to 0 if it went negative.
	if len(rows) > 0 && m.table.Cursor() < 0 {
		m.table.SetCursor(0)
	}
}

func (m ListModel) Init() tea.Cmd {
	return nil
}

func (m ListModel) Update(msg tea.Msg) (ListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		// ctrl+c always quits regardless of search mode
		if key.Matches(msg, listKeys.ForceQuit) {
			return m, tea.Quit
		}

		if m.searching {
			switch {
			case key.Matches(msg, listKeys.ClearSearch):
				m.searching = false
				m.search.Blur()
				m.search.SetValue("")
				m.applyFilter("")
				return m, nil
			case key.Matches(msg, listKeys.SwitchFocus):
				if m.search.Focused() {
					m.search.Blur()
				} else {
					return m, m.search.Focus()
				}
				return m, nil
			case key.Matches(msg, listKeys.Edit):
				if len(m.filtered) == 0 {
					return m, nil
				}
				cursor := m.table.Cursor()
				if cursor < 0 || cursor >= len(m.filtered) {
					return m, nil
				}
				selected := m.filtered[cursor]
				return m, func() tea.Msg { return editMsg{metadata: selected} }
			default:
				if m.search.Focused() {
					var cmd tea.Cmd
					m.search, cmd = m.search.Update(msg)
					m.applyFilter(m.search.Value())
					return m, cmd
				}
				// Table is focused: fall through to m.table.Update so
				// arrow keys navigate the filtered results.
			}
		}

		// Normal (non-search) mode
		switch {
		case key.Matches(msg, listKeys.Quit):
			return m, tea.Quit
		case key.Matches(msg, listKeys.Search):
			m.searching = true
			m.status = ""
			return m, m.search.Focus()
		case key.Matches(msg, listKeys.Edit):
			if len(m.filtered) == 0 {
				return m, nil
			}
			cursor := m.table.Cursor()
			if cursor < 0 || cursor >= len(m.filtered) {
				return m, nil
			}
			selected := m.filtered[cursor]
			return m, func() tea.Msg { return editMsg{metadata: selected} }
		case key.Matches(msg, listKeys.Open):
			if len(m.filtered) == 0 {
				return m, nil
			}
			cursor := m.table.Cursor()
			if cursor < 0 || cursor >= len(m.filtered) {
				return m, nil
			}
			selected := m.filtered[cursor]
			return m, func() tea.Msg { return openReaderMsg{metadata: selected} }
		case key.Matches(msg, listKeys.Reload):
			m.search.SetValue("")
			m.load()
			return m, nil
		}

	case savedMsg:
		// Update the canonical book list entry
		for i, b := range m.books {
			if b.FilePath == msg.metadata.FilePath {
				m.books[i] = msg.metadata
				break
			}
		}
		// Rebuild filtered view to reflect the update
		m.applyFilter(m.search.Value())
		m.status = fmt.Sprintf("Saved: %s", filepath.Base(msg.metadata.FilePath))
		m.statusOK = true
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m ListModel) View() string {
	cwd, _ := os.Getwd()

	header := titleBarStyle.Render("metapunk — EPUB Metadata Editor")
	subheader := helpStyle.Render("Directory: " + cwd)

	var searchBar string
	if m.searching {
		label := searchLabelStyle.Render("Search: ")
		barStyle := searchBarStyle
		if !m.search.Focused() {
			barStyle = searchBarBlurredStyle
		}
		input := barStyle.Render(m.search.View())
		searchBar = lipgloss.JoinHorizontal(lipgloss.Center, label, input)
	}

	var countLine string
	if m.searching || m.search.Value() != "" {
		countLine = searchCountStyle.Render(
			fmt.Sprintf("%d of %d files", len(m.filtered), len(m.books)),
		)
	} else {
		countLine = searchCountStyle.Render(
			fmt.Sprintf("%d files", len(m.books)),
		)
	}

	tableView := tableStyle.Render(m.table.View())

	var statusLine string
	if m.status != "" {
		if m.statusOK {
			statusLine = statusOKStyle.Render("✓ " + m.status)
		} else {
			statusLine = statusErrStyle.Render("✗ " + m.status)
		}
	}

	var help string
	if m.searching {
		if m.search.Focused() {
			help = helpStyle.Render("type to filter  tab → results  esc clear")
		} else {
			help = helpStyle.Render("↑/↓ navigate  enter edit  tab → search  esc clear")
		}
	} else {
		help = helpStyle.Render("↑/k up  ↓/j down  enter edit  o read  / search  r reload  q quit")
	}

	if len(m.books) == 0 {
		empty := lipgloss.NewStyle().Foreground(gray).Italic(true).Render("No .epub files found in this directory.")
		return lipgloss.JoinVertical(lipgloss.Left, header, subheader, countLine, empty, help)
	}

	parts := []string{header, subheader}
	if searchBar != "" {
		parts = append(parts, searchBar)
	}
	if countLine != "" {
		parts = append(parts, countLine)
	}
	parts = append(parts, tableView)
	if statusLine != "" {
		parts = append(parts, statusLine)
	}
	parts = append(parts, help)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) <= max {
		return s
	}
	return string([]rune(s)[:max-1]) + "…"
}
