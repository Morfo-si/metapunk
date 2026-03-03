package epub

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Metadata holds the editable metadata for an EPUB file.
type Metadata struct {
	Title    string
	Author   string
	FilePath string
}

// container.xml structure
type container struct {
	Rootfiles []rootfile `xml:"rootfiles>rootfile"`
}

type rootfile struct {
	FullPath string `xml:"full-path,attr"`
}

// OPF package structure (enough to read/write title and creator)
type opfPackage struct {
	XMLName  xml.Name    `xml:"package"`
	Metadata opfMetadata `xml:"metadata"`
	raw      []byte      // original bytes for round-trip fidelity
}

type opfMetadata struct {
	Titles   []string     `xml:"title"`
	Creators []opfCreator `xml:"creator"`
}

type opfCreator struct {
	Name string `xml:",chardata"`
	Role string `xml:"role,attr,omitempty"`
	ID   string `xml:"id,attr,omitempty"`
}

// ReadMetadata opens an EPUB file and extracts its title and author.
func ReadMetadata(path string) (Metadata, error) {
	m := Metadata{FilePath: path}

	r, err := zip.OpenReader(path)
	if err != nil {
		return m, fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	opfPath, err := findOPFPath(r)
	if err != nil {
		return m, err
	}

	opfBytes, err := readZipEntry(r, opfPath)
	if err != nil {
		return m, fmt.Errorf("read OPF: %w", err)
	}

	var pkg opfPackage
	if err := xml.Unmarshal(opfBytes, &pkg); err != nil {
		return m, fmt.Errorf("parse OPF: %w", err)
	}

	if len(pkg.Metadata.Titles) > 0 {
		m.Title = strings.TrimSpace(pkg.Metadata.Titles[0])
	}
	if len(pkg.Metadata.Creators) > 0 {
		m.Author = strings.TrimSpace(pkg.Metadata.Creators[0].Name)
	}

	return m, nil
}

// WriteMetadata updates the title and author in the EPUB's OPF file.
// It writes atomically: builds a new zip in memory, then replaces the original.
func WriteMetadata(m Metadata) error {
	r, err := zip.OpenReader(m.FilePath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	opfPath, err := findOPFPath(r)
	if err != nil {
		return err
	}

	// Build the new zip in a buffer
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	for _, f := range r.File {
		if f.Name == opfPath {
			// Read original OPF
			opfBytes, err := readZipEntry(r, opfPath)
			if err != nil {
				return fmt.Errorf("read OPF: %w", err)
			}

			// Patch metadata
			updated, err := patchOPF(opfBytes, m.Title, m.Author)
			if err != nil {
				return fmt.Errorf("patch OPF: %w", err)
			}

			// Write patched OPF
			fw, err := w.CreateHeader(&f.FileHeader)
			if err != nil {
				return fmt.Errorf("create OPF entry: %w", err)
			}
			if _, err := fw.Write(updated); err != nil {
				return fmt.Errorf("write OPF entry: %w", err)
			}
		} else {
			// Copy all other files verbatim
			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("open entry %s: %w", f.Name, err)
			}
			fw, err := w.CreateHeader(&f.FileHeader)
			if err != nil {
				rc.Close()
				return fmt.Errorf("create entry %s: %w", f.Name, err)
			}
			if _, err := io.Copy(fw, rc); err != nil {
				rc.Close()
				return fmt.Errorf("copy entry %s: %w", f.Name, err)
			}
			rc.Close()
		}
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("finalize zip: %w", err)
	}

	// Write to temp file then rename for atomicity
	dir := filepath.Dir(m.FilePath)
	tmp, err := os.CreateTemp(dir, ".metapunk-*.epub")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(buf.Bytes()); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpName, m.FilePath); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("replace file: %w", err)
	}

	return nil
}

// ScanDir returns Metadata for every .epub file in dir.
func ScanDir(dir string) ([]Metadata, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var results []Metadata
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".epub") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		m, err := ReadMetadata(path)
		if err != nil {
			// Still include the file, just with empty metadata
			m = Metadata{FilePath: path}
		}
		results = append(results, m)
	}
	return results, nil
}

// findOPFPath reads META-INF/container.xml to locate the OPF file.
func findOPFPath(r *zip.ReadCloser) (string, error) {
	data, err := readZipEntry(r, "META-INF/container.xml")
	if err != nil {
		return "", fmt.Errorf("read container.xml: %w", err)
	}

	var c container
	if err := xml.Unmarshal(data, &c); err != nil {
		return "", fmt.Errorf("parse container.xml: %w", err)
	}
	if len(c.Rootfiles) == 0 {
		return "", fmt.Errorf("no rootfile found in container.xml")
	}
	return c.Rootfiles[0].FullPath, nil
}

// readZipEntry reads a named file from an open zip archive.
func readZipEntry(r *zip.ReadCloser, name string) ([]byte, error) {
	for _, f := range r.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("file %q not found in epub", name)
}

// patchOPF does a targeted string replacement on the raw OPF XML so that
// namespace declarations and other formatting are preserved.
func patchOPF(data []byte, title, author string) ([]byte, error) {
	s := string(data)

	s, err := replaceFirstElement(s, "dc:title", title)
	if err != nil {
		// Element might not exist — inject it into <metadata>
		s = injectElement(s, "dc:title", title)
	}

	if author != "" {
		s, err = replaceFirstElement(s, "dc:creator", author)
		if err != nil {
			s = injectElement(s, "dc:creator", author)
		}
	}

	return []byte(s), nil
}

// replaceFirstElement replaces the text content of the first occurrence of
// <tag ...>...</tag> in s. Returns an error if the tag is not found.
func replaceFirstElement(s, tag, value string) (string, error) {
	open := "<" + tag
	close := "</" + tag + ">"

	start := strings.Index(s, open)
	if start == -1 {
		return s, fmt.Errorf("element %s not found", tag)
	}

	// Find the end of the opening tag (could have attributes)
	gtPos := strings.Index(s[start:], ">")
	if gtPos == -1 {
		return s, fmt.Errorf("malformed element %s", tag)
	}
	contentStart := start + gtPos + 1

	end := strings.Index(s[contentStart:], close)
	if end == -1 {
		return s, fmt.Errorf("closing tag for %s not found", tag)
	}
	contentEnd := contentStart + end

	return s[:contentStart] + xmlEscape(value) + s[contentEnd:], nil
}

// injectElement inserts a new element just before </metadata>.
func injectElement(s, tag, value string) string {
	closeMetadata := "</metadata>"
	pos := strings.Index(s, closeMetadata)
	if pos == -1 {
		return s
	}
	elem := fmt.Sprintf("    <%s>%s</%s>\n    ", tag, xmlEscape(value), tag)
	return s[:pos] + elem + s[pos:]
}

// xmlEscape returns the XML-escaped form of s.
func xmlEscape(s string) string {
	var b strings.Builder
	xml.EscapeText(&b, []byte(s))
	return b.String()
}
