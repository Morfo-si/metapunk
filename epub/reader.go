package epub

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"
)

// Chapter holds the extracted plain text of one spine item.
type Chapter struct {
	Title string
	Text  string
}

// opfFull is a superset of opfPackage used only for content reading.
// It captures manifest and spine in addition to metadata.
type opfFull struct {
	XMLName  xml.Name    `xml:"package"`
	Metadata opfMetadata `xml:"metadata"`
	Manifest struct {
		Items []struct {
			ID        string `xml:"id,attr"`
			Href      string `xml:"href,attr"`
			MediaType string `xml:"media-type,attr"`
		} `xml:"item"`
	} `xml:"manifest"`
	Spine struct {
		ItemRefs []struct {
			IDRef  string `xml:"idref,attr"`
			Linear string `xml:"linear,attr"`
		} `xml:"itemref"`
	} `xml:"spine"`
}

// ReadChapters opens an EPUB and returns its spine chapters as plain text,
// in reading order. Items marked linear="no" are skipped.
func ReadChapters(path string) ([]Chapter, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	opfPath, err := findOPFPath(r)
	if err != nil {
		return nil, err
	}

	opfBytes, err := readZipEntry(r, opfPath)
	if err != nil {
		return nil, fmt.Errorf("read OPF: %w", err)
	}

	var pkg opfFull
	if err := xml.Unmarshal(opfBytes, &pkg); err != nil {
		return nil, fmt.Errorf("parse OPF: %w", err)
	}

	// Build id→href map for XHTML/HTML items from manifest.
	idToHref := make(map[string]string)
	for _, item := range pkg.Manifest.Items {
		mt := item.MediaType
		if strings.HasPrefix(mt, "application/xhtml") || strings.HasPrefix(mt, "text/html") {
			idToHref[item.ID] = item.Href
		}
	}

	opfDir := filepath.Dir(opfPath)

	num := 0
	var chapters []Chapter
	for _, ref := range pkg.Spine.ItemRefs {
		if ref.Linear == "no" {
			continue
		}
		href, ok := idToHref[ref.IDRef]
		if !ok {
			continue
		}

		fullPath := href
		if opfDir != "." {
			fullPath = opfDir + "/" + href
		}

		data, err := readZipEntry(r, fullPath)
		if err != nil {
			continue // skip unreadable chapters rather than aborting
		}

		num++
		text := extractText(data)
		chapters = append(chapters, Chapter{
			Title: fmt.Sprintf("Chapter %d", num),
			Text:  text,
		})
	}

	return chapters, nil
}

// extractText converts XHTML/HTML bytes to readable plain text by walking
// the XML token stream. Block-level tags produce newlines; script/style
// content is discarded.
func extractText(data []byte) string {
	blockTags := map[string]bool{
		"p": true, "div": true, "section": true, "article": true,
		"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
		"li": true, "tr": true, "br": true, "hr": true, "blockquote": true,
	}
	skipTags := map[string]bool{"script": true, "style": true, "head": true}

	var sb strings.Builder
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	dec.AutoClose = xml.HTMLAutoClose
	dec.Entity = xml.HTMLEntity

	skip := 0
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			name := strings.ToLower(t.Name.Local)
			if skipTags[name] {
				skip++
			}
			if skip == 0 && blockTags[name] {
				// Ensure block elements start on their own line.
				s := sb.String()
				if len(s) > 0 && s[len(s)-1] != '\n' {
					sb.WriteByte('\n')
				}
			}
		case xml.EndElement:
			name := strings.ToLower(t.Name.Local)
			if skipTags[name] && skip > 0 {
				skip--
			}
			if skip == 0 && blockTags[name] && name != "br" && name != "hr" {
				sb.WriteByte('\n')
			}
		case xml.CharData:
			if skip == 0 {
				sb.Write(t)
			}
		}
	}

	// Collapse runs of more than one blank line and trim whitespace from each line.
	raw := strings.Split(sb.String(), "\n")
	var out []string
	blanks := 0
	for _, line := range raw {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			blanks++
			if blanks == 1 {
				out = append(out, "")
			}
		} else {
			blanks = 0
			out = append(out, trimmed)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
