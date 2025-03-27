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
// simplified HTML content, plain text paragraphs, and detected content type.
type Article struct {
	Title        string      `json:"title"`
	Byline       string      `json:"byline"`
	Date         time.Time   `json:"date"`
	Content      string      `json:"content"`
	PlainContent string      `json:"plain_content"`
	PlainText    []Block     `json:"plain_text"`
	ContentType  ContentType `json:"content_type"`
}

// ContentType represents the type of content in a document
type ContentType int

// Content type constants
const (
	ContentTypeUnknown ContentType = iota
	ContentTypeReference  // Wikipedia, documentation
	ContentTypeArticle    // News, blog posts
	ContentTypeTechnical  // Code examples, tech blogs
	ContentTypeError      // Error pages, 404s
	ContentTypeMinimal    // Login pages, etc.
)

// String returns a string representation of the content type
func (ct ContentType) String() string {
	switch ct {
	case ContentTypeReference:
		return "Reference"
	case ContentTypeArticle:
		return "Article"
	case ContentTypeTechnical:
		return "Technical"
	case ContentTypeError:
		return "Error"
	case ContentTypeMinimal:
		return "Minimal"
	default:
		return "Unknown"
	}
}

// ExtractionOptions configures the article extraction process.
// It controls whether to include content digests and node indexes,
// and sets limits on buffer size and extraction timeout.
type ExtractionOptions struct {
	UseReadability       bool          // DEPRECATED: No effect - kept for backward compatibility
	ContentDigests       bool          // Add content digest attributes
	NodeIndexes          bool          // Add node index attributes
	MaxBufferSize        int           // Maximum buffer size for content processing
	Timeout              time.Duration // Timeout for extraction process
	PreserveImportantLinks bool        // Preserve important links in cleaned elements (like "More information...")
	DetectContentType    bool          // Whether to enable content type detection
	ContentType          ContentType   // Content type to use for extraction (or auto-detected if DetectContentType is true)
}

// DefaultOptions returns the default extraction options.
// By default, the pure Go implementation is used, content digests and node indexes
// are disabled, buffer size is limited to 1MB, and timeout is set to 30 seconds.
// Important link preservation is disabled by default to match ReadabiliPy behavior.
// Content type detection is enabled by default.
func DefaultOptions() ExtractionOptions {
	return ExtractionOptions{
		UseReadability:       false, // Now uses pure Go implementation by default
		ContentDigests:       false,
		NodeIndexes:          false,
		MaxBufferSize:        1024 * 1024, // 1MB
		Timeout:              time.Second * 30,
		PreserveImportantLinks: false, // Default to false to match ReadabiliPy behavior
		DetectContentType:    true,    // Enable content type detection by default
		ContentType:          ContentTypeUnknown, // Auto-detect by default
	}
}
