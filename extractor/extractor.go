// Package extractor provides the main functionality for extracting readable content from HTML.
// It implements both a JavaScript-based extraction using Readability.js and a pure Go implementation.
package extractor

import (
	"fmt"
	"io"
	"time"

	"github.com/mrjoshuak/readabiligo/internal/extractors"
	"github.com/mrjoshuak/readabiligo/internal/javascript"
	"github.com/mrjoshuak/readabiligo/internal/simplifiers"
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

// WithReadability enables or disables Readability.js usage.
// When enabled (default), the extractor will attempt to use Readability.js for extraction
// if Node.js is available. If disabled or if Node.js is not available, it will fall back
// to the pure Go implementation.
func WithReadability(use bool) Option {
	return func(o *types.ExtractionOptions) {
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
// Node indexes track the position of elements in the original HTML document,
// which can be useful for mapping extracted content back to the source.
func WithNodeIndexes(enable bool) Option {
	return func(o *types.ExtractionOptions) {
		o.NodeIndexes = enable
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

		// If using Readability.js, call the JavaScript implementation
		if options.UseReadability {
			article, err = e.extractUsingJavaScript(html, options)
		} else {
			// Use pure Go implementation
			article, err = e.extractUsingPureGo(html, options)
		}

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

// extractUsingJavaScript extracts article content using Readability.js.
// It requires Node.js to be available on the system. If Node.js is not available,
// it falls back to the pure Go implementation.
func (e *articleExtractor) extractUsingJavaScript(html string, options *types.ExtractionOptions) (*types.Article, error) {
	// Check if Node.js is available
	if !javascript.HaveNode() {
		// Fall back to pure Go implementation
		return e.extractUsingPureGo(html, options)
	}

	// Extract article using Readability.js
	result, err := javascript.ExtractArticle(html)
	if err != nil {
		return nil, fmt.Errorf("failed to extract article using Readability.js: %w", err)
	}

	// Parse the date if available
	var date time.Time
	if result.Date != "" {
		date, err = time.Parse(time.RFC3339, result.Date)
		if err != nil {
			// Try other date formats
			date, err = time.Parse("2006-01-02T15:04:05Z", result.Date)
			if err != nil {
				// If we can't parse the date, use the current time
				date = time.Now()
			}
		}
	}

	// Create the article
	article := &types.Article{
		Title:        result.Title,
		Byline:       result.Byline,
		Date:         date,
		Content:      result.Content,
		PlainContent: "",
		PlainText:    []types.Block{},
	}

	// Generate plain content with content digests and node indexes if requested
	plainContent, err := simplifiers.PlainContent(article.Content, options.ContentDigests, options.NodeIndexes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate plain content: %w", err)
	}
	article.PlainContent = plainContent

	// Extract plain text blocks
	article.PlainText = extractors.ExtractTextBlocks(article.PlainContent, true)

	return article, nil
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
	// Initialize the article with empty values
	article := &types.Article{
		Title:        "",
		Byline:       "",
		Date:         time.Time{},
		Content:      "",
		PlainContent: "",
		PlainText:    []types.Block{},
	}

	// Extract title
	article.Title = extractors.ExtractTitle(html)

	// Extract date
	article.Date = extractors.ExtractDate(html)

	// Extract content using the HTML simplifier
	simpleTree, err := simplifiers.SimpleTreeFromHTMLString(html)
	if err != nil {
		return nil, fmt.Errorf("failed to create simple tree: %w", err)
	}
	article.Content = simpleTree.String()

	// Generate plain content with content digests and node indexes if requested
	plainContent, err := simplifiers.PlainContent(article.Content, options.ContentDigests, options.NodeIndexes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate plain content: %w", err)
	}
	article.PlainContent = plainContent

	// Extract plain text blocks
	article.PlainText = extractors.ExtractTextBlocks(article.PlainContent, false)

	return article, nil
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
