package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Morfo-si/metapunk/epub"
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
}

func NewListModel(dir string) ListModel {
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
			truncate(filepath.Base(b.FilePath), 22),
			truncate(title, 32),
			truncate(author, 24),
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
	case tea.KeyMsg:
		// ctrl+c always quits regardless of search mode
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if m.searching {
			switch msg.String() {
			case "esc":
				m.searching = false
				m.search.Blur()
				m.search.SetValue("")
				m.applyFilter("")
				return m, nil
			case "tab", "shift+tab":
				if m.search.Focused() {
					m.search.Blur()
				} else {
					return m, m.search.Focus()
				}
				return m, nil
			case "enter":
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
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "/":
			m.searching = true
			m.status = ""
			return m, m.search.Focus()
		case "enter":
			if len(m.filtered) == 0 {
				return m, nil
			}
			cursor := m.table.Cursor()
			if cursor < 0 || cursor >= len(m.filtered) {
				return m, nil
			}
			selected := m.filtered[cursor]
			return m, func() tea.Msg { return editMsg{metadata: selected} }
		case "r":
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
		help = helpStyle.Render("↑/k up  ↓/j down  enter edit  / search  r reload  q quit")
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
