// Package readabiligo provides a Go implementation of Mozilla's Readability.js library
// for extracting the main content from HTML pages.
package readabiligo

import "time"

// Version information for the ReadabiliGo library.
const (
	Version = "0.3.0" // Updated version for JavaScript bridge removal
	Name    = "ReadabiliGo"
)

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

// ContentType represents the type of content in a document.
// This type is maintained for backward compatibility but no longer affects extraction.
// The extraction now uses Mozilla's original unified algorithm for all content types.
// Deprecated: This type no longer affects extraction behavior.
type ContentType int

// Content type constants - maintained for backward compatibility
// These constants no longer affect the extraction behavior.
// Deprecated: These constants no longer affect extraction behavior.
const (
	ContentTypeUnknown ContentType = iota
	ContentTypeReference  // Wikipedia, documentation
	ContentTypeArticle    // News, blog posts
	ContentTypeTechnical  // Code examples, tech blogs
	ContentTypeError      // Error pages, 404s
	ContentTypeMinimal    // Login pages, etc.
	ContentTypePaywall    // Paywalled content
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
	case ContentTypePaywall:
		return "Paywall"
	default:
		return "Unknown"
	}
}

// ExtractionOptions configures the article extraction process.
// It controls whether to include content digests and node indexes,
// and sets limits on buffer size and extraction timeout.
type ExtractionOptions struct {
	ContentDigests       bool          // Add content digest attributes
	NodeIndexes          bool          // Add node index attributes
	MaxBufferSize        int           // Maximum buffer size for content processing
	Timeout              time.Duration // Timeout for extraction process
	PreserveImportantLinks bool        // Preserve important links in cleaned elements (like "More information...")
	DetectContentType    bool          // Deprecated: No longer has any effect, maintained for backward compatibility
	ContentType          ContentType   // Deprecated: No longer has any effect, maintained for backward compatibility
}

// DefaultOptions returns the default extraction options.
// By default, the pure Go implementation is used, content digests and node indexes
// are disabled, buffer size is limited to 1MB, and timeout is set to 30 seconds.
// Important link preservation is disabled by default to match ReadabiliPy behavior.
// Note: Content type detection and content type settings no longer have any effect,
// as the implementation now uses Mozilla's unified algorithm for all content.
func DefaultOptions() ExtractionOptions {
	return ExtractionOptions{
		ContentDigests:       false,
		NodeIndexes:          false,
		MaxBufferSize:        1024 * 1024, // 1MB
		Timeout:              time.Second * 30,
		PreserveImportantLinks: false, // Default to false to match ReadabiliPy behavior
		DetectContentType:    false,   // No-op but set to false for clarity
		ContentType:          ContentTypeArticle, // No-op but set to Article for clarity
	}
}

// BuildInfo contains version and build information for the ReadabiliGo library.
// It includes the version number, name, and Go version used to build the library.
type BuildInfo struct {
	Version   string
	Name      string
	GoVersion string
}

// GetBuildInfo returns the current version information for the ReadabiliGo library.
// This is useful for displaying version information in logs or help output.
func GetBuildInfo() BuildInfo {
	return BuildInfo{
		Version:   Version,
		Name:      Name,
		GoVersion: "go1.22", // TODO: Make this dynamic
	}
}