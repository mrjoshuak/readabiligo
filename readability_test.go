package readabiligo_test

import (
	"strings"
	"testing"
	"time"

	"github.com/mrjoshuak/readabiligo"
)

func TestExtractor(t *testing.T) {
	// Create a new extractor
	ext := readabiligo.New()

	// Test HTML
	html := `<html><head><title>Test Title</title></head><body><header><nav><ul><li><a href="#">Home</a></li><li><a href="#">About</a></li></ul></nav></header><main><article><h1>Test Title</h1><p>This is a test paragraph with enough text to be considered relevant content by the Readability algorithm. We need to ensure that this paragraph has sufficient length to be scored highly by the content extraction algorithm. The algorithm looks for blocks of text that appear to be the main content of the page, as opposed to navigation, headers, footers, or other ancillary content.</p><p>Adding another paragraph increases the content score for this article element, making it more likely to be identified as the main content of the page. The Readability algorithm is designed to extract the primary content from a webpage, ignoring elements that are likely to be navigation, ads, or other non-content features.</p></article></main><footer><p>Copyright 2025</p></footer></body></html>`

	// Extract article
	article, err := ext.ExtractFromHTML(html, nil)
	if err != nil {
		t.Fatalf("Failed to extract article: %v", err)
	}

	// Check title
	if article.Title != "Test Title" {
		t.Errorf("Expected title 'Test Title', got '%s'", article.Title)
	}

	// Check if content is extracted
	if len(article.Content) == 0 {
		t.Error("Expected non-empty content")
	}

	// Check if plain text is extracted
	if len(article.PlainText) == 0 {
		t.Error("Expected non-empty plain text")
	}
}

func TestOptions(t *testing.T) {
	// Create a new extractor with options
	ext := readabiligo.New(
		readabiligo.WithContentDigests(true),
		readabiligo.WithNodeIndexes(true),
		readabiligo.WithTimeout(time.Second*5),
	)

	// Test HTML
	html := `<html><head><title>Option Test</title></head><body><header><nav><ul><li><a href="#">Home</a></li><li><a href="#">About</a></li></ul></nav></header><main><article><h1>Option Test</h1><p>This is a test paragraph with enough text to be considered relevant content by the Readability algorithm. We need to ensure that this paragraph has sufficient length to be scored highly by the content extraction algorithm. The algorithm looks for blocks of text that appear to be the main content of the page, as opposed to navigation, headers, footers, or other ancillary content.</p><p>Adding another paragraph increases the content score for this article element, making it more likely to be identified as the main content of the page. The Readability algorithm is designed to extract the primary content from a webpage, ignoring elements that are likely to be navigation, ads, or other non-content features.</p></article></main><footer><p>Copyright 2025</p></footer></body></html>`

	// Extract article
	article, err := ext.ExtractFromHTML(html, nil)
	if err != nil {
		t.Fatalf("Failed to extract article: %v", err)
	}

	// Check if content digests are added (look for data-content-digest attribute)
	if !strings.Contains(article.PlainContent, "data-content-digest") {
		t.Error("Expected content digests to be added")
	}

	// Check if node indexes are added (look for data-node-index attribute)
	if !strings.Contains(article.PlainContent, "data-node-index") {
		t.Error("Expected node indexes to be added")
	}
}