# goblog

[example](https://a-murchison.github.io/goblog)

A minimal static site generator written in Go. Host your site free in Github Pages.

Drop Markdown files into `posts/`, run one command, get a blog.

Configure the items in the templates folder to make it your own.

## Requirements

Go 1.26+

## Quick Start

### 1. Create your repo from the template

Click **Use this template → Create a new repository** on GitHub, then clone your new repo:

```bash
git clone https://github.com/yourusername/your-repo-name
cd your-repo-name
```

Edit `config.json`:

```json
{
  "title": "My Blog",
  "description": "Thoughts on software and life.",
  "baseURL": "https://yourusername.github.io/goblog",
  "author": "Your Name"
}
```

### 2. Write a post

A sample post (`posts/hello-world.md`) is included - delete or replace it.

Create a `.md` file in `posts/`:

```markdown
---
title: My First Post
date: 2026-06-05
description: A short description for SEO and previews.
tags: [go, web]
---

Your post content here...
```

### 3. Build

```bash
go run main.go build
```

Static files are written to `public/`.

### 4. Preview locally

```bash
go run main.go serve        # http://localhost:8080
go run main.go serve 3000   # http://localhost:3000
```

## Deploying to GitHub Pages

1. Push the repo to GitHub
2. Go to **Settings → Pages**
3. Under **Build and deployment → Source**, select **GitHub Actions**

The included workflow builds and deploys on every push to `main`.

## Features

- Markdown → HTML via [gomarkdown](https://github.com/gomarkdown/markdown)
- YAML frontmatter (`title`, `date`, `description`, `tags`)
- Self-contained HTML templates (inline CSS, no external dependencies)
- Optional `static/` directory for assets
- GitHub Actions workflow for auto-deploy to GitHub Pages on push to `main`

## Project Structure

```
goblog/
├── internal/
│   ├── config/config.go        # Loads config.json
│   ├── parser/markdown.go      # Markdown + frontmatter → HTML
│   └── generator/generator.go  # Orchestrates the build
├── templates/
│   ├── base.html               # Shared HTML layout + CSS
│   ├── post.html               # Individual post view
│   ├── index.html              # Post listing
│   └── tag.html                # Tag listing
├── posts/                      # Your Markdown blog posts
│   └── hello-world.md
├── static/                     # Optional: images, fonts, etc.
├── public/                     # Generated output (git-ignored)
├── config.json                 # Site metadata
├── main.go                     # CLI entrypoint
└── .github/workflows/deploy.yml
```

## Static Assets

Place files in `static/` - they are copied to `public/static/` at build time.

Reference them with root-absolute paths - the generator automatically rewrites to the correct base URL at build time:

```markdown
![My image](/static/my-image.png)
```

Paths resolve to `localhost:8080/static/...` when serving, or `baseURL/static/...` when building.

## Frontmatter Reference

| Key           | Required | Example               |
| ------------- | -------- | --------------------- |
| `title`       | No       | `My Post Title`       |
| `date`        | No       | `2026-06-05`          |
| `description` | No       | `A short summary.`    |
| `tags`        | No       | `[go, web, tutorial]` |

If `title` is omitted, the filename is used (hyphens → spaces).
