package config

import (
	"os"

	"gotorrentclient/internal/torrentmgr"
)

// Config holds all runtime configuration for the web daemon.
type Config struct {
	Listen   string
	Username string
	Password string
	Engine   torrentmgr.EngineConfig
}

// Load builds a Config, applying env overrides over the provided values.
// GTC_LISTEN overrides listen (default ":8080").
// GTC_USERNAME overrides username.
// GTC_PASSWORD overrides password.
func Load(engine torrentmgr.EngineConfig, listen, username, password string) Config {
	if v := os.Getenv("GTC_LISTEN"); v != "" {
		listen = v
	}
	if listen == "" {
		listen = ":8080"
	}
	if v := os.Getenv("GTC_USERNAME"); v != "" {
		username = v
	}
	if v := os.Getenv("GTC_PASSWORD"); v != "" {
		password = v
	}
	return Config{
		Listen:   listen,
		Username: username,
		Password: password,
		Engine:   engine,
	}
}
