# Web Interface (EN)

Web mode runs gotorrentclient as a daemon with an embedded torrent engine that
downloads several torrents simultaneously and manages them from a browser.

## Start

```bash
./gotorrentclient --web --username admin --password secret --download-dir ./downloads
```

The server listens on `:8080` by default. Open `http://localhost:8080` and log in.

Omitting `--username` / `--password` starts the server without authentication — the
interface is then open to anyone on the network. A warning is printed to stderr.

## Environment variables

These take precedence over the matching flags:

| Variable | Purpose | Default |
|----------|---------|---------|
| `GTC_LISTEN` | Listen address | `:8080` |
| `GTC_USERNAME` | Login | — |
| `GTC_PASSWORD` | Password | — |

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

**Adding torrents**

- Paste a magnet link into the text field and click **Add**.
- Upload one or more `.torrent` files using the file picker.
- Both can be submitted in the same form. Errors are shown inline without a page reload.

**Torrent table columns**

| Column | Description |
|--------|-------------|
| Name | Torrent name |
| Progress | Completion percentage |
| Downloaded | Bytes completed / total size |
| Down | Current download speed |
| Up | Current upload speed |
| Peers | Number of active peers |
| Uploaded | Total bytes uploaded |
| Status | `Metadata` / `Downloading` / `Checking` / `Paused` / `Done` |

**Per-torrent actions**

| Button | Action |
|--------|--------|
| Pause | Stop data transfer without removing the torrent |
| Resume | Restart data transfer |
| Verify | Re-hash local data to check file integrity |
| Drop | Remove the torrent (downloaded files are kept) |

## Persistence

Torrent state is saved automatically to a `.gotorrentclient/` directory inside the
download directory. On next startup all previously added torrents are restored,
including their paused/active state.

## REST API

The web server exposes a JSON API used internally by the browser UI. All endpoints
require authentication when credentials are configured.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/torrents` | List all torrents |
| POST | `/api/add` | Add magnet or upload `.torrent` file(s) |
| POST | `/api/drop` | Remove a torrent (`id` form field) |
| POST | `/api/pause` | Pause a torrent (`id` form field) |
| POST | `/api/resume` | Resume a torrent (`id` form field) |
| POST | `/api/verify` | Verify torrent data (`id` form field) |

All responses return `{"torrents": [...]}` on success or `{"error": "...", "torrents": [...]}` on failure.
