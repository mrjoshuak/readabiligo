package simplifiers

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// PerformanceOptions defines options for performance optimization
type PerformanceOptions struct {
	EnableCaching            bool
	CacheExpiration          time.Duration
	EnableParallelProcessing bool
	MaxParallelism           int
	EnableLazyLoading        bool
}

// DefaultPerformanceOptions provides default options for performance optimization
var DefaultPerformanceOptions = PerformanceOptions{
	EnableCaching:            true,
	CacheExpiration:          5 * time.Minute,
	EnableParallelProcessing: true,
	MaxParallelism:           4,
	EnableLazyLoading:        true,
}

// cacheItem represents an item in the cache
type cacheItem struct {
	value      interface{}
	expiration time.Time
}

// Cache is a simple in-memory cache
type Cache struct {
	mu      sync.RWMutex
	items   map[string]cacheItem
	maxSize int
}

// NewCache creates a new cache with the specified maximum size
func NewCache(maxSize int) *Cache {
	return &Cache{
		items:   make(map[string]cacheItem),
		maxSize: maxSize,
	}
}

// Get retrieves an item from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return nil, false
	}

	// Check if the item has expired
	if time.Now().After(item.expiration) {
		delete(c.items, key)
		return nil, false
	}

	return item.value, true
}

// Set adds an item to the cache
func (c *Cache) Set(key string, value interface{}, expiration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if the cache is full
	if len(c.items) >= c.maxSize {
		// Remove the oldest item
		var oldestKey string
		var oldestTime time.Time
		for k, v := range c.items {
			if oldestKey == "" || v.expiration.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.expiration
			}
		}
		delete(c.items, oldestKey)
	}

	c.items[key] = cacheItem{
		value:      value,
		expiration: time.Now().Add(expiration),
	}
}

// Clear clears the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]cacheItem)
}

// DocumentCache is a cache for parsed documents
var DocumentCache = NewCache(100)

// ContentNodeCache is a cache for main content nodes
var ContentNodeCache = NewCache(100)

// ContentScoreCache is a cache for content scores
var ContentScoreCache = NewCache(1000)

// stringReader is a simple io.Reader for strings
type stringReader struct {
	*strings.Reader
}

// NewStringReader creates a new string reader
func NewStringReader(s string) io.Reader {
	return &stringReader{strings.NewReader(s)}
}

// calculateStringDigest calculates a digest for a string
func calculateStringDigest(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}

// CachedNewDocumentFromReader parses a document with caching
func CachedNewDocumentFromReader(html string, opts PerformanceOptions) (*goquery.Document, error) {
	if !opts.EnableCaching {
		return goquery.NewDocumentFromReader(NewStringReader(html))
	}

	// Generate a cache key
	key := "doc:" + calculateStringDigest(html)

	// Check if the document is in the cache
	if cachedDoc, found := DocumentCache.Get(key); found {
		return cachedDoc.(*goquery.Document), nil
	}

	// Parse the document
	doc, err := goquery.NewDocumentFromReader(NewStringReader(html))
	if err != nil {
		return nil, err
	}

	// Cache the document
	DocumentCache.Set(key, doc, opts.CacheExpiration)

	return doc, nil
}

// CachedFindMainContentNode finds the main content node with caching
func CachedFindMainContentNode(doc *goquery.Document, opts PerformanceOptions) *goquery.Selection {
	if !opts.EnableCaching {
		return FindMainContentNode(doc)
	}

	// Generate a cache key
	html, err := doc.Html()
	if err != nil {
		return FindMainContentNode(doc)
	}
	key := "main:" + calculateStringDigest(html)

	// Check if the main content node is in the cache
	if cachedNode, found := ContentNodeCache.Get(key); found {
		return cachedNode.(*goquery.Selection)
	}

	// Find the main content node
	mainContent := FindMainContentNode(doc)

	// Cache the main content node
	ContentNodeCache.Set(key, mainContent, opts.CacheExpiration)

	return mainContent
}

// CachedCalculateContentScore calculates a content score with caching
func CachedCalculateContentScore(s *goquery.Selection, opts PerformanceOptions) float64 {
	if !opts.EnableCaching {
		return CalculateContentScore(s)
	}

	// Generate a cache key
	html, err := goquery.OuterHtml(s)
	if err != nil {
		return CalculateContentScore(s)
	}
	key := "score:" + calculateStringDigest(html)

	// Check if the score is in the cache
	if cachedScore, found := ContentScoreCache.Get(key); found {
		return cachedScore.(float64)
	}

	// Calculate the score
	score := CalculateContentScore(s)

	// Cache the score
	ContentScoreCache.Set(key, score, opts.CacheExpiration)

	return score
}

// ParallelProcessNodes processes nodes in parallel
func ParallelProcessNodes(nodes []*goquery.Selection, processor func(*goquery.Selection) interface{}, opts PerformanceOptions) []interface{} {
	if !opts.EnableParallelProcessing || len(nodes) <= 1 {
		// Process sequentially for small number of nodes
		results := make([]interface{}, len(nodes))
		for i, node := range nodes {
			results[i] = processor(node)
		}
		return results
	}

	// Process in parallel
	results := make([]interface{}, len(nodes))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, opts.MaxParallelism)

	for i, node := range nodes {
		wg.Add(1)
		go func(index int, n *goquery.Selection) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			results[index] = processor(n)
		}(i, node)
	}

	wg.Wait()
	return results
}

// LazyLoadImages adds lazy loading to images
func LazyLoadImages(doc *goquery.Document, opts PerformanceOptions) *goquery.Document {
	if !opts.EnableLazyLoading {
		return doc
	}

	// Add loading="lazy" to all images
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		s.SetAttr("loading", "lazy")
	})

	return doc
}

// OptimizeDocument applies all performance optimizations to a document
func OptimizeDocument(doc *goquery.Document, opts PerformanceOptions) *goquery.Document {
	// Lazy load images
	doc = LazyLoadImages(doc, opts)

	// Remove unnecessary attributes
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		// Remove data attributes except for data-src and data-srcset
		for _, attr := range s.Nodes[0].Attr {
			if strings.HasPrefix(attr.Key, "data-") && attr.Key != "data-src" && attr.Key != "data-srcset" {
				s.RemoveAttr(attr.Key)
			}
		}

		// Remove event handlers
		for _, attr := range s.Nodes[0].Attr {
			if strings.HasPrefix(attr.Key, "on") {
				s.RemoveAttr(attr.Key)
			}
		}
	})

	return doc
}
