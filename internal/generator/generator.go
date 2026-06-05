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
	Content     template.HTML
}

// PageData is passed to every template execution.
type PageData struct {
	Site  *config.Site
	Post  *PostView   // populated for post pages
	Posts []*PostView // populated for the index and tag pages
	Tag   string      // populated for tag pages
	Year  int
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

	// Clean and recreate output directory.
	if err := os.RemoveAll(outputDir); err != nil {
		return fmt.Errorf("cleaning output dir: %w", err)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	posts, err := loadPosts(filepath.Join(rootDir, "posts"))
	if err != nil {
		return fmt.Errorf("loading posts: %w", err)
	}

	// Most-recent first.
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date.After(posts[j].Date)
	})

	postsOutDir := filepath.Join(outputDir, "posts")
	if err := os.MkdirAll(postsOutDir, 0755); err != nil {
		return fmt.Errorf("creating posts output dir: %w", err)
	}

	year := time.Now().Year()

	// Render each post page (clean URLs: posts/{slug}/index.html).
	for _, post := range posts {
		tmpl, err := template.ParseFiles(baseTmpl, postTmpl)
		if err != nil {
			return fmt.Errorf("parsing post template: %w", err)
		}
		postDir := filepath.Join(postsOutDir, post.Slug)
		if err := os.MkdirAll(postDir, 0755); err != nil {
			return fmt.Errorf("creating post dir %s: %w", postDir, err)
		}
		outPath := filepath.Join(postDir, "index.html")
		f, err := os.Create(outPath)
		if err != nil {
			return fmt.Errorf("creating %s: %w", outPath, err)
		}
		err = tmpl.ExecuteTemplate(f, "base", PageData{Site: site, Post: post, Year: year})
		f.Close()
		if err != nil {
			return fmt.Errorf("rendering post %s: %w", post.Slug, err)
		}
		fmt.Printf("  Built: posts/%s/\n", post.Slug)
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
		err = tmpl.ExecuteTemplate(f, "base", PageData{Site: site, Posts: posts, Year: year})
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
	for _, post := range posts {
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

	// Generate sitemap.xml.
	if err := renderSitemap(outputDir, site, posts, tagNames); err != nil {
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

func loadPosts(postsDir string) ([]*PostView, error) {
	entries, err := os.ReadDir(postsDir)
	if err != nil {
		return nil, fmt.Errorf("reading posts dir: %w", err)
	}

	var posts []*PostView
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(postsDir, entry.Name())
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

		posts = append(posts, &PostView{
			Title:       title,
			Date:        post.Date,
			Description: post.Description,
			Tags:        post.Tags,
			Slug:        slug,
			Content:     template.HTML(rewriteStaticPaths(post.Content)), // safe: HTML passthrough disabled in renderer
		})
	}
	return posts, nil
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

func renderSitemap(outputDir string, site *config.Site, posts []*PostView, tags []string) error {
	base := strings.TrimRight(site.BaseURL, "/")
	var urls []sitemapURL
	urls = append(urls, sitemapURL{Loc: base + "/"})
	for _, post := range posts {
		urls = append(urls, sitemapURL{Loc: base + "/posts/" + post.Slug + "/"})
	}
	for _, tag := range tags {
		urls = append(urls, sitemapURL{Loc: base + "/tags/" + tag + "/"})
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
