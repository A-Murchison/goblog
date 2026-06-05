package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Site holds site-wide metadata loaded from config.json.
type Site struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	BaseURL     string `json:"baseURL"`
	Author      string `json:"author"`
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
	return &site, nil
}
