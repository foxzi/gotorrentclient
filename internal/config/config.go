package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"gotorrentclient/internal/torrentmgr"
)

// Config holds all runtime configuration for the web daemon.
type Config struct {
	Listen   string
	Username string
	Password string
	Engine   torrentmgr.EngineConfig
}

// fileEngineConfig is the YAML representation of engine settings.
// All fields are pointers so yaml.v3 leaves them nil when the key is absent,
// letting us distinguish "not present" from an explicit zero/false value.
type fileEngineConfig struct {
	DownloadDir      *string  `yaml:"download_dir"`
	MaxPeers         *int     `yaml:"max_peers"`
	DownloadRateMbps *float64 `yaml:"download_rate_mbps"`
	UploadRateMbps   *float64 `yaml:"upload_rate_mbps"`
	EnableSeeding    *bool    `yaml:"enable_seeding"`
	SeedRatio        *float64 `yaml:"seed_ratio"`
	ProxyURL         *string  `yaml:"proxy_url"`
}

// FileConfig is the YAML file schema for all settings.
// Pointer fields are nil when the key is absent from the file.
type FileConfig struct {
	Listen   *string          `yaml:"listen"`
	Username *string          `yaml:"username"`
	Password *string          `yaml:"password"`
	Engine   fileEngineConfig `yaml:"engine"`
}

// Params holds CLI flag values and which flags were explicitly set on the command line.
type Params struct {
	Listen        string
	Username      string
	Password      string
	DownloadDir   string
	MaxPeers      int
	DownloadRate  float64
	UploadRate    float64
	EnableSeeding bool
	SeedRatio     float64
	ProxyURL      string
	// SetFlags maps flag name to true if the flag was explicitly provided.
	SetFlags map[string]bool
}

// LoadFile reads and parses a YAML config file.
// Returns an error if the file does not exist or cannot be parsed.
func LoadFile(path string) (FileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FileConfig{}, fmt.Errorf("reading config file %s: %w", path, err)
	}
	var fc FileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return FileConfig{}, fmt.Errorf("parsing config file %s: %w", path, err)
	}
	return fc, nil
}

// Build creates the final Config applying precedence (lowest to highest):
//
//	built-in defaults -> YAML file -> environment variables -> explicit CLI flags
//
// defaults must be pre-populated with hardcoded defaults (e.g. from flag package defaults).
// fc is the parsed YAML file (zero-value FileConfig if no file was loaded).
// p contains flag values and the set of flags that were explicitly provided.
func Build(defaults Config, fc FileConfig, p Params) Config {
	cfg := defaults

	// Layer 2: YAML file — apply only keys that were present in the file (non-nil pointers).
	if fc.Listen != nil {
		cfg.Listen = *fc.Listen
	}
	if fc.Username != nil {
		cfg.Username = *fc.Username
	}
	if fc.Password != nil {
		cfg.Password = *fc.Password
	}
	if fc.Engine.DownloadDir != nil {
		cfg.Engine.DownloadDir = *fc.Engine.DownloadDir
	}
	if fc.Engine.MaxPeers != nil {
		cfg.Engine.MaxPeers = *fc.Engine.MaxPeers
	}
	if fc.Engine.DownloadRateMbps != nil {
		cfg.Engine.DownloadRateMbps = *fc.Engine.DownloadRateMbps
	}
	if fc.Engine.UploadRateMbps != nil {
		cfg.Engine.UploadRateMbps = *fc.Engine.UploadRateMbps
	}
	if fc.Engine.EnableSeeding != nil {
		cfg.Engine.EnableSeeding = *fc.Engine.EnableSeeding
	}
	if fc.Engine.SeedRatio != nil {
		cfg.Engine.SeedRatio = *fc.Engine.SeedRatio
	}
	if fc.Engine.ProxyURL != nil {
		cfg.Engine.ProxyURL = *fc.Engine.ProxyURL
	}

	// Layer 3: environment variables.
	if v := os.Getenv("GTC_LISTEN"); v != "" {
		cfg.Listen = v
	}
	if v := os.Getenv("GTC_USERNAME"); v != "" {
		cfg.Username = v
	}
	if v := os.Getenv("GTC_PASSWORD"); v != "" {
		cfg.Password = v
	}

	// Layer 4: explicitly-set CLI flags (highest priority).
	if p.SetFlags["listen"] {
		cfg.Listen = p.Listen
	}
	if p.SetFlags["username"] {
		cfg.Username = p.Username
	}
	if p.SetFlags["password"] {
		cfg.Password = p.Password
	}
	if p.SetFlags["download-dir"] {
		cfg.Engine.DownloadDir = p.DownloadDir
	}
	if p.SetFlags["max-peers"] {
		cfg.Engine.MaxPeers = p.MaxPeers
	}
	if p.SetFlags["download-rate"] {
		cfg.Engine.DownloadRateMbps = p.DownloadRate
	}
	if p.SetFlags["upload-rate"] {
		cfg.Engine.UploadRateMbps = p.UploadRate
	}
	if p.SetFlags["enable-seeding"] {
		cfg.Engine.EnableSeeding = p.EnableSeeding
	}
	if p.SetFlags["seed-ratio"] {
		cfg.Engine.SeedRatio = p.SeedRatio
	}
	if p.SetFlags["proxy"] {
		cfg.Engine.ProxyURL = p.ProxyURL
	}

	// Ensure listen has a default if nothing provided it.
	if cfg.Listen == "" {
		cfg.Listen = ":8080"
	}

	return cfg
}
