package book

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/krewire/krewire/pkg/config"
)

// BookConfigFile represents the configuration file schema for standalone books
// (book.yaml or mdbind.yaml), as well as compatible subset of krewire.yaml.
type BookConfigFile struct {
	Title   string   `yaml:"title"`
	Author  string   `yaml:"author"`
	Input   string   `yaml:"input"`
	Output  string   `yaml:"output"`
	Base    string   `yaml:"base"`
	Mount   string   `yaml:"mount"`
	Theme   string   `yaml:"theme"` // auto, light, dark, off
	Nav     []Link   `yaml:"nav"`
	Footer  string   `yaml:"footer"`
	Version string   `yaml:"version"`
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
}

// LoadConfigFile loads and parses a YAML configuration file from path into a Config.
func LoadConfigFile(path string) (*Config, error) {
	var cf BookConfigFile
	if err := config.Load(path, &cf); err != nil {
		return nil, err
	}
	cfg := &Config{
		Title:      cf.Title,
		Author:     cf.Author,
		Input:      cf.Input,
		Output:     cf.Output,
		BasePath:   cf.Base,
		MountPath:  cf.Mount,
		NavLinks:   cf.Nav,
		FooterText: cf.Footer,
		Version:    cf.Version,
		Include:    cf.Include,
		Exclude:    cf.Exclude,
	}
	switch strings.ToLower(strings.TrimSpace(cf.Theme)) {
	case "auto", "default", "":
		cfg.Theme = DefaultTheme()
	case "light":
		th := DefaultTheme()
		th.Default = "light"
		cfg.Theme = th
	case "dark":
		th := DefaultTheme()
		th.Default = "dark"
		cfg.Theme = th
	case "off", "none", "false":
		cfg.Theme = nil
	}
	return cfg, nil
}

// FindConfigFile discovers a configuration file in dir, searching in order:
// book.yaml, book.yml, mdbind.yaml, mdbind.yml, krewire.yaml.
func FindConfigFile(dir string) string {
	candidates := []string{
		"book.yaml",
		"book.yml",
		"mdbind.yaml",
		"mdbind.yml",
		"krewire.yaml",
	}
	for _, c := range candidates {
		p := filepath.Join(dir, c)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}
