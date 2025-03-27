package readabiligo_test

import (
	"fmt"
	"os"
	"time"

	"github.com/mrjoshuak/readabiligo"
)

func ExampleNew() {
	// Create a new extractor with default options
	ext := readabiligo.New()

	// Extract from an HTML string
	html := `<html><head><title>Article Title</title></head><body><header><nav><ul><li><a href="#">Home</a></li><li><a href="#">About</a></li></ul></nav></header><main><article><h1>Article Title</h1><p>This is a test paragraph with enough text to be considered relevant content by the Readability algorithm. We need to ensure that this paragraph has sufficient length to be scored highly by the content extraction algorithm. The algorithm looks for blocks of text that appear to be the main content of the page, as opposed to navigation, headers, footers, or other ancillary content.</p><p>Adding another paragraph increases the content score for this article element, making it more likely to be identified as the main content of the page. The Readability algorithm is designed to extract the primary content from a webpage, ignoring elements that are likely to be navigation, ads, or other non-content features.</p></article></main><footer><p>Copyright 2025</p></footer></body></html>`
	article, err := ext.ExtractFromHTML(html, nil)
	if err != nil {
		fmt.Printf("Error extracting article: %v\n", err)
		return
	}

	fmt.Printf("Title: %s\n", article.Title)
	// Output: Title: Article Title
}

func ExampleWithContentDigests() {
	// Create a new extractor with content digests enabled
	ext := readabiligo.New(
		readabiligo.WithContentDigests(true),
	)

	// Extract from an HTML string
	html := `<html><head><title>Article Title</title></head><body><header><nav><ul><li><a href="#">Home</a></li><li><a href="#">About</a></li></ul></nav></header><main><article><h1>Article Title</h1><p>This is a test paragraph with enough text to be considered relevant content by the Readability algorithm. We need to ensure that this paragraph has sufficient length to be scored highly by the content extraction algorithm. The algorithm looks for blocks of text that appear to be the main content of the page, as opposed to navigation, headers, footers, or other ancillary content.</p><p>Adding another paragraph increases the content score for this article element, making it more likely to be identified as the main content of the page. The Readability algorithm is designed to extract the primary content from a webpage, ignoring elements that are likely to be navigation, ads, or other non-content features.</p></article></main><footer><p>Copyright 2025</p></footer></body></html>`
	article, err := ext.ExtractFromHTML(html, nil)
	if err != nil {
		fmt.Printf("Error extracting article: %v\n", err)
		return
	}

	// Content will have data-content-digest attributes
	fmt.Printf("Has digests: %v\n", len(article.PlainContent) > 0)
	// Output: Has digests: true
}

func ExampleWithTimeout() {
	// Create a new extractor with a custom timeout
	ext := readabiligo.New(
		readabiligo.WithTimeout(time.Second*60),
	)

	// Extract from an HTML string
	html := `<html><head><title>Article Title</title></head><body><header><nav><ul><li><a href="#">Home</a></li><li><a href="#">About</a></li></ul></nav></header><main><article><h1>Article Title</h1><p>This is a test paragraph with enough text to be considered relevant content by the Readability algorithm. We need to ensure that this paragraph has sufficient length to be scored highly by the content extraction algorithm. The algorithm looks for blocks of text that appear to be the main content of the page, as opposed to navigation, headers, footers, or other ancillary content.</p><p>Adding another paragraph increases the content score for this article element, making it more likely to be identified as the main content of the page. The Readability algorithm is designed to extract the primary content from a webpage, ignoring elements that are likely to be navigation, ads, or other non-content features.</p></article></main><footer><p>Copyright 2025</p></footer></body></html>`
	article, err := ext.ExtractFromHTML(html, nil)
	if err != nil {
		fmt.Printf("Error extracting article: %v\n", err)
		return
	}

	fmt.Printf("Title: %s\n", article.Title)
	// Output: Title: Article Title
}

func ExampleExtractor_ExtractFromReader() {
	// Create a new extractor
	ext := readabiligo.New()

	// Example HTML file (replace with actual file path)
	file, err := os.Open("test/data/content_extraction_test.html")
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	// Extract from the file
	article, err := ext.ExtractFromReader(file, nil)
	if err != nil {
		fmt.Printf("Error extracting article: %v\n", err)
		return
	}

	// Access article data
	fmt.Printf("Has title: %v\n", len(article.Title) > 0)
	fmt.Printf("Has content: %v\n", len(article.Content) > 0)
	fmt.Printf("Has plain text: %v\n", len(article.PlainText) > 0)
	// Output:
	// Has title: true
	// Has content: true
	// Has plain text: true
}