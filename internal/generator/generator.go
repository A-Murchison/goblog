package generator

import (
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
	Posts []*PostView // populated for the index page
	Year  int
}

// Build reads posts from rootDir/posts/, renders HTML, and writes to outputDir.
func Build(rootDir, outputDir string) error {
	site, err := config.Load(rootDir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	tmplDir := filepath.Join(rootDir, "templates")
	baseTmpl := filepath.Join(tmplDir, "base.html")
	postTmpl := filepath.Join(tmplDir, "post.html")
	indexTmpl := filepath.Join(tmplDir, "index.html")

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

	// Render each post page.
	for _, post := range posts {
		tmpl, err := template.ParseFiles(baseTmpl, postTmpl)
		if err != nil {
			return fmt.Errorf("parsing post template: %w", err)
		}
		outPath := filepath.Join(postsOutDir, post.Slug+".html")
		f, err := os.Create(outPath)
		if err != nil {
			return fmt.Errorf("creating %s: %w", outPath, err)
		}
		err = tmpl.ExecuteTemplate(f, "base", PageData{Site: site, Post: post, Year: year})
		f.Close()
		if err != nil {
			return fmt.Errorf("rendering post %s: %w", post.Slug, err)
		}
		fmt.Printf("  Built: posts/%s.html\n", post.Slug)
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

	// Copy static/ directory if it exists.
	staticDir := filepath.Join(rootDir, "static")
	if _, err := os.Stat(staticDir); err == nil {
		if err := copyDir(staticDir, filepath.Join(outputDir, "static")); err != nil {
			return fmt.Errorf("copying static assets: %w", err)
		}
		fmt.Println("  Copied: static/")
	}

	return nil
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
			Content:     template.HTML(post.Content), // safe: generated from our own markdown
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
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}
