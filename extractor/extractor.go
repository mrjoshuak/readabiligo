// Package extractor provides the main functionality for extracting readable content from HTML.
// It implements a pure Go extraction algorithm based on Mozilla's Readability.js.
package extractor

import (
	"fmt"
	"io"
	"time"

	"github.com/mrjoshuak/readabiligo/internal/readability"
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
// This follows the functional options pattern for configuring the extractor.
type Option func(*types.ExtractionOptions)

// WithReadability formerly enabled or disabled Readability.js usage.
// DEPRECATED: This option is now a no-op. The JavaScript bridge has been removed.
// The pure Go implementation is now the only available approach.
//
// This option is kept for backward compatibility but has no effect.
// All extraction is performed using the pure Go implementation.
func WithReadability(use bool) Option {
	return func(o *types.ExtractionOptions) {
		// No-op - JavaScript support has been removed
		// The UseReadability field is still set for backward compatibility
		o.UseReadability = use
	}
}

// WithContentDigests enables or disables content digest attributes.
// Content digests are SHA256 hashes of the content, which can be used to
// identify and track content across different versions of the same document.
func WithContentDigests(enable bool) Option {
	return func(o *types.ExtractionOptions) {
		o.ContentDigests = enable
	}
}

// WithNodeIndexes enables or disables node index attributes.
// Node indexes are unique identifiers assigned to HTML elements during extraction,
// which can be used to track the source of specific content blocks.
func WithNodeIndexes(enable bool) Option {
	return func(o *types.ExtractionOptions) {
		o.NodeIndexes = enable
	}
}

// WithPreserveImportantLinks enables or disables the preservation of important links
// (like "More information..." links) from elements that would normally be removed
// like footers, navigation, and asides. This is a ReadabiliGo-specific feature
// that is not present in the original Readability.js algorithm.
func WithPreserveImportantLinks(enable bool) Option {
	return func(o *types.ExtractionOptions) {
		o.PreserveImportantLinks = enable
	}
}

// WithDetectContentType enables or disables automatic content type detection.
// When enabled, the extractor will analyze the document structure to determine
// the appropriate content type (Reference, Article, Technical, Error, Minimal)
// and apply optimized extraction rules for that content type.
func WithDetectContentType(enable bool) Option {
	return func(o *types.ExtractionOptions) {
		o.DetectContentType = enable
	}
}

// WithContentType sets a specific content type for extraction, bypassing automatic detection.
// This is useful when you know in advance what type of content you're extracting.
// Has no effect if WithDetectContentType is enabled.
func WithContentType(contentType types.ContentType) Option {
	return func(o *types.ExtractionOptions) {
		o.ContentType = contentType
	}
}

// WithMaxBufferSize sets the maximum buffer size for content processing.
// This limits the amount of memory used during extraction for very large documents.
func WithMaxBufferSize(size int) Option {
	return func(o *types.ExtractionOptions) {
		o.MaxBufferSize = size
	}
}

// WithTimeout sets the timeout duration for extraction.
// This prevents extraction from hanging indefinitely on problematic documents.
func WithTimeout(timeout time.Duration) Option {
	return func(o *types.ExtractionOptions) {
		o.Timeout = timeout
	}
}

// articleExtractor is the concrete implementation of the Extractor interface.
// It handles both JavaScript-based and pure Go extraction methods.
type articleExtractor struct {
	options types.ExtractionOptions
}

// ExtractFromHTML extracts article content from an HTML string.
// It returns an Article containing the extracted content and metadata.
// The extraction process can be configured using the provided options.
func (e *articleExtractor) ExtractFromHTML(html string, options *types.ExtractionOptions) (*types.Article, error) {
	if options == nil {
		options = &e.options
	}

	// Create a channel for the result
	resultCh := make(chan struct {
		article *types.Article
		err     error
	})

	// Start the extraction in a goroutine
	go func() {
		var article *types.Article
		var err error

		// If using Readability.js option is set (deprecated), provide a warning but continue with pure Go
		if options.UseReadability {
			// Log warning in debug mode here if needed
			// Silently fall back to pure Go implementation
		}
		
		// Use pure Go implementation
		article, err = e.extractUsingPureGo(html, options)

		// Send the result to the channel
		resultCh <- struct {
			article *types.Article
			err     error
		}{article, err}
	}()

	// Wait for the result or timeout
	select {
	case result := <-resultCh:
		return result.article, result.err
	case <-time.After(options.Timeout):
		return nil, fmt.Errorf("extraction timed out after %v", options.Timeout)
	}
}

// ExtractFromReader extracts article content from an io.Reader.
// It reads the entire content from the reader and passes it to ExtractFromHTML.
func (e *articleExtractor) ExtractFromReader(r io.Reader, options *types.ExtractionOptions) (*types.Article, error) {
	if options == nil {
		options = &e.options
	}

	// Read the entire content from the reader
	html, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return e.ExtractFromHTML(string(html), options)
}

// extractUsingPureGo implements the pure Go extraction logic.
// This is used when Readability.js is not available or when explicitly requested.
func (e *articleExtractor) extractUsingPureGo(html string, options *types.ExtractionOptions) (*types.Article, error) {
	// Use our pure Go Readability implementation
	return readability.ExtractFromHTML(html, options)
}

// New creates a new Extractor instance with the provided options.
// It returns an implementation of the Extractor interface that can be used
// to extract article content from HTML.
func New(opts ...Option) Extractor {
	options := types.DefaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	return &articleExtractor{
		options: options,
	}
}