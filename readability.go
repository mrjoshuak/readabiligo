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
	"fmt"
	"io"
	"time"

	"github.com/mrjoshuak/readabiligo/internal/readability"
)

// Extractor defines the interface for article extraction.
// It provides methods to extract article content from HTML strings or io.Readers.
type Extractor interface {
	// ExtractFromHTML extracts article content from an HTML string
	ExtractFromHTML(html string, options *ExtractionOptions) (*Article, error)

	// ExtractFromReader extracts article content from an io.Reader
	ExtractFromReader(r io.Reader, options *ExtractionOptions) (*Article, error)
}

// Option represents a function that modifies ExtractionOptions.
// This follows the functional options pattern for configuring the extractor.
type Option func(*ExtractionOptions)


// WithContentDigests enables or disables content digest attributes.
// Content digests are SHA256 hashes of the content, which can be used to
// identify and track content across different versions of the same document.
func WithContentDigests(enable bool) Option {
	return func(o *ExtractionOptions) {
		o.ContentDigests = enable
	}
}

// WithNodeIndexes enables or disables node index attributes.
// Node indexes are unique identifiers assigned to HTML elements during extraction,
// which can be used to track the source of specific content blocks.
func WithNodeIndexes(enable bool) Option {
	return func(o *ExtractionOptions) {
		o.NodeIndexes = enable
	}
}

// WithPreserveImportantLinks enables or disables the preservation of important links
// (like "More information..." links) from elements that would normally be removed
// like footers, navigation, and asides. This is a ReadabiliGo-specific feature
// that is not present in the original Readability.js algorithm.
func WithPreserveImportantLinks(enable bool) Option {
	return func(o *ExtractionOptions) {
		o.PreserveImportantLinks = enable
	}
}

// WithDetectContentType is maintained for backward compatibility but does nothing.
// The content type detection has been removed to follow Mozilla's Readability.js algorithm,
// which uses a unified approach for all content types.
// Deprecated: This option no longer has any effect.
func WithDetectContentType(enable bool) Option {
	return func(o *ExtractionOptions) {
		// No-op for backward compatibility
		o.DetectContentType = enable
	}
}

// WithContentType is maintained for backward compatibility but does nothing.
// The content type specialization has been removed to follow Mozilla's Readability.js algorithm,
// which uses a unified approach for all content types.
// Deprecated: This option no longer has any effect.
func WithContentType(contentType ContentType) Option {
	return func(o *ExtractionOptions) {
		// No-op for backward compatibility
		o.ContentType = contentType
	}
}

// WithMaxBufferSize sets the maximum buffer size for content processing.
// This limits the amount of memory used during extraction for very large documents.
func WithMaxBufferSize(size int) Option {
	return func(o *ExtractionOptions) {
		o.MaxBufferSize = size
	}
}

// WithTimeout sets the timeout duration for extraction.
// This prevents extraction from hanging indefinitely on problematic documents.
func WithTimeout(timeout time.Duration) Option {
	return func(o *ExtractionOptions) {
		o.Timeout = timeout
	}
}

// articleExtractor is the concrete implementation of the Extractor interface.
// It handles the pure Go extraction method.
type articleExtractor struct {
	options ExtractionOptions
}

// ExtractFromHTML extracts article content from an HTML string.
// It returns an Article containing the extracted content and metadata.
// The extraction process can be configured using the provided options.
func (e *articleExtractor) ExtractFromHTML(html string, options *ExtractionOptions) (*Article, error) {
	if options == nil {
		options = &e.options
	}

	// Create a channel for the result
	resultCh := make(chan struct {
		article *Article
		err     error
	})

	// Start the extraction in a goroutine
	go func() {
		var article *Article
		var err error

		
		// Use pure Go implementation
		article, err = e.extractUsingPureGo(html, options)

		// Send the result to the channel
		resultCh <- struct {
			article *Article
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
func (e *articleExtractor) ExtractFromReader(r io.Reader, options *ExtractionOptions) (*Article, error) {
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
func (e *articleExtractor) extractUsingPureGo(html string, options *ExtractionOptions) (*Article, error) {
	// Convert our options to internal options
	internalOptions := &readability.ExtractionOptions{
		ContentDigests:        options.ContentDigests,
		NodeIndexes:           options.NodeIndexes,
		MaxBufferSize:         options.MaxBufferSize,
		Timeout:               int(options.Timeout.Seconds()),
		PreserveImportantLinks: options.PreserveImportantLinks,
		DetectContentType:     options.DetectContentType,
		ContentType:           readability.ContentType(options.ContentType),
	}

	// Use our pure Go Readability implementation
	internalArticle, err := readability.ExtractFromHTML(html, internalOptions)
	if err != nil {
		return nil, err
	}

	// Convert internal article to our public type
	article := &Article{
		Title:        internalArticle.Title,
		Byline:       internalArticle.Byline,
		Content:      internalArticle.Content,
		PlainContent: internalArticle.PlainContent,
		ContentType:  ContentType(internalArticle.ContentType),
	}

	// Convert internal blocks to our blocks
	article.PlainText = make([]Block, len(internalArticle.PlainText))
	for i, block := range internalArticle.PlainText {
		article.PlainText[i] = Block{
			Text:      block.Text,
			NodeIndex: block.NodeIndex,
		}
	}

	// Set date if available
	if date, ok := internalArticle.Date.(time.Time); ok {
		article.Date = date
	}

	return article, nil
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
	options := DefaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	return &articleExtractor{
		options: options,
	}
}