package main

import (
	"context" // Added for HTTPDialContext
	"flag"
	"fmt"
	"log"
	"net"      // Added for proxy dialing
	"net/http" // Added for HTTP client proxy
	"net/url"  // Added for parsing proxy URL
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anacrolix/torrent"
	"golang.org/x/net/proxy"
	"golang.org/x/time/rate"
)

func main() {
	// Define command-line flags
	maxPeers := flag.Int("max-peers", 50, "Maximum number of peers to connect to per torrent")
	downloadDir := flag.String("download-dir", "./downloads", "Directory to download torrents to")
	downloadRateMbps := flag.Float64("download-rate", 0, "Maximum download rate in Mbps (0 for unlimited)")
	uploadRateMbps := flag.Float64("upload-rate", 0, "Maximum upload rate in Mbps (0 for unlimited)")
	seedRatio := flag.Float64("seed-ratio", 0, "Seed ratio (e.g., 1.0 means seed until you've uploaded as much as you've downloaded, 0 for unlimited)")
	enableSeeding := flag.Bool("enable-seeding", false, "Enable seeding after download completes")
	proxyURL := flag.String("proxy", "", "Proxy URL (e.g., socks5://user:pass@host:port or http://host:port)") // New flag
	flag.Parse()

	if len(flag.Args()) < 1 {
		log.Fatalf("Usage: %s [options] <magnet link or torrent file>", os.Args[0])
	}
	uri := flag.Arg(0)

	cfg := torrent.NewDefaultClientConfig()
	// Only disable upload if seeding is explicitly disabled
	cfg.NoUpload = !*enableSeeding
	cfg.DataDir = *downloadDir // Use the flag value
	// Set max established connections per torrent
	cfg.EstablishedConnsPerTorrent = *maxPeers

	// Set download rate limit if specified
	if *downloadRateMbps > 0 {
		limit := rate.Limit(*downloadRateMbps * 1024 * 1024 / 8) // Convert Mbps to bytes/sec
		burst := 512 * 1024
		cfg.DownloadRateLimiter = rate.NewLimiter(limit, burst)
		log.Printf("Download rate limited to %.2f Mbps", *downloadRateMbps)
	}

	// Set upload rate limit if specified
	if *uploadRateMbps > 0 {
		limit := rate.Limit(*uploadRateMbps * 1024 * 1024 / 8) // Convert Mbps to bytes/sec
		burst := 512 * 1024
		cfg.UploadRateLimiter = rate.NewLimiter(limit, burst)
		log.Printf("Upload rate limited to %.2f Mbps", *uploadRateMbps)
	}

	// Log seeding configuration
	if *enableSeeding {
		if *seedRatio > 0 {
			log.Printf("Seeding enabled with ratio %.2f", *seedRatio)
		} else {
			log.Printf("Seeding enabled with unlimited ratio")
		}
	} else {
		log.Println("Seeding disabled")
	}

	// Configure proxy if specified
	if *proxyURL != "" {
		parsedProxyURL, err := url.Parse(*proxyURL)
		if err != nil {
			log.Fatalf("Error parsing proxy URL %s: %v", *proxyURL, err)
		}

		switch parsedProxyURL.Scheme {
		case "http", "https":
			cfg.HTTPProxy = http.ProxyURL(parsedProxyURL) // This function returns the correct type for ClientConfig.HTTPProxy
			log.Printf("Using HTTP(S) proxy for HTTP trackers: %s", parsedProxyURL.Host)
		case "socks5":
			// Create a SOCKS5 dialer for use with HTTP transport
			socksDialer, err := proxy.FromURL(parsedProxyURL, proxy.Direct)
			if err != nil {
				log.Fatalf("Error creating SOCKS5 dialer from URL %s: %v", *proxyURL, err)
			}

			// Configure SOCKS5 proxy for HTTP requests (trackers, etc.)
			cfg.HTTPDialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return socksDialer.Dial(network, addr)
			}

			log.Printf("Using SOCKS5 proxy for HTTP trackers and peer connections: %s. UDP trackers will not use this proxy.", parsedProxyURL.Host)
		default:
			log.Fatalf("Unsupported proxy scheme: %s. Only http, https, and socks5 are supported.", parsedProxyURL.Scheme)
		}
	}

	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory %s: %v", cfg.DataDir, err)
	}

	client, err := torrent.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	// Graceful shutdown on interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Printf("Received signal %s, shutting down...", sig)
		client.Close() // This will allow torrents to save resume data
		// os.Exit(0) // client.Close() will unblock WaitAll or other blocking calls
	}()
	// Defer client.Close() to ensure it's called on normal exit or panic
	// but the signal handler will call it first on interrupt.
	defer client.Close()

	var t *torrent.Torrent
	var errAdd error                                // Use a different variable name to avoid conflict with outer scope err
	if _, statErr := os.Stat(uri); statErr == nil { // Renamed err to statErr to avoid confusion
		// Assume it's a torrent file
		t, errAdd = client.AddTorrentFromFile(uri)
	} else {
		// Assume it's a magnet link
		t, errAdd = client.AddMagnet(uri)
	}

	// If a proxy is configured, and it's SOCKS5, we might want to explicitly pass proxy info
	// to tracker announce calls if the library supports it and cfg.Dial isn't enough for all cases.
	// However, anacrolix/torrent is expected to use the cfg.Dial for its HTTP tracker requests if no HTTPProxy is set.

	if errAdd != nil { // Check the new error variable
		log.Fatalf("Error adding torrent: %v", errAdd)
	}

	log.Println("Waiting for torrent info...")
	<-t.GotInfo()
	log.Printf("Downloading %s to %s/%s...", t.Name(), cfg.DataDir, t.Name())

	// Start the download
	t.DownloadAll()

	// Print progress and monitor seeding
	initialBytesCompleted := int64(0)
	var bytesUploaded int64
	seedingStarted := false

	go func() {
		for {
			stats := t.Stats()
			bytesCompleted := t.BytesCompleted()
			bytesUploaded = stats.BytesWrittenData.Int64()

			// If download is complete and we're starting to seed
			if bytesCompleted == t.Length() && t.Length() > 0 {
				if !seedingStarted && *enableSeeding {
					seedingStarted = true
					initialBytesCompleted = bytesCompleted
					log.Printf("Download complete. Starting to seed: %s", t.Name())
				}

				// If seeding is enabled and ratio is set, check if we've reached the desired ratio
				if seedingStarted && *seedRatio > 0 {
					// Calculate current ratio
					currentRatio := float64(bytesUploaded) / float64(initialBytesCompleted)

					log.Printf("Seeding: %s, Ratio: %.2f/%.2f, Uploaded: %s",
						t.Name(),
						currentRatio,
						*seedRatio,
						formatBytes(bytesUploaded))

					// If we've reached the desired ratio, stop seeding
					if currentRatio >= *seedRatio {
						log.Printf("Reached target seed ratio of %.2f. Stopping seeding.", *seedRatio)
						// Since we're in a goroutine, we need to request shutdown
						go func() {
							sigChan <- syscall.SIGTERM
						}()
						return
					}
				} else if seedingStarted {
					// Just show seeding status without ratio check
					log.Printf("Seeding: %s, Uploaded: %s",
						t.Name(),
						formatBytes(bytesUploaded))
				}

				if !*enableSeeding {
					// If seeding is disabled, exit the loop once download is complete
					return
				}

				time.Sleep(5 * time.Second)
				continue
			}

			// Display download progress
			if t.Info() == nil { // Check if t.Info() is nil
				time.Sleep(1 * time.Second)
				continue
			}

			percent := float64(bytesCompleted) / float64(t.Length()) * 100
			if t.Length() == 0 && bytesCompleted > 0 { // Handle cases where length might initially be 0 for magnets
				log.Printf("Downloaded %d bytes (metadata not fully resolved yet)", bytesCompleted)
			} else if t.Length() > 0 {
				log.Printf("%.2f%% complete. Downloaded: %s / %s. Peers: %d, Uploaded: %s",
					percent,
					formatBytes(bytesCompleted),
					formatBytes(t.Length()),
					stats.ActivePeers,
					formatBytes(bytesUploaded),
				)
			}
			time.Sleep(2 * time.Second)
		}
	}()

	// Wait for download to complete or for client to be closed by signal
	if ok := client.WaitAll(); !ok {
		// This can happen if client.Close() is called by the signal handler
		// before all torrents complete.
		log.Println("Shutdown initiated or not all torrents downloaded successfully.")
	} else {
		log.Printf("Torrent %s downloaded successfully to %s/%s", t.Name(), cfg.DataDir, t.Name())
	}
	// If shutdown was initiated by a signal, the goroutine above will handle exit.
	// If downloads completed normally, we reach here.
	log.Println("Download process finished.")
}

// formatBytes converts bytes to a human-readable string
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
