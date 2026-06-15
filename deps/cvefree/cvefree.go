// Package cvefree provides access to Kaze's CVE vulnerability dataset.
// Data sourced from https://github.com/kaze-technologies/cvefree.
package cvefree

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// DefaultDataURL is the upstream CVE dataset URL.
const DefaultDataURL = "https://kazepublic.blob.core.windows.net/cvefree/data.json"

// DefaultCacheDir is the default cache directory for CVE data.
const DefaultCacheDir = ".talon/cve-cache"

// CVE represents a single CVE entry from the dataset.
type CVE struct {
	CVEID              string   `json:"cve"`
	LastModified       string   `json:"last_modified_datetime"`
	Published          string   `json:"published_datetime"`
	Description        string   `json:"description"`
	Vendors            []string `json:"vendors"`
	CVSSv2             *float64 `json:"cvssv2"`
	CVSSv3             *float64 `json:"cvssv3"`
	EPSS               *float64 `json:"epss"`
	VScore             *float64 `json:"v_score"`
	CTICount           int      `json:"cti_count"`
	SocialMediaAudience int64   `json:"social_media_audience"`
	SoftwareCPEs       []string `json:"software_cpes"`
	CISA               bool     `json:"cisa"`
	Metasploit         bool     `json:"metasploit"`
}

// Severity returns the severity level based on CVSSv3 (or v2 fallback).
func (c CVE) Severity() string {
	s := c.CVSSv3
	if s == nil {
		s = c.CVSSv2
	}
	if s == nil {
		return "unknown"
	}
	switch {
	case *s >= 9.0:
		return "critical"
	case *s >= 7.0:
		return "high"
	case *s >= 4.0:
		return "medium"
	case *s > 0:
		return "low"
	default:
		return "unknown"
	}
}

// CVSS returns the best available CVSS score (v3 preferred, v2 fallback).
func (c CVE) CVSS() float64 {
	if c.CVSSv3 != nil {
		return *c.CVSSv3
	}
	if c.CVSSv2 != nil {
		return *c.CVSSv2
	}
	return 0
}

// DB holds the CVE dataset in memory, with lazy loading from cache or network.
type DB struct {
	mu       sync.RWMutex
	entries  []CVE
	indexed  map[string]*CVE // by CVE ID
	loaded   bool
	cacheDir string
	dataURL  string
}

// Option configures the DB.
type Option func(*DB)

// WithCacheDir sets a custom cache directory.
func WithCacheDir(dir string) Option {
	return func(d *DB) { d.cacheDir = dir }
}

// WithDataURL sets a custom data source URL.
func WithDataURL(url string) Option {
	return func(d *DB) { d.dataURL = url }
}

// New creates a new CVE database.
func New(opts ...Option) *DB {
	d := &DB{
		cacheDir: DefaultCacheDir,
		dataURL:  DefaultDataURL,
		indexed:  make(map[string]*CVE),
	}
	for _, o := range opts {
		o(d)
	}
	return d
}

// Load loads the CVE dataset from cache or downloads if needed.
func (d *DB) Load() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.loaded {
		return nil
	}

	// Try cache first
	cachePath := d.cachePath()
	if data, err := os.ReadFile(cachePath); err == nil {
		if err := json.Unmarshal(data, &d.entries); err == nil {
			d.buildIndex()
			d.loaded = true
			return nil
		}
	}

	// Download from upstream
	if err := d.download(); err != nil {
		return fmt.Errorf("downloading cve data: %w", err)
	}

	d.buildIndex()
	d.loaded = true
	return nil
}

// SearchResult holds a single search result.
type SearchResult struct {
	Total int   `json:"total"`
	Page  int   `json:"page"`
	Size  int   `json:"size"`
	CVEs  []CVE `json:"cves"`
}

// Query filters for searching CVEs.
type Query struct {
	Search    string   `json:"search,omitempty"`    // free-text search in description/CVE ID
	Vendor    string   `json:"vendor,omitempty"`    // filter by vendor
	Severity  string   `json:"severity,omitempty"`  // critical/high/medium/low
	CISA      *bool    `json:"cisa,omitempty"`      // filter by CISA KEV
	Metasploit *bool   `json:"metasploit,omitempty"`// filter by Metasploit
	MinCVSS   *float64 `json:"min_cvss,omitempty"`  // minimum CVSS score
	MaxCVSS   *float64 `json:"max_cvss,omitempty"`  // maximum CVSS score
	SortBy    string   `json:"sort_by,omitempty"`   // published, cvss, epss, vscore
	SortDir   string   `json:"sort_dir,omitempty"`  // asc, desc
	Page      int      `json:"page"`
	Size      int      `json:"size"`
}

// Search queries the CVE database with filters.
func (d *DB) Search(q Query) (*SearchResult, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if !d.loaded {
		return nil, fmt.Errorf("database not loaded")
	}

	if q.Page <= 0 {
		q.Page = 1
	}
	if q.Size <= 0 || q.Size > 1000 {
		q.Size = 50
	}

	searchLower := strings.ToLower(q.Search)
	vendorLower := strings.ToLower(q.Vendor)

	var filtered []CVE
	for _, c := range d.entries {
		// Free-text search
		if searchLower != "" {
			if !strings.Contains(strings.ToLower(c.CVEID), searchLower) &&
				!strings.Contains(strings.ToLower(c.Description), searchLower) {
				continue
			}
		}

		// Vendor filter
		if vendorLower != "" {
			found := false
			for _, v := range c.Vendors {
				if strings.Contains(strings.ToLower(v), vendorLower) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Severity filter
		if q.Severity != "" {
			if c.Severity() != q.Severity {
				continue
			}
		}

		// CISA filter
		if q.CISA != nil && c.CISA != *q.CISA {
			continue
		}

		// Metasploit filter
		if q.Metasploit != nil && c.Metasploit != *q.Metasploit {
			continue
		}

		// CVSS range
		cvss := c.CVSS()
		if q.MinCVSS != nil && cvss < *q.MinCVSS {
			continue
		}
		if q.MaxCVSS != nil && cvss > *q.MaxCVSS {
			continue
		}

		filtered = append(filtered, c)
	}

	// Sort
	d.sortCVEs(filtered, q.SortBy, q.SortDir)

	// Paginate
	total := len(filtered)
	start := (q.Page - 1) * q.Size
	if start >= total {
		return &SearchResult{Total: total, Page: q.Page, Size: q.Size, CVEs: []CVE{}}, nil
	}
	end := start + q.Size
	if end > total {
		end = total
	}

	return &SearchResult{
		Total: total,
		Page:  q.Page,
		Size:  q.Size,
		CVEs:  filtered[start:end],
	}, nil
}

// GetByID returns a single CVE by its ID.
func (d *DB) GetByID(id string) (*CVE, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	c, ok := d.indexed[strings.ToUpper(id)]
	if ok {
		return c, true
	}
	return nil, false
}

// Stats holds dataset statistics.
type Stats struct {
	Total      int `json:"total"`
	Critical   int `json:"critical"`
	High       int `json:"high"`
	Medium     int `json:"medium"`
	Low        int `json:"low"`
	Unknown    int `json:"unknown"`
	CISA       int `json:"cisa"`
	Metasploit int `json:"metasploit"`
}

// Stats returns aggregate statistics about the dataset.
func (d *DB) Stats() (*Stats, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if !d.loaded {
		return nil, fmt.Errorf("database not loaded")
	}

	s := &Stats{Total: len(d.entries)}
	for _, c := range d.entries {
		switch c.Severity() {
		case "critical":
			s.Critical++
		case "high":
			s.High++
		case "medium":
			s.Medium++
		case "low":
			s.Low++
		default:
			s.Unknown++
		}
		if c.CISA {
			s.CISA++
		}
		if c.Metasploit {
			s.Metasploit++
		}
	}
	return s, nil
}

// LastUpdated returns the latest published date in the dataset.
func (d *DB) LastUpdated() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var latest string
	for _, c := range d.entries {
		if c.Published > latest {
			latest = c.Published
		}
	}
	return latest
}

// --- internal ---

func (d *DB) cachePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, d.cacheDir, "cve-data.json")
}

func (d *DB) download() error {
	fmt.Printf("Downloading CVE data from %s ...\n", d.dataURL)

	resp, err := http.Get(d.dataURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Parse to validate
	var entries []CVE
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}

	d.entries = entries

	// Cache to disk
	cachePath := d.cachePath()
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return err
	}

	fmt.Printf("Downloaded %d CVEs\n", len(entries))
	return nil
}

func (d *DB) buildIndex() {
	d.indexed = make(map[string]*CVE, len(d.entries))
	for i := range d.entries {
		d.indexed[strings.ToUpper(d.entries[i].CVEID)] = &d.entries[i]
	}
}

func (d *DB) sortCVEs(cves []CVE, sortBy, sortDir string) {
	// Simple insertion sort for now; dataset is already roughly sorted by published
	if sortBy == "" {
		sortBy = "published"
	}
	desc := strings.ToLower(sortDir) != "asc"

	for i := 1; i < len(cves); i++ {
		for j := i; j > 0; j-- {
			less := compareCVE(cves[j-1], cves[j], sortBy)
			if desc {
				less = !less
			}
			if !less {
				break
			}
			cves[j-1], cves[j] = cves[j], cves[j-1]
		}
	}
}

func compareCVE(a, b CVE, sortBy string) bool {
	switch sortBy {
	case "cvss":
		return a.CVSS() < b.CVSS()
	case "epss":
		return getVal(a.EPSS) < getVal(b.EPSS)
	case "vscore":
		return getVal(a.VScore) < getVal(b.VScore)
	default: // published
		return a.Published < b.Published
	}
}

func getVal(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

// Refresh redownloads the dataset from upstream.
func (d *DB) Refresh() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Remove cache
	cachePath := d.cachePath()
	os.Remove(cachePath)

	if err := d.download(); err != nil {
		return err
	}

	d.buildIndex()
	d.loaded = true
	return nil
}

// Vendors returns a list of all unique vendors in the dataset.
func (d *DB) Vendors() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	seen := make(map[string]bool)
	var vendors []string
	for _, c := range d.entries {
		for _, v := range c.Vendors {
			if !seen[v] {
				seen[v] = true
				vendors = append(vendors, v)
			}
		}
	}
	return vendors
}
