// Package readabiligo provides a Go implementation of Mozilla's Readability.js library
// for extracting the main content from HTML pages. It is a port of ReadabiliPy by
// The Alan Turing Institute with a pure Go implementation.
//
// Usage:
//
//	import "github.com/mrjoshuak/readabiligo"
//
//	// Create extractor
//	extractor := readabiligo.New()
//
//	// Extract from HTML
//	article, err := extractor.ExtractFromHTML(htmlString, nil)
//
//	// Use article data
//	fmt.Println(article.Title)
//	fmt.Println(article.Content)
package readabiligo

import (
	"io"
	"time"

	"github.com/mrjoshuak/readabiligo/extractor"
	"github.com/mrjoshuak/readabiligo/types"
)

// Extractor defines the interface for article extraction.
// It provides methods to extract article content from HTML strings or io.Readers.
type Extractor interface {
	// ExtractFromHTML extracts article content from an HTML string
	ExtractFromHTML(html string, options *types.ExtractionOptions) (*types.Article, error)

	// ExtractFromReader extracts article content from an io.Reader
	ExtractFromReader(r io.Reader, options *types.ExtractionOptions) (*types.Article, error)
}

// Option represents a function that modifies ExtractionOptions.
type Option = extractor.Option

// WithReadability enables or disables Readability.js usage.
// When enabled (default), the extractor will attempt to use Readability.js for extraction
// if Node.js is available. If disabled or if Node.js is not available, it will fall back
// to the pure Go implementation.
func WithReadability(use bool) Option {
	return extractor.WithReadability(use)
}

// WithContentDigests enables or disables content digest attributes.
// Content digests are SHA256 hashes of the content, which can be used to
// identify and track content across different versions of the same document.
func WithContentDigests(enable bool) Option {
	return extractor.WithContentDigests(enable)
}

// WithNodeIndexes enables or disables node index attributes.
// Node indexes track the position of elements in the original HTML document,
// which can be useful for mapping extracted content back to the source.
func WithNodeIndexes(enable bool) Option {
	return extractor.WithNodeIndexes(enable)
}

// WithMaxBufferSize sets the maximum buffer size for content processing.
// This limits the amount of memory used during extraction for very large documents.
func WithMaxBufferSize(size int) Option {
	return extractor.WithMaxBufferSize(size)
}

// WithTimeout sets the timeout duration for extraction.
// This prevents extraction from hanging indefinitely on problematic documents.
func WithTimeout(timeout time.Duration) Option {
	return extractor.WithTimeout(timeout)
}

// New creates a new Extractor instance with the provided options.
// It returns an implementation of the Extractor interface that can be used
// to extract article content from HTML.
//
// Example:
//
//	extractor := readabiligo.New(
//	    readabiligo.WithContentDigests(true),
//	    readabiligo.WithTimeout(time.Second*60),
//	)
func New(opts ...Option) Extractor {
	return extractor.New(opts...)
}