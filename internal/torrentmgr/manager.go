package torrentmgr

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
)

// seedCheckInterval is how often the seed ratio is re-evaluated.
const seedCheckInterval = 5 * time.Second

const (
	stateDirName = ".gotorrentclient"
	torrentsDir  = "torrents"
	magnetsFile  = "magnets.txt"
)

type speedSample struct {
	downloaded int64
	uploaded   int64
	at         time.Time
}

// Manager wraps a torrent.Client and exposes simple operations for the web UI.
type Manager struct {
	client  *torrent.Client
	cfg     EngineConfig
	mu      sync.Mutex
	paused  map[string]bool
	samples map[string]speedSample
}

// TorrentInfo is a snapshot of a single torrent's state.
type TorrentInfo struct {
	ID            string
	Name          string
	Completed     int64
	Length        int64
	Percent       float64
	Peers         int
	Uploaded      int64
	GotInfo       bool
	Done          bool
	Paused        bool
	Checking      bool
	Status        string
	DownloadSpeed int64
	UploadSpeed   int64
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
	mgr := &Manager{client: client, cfg: ec, paused: make(map[string]bool), samples: make(map[string]speedSample)}
	if err := mgr.ensureStateDirs(); err != nil {
		client.Close()
		return nil, err
	}
	mgr.restorePersisted()
	return mgr, nil
}

// AddMagnet adds a torrent from a magnet URI and starts downloading.
func (m *Manager) AddMagnet(uri string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, err := m.client.AddMagnet(uri)
	if err != nil {
		return err
	}
	if err := m.persistMagnet(uri); err != nil {
		return err
	}
	m.startWhenReady(t)
	return nil
}

// AddFile adds a torrent from a .torrent file path and starts downloading.
func (m *Manager) AddFile(path string) error {
	mi, err := metainfo.LoadFromFile(path)
	if err != nil {
		return err
	}
	return m.addMetaInfo(mi, true)
}

func (m *Manager) addMetaInfo(mi *metainfo.MetaInfo, persist bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, err := m.client.AddTorrent(mi)
	if err != nil {
		return err
	}
	if persist {
		if err := m.persistMetaInfo(mi); err != nil {
			return err
		}
	}
	m.startWhenReady(t)
	return nil
}

// startWhenReady waits for metadata then begins downloading all pieces.
func (m *Manager) startWhenReady(t *torrent.Torrent) {
	go func() {
		<-t.GotInfo()
		m.persistTorrentMetaInfo(t)
		t.VerifyData()
		if m.isPaused(t.InfoHash().HexString()) {
			t.DisallowDataDownload()
			t.DisallowDataUpload()
			t.CancelPieces(0, t.NumPieces())
			return
		}
		t.AllowDataDownload()
		if m.cfg.EnableSeeding {
			t.AllowDataUpload()
		} else {
			t.DisallowDataUpload()
		}
		t.DownloadAll()
		if m.cfg.EnableSeeding && m.cfg.SeedRatio > 0 {
			m.monitorSeedRatio(t)
		}
	}()
}

func (m *Manager) stateDir() string {
	return filepath.Join(m.cfg.DownloadDir, stateDirName)
}

func (m *Manager) torrentStateDir() string {
	return filepath.Join(m.stateDir(), torrentsDir)
}

func (m *Manager) magnetsPath() string {
	return filepath.Join(m.stateDir(), magnetsFile)
}

func (m *Manager) ensureStateDirs() error {
	if err := os.MkdirAll(m.torrentStateDir(), 0755); err != nil {
		return err
	}
	return os.MkdirAll(m.cfg.DownloadDir, 0755)
}

func (m *Manager) restorePersisted() {
	for _, path := range m.persistedTorrentFiles() {
		mi, err := metainfo.LoadFromFile(path)
		if err != nil {
			continue
		}
		_ = m.addMetaInfo(mi, false)
	}

	for _, uri := range m.persistedMagnets() {
		if id, ok := magnetInfoHashHex(uri); ok {
			if _, err := os.Stat(filepath.Join(m.torrentStateDir(), id+".torrent")); err == nil {
				continue
			}
		}
		m.mu.Lock()
		t, err := m.client.AddMagnet(uri)
		m.mu.Unlock()
		if err == nil {
			m.startWhenReady(t)
		}
	}
}

func (m *Manager) persistedTorrentFiles() []string {
	entries, err := os.ReadDir(m.torrentStateDir())
	if err != nil {
		return nil
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".torrent" {
			continue
		}
		paths = append(paths, filepath.Join(m.torrentStateDir(), entry.Name()))
	}
	return paths
}

func (m *Manager) persistedMagnets() []string {
	file, err := os.Open(m.magnetsPath())
	if err != nil {
		return nil
	}
	defer file.Close()

	var uris []string
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		uri := strings.TrimSpace(scanner.Text())
		if uri == "" || seen[uri] {
			continue
		}
		seen[uri] = true
		uris = append(uris, uri)
	}
	return uris
}

func (m *Manager) persistMetaInfo(mi *metainfo.MetaInfo) error {
	if err := m.ensureStateDirs(); err != nil {
		return err
	}
	path := filepath.Join(m.torrentStateDir(), mi.HashInfoBytes().HexString()+".torrent")
	tmpPath := path + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	writeErr := mi.Write(file)
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

func (m *Manager) persistTorrentMetaInfo(t *torrent.Torrent) {
	mi := t.Metainfo()
	if mi.InfoBytes == nil {
		return
	}
	_ = m.persistMetaInfo(&mi)
	m.removePersistedMagnet(t.InfoHash().HexString())
}

func (m *Manager) persistMagnet(uri string) error {
	if err := m.ensureStateDirs(); err != nil {
		return err
	}
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return nil
	}
	for _, existing := range m.persistedMagnets() {
		if existing == uri {
			return nil
		}
	}
	file, err := os.OpenFile(m.magnetsPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(uri + "\n")
	return err
}

func (m *Manager) removePersisted(id string) {
	_ = os.Remove(filepath.Join(m.torrentStateDir(), id+".torrent"))
	m.removePersistedMagnet(id)
}

func (m *Manager) removePersistedMagnet(id string) {
	magnets := m.persistedMagnets()
	if len(magnets) == 0 {
		return
	}

	keep := magnets[:0]
	for _, uri := range magnets {
		if magnetMatchesInfoHash(uri, id) {
			continue
		}
		keep = append(keep, uri)
	}
	if len(keep) == len(magnets) {
		return
	}

	tmpPath := m.magnetsPath() + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return
	}
	for _, uri := range keep {
		if _, err := file.WriteString(uri + "\n"); err != nil {
			file.Close()
			os.Remove(tmpPath)
			return
		}
	}
	if err := file.Close(); err != nil {
		os.Remove(tmpPath)
		return
	}
	_ = os.Rename(tmpPath, m.magnetsPath())
}

func magnetMatchesInfoHash(uri, id string) bool {
	if magnetID, ok := magnetInfoHashHex(uri); ok {
		return strings.EqualFold(magnetID, id)
	}
	return strings.Contains(strings.ToLower(uri), strings.ToLower(id))
}

func magnetInfoHashHex(uri string) (string, bool) {
	magnet, err := metainfo.ParseMagnetUri(uri)
	if err != nil {
		return "", false
	}
	return magnet.InfoHash.HexString(), true
}

func (m *Manager) isPaused(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.paused[id]
}

// calcSpeed returns download and upload speeds in bytes/s based on delta since last call.
func (m *Manager) calcSpeed(id string, downloaded, uploaded int64) (int64, int64) {
	now := time.Now()
	m.mu.Lock()
	prev, ok := m.samples[id]
	m.samples[id] = speedSample{downloaded: downloaded, uploaded: uploaded, at: now}
	m.mu.Unlock()
	if !ok {
		return 0, 0
	}
	elapsed := now.Sub(prev.at).Seconds()
	if elapsed < 0.1 {
		return 0, 0
	}
	dlSpeed := int64(float64(downloaded-prev.downloaded) / elapsed)
	ulSpeed := int64(float64(uploaded-prev.uploaded) / elapsed)
	if dlSpeed < 0 {
		dlSpeed = 0
	}
	if ulSpeed < 0 {
		ulSpeed = 0
	}
	return dlSpeed, ulSpeed
}

// monitorSeedRatio drops the torrent once the upload/size ratio reaches the
// configured SeedRatio. It exits early if the torrent is dropped elsewhere.
func (m *Manager) monitorSeedRatio(t *torrent.Torrent) {
	ticker := time.NewTicker(seedCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-t.Closed():
			return
		case <-ticker.C:
			length := t.Length()
			if length <= 0 || t.BytesCompleted() < length {
				continue
			}
			uploaded := t.Stats().BytesWrittenData
			if float64(uploaded.Int64())/float64(length) >= m.cfg.SeedRatio {
				t.Drop()
				return
			}
		}
	}
}

// List returns a snapshot of all torrents managed by the client.
func (m *Manager) List() []TorrentInfo {
	torrents := m.client.Torrents()
	infos := make([]TorrentInfo, 0, len(torrents))
	for _, t := range torrents {
		id := t.InfoHash().HexString()
		info := TorrentInfo{
			ID:      id,
			Name:    t.Name(),
			GotInfo: t.Info() != nil,
			Paused:  m.isPaused(id),
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
			info.DownloadSpeed, info.UploadSpeed = m.calcSpeed(id, info.Completed, info.Uploaded)
			for _, run := range t.PieceStateRuns() {
				if run.Checking || run.Hashing || run.QueuedForHash {
					info.Checking = true
					break
				}
			}
		}
		info.Status = torrentStatus(info)
		infos = append(infos, info)
	}
	return infos
}

func torrentStatus(info TorrentInfo) string {
	switch {
	case !info.GotInfo:
		return "Metadata"
	case info.Paused:
		return "Paused"
	case info.Checking:
		return "Checking"
	case info.Done:
		return "Done"
	default:
		return "Downloading"
	}
}

func (m *Manager) find(id string) *torrent.Torrent {
	for _, t := range m.client.Torrents() {
		if t.InfoHash().HexString() == id {
			return t
		}
	}
	return nil
}

// Pause stops data transfer for a torrent without removing it.
func (m *Manager) Pause(id string) bool {
	t := m.find(id)
	if t == nil {
		return false
	}

	m.mu.Lock()
	m.paused[id] = true
	m.mu.Unlock()

	t.DisallowDataDownload()
	t.DisallowDataUpload()
	if t.Info() != nil {
		t.CancelPieces(0, t.NumPieces())
	}
	return true
}

// Resume allows data transfer and requests all pieces again.
func (m *Manager) Resume(id string) bool {
	t := m.find(id)
	if t == nil {
		return false
	}

	m.mu.Lock()
	delete(m.paused, id)
	m.mu.Unlock()

	t.AllowDataDownload()
	if m.cfg.EnableSeeding {
		t.AllowDataUpload()
	} else {
		t.DisallowDataUpload()
	}
	if t.Info() != nil {
		t.DownloadAll()
	}
	return true
}

// Verify re-hashes local data for a torrent to check file consistency.
func (m *Manager) Verify(id string) bool {
	t := m.find(id)
	if t == nil || t.Info() == nil {
		return false
	}
	t.VerifyData()
	return true
}

// Drop stops and removes a torrent by its infohash hex id.
func (m *Manager) Drop(id string) bool {
	t := m.find(id)
	if t == nil {
		return false
	}
	t.Drop()
	m.mu.Lock()
	delete(m.paused, id)
	m.mu.Unlock()
	m.removePersisted(id)
	return true
}

// Close shuts down the underlying client.
func (m *Manager) Close() {
	m.client.Close()
}
