# metapunk

<p align="center">
  <img src="metapunk-logo.png" alt="metapunk logo" width="200" />
</p>

A terminal UI for editing EPUB metadata — fix titles and authors before uploading to Kindle or any other e-reader app.

![Go version](https://img.shields.io/badge/go-1.26-blue)
[![PR Checks](https://github.com/morfo-si/metapunk/actions/workflows/pr.yml/badge.svg)](https://github.com/morfo-si/metapunk/actions/workflows/pr.yml)

## Features

- Scans the current directory for `.epub` files automatically
- Displays a table with each file's filename, title, and author
- Edit title, author, publisher, language, description, and subject with a clean form UI
- Saves changes back into the EPUB file atomically (temp file + rename)
- No external tools or epub libraries required — uses Go's standard library only

## Demo

**List view** — browse all EPUBs in the current directory:

```
╭──────────────────────────────────────────────────────────────────────╮
│  metapunk — EPUB Metadata Editor                                     │
├──────────────────────────┬──────────────────────┬────────────────────┤
│ File                     │ Title                │ Author             │
├──────────────────────────┼──────────────────────┼────────────────────┤
│ a-fire-upon-the-deep.epub│ A Fire Upon the Deep │ Vernor Vinge       │
│ clean-code.epub          │ Clean Code           │ Robert C. Martin   │
│ unknown.epub             │ (unknown)            │ (unknown)          │
└──────────────────────────┴──────────────────────┴────────────────────┘
↑/k up  ↓/j down  enter edit  r reload  q quit
```

**Editor view** — press `Enter` on any row to edit all metadata fields:

```
╭──────────────────────────────────────────────────────────────╮
│  Editing: clean-code.epub                                    │
│                                                              │
│  Title         [ Clean Code                               ]  │
│  Author        [ Robert C. Martin                         ]  │
│  Publisher     [ Prentice Hall                            ]  │
│  Language      [ en                                       ]  │
│  Description   [ A handbook of agile software craftsman…  ]  │
│  Subject       [ Software Engineering, Programming        ]  │
│                                                              │
│  tab next  shift+tab prev  ctrl+s save  esc cancel           │
╰──────────────────────────────────────────────────────────────╯
```

## Installation

### From source

Requires Go 1.26 or later.

```bash
git clone https://github.com/morfo-si/metapunk.git
cd metapunk
make install
```

### Build locally

```bash
make build
# produces ./metapunk
```

## Usage

Run `metapunk` in any directory that contains `.epub` files:

```bash
cd ~/Books
metapunk
```

### Key bindings

#### List view

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Enter` | Edit selected file |
| `r` | Reload directory |
| `q` / `Ctrl+C` | Quit |

#### Editor view

Six fields are available: Title, Author, Publisher, Language, Description, and Subject.
For Subject, multiple values can be entered separated by commas (e.g. `Science Fiction, Fantasy`).

| Key | Action |
|-----|--------|
| `Tab` / `↓` | Next field |
| `Shift+Tab` / `↑` | Previous field |
| `Ctrl+S` | Save changes |
| `Esc` | Cancel and return to list |

## Development

### Prerequisites

- Go 1.26+
- `make`

### Common tasks

```bash
make build          # compile the binary
make run            # build and run
make test           # run all tests
make test-verbose   # run tests with -v
make test-cover     # run tests and print coverage by function
make test-cover-html # open an HTML coverage report in the browser
make lint           # fmt + vet
make tidy           # go mod tidy + verify
make clean          # remove binary and coverage artefacts
```

### Project structure

```
metapunk/
├── main.go              — entry point
├── epub/
│   ├── epub.go          — ReadMetadata, WriteMetadata, ScanDir
│   └── epub_test.go     — unit and integration tests
└── tui/
    ├── app.go           — root model, routes between views
    ├── list.go          — file browser (bubbles/table)
    ├── editor.go        — metadata form (bubbles/textinput)
    ├── keys.go          — key binding definitions
    ├── styles.go        — lipgloss colour palette and styles
    └── list_test.go     — tests for pure TUI helpers
```

### Tech stack

| Library | Purpose |
|---------|---------|
| [bubbletea](https://github.com/charmbracelet/bubbletea) | Elm-architecture TUI framework |
| [bubbles](https://github.com/charmbracelet/bubbles) | Table, text input, and key binding components |
| [lipgloss](https://github.com/charmbracelet/lipgloss) | Styles, borders, and layout |
| `archive/zip` + `encoding/xml` | EPUB read/write (stdlib only) |

## How EPUB metadata is stored

An EPUB file is a ZIP archive. Inside it, `META-INF/container.xml` points to an OPF file (e.g. `OEBPS/content.opf`) that holds Dublin Core metadata:

```xml
<metadata>
  <dc:title>My Book</dc:title>
  <dc:creator opf:role="aut">Author Name</dc:creator>
  <dc:publisher>Publisher Name</dc:publisher>
  <dc:language>en</dc:language>
  <dc:description>A short blurb about the book.</dc:description>
  <dc:subject>Science Fiction</dc:subject>
</metadata>
```

`metapunk` edits these elements in place and re-packages the ZIP, leaving every other file in the archive untouched. Fields absent from the original OPF are injected automatically on first save.

## License

MIT — see [LICENSE](LICENSE).
