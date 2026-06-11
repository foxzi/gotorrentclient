package torrentmgr

import (
	"sync"

	"github.com/anacrolix/torrent"
)

// Manager wraps a torrent.Client and exposes simple operations for the web UI.
type Manager struct {
	client *torrent.Client
	cfg    EngineConfig
	mu     sync.Mutex
}

// TorrentInfo is a snapshot of a single torrent's state.
type TorrentInfo struct {
	ID        string
	Name      string
	Completed int64
	Length    int64
	Percent   float64
	Peers     int
	Uploaded  int64
	GotInfo   bool
	Done      bool
}

// New creates a Manager with an embedded torrent.Client.
func New(ec EngineConfig) (*Manager, error) {
	cfg, err := BuildClientConfig(ec)
	if err != nil {
		return nil, err
	}
	client, err := torrent.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &Manager{client: client, cfg: ec}, nil
}

// AddMagnet adds a torrent from a magnet URI and starts downloading.
func (m *Manager) AddMagnet(uri string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, err := m.client.AddMagnet(uri)
	if err != nil {
		return err
	}
	m.startWhenReady(t)
	return nil
}

// AddFile adds a torrent from a .torrent file path and starts downloading.
func (m *Manager) AddFile(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, err := m.client.AddTorrentFromFile(path)
	if err != nil {
		return err
	}
	m.startWhenReady(t)
	return nil
}

// startWhenReady waits for metadata then begins downloading all pieces.
func (m *Manager) startWhenReady(t *torrent.Torrent) {
	go func() {
		<-t.GotInfo()
		t.DownloadAll()
	}()
}

// List returns a snapshot of all torrents managed by the client.
func (m *Manager) List() []TorrentInfo {
	torrents := m.client.Torrents()
	infos := make([]TorrentInfo, 0, len(torrents))
	for _, t := range torrents {
		info := TorrentInfo{
			ID:      t.InfoHash().HexString(),
			Name:    t.Name(),
			GotInfo: t.Info() != nil,
		}
		if info.GotInfo {
			info.Length = t.Length()
			info.Completed = t.BytesCompleted()
			if info.Length > 0 {
				info.Percent = float64(info.Completed) / float64(info.Length) * 100
				info.Done = info.Completed >= info.Length
			}
			stats := t.Stats()
			info.Peers = stats.ActivePeers
			info.Uploaded = stats.BytesWrittenData.Int64()
		}
		infos = append(infos, info)
	}
	return infos
}

// Drop stops and removes a torrent by its infohash hex id.
func (m *Manager) Drop(id string) bool {
	for _, t := range m.client.Torrents() {
		if t.InfoHash().HexString() == id {
			t.Drop()
			return true
		}
	}
	return false
}

// Close shuts down the underlying client.
func (m *Manager) Close() {
	m.client.Close()
}
