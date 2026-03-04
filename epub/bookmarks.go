package epub

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Bookmark is a saved reading position with an optional note.
type Bookmark struct {
	Chapter int    `json:"chapter"`
	Offset  int    `json:"offset"`
	Note    string `json:"note,omitempty"`
}

// BookmarkState holds the full reading state for one epub file.
type BookmarkState struct {
	LastChapter int        `json:"last_chapter"`
	LastOffset  int        `json:"last_offset"`
	Marks       []Bookmark `json:"marks,omitempty"`
}

// bookmarkFile is the on-disk structure: epub path → state.
type bookmarkFile map[string]BookmarkState

func bookmarkPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "metapunk", "bookmarks.json"), nil
}

// LoadBookmarks returns the persisted reading state for the given epub path.
// Returns a zero-value BookmarkState on any error.
func LoadBookmarks(epubPath string) BookmarkState {
	p, err := bookmarkPath()
	if err != nil {
		return BookmarkState{}
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return BookmarkState{}
	}
	var bf bookmarkFile
	if err := json.Unmarshal(data, &bf); err != nil {
		return BookmarkState{}
	}
	return bf[epubPath]
}

// SaveBookmarks persists the reading state for the given epub path.
func SaveBookmarks(epubPath string, state BookmarkState) error {
	p, err := bookmarkPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}

	// Load existing data so we don't overwrite other books.
	var bf bookmarkFile
	if data, err := os.ReadFile(p); err == nil {
		_ = json.Unmarshal(data, &bf)
	}
	if bf == nil {
		bf = make(bookmarkFile)
	}

	bf[epubPath] = state
	data, err := json.MarshalIndent(bf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}
