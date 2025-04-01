// Package readability provides a pure Go implementation of Mozilla's Readability.js
// for extracting the main content from web pages.
package readability

import (
	"github.com/PuerkitoBio/goquery"
)

// ContentType is maintained for API compatibility but no longer affects the extraction algorithm.
// Mozilla's Readability.js doesn't use content type detection, so we use a unified algorithm.
type ContentType int

// Content type constants - kept for API compatibility
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

// DetectContentType returns ContentTypeArticle for all content.
// This function is maintained for API compatibility but no longer performs detection.
// Mozilla's Readability.js doesn't classify content by type, so we use a unified algorithm.
func DetectContentType(doc *goquery.Document) ContentType {
	// Always return Article to use the standard algorithm for all content
	return ContentTypeArticle
}