package torrentmgr

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/anacrolix/torrent"
	"golang.org/x/net/proxy"
	"golang.org/x/time/rate"
)

// EngineConfig holds settings for the underlying torrent client.
// It is shared between the CLI and the web daemon.
type EngineConfig struct {
	DownloadDir      string
	MaxPeers         int
	DownloadRateMbps float64
	UploadRateMbps   float64
	EnableSeeding    bool
	SeedRatio        float64
	ProxyURL         string
}

// BuildClientConfig converts an EngineConfig into a *torrent.ClientConfig.
func BuildClientConfig(ec EngineConfig) (*torrent.ClientConfig, error) {
	cfg := torrent.NewDefaultClientConfig()
	cfg.NoUpload = !ec.EnableSeeding
	cfg.DataDir = ec.DownloadDir
	cfg.EstablishedConnsPerTorrent = ec.MaxPeers

	if ec.DownloadRateMbps > 0 {
		limit := rate.Limit(ec.DownloadRateMbps * 1024 * 1024 / 8)
		cfg.DownloadRateLimiter = rate.NewLimiter(limit, 512*1024)
	}
	if ec.UploadRateMbps > 0 {
		limit := rate.Limit(ec.UploadRateMbps * 1024 * 1024 / 8)
		cfg.UploadRateLimiter = rate.NewLimiter(limit, 512*1024)
	}

	if ec.ProxyURL != "" {
		if err := applyProxy(cfg, ec.ProxyURL); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func applyProxy(cfg *torrent.ClientConfig, proxyURL string) error {
	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return fmt.Errorf("parsing proxy URL %s: %w", proxyURL, err)
	}

	switch parsed.Scheme {
	case "http", "https":
		cfg.HTTPProxy = http.ProxyURL(parsed)
	case "socks5":
		socksDialer, err := proxy.FromURL(parsed, proxy.Direct)
		if err != nil {
			return fmt.Errorf("creating SOCKS5 dialer from %s: %w", proxyURL, err)
		}
		cfg.HTTPDialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return socksDialer.Dial(network, addr)
		}
	default:
		return fmt.Errorf("unsupported proxy scheme: %s (only http, https, socks5)", parsed.Scheme)
	}
	return nil
}
