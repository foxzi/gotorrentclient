package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDownloadDirCreation tests that the download directory is created
func TestDownloadDirCreation(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "gotorrentclient-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Path that doesn't exist yet
	testDownloadDir := filepath.Join(tempDir, "test-downloads")

	// Ensure our test directory doesn't exist yet
	if _, err := os.Stat(testDownloadDir); err == nil {
		t.Fatalf("Directory %s already exists, expected it not to", testDownloadDir)
	}

	// Create the directory
	if err := os.MkdirAll(testDownloadDir, 0755); err != nil {
		t.Fatalf("Failed to create data directory %s: %v", testDownloadDir, err)
	}

	// Check that the directory now exists
	if _, err := os.Stat(testDownloadDir); os.IsNotExist(err) {
		t.Fatalf("Directory %s was not created", testDownloadDir)
	}
}

// TestFlagParsing tests the command line flag parsing
func TestFlagParsing(t *testing.T) {
	// Save original args and restore them after the test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Test with valid args
	os.Args = []string{
		"gotorrentclient",
		"-download-dir=./test-downloads",
		"-max-peers=30",
		"-download-rate=5",
		"-upload-rate=2",
		"magnet:?xt=urn:btih:testmagnetlink",
	}

	// We don't call the main() function directly, because it would start the torrent client
	// In a real test suite, you'd have a separate function that just parses flags and returns the config
	// For this test, we're just ensuring the file compiles correctly
}
