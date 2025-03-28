package simplifiers

import (
	"container/list"
	"crypto/md5"
	"encoding/hex"
	"io"
	"runtime"
	"strconv"
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
	element    *list.Element // Reference to the list element for quick access
}

// Cache is an improved in-memory cache with LRU eviction
type Cache struct {
	mu           sync.RWMutex
	items        map[string]*cacheItem
	lruList      *list.List    // Tracks least recently used items
	maxSize      int
	cleanupTick  *time.Ticker
	done         chan struct{} // Signal to stop the cleanup goroutine
	cleanupEvery time.Duration
	hitCount     int64 // Cache hit statistics
	missCount    int64 // Cache miss statistics
}

// NewCache creates a new cache with the specified maximum size
// and starts a background cleanup routine
func NewCache(maxSize int) *Cache {
	if maxSize <= 0 {
		maxSize = 100 // Default size
	}

	cache := &Cache{
		items:        make(map[string]*cacheItem, maxSize),
		lruList:      list.New(),
		maxSize:      maxSize,
		cleanupEvery: 30 * time.Second, // Clean expired items every 30 seconds
		done:         make(chan struct{}),
	}

	// Start background cleanup
	cache.cleanupTick = time.NewTicker(cache.cleanupEvery)
	go cache.cleanupRoutine()

	// Set finalizer to ensure cleanup ticker is stopped
	runtime.SetFinalizer(cache, func(c *Cache) {
		c.cleanupTick.Stop()
		close(c.done)
	})

	return cache
}

// cleanupRoutine periodically removes expired items from the cache
func (c *Cache) cleanupRoutine() {
	for {
		select {
		case <-c.cleanupTick.C:
			c.removeExpired()
		case <-c.done:
			return
		}
	}
}

// removeExpired removes all expired items from the cache
func (c *Cache) removeExpired() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()

	// Use list iteration for better performance than map iteration
	for e := c.lruList.Front(); e != nil; {
		next := e.Next() // Store next before potentially removing e
		key := e.Value.(string)
		if item, found := c.items[key]; found && now.After(item.expiration) {
			delete(c.items, key)
			c.lruList.Remove(e)
		}
		e = next
	}
}

// Get retrieves an item from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	item, found := c.items[key]
	if !found {
		c.mu.RUnlock()
		c.mu.Lock()
		c.missCount++
		c.mu.Unlock()
		return nil, false
	}
	
	now := time.Now()
	if now.After(item.expiration) {
		c.mu.RUnlock()
		c.mu.Lock()
		defer c.mu.Unlock()
		// Double-check expiration under write lock
		if item, found := c.items[key]; found && now.After(item.expiration) {
			delete(c.items, key)
			c.lruList.Remove(item.element)
			c.missCount++
			return nil, false
		}
		return nil, false
	}
	
	c.mu.RUnlock()
	// Update LRU list under write lock
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Move to front of LRU list to mark as recently used
	c.lruList.MoveToFront(item.element)
	c.hitCount++
	
	return item.value, true
}

// Set adds an item to the cache
func (c *Cache) Set(key string, value interface{}, expiration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If item already exists, update it
	if existingItem, found := c.items[key]; found {
		existingItem.value = value
		existingItem.expiration = time.Now().Add(expiration)
		c.lruList.MoveToFront(existingItem.element)
		return
	}

	// Check if the cache is full
	if len(c.items) >= c.maxSize {
		// Remove the least recently used item (back of the list)
		oldest := c.lruList.Back()
		if oldest != nil {
			oldestKey := oldest.Value.(string)
			delete(c.items, oldestKey)
			c.lruList.Remove(oldest)
		}
	}

	// Add to front of LRU list
	element := c.lruList.PushFront(key)
	
	// Create new cache item
	newItem := &cacheItem{
		value:      value,
		expiration: time.Now().Add(expiration),
		element:    element,
	}
	
	// Add to items map
	c.items[key] = newItem
}

// Clear clears the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*cacheItem, c.maxSize)
	c.lruList.Init() // Reset the LRU list
	c.hitCount = 0
	c.missCount = 0
}

// Stats returns the cache statistics
func (c *Cache) Stats() (size int, hits int64, misses int64, ratio float64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	size = len(c.items)
	hits = c.hitCount
	misses = c.missCount
	
	if hits+misses > 0 {
		ratio = float64(hits) / float64(hits+misses)
	}
	
	return
}

// Caches for different types of content
var (
	// DocumentCache is a cache for parsed documents
	DocumentCache = NewCache(200)

	// ContentNodeCache is a cache for main content nodes
	ContentNodeCache = NewCache(200)

	// ContentScoreCache is a cache for content scores
	ContentScoreCache = NewCache(2000)
	
	// ElementCache is a cache for element selector results
	ElementCache = NewCache(500)
)

// stringReader is a simple io.Reader for strings
type stringReader struct {
	*strings.Reader
}

// NewStringReader creates a new string reader
func NewStringReader(s string) io.Reader {
	return &stringReader{strings.NewReader(s)}
}

// calculateStringDigest calculates a digest for a string
// Uses a fast but efficient hash for cache keys
func calculateStringDigest(s string) string {
	// For long strings, use a prefix to improve performance
	if len(s) > 1024 {
		// Use first 1KB + length as a unique identifier
		// This is a compromise between uniqueness and performance
		prefix := s[:1024]
		lenStr := len(s)
		
		// Combine prefix with length for better uniqueness
		hash := md5.Sum([]byte(prefix + ":" + strconv.Itoa(lenStr)))
		return hex.EncodeToString(hash[:])
	}
	
	// For shorter strings, hash the entire string
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}

// documentCacheKey generates a cache key for a document using a
// more efficient approach than hashing the entire HTML
func documentCacheKey(html string) string {
	return "doc:" + calculateStringDigest(html)
}

// nodeCacheKey generates a cache key for a node using node-specific attributes
// rather than the full HTML content
func nodeCacheKey(doc *goquery.Document) string {
	// Get basic document info - more efficient than full HTML
	title := doc.Find("title").Text()
	bodyStart, _ := doc.Find("body").Html()
	
	// Limit body content to first 512 bytes
	if len(bodyStart) > 512 {
		bodyStart = bodyStart[:512]
	}
	
	// Use document features plus length as signature
	signature := title + "|" + bodyStart + "|" + strconv.Itoa(len(doc.Text()))
	return "node:" + calculateStringDigest(signature)
}

// scoreCacheKey generates a cache key for content scoring
func scoreCacheKey(s *goquery.Selection) string {
	// Try to get attributes first as they're more stable for cache keys
	id, hasID := s.Attr("id")
	class, hasClass := s.Attr("class")
	
	// Generate a selection signature
	var signature string
	if hasID {
		signature += "id:" + id + "|"
	}
	if hasClass {
		signature += "class:" + class + "|"
	}
	
	// Add tag name
	signature += "tag:" + goquery.NodeName(s) + "|"
	
	// Add content sample and text length
	text := s.Text()
	textLen := len(text)
	
	// Add text sample (first 256 chars) if available
	if textLen > 0 {
		sampleSize := 256
		if textLen < sampleSize {
			sampleSize = textLen
		}
		signature += "sample:" + text[:sampleSize] + "|"
	}
	
	signature += "len:" + strconv.Itoa(textLen)
	
	return "score:" + calculateStringDigest(signature)
}

// CachedNewDocumentFromReader parses a document with caching
func CachedNewDocumentFromReader(html string, opts PerformanceOptions) (*goquery.Document, error) {
	if !opts.EnableCaching {
		return goquery.NewDocumentFromReader(NewStringReader(html))
	}

	// Generate a cache key more efficiently
	key := documentCacheKey(html)

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

	// Generate a more efficient cache key
	key := nodeCacheKey(doc)

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

	// Generate a more efficient cache key
	key := scoreCacheKey(s)

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

// CachedFind performs a cached element selection
func CachedFind(doc *goquery.Document, selector string, opts PerformanceOptions) *goquery.Selection {
	if !opts.EnableCaching {
		return doc.Find(selector)
	}
	
	// Create a cache key based on document signature and selector
	docSig := nodeCacheKey(doc)
	key := "find:" + docSig + "|" + selector
	
	// Check if the selection is in the cache
	if cachedSelection, found := ElementCache.Get(key); found {
		return cachedSelection.(*goquery.Selection)
	}
	
	// Perform the selection
	selection := doc.Find(selector)
	
	// Only cache if the selection contains elements
	if selection.Length() > 0 {
		ElementCache.Set(key, selection, opts.CacheExpiration)
	}
	
	return selection
}

// BatchProcessSelections processes multiple selections with a single DOM walk
// This helps avoid redundant DOM traversals when multiple operations
// need to be performed on the same document
func BatchProcessSelections(doc *goquery.Document, operations map[string]func(*goquery.Selection), opts PerformanceOptions) {
	if doc == nil || len(operations) == 0 {
		return
	}
	
	// Process all elements in the document with a single traversal
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		// Get the element name
		tagName := goquery.NodeName(s)
		
		// Apply all relevant operations to this element
		for selector, operation := range operations {
			// Check if this element matches the selector
			// For simple tag selectors, we can optimize with a direct check
			if selector == tagName || selector == "*" {
				operation(s)
				continue
			}
			
			// For complex selectors, we need to do a proper match test
			if s.Is(selector) {
				operation(s)
			}
		}
	})
}

// ParallelProcessNodes processes nodes in parallel with improved batching
func ParallelProcessNodes(nodes []*goquery.Selection, processor func(*goquery.Selection) interface{}, opts PerformanceOptions) []interface{} {
	nodeCount := len(nodes)
	
	// For small datasets or when parallelism is disabled, process sequentially
	if !opts.EnableParallelProcessing || nodeCount <= 1 {
		results := make([]interface{}, nodeCount)
		for i, node := range nodes {
			results[i] = processor(node)
		}
		return results
	}

	// For medium datasets, use a more efficient approach with fewer goroutines
	if nodeCount < 20 {
		// Determine optimal number of goroutines based on node count and available cores
		numWorkers := opts.MaxParallelism
		if numWorkers > nodeCount {
			numWorkers = nodeCount
		}
		
		// Create results slice
		results := make([]interface{}, nodeCount)
		
		// Use a worker pool pattern
		var wg sync.WaitGroup
		wg.Add(numWorkers)
		
		// Calculate batch size for each worker
		batchSize := (nodeCount + numWorkers - 1) / numWorkers
		
		// Start workers
		for w := 0; w < numWorkers; w++ {
			go func(workerID int) {
				defer wg.Done()
				
				// Calculate start and end indices for this worker
				start := workerID * batchSize
				end := start + batchSize
				if end > nodeCount {
					end = nodeCount
				}
				
				// Process assigned nodes
				for i := start; i < end; i++ {
					results[i] = processor(nodes[i])
				}
			}(w)
		}
		
		wg.Wait()
		return results
	}

	// For larger datasets, use worker queue approach for better load balancing
	results := make([]interface{}, nodeCount)
	jobs := make(chan int, nodeCount)
	var wg sync.WaitGroup
	
	// Create worker pool
	numWorkers := opts.MaxParallelism
	wg.Add(numWorkers)
	
	// Start workers
	for w := 0; w < numWorkers; w++ {
		go func() {
			defer wg.Done()
			for i := range jobs {
				results[i] = processor(nodes[i])
			}
		}()
	}
	
	// Send jobs to workers
	for i := range nodes {
		jobs <- i
	}
	close(jobs)
	
	// Wait for all workers to complete
	wg.Wait()
	return results
}

// LazyLoadImages adds lazy loading to images more efficiently
func LazyLoadImages(doc *goquery.Document, opts PerformanceOptions) *goquery.Document {
	if !opts.EnableLazyLoading {
		return doc
	}

	// Use caching to avoid reprocessing the same document
	key := "lazy_" + nodeCacheKey(doc)
	if cachedDoc, found := DocumentCache.Get(key); found {
		return cachedDoc.(*goquery.Document)
	}

	// Add loading="lazy" to all images
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		s.SetAttr("loading", "lazy")
	})

	// Cache the result if caching is enabled
	if opts.EnableCaching {
		DocumentCache.Set(key, doc, opts.CacheExpiration)
	}

	return doc
}

// OptimizeDocument applies all performance optimizations to a document
func OptimizeDocument(doc *goquery.Document, opts PerformanceOptions) *goquery.Document {
	if doc == nil {
		return nil
	}
	
	// Generate a cache key for optimized document
	key := "opt_" + nodeCacheKey(doc)
	
	// Check if we already have an optimized version
	if opts.EnableCaching {
		if cachedDoc, found := DocumentCache.Get(key); found {
			return cachedDoc.(*goquery.Document)
		}
	}

	// Lazy load images
	doc = LazyLoadImages(doc, opts)

	// Prepare attribute lists for faster filtering
	keepDataAttrs := map[string]bool{
		"data-src":    true,
		"data-srcset": true,
		"data-content-digest": true,
		"data-node-index":     true,
	}

	// Remove unnecessary attributes more efficiently
	// Use custom attribute filtering to avoid creating intermediary slices
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		if s.Length() == 0 || len(s.Nodes) == 0 {
			return
		}
		
		// Group attribute operations to reduce DOM manipulations
		var attrsToRemove []string
		
		for _, attr := range s.Nodes[0].Attr {
			// Check data attributes - only keep specific ones 
			if strings.HasPrefix(attr.Key, "data-") && !keepDataAttrs[attr.Key] {
				attrsToRemove = append(attrsToRemove, attr.Key)
				continue
			}
			
			// Remove event handlers
			if strings.HasPrefix(attr.Key, "on") {
				attrsToRemove = append(attrsToRemove, attr.Key)
				continue
			}
		}
		
		// Batch removal of attributes
		for _, attrName := range attrsToRemove {
			s.RemoveAttr(attrName)
		}
	})

	// Cache the optimized document if caching is enabled
	if opts.EnableCaching {
		DocumentCache.Set(key, doc, opts.CacheExpiration)
	}

	return doc
}