package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Site holds site-wide metadata loaded from config.json.
type Site struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	BaseURL      string   `json:"baseURL"`
	Author       string   `json:"author"`
	Tagline      string   `json:"tagline"`
	GitHub       string   `json:"github"`
	LinkedIn     string   `json:"linkedin"`
	ContentTypes []string `json:"contentTypes"`
}

// Load reads config.json from rootDir and returns a Site.
func Load(rootDir string) (*Site, error) {
	path := filepath.Join(rootDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config.json: %w", err)
	}
	var site Site
	if err := json.Unmarshal(data, &site); err != nil {
		return nil, fmt.Errorf("parsing config.json: %w", err)
	}
	// Ensure BaseURL always has a trailing slash for correct <base> href resolution.
	site.BaseURL = strings.TrimRight(site.BaseURL, "/") + "/"
	// Default to "posts" if no content types are configured.
	if len(site.ContentTypes) == 0 {
		site.ContentTypes = []string{"posts"}
	}
	return &site, nil
}
