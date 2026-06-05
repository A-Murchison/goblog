# goblog

A fast, minimal **Static Site Generator** written in Go. Drop Markdown files into `posts/`, run one command, and get a static HTML blog ready to deploy on GitHub Pages - no database, no server, no build tools beyond Go itself.

## Features

- Markdown → HTML via [gomarkdown](https://github.com/gomarkdown/markdown)
- YAML frontmatter (`title`, `date`, `description`, `tags`)
- Clean, self-contained HTML templates (inline CSS - no external dependencies)
- Optional `static/` directory for images and other assets
- GitHub Actions workflow - auto-builds and deploys to GitHub Pages on every push to `main`

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
│   └── index.html              # Post listing
├── posts/                      # Your Markdown blog posts
│   └── hello-world.md
├── static/                     # Optional: images, fonts, etc.
├── public/                     # Generated output (git-ignored)
├── config.json                 # Site metadata
├── main.go                     # CLI entrypoint
└── .github/workflows/deploy.yml
```

## Quick Start

### 1. Clone and configure

```bash
git clone https://github.com/A-Murchison/goblog
cd goblog
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
go run main.go serve
```

Open [http://localhost:8080](http://localhost:8080).

Or with a custom port

```bash
go run main.go serve 3000
```

Open [http://localhost:3000](http://localhost:3000).

## Deploying to GitHub Pages

### One-time setup

1. Push this repo to GitHub
2. Go to **Settings → Pages**
3. Under **Source**, select **GitHub Actions**

The `.github/workflows/deploy.yml` workflow builds the site and deploys it automatically on every push to `main`.

## Adding Static Assets

Place files in the `static/` directory - they are copied to `public/static/` during the build.

Use root-absolute paths in your Markdown - the generator automatically rewrites them to the correct base URL at build time:

```markdown
![My image](/static/my-image.png)
```

When running `go run main.go serve`, paths resolve to `http://localhost:8080/static/...`. When running `go run main.go build`, they resolve to the `baseURL` in `config.json`.

## Frontmatter Reference

| Key           | Required | Example               |
| ------------- | -------- | --------------------- |
| `title`       | No       | `My Post Title`       |
| `date`        | No       | `2026-06-05`          |
| `description` | No       | `A short summary.`    |
| `tags`        | No       | `[go, web, tutorial]` |

If `title` is omitted, the filename is used (hyphens replaced with spaces).

├── templates/ ← HTML templates (header, footer, layout)
│ └── layout.html
│
├── main.go ← Your Go CLI tool
│
└── public/ ← GENERATED output (HTML files)
├── index.html
├── hello-world/
│ └── index.html
└── my-go-journey/
└── index.html

```

```
