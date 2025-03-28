package simplifiers

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Test HTML sample
const testHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>Test Document</title>
</head>
<body>
    <article id="content">
        <h1>Test Article</h1>
        <p>This is a test paragraph with <a href="#">a link</a>.</p>
        <div class="section">
            <h2>Test Section</h2>
            <p>Another paragraph with <img src="test.jpg" alt="Test Image"> and <a href="#">another link</a>.</p>
        </div>
    </article>
    <aside>
        <div class="sidebar">
            <h3>Related</h3>
            <ul>
                <li><a href="#">Link 1</a></li>
                <li><a href="#">Link 2</a></li>
            </ul>
        </div>
    </aside>
</body>
</html>
`

// Setup helper for creating test document
func setupTestDoc() (*goquery.Document, error) {
	return goquery.NewDocumentFromReader(strings.NewReader(testHTML))
}

// TestCache tests the basic cache functionality
func TestCache(t *testing.T) {
	cache := NewCache(10)
	
	// Set an item
	cache.Set("key1", "value1", 1*time.Minute)
	
	// Get the item
	value, found := cache.Get("key1")
	if !found {
		t.Error("Item not found in cache")
	}
	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}
	
	// Test expiration
	cache.Set("key2", "value2", 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	_, found = cache.Get("key2")
	if found {
		t.Error("Item should have expired")
	}
	
	// Test max size
	for i := 0; i < 15; i++ {
		cache.Set("item"+strconv.Itoa(i), i, 1*time.Minute)
	}
	
	// After adding 15 items to a cache with max size 10,
	// the first few should be evicted
	_, found = cache.Get("key1")
	if found {
		t.Error("Item should have been evicted due to max size")
	}
	
	// Test LRU - add 9 items to a fresh cache, then access the first one
	// to make it recently used
	cache = NewCache(10)
	cache.Set("item0", 0, 1*time.Minute)
	for i := 1; i < 9; i++ {
		cache.Set("item"+strconv.Itoa(i), i, 1*time.Minute)
	}
	
	// Access the first item to make it recently used
	cache.Get("item0")
	
	// Add 2 more items to push out the LRU item
	cache.Set("itemA", "A", 1*time.Minute)
	cache.Set("itemB", "B", 1*time.Minute)
	
	// The first item should still be in the cache since it was recently used
	_, found = cache.Get("item0")
	if !found {
		t.Error("Recently used item should not have been evicted")
	}
}

// TestCacheStats tests the cache statistics functionality
func TestCacheStats(t *testing.T) {
	cache := NewCache(10)
	
	// Set some items
	cache.Set("key1", "value1", 1*time.Minute)
	cache.Set("key2", "value2", 1*time.Minute)
	
	// Access some items (creating hits)
	cache.Get("key1") // Hit
	cache.Get("key1") // Hit
	cache.Get("key2") // Hit
	cache.Get("key3") // Miss
	
	// Check stats
	size, hits, misses, ratio := cache.Stats()
	
	if size != 2 {
		t.Errorf("Expected size 2, got %d", size)
	}
	if hits != 3 {
		t.Errorf("Expected 3 hits, got %d", hits)
	}
	if misses != 1 {
		t.Errorf("Expected 1 miss, got %d", misses)
	}
	if ratio != 0.75 {
		t.Errorf("Expected hit ratio 0.75, got %f", ratio)
	}
}

// TestCachedDocumentFromReader tests the document caching functionality
func TestCachedDocumentFromReader(t *testing.T) {
	opts := DefaultPerformanceOptions
	
	// First access should parse the document
	doc1, err := CachedNewDocumentFromReader(testHTML, opts)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}
	
	// Second access should use the cache
	doc2, err := CachedNewDocumentFromReader(testHTML, opts)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}
	
	// The two documents should be the same instance
	if doc1 != doc2 {
		t.Error("Document was not cached properly")
	}
	
	// Check with a different document
	differentHTML := "<html><body><p>Different</p></body></html>"
	doc3, err := CachedNewDocumentFromReader(differentHTML, opts)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}
	
	// This should be a different instance
	if doc1 == doc3 {
		t.Error("Different documents should be different instances")
	}
}

// TestBatchProcessSelections tests the batch processing of selections
func TestBatchProcessSelections(t *testing.T) {
	doc, err := setupTestDoc()
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}
	
	// To avoid declared and not used error with doc
	if doc == nil {
		t.Fatal("Document should not be nil")
	}
	
	// Define test operations
	linkCount := 0
	imgCount := 0
	headerCount := 0
	
	operations := map[string]func(*goquery.Selection){
		"a": func(s *goquery.Selection) {
			linkCount++
		},
		"img": func(s *goquery.Selection) {
			imgCount++
		},
		"h1, h2, h3": func(s *goquery.Selection) {
			headerCount++
		},
	}
	
	// Run batch processing
	BatchProcessSelections(doc, operations, DefaultPerformanceOptions)
	
	// Verify counts match what we expect
	expectedLinks := 4    // There are 4 links in the test HTML
	expectedImages := 1   // There is 1 image in the test HTML
	expectedHeaders := 3  // There are 3 headers (h1, h2, h3) in the test HTML
	
	if linkCount != expectedLinks {
		t.Errorf("Expected %d links, got %d", expectedLinks, linkCount)
	}
	if imgCount != expectedImages {
		t.Errorf("Expected %d images, got %d", expectedImages, imgCount)
	}
	if headerCount != expectedHeaders {
		t.Errorf("Expected %d headers, got %d", expectedHeaders, headerCount)
	}
}

// TestParallelProcessNodes tests the parallel node processing
func TestParallelProcessNodes(t *testing.T) {
	doc, err := setupTestDoc()
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}
	
	// Find all paragraphs
	paragraphs := []*goquery.Selection{}
	doc.Find("p").Each(func(i int, s *goquery.Selection) {
		paragraphs = append(paragraphs, s)
	})
	
	// Define a processor function
	processor := func(s *goquery.Selection) interface{} {
		return len(s.Text())
	}
	
	// Process in parallel
	opts := DefaultPerformanceOptions
	results := ParallelProcessNodes(paragraphs, processor, opts)
	
	// Verify results
	if len(results) != len(paragraphs) {
		t.Errorf("Expected %d results, got %d", len(paragraphs), len(results))
	}
	
	// Each result should be the length of the paragraph text
	for i, result := range results {
		expected := len(paragraphs[i].Text())
		if result.(int) != expected {
			t.Errorf("Expected result %d to be %d, got %d", i, expected, result.(int))
		}
	}
}

// Benchmarks

// BenchmarkCache benchmarks the cache performance
func BenchmarkCache(b *testing.B) {
	cache := NewCache(1000)
	
	// Prepare some test values
	values := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		values[i] = strconv.Itoa(i)
	}
	
	b.ResetTimer()
	
	// Benchmark set operations
	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			idx := i % 1000
			cache.Set(values[idx], i, 1*time.Minute)
		}
	})
	
	// Prepare cache with values
	for i := 0; i < 1000; i++ {
		cache.Set(values[i], i, 1*time.Minute)
	}
	
	// Benchmark get operations (80% hits, 20% misses)
	b.Run("Get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			idx := i % 1250 // Creates 20% misses (indices 1000-1249)
			cache.Get(values[idx%1000])
		}
	})
}

// BenchmarkCachedDocumentFromReader benchmarks document caching
func BenchmarkCachedDocumentFromReader(b *testing.B) {
	opts := DefaultPerformanceOptions
	
	b.Run("WithoutCaching", func(b *testing.B) {
		optsCopy := opts
		optsCopy.EnableCaching = false
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = CachedNewDocumentFromReader(testHTML, optsCopy)
		}
	})
	
	b.Run("WithCaching", func(b *testing.B) {
		optsCopy := opts
		optsCopy.EnableCaching = true
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = CachedNewDocumentFromReader(testHTML, optsCopy)
		}
	})
}

// BenchmarkBatchProcessSelections benchmarks batch processing
func BenchmarkBatchProcessSelections(b *testing.B) {
	doc, err := setupTestDoc()
	if err != nil {
		b.Fatalf("Failed to parse document: %v", err)
	}
	
	// To avoid declared and not used error with doc
	if doc == nil {
		b.Fatal("Document should not be nil")
	}
	
	// Define operations
	operations := map[string]func(*goquery.Selection){
		"a": func(s *goquery.Selection) {
			s.SetAttr("rel", "nofollow")
		},
		"img": func(s *goquery.Selection) {
			s.SetAttr("loading", "lazy")
		},
		"h1, h2, h3": func(s *goquery.Selection) {
			s.RemoveAttr("style")
		},
	}
	
	// Test batch processing vs. individual operations
	b.Run("IndividualOperations", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Clone document to avoid modifications affecting benchmarks
			docClone, _ := goquery.NewDocumentFromReader(strings.NewReader(testHTML))
			
			// Apply operations individually
			docClone.Find("a").Each(func(i int, s *goquery.Selection) {
				s.SetAttr("rel", "nofollow")
			})
			docClone.Find("img").Each(func(i int, s *goquery.Selection) {
				s.SetAttr("loading", "lazy")
			})
			docClone.Find("h1, h2, h3").Each(func(i int, s *goquery.Selection) {
				s.RemoveAttr("style")
			})
		}
	})
	
	b.Run("BatchOperations", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Clone document to avoid modifications affecting benchmarks
			docClone, _ := goquery.NewDocumentFromReader(strings.NewReader(testHTML))
			
			// Apply operations in batch
			BatchProcessSelections(docClone, operations, DefaultPerformanceOptions)
		}
	})
}

// BenchmarkParallelProcessNodes benchmarks node parallel processing
func BenchmarkParallelProcessNodes(b *testing.B) {
	// Create a larger test document with more nodes
	var testNodes strings.Builder
	for i := 0; i < 100; i++ {
		testNodes.WriteString(fmt.Sprintf("<div class='item-%d'>Test content %d</div>", i, i))
	}
	largeHTML := fmt.Sprintf("<html><body>%s</body></html>", testNodes.String())
	
	// To avoid declared and not used error with doc
	if len(largeHTML) == 0 {
		b.Fatal("HTML should not be empty")
	}
	
	// Parse the document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(largeHTML))
	if err != nil {
		b.Fatalf("Failed to parse document: %v", err)
	}
	
	// Select all divs
	nodes := []*goquery.Selection{}
	doc.Find("div").Each(func(i int, s *goquery.Selection) {
		nodes = append(nodes, s)
	})
	
	// Define a processor that does some work
	processor := func(s *goquery.Selection) interface{} {
		// Simulate some work
		time.Sleep(100 * time.Microsecond)
		return len(s.Text())
	}
	
	// Test sequential vs parallel processing
	b.Run("Sequential", func(b *testing.B) {
		opts := DefaultPerformanceOptions
		opts.EnableParallelProcessing = false
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ParallelProcessNodes(nodes, processor, opts)
		}
	})
	
	b.Run("Parallel", func(b *testing.B) {
		opts := DefaultPerformanceOptions
		opts.EnableParallelProcessing = true
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ParallelProcessNodes(nodes, processor, opts)
		}
	})
}