# Web Interface (EN)

Web mode runs gotorrentclient as a daemon with an embedded torrent engine that
downloads several torrents simultaneously and manages them from a browser.

## Start

```bash
./gotorrentclient --web --username admin --password secret --download-dir ./downloads
```

The server listens on `:8080` by default. Open `http://localhost:8080` and log in.

## Environment variables

These take precedence over the matching flags:

| Variable | Purpose | Default |
|----------|---------|---------|
| `GTC_LISTEN` | Listen address | `:8080` |
| `GTC_USERNAME` | Login | — |
| `GTC_PASSWORD` | Password | — |

The daemon refuses to start without a username and password.

## Engine flags

Set at daemon start and applied to all torrents:

| Flag | Purpose |
|------|---------|
| `--download-dir` | Download directory |
| `--max-peers` | Max peers per torrent |
| `--download-rate` | Download limit, Mbps (0 = unlimited) |
| `--upload-rate` | Upload limit, Mbps (0 = unlimited) |
| `--enable-seeding` | Seed after completion |
| `--seed-ratio` | Target seed ratio |
| `--proxy` | Proxy (http, https, socks5) |

## Authentication

A single configured user. After login a cookie session (HttpOnly) is issued.
Every page except login and static assets requires authentication. The logout
button ends the session.

## Interface

- Add a torrent via a magnet link or by uploading a `.torrent` file.
- The table lists all active downloads: name, progress, size, peer count, uploaded.
- Multiple torrents download in parallel.
- A drop button removes the selected torrent.
- The page auto-refreshes every 2 seconds.
