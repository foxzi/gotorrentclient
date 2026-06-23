package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anacrolix/torrent"

	"gotorrentclient/internal/config"
	"gotorrentclient/internal/torrentmgr"
	"gotorrentclient/internal/web"
	"gotorrentclient/utils"
)

// Version information - will be set during build
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "Show version information and exit")
	configFile := flag.String("config", "", "Path to YAML config file (or GTC_CONFIG env var)")
	maxPeers := flag.Int("max-peers", 50, "Maximum number of peers per torrent")
	downloadDir := flag.String("download-dir", "./downloads", "Directory to download torrents to")
	downloadRateMbps := flag.Float64("download-rate", 0, "Maximum download rate in Mbps (0 = unlimited)")
	uploadRateMbps := flag.Float64("upload-rate", 0, "Maximum upload rate in Mbps (0 = unlimited)")
	seedRatio := flag.Float64("seed-ratio", 0, "Seed ratio (0 = unlimited)")
	enableSeeding := flag.Bool("enable-seeding", false, "Enable seeding after download completes")
	proxyURL := flag.String("proxy", "", "Proxy URL (e.g. socks5://user:pass@host:port)")
	webMode := flag.Bool("web", false, "Start web UI daemon instead of CLI download")
	listen := flag.String("listen", "", "Web listen address (default :8080, or GTC_LISTEN)")
	username := flag.String("username", "", "Web UI username (or GTC_USERNAME)")
	password := flag.String("password", "", "Web UI password (or GTC_PASSWORD)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("gotorrentclient %s\n", version)
		os.Exit(0)
	}

	// Collect flags that were explicitly provided on the command line.
	setFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	// Determine config file path: flag takes precedence over env var.
	cfgPath := *configFile
	if cfgPath == "" {
		cfgPath = os.Getenv("GTC_CONFIG")
	}

	// Load YAML config file if a path was provided.
	var fc config.FileConfig
	if cfgPath != "" {
		var err error
		fc, err = config.LoadFile(cfgPath)
		if err != nil {
			log.Fatalf("Config error: %v", err)
		}
	}

	// Defaults come from the flag package defaults (already reflected in pointer values).
	defaults := config.Config{
		Listen:   *listen,
		Username: *username,
		Password: *password,
		Engine: torrentmgr.EngineConfig{
			DownloadDir:      *downloadDir,
			MaxPeers:         *maxPeers,
			DownloadRateMbps: *downloadRateMbps,
			UploadRateMbps:   *uploadRateMbps,
			EnableSeeding:    *enableSeeding,
			SeedRatio:        *seedRatio,
			ProxyURL:         *proxyURL,
		},
	}

	params := config.Params{
		Listen:        *listen,
		Username:      *username,
		Password:      *password,
		DownloadDir:   *downloadDir,
		MaxPeers:      *maxPeers,
		DownloadRate:  *downloadRateMbps,
		UploadRate:    *uploadRateMbps,
		EnableSeeding: *enableSeeding,
		SeedRatio:     *seedRatio,
		ProxyURL:      *proxyURL,
		SetFlags:      setFlags,
	}

	cfg := config.Build(defaults, fc, params)

	if *webMode {
		runWeb(cfg)
		return
	}

	runCLI(cfg.Engine)
}

func runWeb(cfg config.Config) {
	if cfg.Username == "" || cfg.Password == "" {
		log.Println("WARNING: running without authentication -- web interface is open to anyone")
	}

	if err := os.MkdirAll(cfg.Engine.DownloadDir, 0755); err != nil {
		log.Fatalf("Failed to create download directory: %v", err)
	}

	mgr, err := torrentmgr.New(cfg.Engine)
	if err != nil {
		log.Fatalf("Failed to create torrent manager: %v", err)
	}
	defer mgr.Close()

	srv, err := web.NewServer(cfg, mgr)
	if err != nil {
		log.Fatalf("Failed to create web server: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Printf("Received signal %s, shutting down...", sig)
		mgr.Close()
		os.Exit(0)
	}()

	log.Printf("Web UI listening on %s", cfg.Listen)
	if err := srv.Start(); err != nil {
		log.Fatalf("Web server error: %v", err)
	}
}

func runCLI(engineCfg torrentmgr.EngineConfig) {
	if len(flag.Args()) < 1 {
		log.Fatalf("Usage: %s [options] <magnet link or torrent file>", os.Args[0])
	}
	uri := flag.Arg(0)

	if engineCfg.EnableSeeding {
		if engineCfg.SeedRatio > 0 {
			log.Printf("Seeding enabled with ratio %.2f", engineCfg.SeedRatio)
		} else {
			log.Printf("Seeding enabled with unlimited ratio")
		}
	} else {
		log.Println("Seeding disabled")
	}

	cfg, err := torrentmgr.BuildClientConfig(engineCfg)
	if err != nil {
		log.Fatalf("Error building client config: %v", err)
	}

	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory %s: %v", cfg.DataDir, err)
	}

	client, err := torrent.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Printf("Received signal %s, shutting down...", sig)
		client.Close()
	}()

	var t *torrent.Torrent
	var errAdd error
	if _, statErr := os.Stat(uri); statErr == nil {
		t, errAdd = client.AddTorrentFromFile(uri)
	} else {
		t, errAdd = client.AddMagnet(uri)
	}
	if errAdd != nil {
		log.Fatalf("Error adding torrent: %v", errAdd)
	}

	log.Println("Waiting for torrent info...")
	<-t.GotInfo()
	log.Printf("Downloading %s to %s/%s...", t.Name(), cfg.DataDir, t.Name())

	t.DownloadAll()

	initialBytesCompleted := int64(0)
	var bytesUploaded int64
	seedingStarted := false

	go func() {
		for {
			stats := t.Stats()
			bytesCompleted := t.BytesCompleted()
			bytesUploaded = stats.BytesWrittenData.Int64()

			if bytesCompleted == t.Length() && t.Length() > 0 {
				if !seedingStarted && engineCfg.EnableSeeding {
					seedingStarted = true
					initialBytesCompleted = bytesCompleted
					log.Printf("Download complete. Starting to seed: %s", t.Name())
				}

				if seedingStarted && engineCfg.SeedRatio > 0 {
					currentRatio := float64(bytesUploaded) / float64(initialBytesCompleted)
					log.Printf("Seeding: %s, Ratio: %.2f/%.2f, Uploaded: %s",
						t.Name(), currentRatio, engineCfg.SeedRatio, utils.FormatBytes(bytesUploaded))
					if currentRatio >= engineCfg.SeedRatio {
						log.Printf("Reached target seed ratio of %.2f. Stopping seeding.", engineCfg.SeedRatio)
						go func() { sigChan <- syscall.SIGTERM }()
						return
					}
				} else if seedingStarted {
					log.Printf("Seeding: %s, Uploaded: %s", t.Name(), utils.FormatBytes(bytesUploaded))
				}

				if !engineCfg.EnableSeeding {
					return
				}
				time.Sleep(5 * time.Second)
				continue
			}

			if t.Info() == nil {
				time.Sleep(1 * time.Second)
				continue
			}

			percent := float64(bytesCompleted) / float64(t.Length()) * 100
			if t.Length() == 0 && bytesCompleted > 0 {
				log.Printf("Downloaded %d bytes (metadata not fully resolved yet)", bytesCompleted)
			} else if t.Length() > 0 {
				log.Printf("%.2f%% complete. Downloaded: %s / %s. Peers: %d, Uploaded: %s",
					percent,
					utils.FormatBytes(bytesCompleted),
					utils.FormatBytes(t.Length()),
					stats.ActivePeers,
					utils.FormatBytes(bytesUploaded),
				)
			}
			time.Sleep(2 * time.Second)
		}
	}()

	if ok := client.WaitAll(); !ok {
		log.Println("Shutdown initiated or not all torrents downloaded successfully.")
	} else {
		log.Printf("Torrent %s downloaded successfully to %s/%s", t.Name(), cfg.DataDir, t.Name())
	}
	log.Println("Download process finished.")
}
