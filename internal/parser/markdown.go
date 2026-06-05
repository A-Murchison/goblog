package parser

import (
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// Post holds the parsed frontmatter and rendered HTML body of a Markdown file.
type Post struct {
	Title       string
	Date        time.Time
	Description string
	Tags        []string
	Content     string // rendered HTML
}

// ParseFile parses a Markdown file with optional YAML frontmatter.
// src is the raw file bytes; filename is used only for error messages.
func ParseFile(_ string, src []byte) (*Post, error) {
	post := &Post{}
	body := parseFrontmatter(src, post)
	post.Content = string(mdToHTML(body))
	return post, nil
}

// parseFrontmatter extracts --- delimited key: value pairs from the top of src,
// populates post fields, and returns the remaining Markdown body.
func parseFrontmatter(src []byte, post *Post) []byte {
	text := string(src)
	lines := strings.Split(text, "\n")

	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return src
	}

	endIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIdx = i
			break
		}
	}
	if endIdx == -1 {
		return src // no closing delimiter
	}

	for _, line := range lines[1:endIdx] {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch strings.ToLower(key) {
		case "title":
			post.Title = strings.Trim(value, `"'`)
		case "date":
			if t, err := time.Parse("2006-01-02", value); err == nil {
				post.Date = t
			}
		case "description":
			post.Description = strings.Trim(value, `"'`)
		case "tags":
			// Accepts: [go, blog] or go, blog
			value = strings.Trim(value, "[]")
			for _, tag := range strings.Split(value, ",") {
				if t := strings.TrimSpace(tag); t != "" {
					post.Tags = append(post.Tags, t)
				}
			}
		}
	}

	body := strings.Join(lines[endIdx+1:], "\n")
	return []byte(body)
}

// mdToHTML converts Markdown bytes to HTML bytes using gomarkdown.
func mdToHTML(src []byte) []byte {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(src)

	flags := html.CommonFlags | html.HrefTargetBlank
	renderer := html.NewRenderer(html.RendererOptions{Flags: flags})
	return markdown.Render(doc, renderer)
}
