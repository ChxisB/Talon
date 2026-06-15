package promptcache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Config controls cache behavior.
type Config struct {
	// CacheDir is the directory for cache storage. Defaults to ~/.talon/prompt-cache/.
	CacheDir string
	// TTL is how long cached entries live before expiring. Default: 24h.
	TTL time.Duration
	// MaxEntries is the maximum number of cached entries (LRU eviction). Default: 10000.
	MaxEntries int
	// HighThreshold is the cosine similarity threshold for a semantic cache hit.
	// Must be between 0 and 1. Higher = stricter match. Default: 0.92.
	HighThreshold float32
	// LowThreshold is the lower bound for the gray zone. Below this is a clear miss.
	// Default: 0.70.
	LowThreshold float32
	// EnableGrayZoneVerifier enables the secondary LLM verification for gray-zone hits.
	// When disabled, gray-zone results are treated as misses. Default: true.
	EnableGrayZoneVerifier bool
	// FlushInterval is how often the cache is flushed to disk. Default: 30s.
	FlushInterval time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	cacheDir := filepath.Join(home, ".talon", "prompt-cache")
	return Config{
		CacheDir:               cacheDir,
		TTL:                    24 * time.Hour,
		MaxEntries:             10000,
		HighThreshold:          0.92,
		LowThreshold:           0.70,
		EnableGrayZoneVerifier: true,
		FlushInterval:          30 * time.Second,
	}
}

// Embedder generates an embedding vector for a text string.
type Embedder func(ctx context.Context, text string) ([]float32, error)

// GrazzZoneVerifier checks whether two prompts are semantically equivalent.
type GrayZoneVerifier func(ctx context.Context, a, b string) (bool, error)

// Entry is a cached prompt-response pair returned from Check.
type Entry struct {
	// Prompt is the original prompt text.
	Prompt string
	// Response is the cached LLM response.
	Response string
	// Similarity is the cosine similarity score (0-1) for semantic matches.
	// For exact hash matches, this is 1.0.
	Similarity float32
	// MatchedBy indicates how the match was found ("exact" or "semantic").
	MatchedBy string
	// CreatedAt is when this entry was cached.
	CreatedAt time.Time
	// Hash is the SHA-256 hash of the prompt.
	Hash string
}

// Stats provides cache usage statistics.
type Stats struct {
	Entries    int     `json:"entries"`
	MaxEntries int     `json:"max_entries"`
	Hits       int64   `json:"hits"`
	Misses     int64   `json:"misses"`
	HitRate    float64 `json:"hit_rate"`
}

// Cache is a semantic prompt-response cache.
type Cache struct {
	config     Config
	store      *store
	embedder   Embedder
	verifier   GrayZoneVerifier
	hits       int64
	misses     int64
	mu         sync.RWMutex
	stopFlush  chan struct{}
	flushDone  chan struct{}
	initOnce   sync.Once
}

// New creates a new Cache with the given config and embedding function.
// The embedder is optional — if nil, only exact hash matching is used.
func New(config Config, embedder Embedder) *Cache {
	if config.CacheDir == "" {
		config.CacheDir = DefaultConfig().CacheDir
	}
	if config.TTL == 0 {
		config.TTL = DefaultConfig().TTL
	}
	if config.MaxEntries == 0 {
		config.MaxEntries = DefaultConfig().MaxEntries
	}
	if config.HighThreshold == 0 {
		config.HighThreshold = DefaultConfig().HighThreshold
	}
	if config.LowThreshold == 0 {
		config.LowThreshold = DefaultConfig().LowThreshold
	}
	if config.FlushInterval == 0 {
		config.FlushInterval = DefaultConfig().FlushInterval
	}

	return &Cache{
		config:    config,
		embedder:  embedder,
		verifier:  nil, // optional, set via SetGrayZoneVerifier
		stopFlush: make(chan struct{}),
		flushDone: make(chan struct{}),
	}
}

// SetGrayZoneVerifier sets an optional verifier for gray-zone matches.
// If set, prompts in the gray zone (between LowThreshold and HighThreshold)
// are verified using this function before returning a hit.
func (c *Cache) SetGrayZoneVerifier(v GrayZoneVerifier) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.verifier = v
}

// SetEmbedder sets or updates the embedding function for semantic matching.
func (c *Cache) SetEmbedder(e Embedder) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.embedder = e
}

// lazyInit creates the store on first use.
func (c *Cache) lazyInit() {
	c.initOnce.Do(func() {
		c.store = newStore(c.config.CacheDir, c.config.MaxEntries)
		go c.flushLoop()
	})
}

// GenerateKey returns a SHA-256 hex hash of the input.
func GenerateKey(input string) string {
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:])
}

// Check looks for a cached response matching the prompt.
// It first tries exact hash match, then semantic (embedding) match if an embedder is set.
// Returns the entry and true if found, nil and false otherwise.
func (c *Cache) Check(ctx context.Context, prompt string) (*Entry, bool) {
	c.lazyInit()

	hash := GenerateKey(prompt)

	// 1. Try exact hash match
	if entry := c.store.get(hash); entry != nil {
		c.store.touch(hash)
		c.recordHit()
		return &Entry{
			Prompt:     entry.Prompt,
			Response:   entry.Response,
			Similarity: 1.0,
			MatchedBy:  "exact",
			CreatedAt:  entry.CreatedAt,
			Hash:       hash,
		}, true
	}

	// 2. Try semantic match if embedder is available
	if c.embedder != nil {
		return c.checkSemantic(ctx, prompt, hash)
	}

	c.recordMiss()
	return nil, false
}

// checkSemantic performs embedding-based similarity search.
func (c *Cache) checkSemantic(ctx context.Context, prompt, hash string) (*Entry, bool) {
	queryEmb, err := c.embedder(ctx, prompt)
	if err != nil {
		c.recordMiss()
		return nil, false
	}

	similarHash, sim, found := c.store.findByEmbedding(queryEmb, c.config.LowThreshold)
	if !found {
		c.recordMiss()
		return nil, false
	}

	// Clear hit
	if sim >= c.config.HighThreshold {
		entry := c.store.get(similarHash)
		if entry != nil {
			c.store.touch(similarHash)
			c.recordHit()
			return &Entry{
				Prompt:     entry.Prompt,
				Response:   entry.Response,
				Similarity: sim,
				MatchedBy:  "semantic",
				CreatedAt:  entry.CreatedAt,
				Hash:       similarHash,
			}, true
		}
		c.recordMiss()
		return nil, false
	}

	// Gray zone — use verifier if available
	if c.config.EnableGrayZoneVerifier {
		c.mu.RLock()
		verifier := c.verifier
		c.mu.RUnlock()

		if verifier != nil {
			entry := c.store.get(similarHash)
			if entry != nil {
				match, err := verifier(ctx, prompt, entry.Prompt)
				if err == nil && match {
					c.store.touch(similarHash)
					c.recordHit()
					return &Entry{
						Prompt:     entry.Prompt,
						Response:   entry.Response,
						Similarity: sim,
						MatchedBy:  "semantic",
						CreatedAt:  entry.CreatedAt,
						Hash:       similarHash,
					}, true
				}
			}
		}
	}

	c.recordMiss()
	return nil, false
}

// Store caches a prompt-response pair. If an embedder is set, it also computes
// and stores the embedding for future semantic lookups.
func (c *Cache) Store(ctx context.Context, prompt, response string) error {
	c.lazyInit()

	hash := GenerateKey(prompt)

	var emb []byte
	if c.embedder != nil {
		vec, err := c.embedder(ctx, prompt)
		if err == nil {
			emb = Float32ToBytes(vec)
		}
	}

	entry := diskEntry{
		Prompt:    prompt,
		Response:  response,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		TTL:       c.config.TTL.String(),
		Hash:      hash,
		Embedding: emb,
	}

	c.store.set(hash, entry)
	return nil
}

// Flush writes all dirty entries to disk.
func (c *Cache) Flush() error {
	c.lazyInit()
	return c.store.flush()
}

// Stats returns current cache statistics.
func (c *Cache) Stats() Stats {
	c.lazyInit()
	c.mu.RLock()
	hits := c.hits
	misses := c.misses
	c.mu.RUnlock()

	total := hits + misses
	rate := 0.0
	if total > 0 {
		rate = float64(hits) / float64(total) * 100
	}

	return Stats{
		Entries:    c.store.count(),
		MaxEntries: c.config.MaxEntries,
		Hits:       hits,
		Misses:     misses,
		HitRate:    rate,
	}
}

// Clear removes all cached entries.
func (c *Cache) Clear() error {
	c.lazyInit()
	c.store.clear()
	return c.store.flush()
}

// Close flushes and stops background goroutines.
func (c *Cache) Close() error {
	select {
	case <-c.stopFlush:
		// already closed
	default:
		close(c.stopFlush)
		<-c.flushDone
	}
	return c.store.flush()
}

// recordHit increments the hit counter.
func (c *Cache) recordHit() {
	c.mu.Lock()
	c.hits++
	c.mu.Unlock()
}

// recordMiss increments the miss counter.
func (c *Cache) recordMiss() {
	c.mu.Lock()
	c.misses++
	c.mu.Unlock()
}

// flushLoop periodically flushes dirty entries to disk.
func (c *Cache) flushLoop() {
	ticker := time.NewTicker(c.config.FlushInterval)
	defer ticker.Stop()
	defer close(c.flushDone)

	for {
		select {
		case <-ticker.C:
			_ = c.store.flush()
		case <-c.stopFlush:
			return
		}
	}
}


