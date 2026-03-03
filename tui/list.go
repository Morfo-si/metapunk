package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Morfo-si/metapunk/epub"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// editMsg is sent when the user selects a file to edit.
type editMsg struct {
	metadata epub.Metadata
}

// reloadMsg triggers a rescan of the directory.
type reloadMsg struct{}

// ListModel is the file-browser view.
type ListModel struct {
	table    table.Model
	books    []epub.Metadata
	dir      string
	status   string
	statusOK bool
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

	m := ListModel{
		table: t,
		dir:   dir,
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

	rows := make([]table.Row, len(books))
	for i, b := range books {
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
}

func (m ListModel) Init() tea.Cmd {
	return nil
}

func (m ListModel) Update(msg tea.Msg) (ListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "q" || msg.String() == "ctrl+c":
			return m, tea.Quit
		case msg.String() == "enter":
			if len(m.books) == 0 {
				return m, nil
			}
			selected := m.books[m.table.Cursor()]
			return m, func() tea.Msg { return editMsg{metadata: selected} }
		case msg.String() == "r":
			m.load()
			return m, nil
		}
	case savedMsg:
		// Update the in-memory book list entry and table row
		for i, b := range m.books {
			if b.FilePath == msg.metadata.FilePath {
				m.books[i] = msg.metadata
				row := m.table.Rows()[i]
				title := msg.metadata.Title
				if title == "" {
					title = "(unknown)"
				}
				author := msg.metadata.Author
				if author == "" {
					author = "(unknown)"
				}
				row[1] = truncate(title, 32)
				row[2] = truncate(author, 24)
				rows := m.table.Rows()
				rows[i] = row
				m.table.SetRows(rows)
				break
			}
		}
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

	tableView := tableStyle.Render(m.table.View())

	var statusLine string
	if m.status != "" {
		if m.statusOK {
			statusLine = statusOKStyle.Render("✓ " + m.status)
		} else {
			statusLine = statusErrStyle.Render("✗ " + m.status)
		}
	}

	help := helpStyle.Render("↑/k up  ↓/j down  enter edit  r reload  q quit")

	if len(m.books) == 0 {
		empty := lipgloss.NewStyle().Foreground(gray).Italic(true).Render("No .epub files found in this directory.")
		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			subheader,
			empty,
			help,
		)
	}

	parts := []string{header, subheader, tableView}
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
