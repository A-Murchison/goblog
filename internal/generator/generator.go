package generator

import (
	"encoding/xml"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"goblog/internal/config"
	"goblog/internal/parser"
)

// PostView is a post ready for template rendering (Content is safe HTML).
type PostView struct {
	Title       string
	Date        time.Time
	Description string
	Tags        []string
	Slug        string
	Type        string
	Image       string
	Link        string
	Content     template.HTML
}

// PageData is passed to every template execution.
type PageData struct {
	Site        *config.Site
	Post        *PostView   // populated for post pages
	Posts       []*PostView // populated for the index and tag pages
	Tag         string      // populated for tag pages
	ContentType string      // populated for content-type listing pages
	Year        int
}

// Build reads posts from rootDir/posts/, renders HTML, and writes to outputDir.
// baseURLOverride, if non-empty, replaces site.BaseURL (useful for local serve).
func Build(rootDir, outputDir, baseURLOverride string) error {
	site, err := config.Load(rootDir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if baseURLOverride != "" {
		site.BaseURL = strings.TrimRight(baseURLOverride, "/") + "/"
	}

	tmplDir := filepath.Join(rootDir, "templates")
	baseTmpl := filepath.Join(tmplDir, "base.html")
	postTmpl := filepath.Join(tmplDir, "post.html")
	indexTmpl := filepath.Join(tmplDir, "index.html")
	tagTmpl := filepath.Join(tmplDir, "tag.html")
	pageTmpl := filepath.Join(tmplDir, "page.html")

	// Clean and recreate output directory.
	if err := os.RemoveAll(outputDir); err != nil {
		return fmt.Errorf("cleaning output dir: %w", err)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	year := time.Now().Year()

	// Reserved output directory names that content types must not collide with.
	reservedNames := map[string]bool{"tags": true, "static": true, "pages": true}

	// Validate all content types before doing any filesystem work.
	for _, contentType := range site.ContentTypes {
		if contentType == "" {
			return fmt.Errorf("invalid content type: name must not be empty")
		}
		// Reject anything that is not a plain single directory name (no separators, no . or ..).
		if filepath.Base(contentType) != contentType || contentType == "." || contentType == ".." {
			return fmt.Errorf("invalid content type %q: must be a single directory name with no path separators", contentType)
		}
		if reservedNames[contentType] {
			return fmt.Errorf("invalid content type %q: name is reserved", contentType)
		}
	}

	// Load and render all configured content types.
	// contentItems maps each type to its sorted items for use by the homepage and tag pages.
	contentItems := make(map[string][]*PostView)
	var allItems []*PostView

	for _, contentType := range site.ContentTypes {
		items, err := loadContent(filepath.Join(rootDir, contentType), contentType)
		if err != nil {
			return fmt.Errorf("loading %s: %w", contentType, err)
		}
		// Most-recent first.
		sort.Slice(items, func(i, j int) bool {
			return items[i].Date.After(items[j].Date)
		})
		contentItems[contentType] = items
		allItems = append(allItems, items...)

		typeOutDir := filepath.Join(outputDir, contentType)
		if err := os.MkdirAll(typeOutDir, 0755); err != nil {
			return fmt.Errorf("creating %s output dir: %w", contentType, err)
		}

		// Render each item page (clean URLs: {type}/{slug}/index.html).
		tmpl, err := template.ParseFiles(baseTmpl, postTmpl)
		if err != nil {
			return fmt.Errorf("parsing post template: %w", err)
		}
		for _, item := range items {
			itemDir := filepath.Join(typeOutDir, item.Slug)
			if err := os.MkdirAll(itemDir, 0755); err != nil {
				return fmt.Errorf("creating item dir %s: %w", itemDir, err)
			}
			outPath := filepath.Join(itemDir, "index.html")
			f, err := os.Create(outPath)
			if err != nil {
				return fmt.Errorf("creating %s: %w", outPath, err)
			}
			err = tmpl.ExecuteTemplate(f, "base", PageData{Site: site, Post: item, Year: year})
			f.Close()
			if err != nil {
				return fmt.Errorf("rendering %s/%s: %w", contentType, item.Slug, err)
			}
			fmt.Printf("  Built: %s/%s/\n", contentType, item.Slug)
		}
	}

	// Homepage shows the 5 most recent posts
	homepageItems := contentItems["posts"]
	const maxHomepagePosts = 5
	if len(homepageItems) > maxHomepagePosts {
		homepageItems = homepageItems[:maxHomepagePosts]
	}

	// Render index page.
	{
		tmpl, err := template.ParseFiles(baseTmpl, indexTmpl)
		if err != nil {
			return fmt.Errorf("parsing index template: %w", err)
		}
		outPath := filepath.Join(outputDir, "index.html")
		f, err := os.Create(outPath)
		if err != nil {
			return fmt.Errorf("creating index.html: %w", err)
		}
		err = tmpl.ExecuteTemplate(f, "base", PageData{Site: site, Posts: homepageItems, Year: year})
		f.Close()
		if err != nil {
			return fmt.Errorf("rendering index: %w", err)
		}
		fmt.Println("  Built: index.html")
	}

	// Build tag index: group posts by tag.
	// tagsOutDir is defined here so tags can be validated against it before use.
	tagsOutDir := filepath.Join(outputDir, "tags")
	tagMap := make(map[string][]*PostView)
	for _, post := range allItems {
		for _, tag := range post.Tags {
			// Safety: reject tags that would escape the tags output directory.
			candidate := filepath.Join(tagsOutDir, tag)
			rel, relErr := filepath.Rel(tagsOutDir, candidate)
			if relErr != nil || rel == "." || strings.HasPrefix(rel, "..") {
				fmt.Printf("  Skipping unsafe tag %q\n", tag)
				continue
			}
			tagMap[tag] = append(tagMap[tag], post)
		}
	}

	// Render tag pages (clean URLs: tags/{tag}/index.html).
	if err := os.MkdirAll(tagsOutDir, 0755); err != nil {
		return fmt.Errorf("creating tags output dir: %w", err)
	}
	var tagNames []string
	for tag := range tagMap {
		tagNames = append(tagNames, tag)
	}
	sort.Strings(tagNames)
	for _, tag := range tagNames {
		tmpl, err := template.ParseFiles(baseTmpl, tagTmpl)
		if err != nil {
			return fmt.Errorf("parsing tag template: %w", err)
		}
		tagDir := filepath.Join(tagsOutDir, tag)
		if err := os.MkdirAll(tagDir, 0755); err != nil {
			return fmt.Errorf("creating tag dir %s: %w", tagDir, err)
		}
		outPath := filepath.Join(tagDir, "index.html")
		f, err := os.Create(outPath)
		if err != nil {
			return fmt.Errorf("creating %s: %w", outPath, err)
		}
		err = tmpl.ExecuteTemplate(f, "base", PageData{Site: site, Posts: tagMap[tag], Tag: tag, Year: year})
		f.Close()
		if err != nil {
			return fmt.Errorf("rendering tag %s: %w", tag, err)
		}
		fmt.Printf("  Built: tags/%s/\n", tag)
	}

	// Copy static/ directory if it exists.
	staticDir := filepath.Join(rootDir, "static")
	if _, err := os.Stat(staticDir); err == nil {
		if err := copyDir(staticDir, filepath.Join(outputDir, "static")); err != nil {
			return fmt.Errorf("copying static assets: %w", err)
		}
		fmt.Println("  Copied: static/")
	}

	// Load and render pages/ (static pages such as About).
	var pageItems []*PostView
	pagesDir := filepath.Join(rootDir, "pages")
	if _, statErr := os.Stat(pagesDir); statErr == nil {
		pages, err := loadContent(pagesDir, "page")
		if err != nil {
			return fmt.Errorf("loading pages: %w", err)
		}
		pageItems = pages
		pageT, err := template.ParseFiles(baseTmpl, pageTmpl)
		if err != nil {
			return fmt.Errorf("parsing page template: %w", err)
		}
		for _, pg := range pages {
			if reservedNames[pg.Slug] {
				return fmt.Errorf("invalid page slug %q: name is reserved", pg.Slug)
			}
			for _, ct := range site.ContentTypes {
				if pg.Slug == ct {
					return fmt.Errorf("invalid page slug %q: conflicts with content type", pg.Slug)
				}
			}
			pgDir := filepath.Join(outputDir, pg.Slug)
			if err := os.MkdirAll(pgDir, 0755); err != nil {
				return fmt.Errorf("creating page dir %s: %w", pgDir, err)
			}
			outPath := filepath.Join(pgDir, "index.html")
			f, err := os.Create(outPath)
			if err != nil {
				return fmt.Errorf("creating %s: %w", outPath, err)
			}
			err = pageT.ExecuteTemplate(f, "base", PageData{Site: site, Post: pg, Year: year})
			f.Close()
			if err != nil {
				return fmt.Errorf("rendering page %s: %w", pg.Slug, err)
			}
			fmt.Printf("  Built: %s/\n", pg.Slug)
		}
	}

	// Generate sitemap.xml.
	if err := renderSitemap(outputDir, site, site.ContentTypes, allItems, tagNames, pageItems); err != nil {
		return fmt.Errorf("generating sitemap: %w", err)
	}
	fmt.Println("  Built: sitemap.xml")

	return nil
}

func rewriteStaticPaths(content string) string {
	content = strings.ReplaceAll(content, `src="/static/`, `src="static/`)
	content = strings.ReplaceAll(content, `href="/static/`, `href="static/`)
	return content
}

func loadContent(dir, contentType string) ([]*PostView, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s dir: %w", contentType, err)
	}

	var items []*PostView
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", entry.Name(), err)
		}
		post, err := parser.ParseFile(entry.Name(), data)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}

		slug := strings.TrimSuffix(entry.Name(), ".md")
		title := post.Title
		if title == "" {
			title = strings.ReplaceAll(slug, "-", " ")
		}

		items = append(items, &PostView{
			Title:       title,
			Date:        post.Date,
			Description: post.Description,
			Tags:        post.Tags,
			Slug:        slug,
			Type:        contentType,
			Image:       post.Image,
			Link:        post.Link,
			Content:     template.HTML(rewriteStaticPaths(post.Content)), // safe: HTML passthrough disabled in renderer
		})
	}
	return items, nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}

type sitemapURL struct {
	Loc string `xml:"loc"`
}

type urlSet struct {
	XMLName   xml.Name     `xml:"urlset"`
	Namespace string       `xml:"xmlns,attr"`
	URLs      []sitemapURL `xml:"url"`
}

func renderSitemap(outputDir string, site *config.Site, contentTypes []string, posts []*PostView, tags []string, pages []*PostView) error {
	base := strings.TrimRight(site.BaseURL, "/")
	var urls []sitemapURL
	urls = append(urls, sitemapURL{Loc: base + "/"})
	for _, ct := range contentTypes {
		urls = append(urls, sitemapURL{Loc: base + "/" + ct + "/"})
	}
	for _, post := range posts {
		urls = append(urls, sitemapURL{Loc: base + "/" + post.Type + "/" + post.Slug + "/"})
	}
	for _, tag := range tags {
		urls = append(urls, sitemapURL{Loc: base + "/tags/" + tag + "/"})
	}
	for _, pg := range pages {
		urls = append(urls, sitemapURL{Loc: base + "/" + pg.Slug + "/"})
	}
	data, err := xml.MarshalIndent(urlSet{
		Namespace: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:      urls,
	}, "", "  ")
	if err != nil {
		return err
	}
	content := []byte(xml.Header + string(data))
	return os.WriteFile(filepath.Join(outputDir, "sitemap.xml"), content, 0644)
}
