package epub

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// ── helpers ──────────────────────────────────────────────────────────────────

const containerXML = `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`

// opfXML builds a minimal OPF with only title and author.
func opfXML(title, author string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
    <dc:title>` + title + `</dc:title>
    <dc:creator opf:role="aut">` + author + `</dc:creator>
  </metadata>
</package>`
}

// opfXMLFull builds an OPF with all supported metadata fields.
func opfXMLFull(m Metadata) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
    <dc:title>` + m.Title + `</dc:title>
    <dc:creator opf:role="aut">` + m.Author + `</dc:creator>
    <dc:publisher>` + m.Publisher + `</dc:publisher>
    <dc:language>` + m.Language + `</dc:language>
    <dc:description>` + m.Description + `</dc:description>
    <dc:subject>` + m.Subject + `</dc:subject>
  </metadata>
</package>`
}

// buildTestEPUB creates a minimal valid epub in a temp dir and returns its path.
func buildTestEPUB(t *testing.T, title, author string) string {
	t.Helper()
	return buildTestEPUBFull(t, Metadata{Title: title, Author: author})
}

// buildTestEPUBFull creates an epub with all supported metadata fields.
func buildTestEPUBFull(t *testing.T, m Metadata) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.epub")

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	add := func(name, content string) {
		t.Helper()
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}

	add("mimetype", "application/epub+zip")
	add("META-INF/container.xml", containerXML)
	if m.Publisher != "" || m.Language != "" || m.Description != "" || m.Subject != "" {
		add("OEBPS/content.opf", opfXMLFull(m))
	} else {
		add("OEBPS/content.opf", opfXML(m.Title, m.Author))
	}

	if err := w.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatalf("write epub: %v", err)
	}
	return path
}

// ── xmlEscape ────────────────────────────────────────────────────────────────

func TestXMLEscape(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"plain text", "plain text"},
		{"Rock & Roll", "Rock &amp; Roll"},
		{"<em>bold</em>", "&lt;em&gt;bold&lt;/em&gt;"},
		{`"quoted"`, "&#34;quoted&#34;"},
		{"apostrophe's", "apostrophe&#39;s"},
		{"", ""},
	}
	for _, tc := range cases {
		got := xmlEscape(tc.input)
		if got != tc.want {
			t.Errorf("xmlEscape(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ── replaceFirstElement ───────────────────────────────────────────────────────

func TestReplaceFirstElement(t *testing.T) {
	cases := []struct {
		name    string
		s       string
		tag     string
		value   string
		want    string
		wantErr bool
	}{
		{
			name:  "simple replace",
			s:     `<dc:title>Old Title</dc:title>`,
			tag:   "dc:title",
			value: "New Title",
			want:  `<dc:title>New Title</dc:title>`,
		},
		{
			name:  "tag with attributes",
			s:     `<dc:creator opf:role="aut">Old Author</dc:creator>`,
			tag:   "dc:creator",
			value: "New Author",
			want:  `<dc:creator opf:role="aut">New Author</dc:creator>`,
		},
		{
			name:  "value with special chars",
			s:     `<dc:title>Old</dc:title>`,
			tag:   "dc:title",
			value: "A & B",
			want:  `<dc:title>A &amp; B</dc:title>`,
		},
		{
			name:    "tag not found",
			s:       `<dc:creator>Author</dc:creator>`,
			tag:     "dc:title",
			wantErr: true,
		},
		{
			name:    "malformed — no closing gt on opening tag",
			s:       `<dc:title`,
			tag:     "dc:title",
			wantErr: true,
		},
		{
			name:    "missing closing tag",
			s:       `<dc:title>No close`,
			tag:     "dc:title",
			wantErr: true,
		},
		{
			name:  "only first occurrence replaced",
			s:     `<dc:title>First</dc:title><dc:title>Second</dc:title>`,
			tag:   "dc:title",
			value: "New",
			want:  `<dc:title>New</dc:title><dc:title>Second</dc:title>`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := replaceFirstElement(tc.s, tc.tag, tc.value)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil (result: %q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// ── injectElement ─────────────────────────────────────────────────────────────

func TestInjectElement(t *testing.T) {
	base := `<metadata>
  <dc:creator>Author</dc:creator>
</metadata>`

	got := injectElement(base, "dc:title", "My Title")
	if !bytes.Contains([]byte(got), []byte("<dc:title>My Title</dc:title>")) {
		t.Errorf("injected element not found in output:\n%s", got)
	}
	// Must appear before </metadata>
	titlePos := bytes.Index([]byte(got), []byte("<dc:title>"))
	closePos := bytes.Index([]byte(got), []byte("</metadata>"))
	if titlePos == -1 || titlePos >= closePos {
		t.Errorf("injected element is not before </metadata>:\n%s", got)
	}
}

func TestInjectElementNoMetadata(t *testing.T) {
	s := `<package></package>`
	got := injectElement(s, "dc:title", "Title")
	if got != s {
		t.Errorf("expected original string unchanged, got %q", got)
	}
}

func TestInjectElementEscapesValue(t *testing.T) {
	base := `<metadata></metadata>`
	got := injectElement(base, "dc:title", "Rock & Roll")
	if !bytes.Contains([]byte(got), []byte("Rock &amp; Roll")) {
		t.Errorf("expected XML-escaped value in output:\n%s", got)
	}
}

// ── patchOPF ─────────────────────────────────────────────────────────────────

func TestPatchOPF_Replace(t *testing.T) {
	orig := []byte(opfXML("Old Title", "Old Author"))
	out, err := patchOPF(orig, Metadata{Title: "New Title", Author: "New Author"})
	if err != nil {
		t.Fatalf("patchOPF: %v", err)
	}
	if !bytes.Contains(out, []byte("New Title")) {
		t.Errorf("new title not in output:\n%s", out)
	}
	if bytes.Contains(out, []byte("Old Title")) {
		t.Errorf("old title still in output:\n%s", out)
	}
	if !bytes.Contains(out, []byte("New Author")) {
		t.Errorf("new author not in output:\n%s", out)
	}
}

func TestPatchOPF_EmptyAuthorSkipped(t *testing.T) {
	orig := []byte(opfXML("Title", "Original Author"))
	out, err := patchOPF(orig, Metadata{Title: "Title"})
	if err != nil {
		t.Fatalf("patchOPF: %v", err)
	}
	if !bytes.Contains(out, []byte("Original Author")) {
		t.Errorf("author should be unchanged when empty author passed:\n%s", out)
	}
}

func TestPatchOPF_InjectMissingTitle(t *testing.T) {
	noTitle := `<?xml version="1.0"?>
<package>
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:creator>Author</dc:creator>
  </metadata>
</package>`
	out, err := patchOPF([]byte(noTitle), Metadata{Title: "Injected Title"})
	if err != nil {
		t.Fatalf("patchOPF: %v", err)
	}
	if !bytes.Contains(out, []byte("Injected Title")) {
		t.Errorf("title was not injected:\n%s", out)
	}
}

func TestPatchOPF_AllFields(t *testing.T) {
	orig := []byte(opfXMLFull(Metadata{
		Title: "Old", Author: "Old", Publisher: "Old Publisher",
		Language: "fr", Description: "Old desc", Subject: "Old subject",
	}))
	m := Metadata{
		Title:       "New Title",
		Author:      "New Author",
		Publisher:   "New Publisher",
		Language:    "en",
		Description: "New description",
		Subject:     "New subject",
	}
	out, err := patchOPF(orig, m)
	if err != nil {
		t.Fatalf("patchOPF: %v", err)
	}
	for _, want := range []string{
		"New Title", "New Author", "New Publisher", "en", "New description", "New subject",
	} {
		if !bytes.Contains(out, []byte(want)) {
			t.Errorf("expected %q in patched OPF:\n%s", want, out)
		}
	}
}

// ── ReadMetadata ──────────────────────────────────────────────────────────────

func TestReadMetadata(t *testing.T) {
	path := buildTestEPUB(t, "The Go Programming Language", "Alan Donovan")
	m, err := ReadMetadata(path)
	if err != nil {
		t.Fatalf("ReadMetadata: %v", err)
	}
	if m.Title != "The Go Programming Language" {
		t.Errorf("Title = %q, want %q", m.Title, "The Go Programming Language")
	}
	if m.Author != "Alan Donovan" {
		t.Errorf("Author = %q, want %q", m.Author, "Alan Donovan")
	}
	if m.FilePath != path {
		t.Errorf("FilePath = %q, want %q", m.FilePath, path)
	}
}

func TestReadMetadata_AllFields(t *testing.T) {
	original := Metadata{
		Title:       "Dune",
		Author:      "Frank Herbert",
		Publisher:   "Chilton Books",
		Language:    "en",
		Description: "Epic science fiction novel set in the far future.",
		Subject:     "Science Fiction",
	}
	path := buildTestEPUBFull(t, original)
	m, err := ReadMetadata(path)
	if err != nil {
		t.Fatalf("ReadMetadata: %v", err)
	}
	if m.Title != original.Title {
		t.Errorf("Title = %q, want %q", m.Title, original.Title)
	}
	if m.Author != original.Author {
		t.Errorf("Author = %q, want %q", m.Author, original.Author)
	}
	if m.Publisher != original.Publisher {
		t.Errorf("Publisher = %q, want %q", m.Publisher, original.Publisher)
	}
	if m.Language != original.Language {
		t.Errorf("Language = %q, want %q", m.Language, original.Language)
	}
	if m.Description != original.Description {
		t.Errorf("Description = %q, want %q", m.Description, original.Description)
	}
	if m.Subject != original.Subject {
		t.Errorf("Subject = %q, want %q", m.Subject, original.Subject)
	}
}

func TestReadMetadata_SpecialCharsInOPF(t *testing.T) {
	path := buildTestEPUB(t, "Lords &amp; Ladies", "Terry Pratchett")
	m, err := ReadMetadata(path)
	if err != nil {
		t.Fatalf("ReadMetadata: %v", err)
	}
	// xml.Unmarshal should decode the entity
	if m.Title != "Lords & Ladies" {
		t.Errorf("Title = %q, want %q", m.Title, "Lords & Ladies")
	}
}

func TestReadMetadata_NotFound(t *testing.T) {
	_, err := ReadMetadata("/nonexistent/path/book.epub")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestReadMetadata_NotAnEPUB(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "*.epub")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("this is not a zip file")
	f.Close()

	_, err = ReadMetadata(f.Name())
	if err == nil {
		t.Error("expected error for invalid epub, got nil")
	}
}

// ── WriteMetadata ─────────────────────────────────────────────────────────────

func TestWriteMetadata_RoundTrip(t *testing.T) {
	path := buildTestEPUB(t, "Original Title", "Original Author")

	err := WriteMetadata(Metadata{
		FilePath: path,
		Title:    "Updated Title",
		Author:   "Updated Author",
	})
	if err != nil {
		t.Fatalf("WriteMetadata: %v", err)
	}

	m, err := ReadMetadata(path)
	if err != nil {
		t.Fatalf("ReadMetadata after write: %v", err)
	}
	if m.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", m.Title, "Updated Title")
	}
	if m.Author != "Updated Author" {
		t.Errorf("Author = %q, want %q", m.Author, "Updated Author")
	}
}

func TestWriteMetadata_AllFieldsRoundTrip(t *testing.T) {
	original := Metadata{
		Title:       "Neuromancer",
		Author:      "William Gibson",
		Publisher:   "Ace Books",
		Language:    "en",
		Description: "A seminal cyberpunk novel.",
		Subject:     "Cyberpunk, Science Fiction",
	}
	path := buildTestEPUBFull(t, original)

	updated := Metadata{
		FilePath:    path,
		Title:       "Neuromancer (Updated)",
		Author:      "W. Gibson",
		Publisher:   "Ace",
		Language:    "en-US",
		Description: "A groundbreaking cyberpunk novel.",
		Subject:     "Science Fiction",
	}
	if err := WriteMetadata(updated); err != nil {
		t.Fatalf("WriteMetadata: %v", err)
	}

	m, err := ReadMetadata(path)
	if err != nil {
		t.Fatalf("ReadMetadata: %v", err)
	}
	if m.Title != updated.Title {
		t.Errorf("Title = %q, want %q", m.Title, updated.Title)
	}
	if m.Publisher != updated.Publisher {
		t.Errorf("Publisher = %q, want %q", m.Publisher, updated.Publisher)
	}
	if m.Language != updated.Language {
		t.Errorf("Language = %q, want %q", m.Language, updated.Language)
	}
	if m.Description != updated.Description {
		t.Errorf("Description = %q, want %q", m.Description, updated.Description)
	}
	if m.Subject != updated.Subject {
		t.Errorf("Subject = %q, want %q", m.Subject, updated.Subject)
	}
}

func TestWriteMetadata_PreservesOtherFiles(t *testing.T) {
	path := buildTestEPUB(t, "Title", "Author")

	if err := WriteMetadata(Metadata{FilePath: path, Title: "New", Author: "New"}); err != nil {
		t.Fatalf("WriteMetadata: %v", err)
	}

	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open result zip: %v", err)
	}
	defer r.Close()

	names := map[string]bool{}
	for _, f := range r.File {
		names[f.Name] = true
	}
	for _, want := range []string{"mimetype", "META-INF/container.xml", "OEBPS/content.opf"} {
		if !names[want] {
			t.Errorf("entry %q missing from written epub", want)
		}
	}
}

func TestWriteMetadata_SpecialChars(t *testing.T) {
	path := buildTestEPUB(t, "Old", "Old")

	if err := WriteMetadata(Metadata{FilePath: path, Title: "Rock & Roll", Author: "AC/DC"}); err != nil {
		t.Fatalf("WriteMetadata: %v", err)
	}

	m, err := ReadMetadata(path)
	if err != nil {
		t.Fatalf("ReadMetadata: %v", err)
	}
	if m.Title != "Rock & Roll" {
		t.Errorf("Title = %q, want %q", m.Title, "Rock & Roll")
	}
	if m.Author != "AC/DC" {
		t.Errorf("Author = %q, want %q", m.Author, "AC/DC")
	}
}

// ── ScanDir ───────────────────────────────────────────────────────────────────

func TestScanDir(t *testing.T) {
	dir := t.TempDir()

	// Create two epubs directly in dir (not via buildTestEPUB's own subdir)
	writeEPUBTo := func(filename, title, author string) {
		t.Helper()
		path := filepath.Join(dir, filename)
		var buf bytes.Buffer
		w := zip.NewWriter(&buf)
		add := func(name, content string) {
			f, _ := w.Create(name)
			f.Write([]byte(content))
		}
		add("mimetype", "application/epub+zip")
		add("META-INF/container.xml", containerXML)
		add("OEBPS/content.opf", opfXML(title, author))
		w.Close()
		os.WriteFile(path, buf.Bytes(), 0644)
	}

	writeEPUBTo("alpha.epub", "Alpha Book", "Author A")
	writeEPUBTo("beta.epub", "Beta Book", "Author B")
	// A non-epub file that must be ignored
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignore me"), 0644)

	books, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}
	if len(books) != 2 {
		t.Fatalf("got %d books, want 2", len(books))
	}
}

func TestScanDir_Empty(t *testing.T) {
	books, err := ScanDir(t.TempDir())
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}
	if len(books) != 0 {
		t.Errorf("expected 0 books in empty dir, got %d", len(books))
	}
}

func TestScanDir_InvalidDir(t *testing.T) {
	_, err := ScanDir("/nonexistent/directory")
	if err == nil {
		t.Error("expected error for nonexistent directory, got nil")
	}
}

func TestReadMetadata_MalformedContainerXML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.epub")

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	add := func(name, content string) {
		f, _ := w.Create(name)
		f.Write([]byte(content))
	}
	add("mimetype", "application/epub+zip")
	add("META-INF/container.xml", "<<<not xml>>>")
	w.Close()
	os.WriteFile(path, buf.Bytes(), 0644)

	_, err := ReadMetadata(path)
	if err == nil {
		t.Error("expected error for malformed container.xml, got nil")
	}
}

func TestReadMetadata_MalformedOPFXML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.epub")

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	add := func(name, content string) {
		f, _ := w.Create(name)
		f.Write([]byte(content))
	}
	add("mimetype", "application/epub+zip")
	add("META-INF/container.xml", containerXML)
	add("OEBPS/content.opf", "<<<not xml>>>")
	w.Close()
	os.WriteFile(path, buf.Bytes(), 0644)

	_, err := ReadMetadata(path)
	if err == nil {
		t.Error("expected error for malformed OPF XML, got nil")
	}
}

func TestReadMetadata_MissingOPFFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.epub")

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	add := func(name, content string) {
		f, _ := w.Create(name)
		f.Write([]byte(content))
	}
	add("mimetype", "application/epub+zip")
	// container.xml points to OEBPS/content.opf but the file is absent
	add("META-INF/container.xml", containerXML)
	w.Close()
	os.WriteFile(path, buf.Bytes(), 0644)

	_, err := ReadMetadata(path)
	if err == nil {
		t.Error("expected error when OPF file is missing from zip, got nil")
	}
}

func TestScanDir_CorruptEPUBIncluded(t *testing.T) {
	dir := t.TempDir()
	// Write a file with .epub extension that is not a valid zip
	os.WriteFile(filepath.Join(dir, "broken.epub"), []byte("not a zip"), 0644)

	books, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}
	// The broken file should still appear (with empty metadata)
	if len(books) != 1 {
		t.Fatalf("got %d books, want 1 (the corrupt epub)", len(books))
	}
	if books[0].Title != "" || books[0].Author != "" {
		t.Errorf("corrupt epub should have empty metadata, got title=%q author=%q",
			books[0].Title, books[0].Author)
	}
}
