// Package types provides the core data structures for the ReadabiliGo library.
package types

import "time"

// Block represents a block of text with optional metadata.
// It is used to store paragraphs of plain text extracted from an article,
// with optional node index information for tracking the source HTML elements.
type Block struct {
	Text      string `json:"text"`
	NodeIndex string `json:"node_index,omitempty"`
}

// Article represents the extracted content and metadata from a webpage.
// It contains the article title, byline, publication date, HTML content,
// simplified HTML content, and plain text paragraphs.
type Article struct {
	Title        string    `json:"title"`
	Byline       string    `json:"byline"`
	Date         time.Time `json:"date"`
	Content      string    `json:"content"`
	PlainContent string    `json:"plain_content"`
	PlainText    []Block   `json:"plain_text"`
}

// ExtractionOptions configures the article extraction process.
// It controls whether to use Readability.js, whether to include content digests
// and node indexes, and sets limits on buffer size and extraction timeout.
type ExtractionOptions struct {
	UseReadability bool          // Use Readability.js instead of pure Go
	ContentDigests bool          // Add content digest attributes
	NodeIndexes    bool          // Add node index attributes
	MaxBufferSize  int           // Maximum buffer size for content processing
	Timeout        time.Duration // Timeout for extraction process
}

// DefaultOptions returns the default extraction options.
// By default, Readability.js is used if available, content digests and node indexes
// are disabled, buffer size is limited to 1MB, and timeout is set to 30 seconds.
func DefaultOptions() ExtractionOptions {
	return ExtractionOptions{
		UseReadability: true,
		ContentDigests: false,
		NodeIndexes:    false,
		MaxBufferSize:  1024 * 1024, // 1MB
		Timeout:        time.Second * 30,
	}
}
