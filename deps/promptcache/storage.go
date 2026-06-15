package promptcache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// diskEntry is a single cache entry persisted to disk.
type diskEntry struct {
	Prompt    string    `json:"prompt"`
	Response  string    `json:"response"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used"`
	TTL       string    `json:"ttl"` // duration string, e.g. "24h"
	Hash      string    `json:"hash"`
	Embedding []byte    `json:"embedding,omitempty"` // serialized float32 slice
}

// diskStore is the on-disk format for the entire cache.
type diskStore struct {
	Entries []diskEntry `json:"entries"`
}

// store manages the JSON-file backed persistent cache storage.
// Thread-safe with sync.RWMutex.
type store struct {
	mu        sync.RWMutex
	dir       string
	filePath  string
	entries   []diskEntry
	keyIndex  map[string]int // hash → index in entries
	maxSize   int
	dirty     bool
}

func newStore(dir string, maxSize int) *store {
	s := &store{
		dir:      dir,
		filePath: filepath.Join(dir, "cache.json"),
		keyIndex: make(map[string]int),
		maxSize:  maxSize,
	}
	s.load()
	return s
}

func (s *store) load() {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		s.entries = make([]diskEntry, 0)
		return
	}

	var ds diskStore
	if err := json.Unmarshal(data, &ds); err != nil {
		s.entries = make([]diskEntry, 0)
		return
	}

	s.entries = ds.Entries
	if s.entries == nil {
		s.entries = make([]diskEntry, 0)
	}

	// Rebuild index, filter expired
	now := time.Now()
	valid := make([]diskEntry, 0, len(s.entries))
	for _, e := range s.entries {
		ttl, err := time.ParseDuration(e.TTL)
		if err == nil && ttl > 0 && now.After(e.CreatedAt.Add(ttl)) {
			continue // expired, skip
		}
		s.keyIndex[e.Hash] = len(valid)
		valid = append(valid, e)
	}
	s.entries = valid
}

func (s *store) save() error {
	s.mu.RLock()
	entries := s.entries
	s.mu.RUnlock()

	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}

	ds := diskStore{Entries: entries}
	data, err := json.MarshalIndent(ds, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0o644)
}

// get retrieves an entry by hash. Returns nil if not found or expired.
func (s *store) get(hash string) *diskEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idx, ok := s.keyIndex[hash]
	if !ok || idx >= len(s.entries) {
		return nil
	}

	e := s.entries[idx]

	// Check TTL expiry
	ttl, err := time.ParseDuration(e.TTL)
	if err == nil && ttl > 0 && time.Now().After(e.CreatedAt.Add(ttl)) {
		return nil
	}

	return &e
}

// set adds or updates an entry. If maxSize is exceeded, evicts LRU entries.
func (s *store) set(hash string, entry diskEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update existing
	if idx, ok := s.keyIndex[hash]; ok && idx < len(s.entries) {
		s.entries[idx] = entry
		s.dirty = true
		return
	}

	// Evict if at capacity
	for len(s.entries) >= s.maxSize {
		// Find LRU (oldest LastUsed)
		oldestIdx := 0
		for i := 1; i < len(s.entries); i++ {
			if s.entries[i].LastUsed.Before(s.entries[oldestIdx].LastUsed) {
				oldestIdx = i
			}
		}
		delete(s.keyIndex, s.entries[oldestIdx].Hash)
		s.entries = append(s.entries[:oldestIdx], s.entries[oldestIdx+1:]...)
		// Rebuild keyIndex after removal
		for i := range s.entries {
			s.keyIndex[s.entries[i].Hash] = i
		}
	}

	s.keyIndex[hash] = len(s.entries)
	s.entries = append(s.entries, entry)
	s.dirty = true
}

// touch updates the LastUsed timestamp for an entry (for LRU tracking).
func (s *store) touch(hash string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if idx, ok := s.keyIndex[hash]; ok && idx < len(s.entries) {
		s.entries[idx].LastUsed = time.Now()
		s.dirty = true
	}
}

// count returns the number of entries.
func (s *store) count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

// all returns a copy of all entries (for stats / inspection).
func (s *store) all() []diskEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]diskEntry, len(s.entries))
	copy(result, s.entries)
	return result
}

// flush writes to disk if dirty.
func (s *store) flush() error {
	s.mu.RLock()
	dirty := s.dirty
	s.mu.RUnlock()

	if !dirty {
		return nil
	}

	if err := s.save(); err != nil {
		return err
	}

	s.mu.Lock()
	s.dirty = false
	s.mu.Unlock()

	return nil
}

// findByEmbedding performs a linear scan over all entries that have embeddings,
// returning the best match above the threshold.
func (s *store) findByEmbedding(query []float32, threshold float32) (string, float32, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var bestKey string
	var bestSim float32

	for _, e := range s.entries {
		if len(e.Embedding) == 0 {
			continue
		}
		emb := BytesToFloat32(e.Embedding)
		if emb == nil {
			continue
		}
		sim := CosineSimilarity(query, emb)
		if sim > bestSim {
			bestSim = sim
			bestKey = e.Hash
		}
	}

	if bestKey != "" && bestSim >= threshold {
		return bestKey, bestSim, true
	}

	return "", 0, false
}

// clear removes all entries.
func (s *store) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = make([]diskEntry, 0)
	s.keyIndex = make(map[string]int)
	s.dirty = true
}

// Sort interface for LRU eviction
type byLastUsed []diskEntry

func (b byLastUsed) Len() int           { return len(b) }
func (b byLastUsed) Less(i, j int) bool { return b[i].LastUsed.Before(b[j].LastUsed) }
func (b byLastUsed) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

var _ sort.Interface = byLastUsed(nil)
