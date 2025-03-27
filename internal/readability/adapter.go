// Package readability provides a pure Go implementation of Mozilla's Readability.js
// algorithm for extracting article content from web pages.
//
// This implementation follows the same content extraction logic as the original
// JavaScript implementation, including scoring elements based on content quality,
// handling special cases, and cleaning up the final article content.
//
// Key features:
// - No JavaScript dependencies (100% Go)
// - Compatible with Mozilla's Readability algorithm
// - Proper handling of important links, headings, and navigation elements
// - Built-in adapters for integration with the main extractor package
package readability

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/mrjoshuak/readabiligo/internal/simplifiers"
	"github.com/mrjoshuak/readabiligo/types"
)

// ExtractFromHTML extracts readable content from HTML using pure Go Readability
// This function adapts our implementation to match the expected interface
func ExtractFromHTML(html string, options *types.ExtractionOptions) (*types.Article, error) {
	// Set options for Readability parser
	opts := defaultReadabilityOptions()
	if options != nil {
		// Apply relevant options from the extraction options
		if options.PreserveImportantLinks {
			opts.PreserveImportantLinks = true
		}
		
		// Apply content type detection options
		opts.DetectContentType = options.DetectContentType
		opts.ContentType = ContentType(options.ContentType)
		
		// Add any other option mappings here in the future
	}

	// Parse HTML using Readability algorithm
	readabilityArticle, err := ParseHTML(html, &opts)
	if err != nil {
		return nil, WrapExtractionError(err, "ExtractFromHTML", "failed to parse HTML content")
	}

	// Convert to standard article format
	result := readabilityArticle.ToStandardArticle()
	
	// Set the detected content type in the result
	result.ContentType = types.ContentType(readabilityArticle.ContentType)

	// Generate plain content with content digests and node indexes if requested
	plainContent, err := simplifiers.PlainContent(result.Content, options.ContentDigests, options.NodeIndexes)
	if err != nil {
		return nil, WrapExtractionError(err, "ExtractFromHTML", "failed to generate plain content")
	}
	result.PlainContent = plainContent

	// Extract plain text blocks
	result.PlainText = extractTextBlocks(result.PlainContent)

	return result, nil
}

// ParseHTML parses HTML content and returns a ReadabilityArticle
func ParseHTML(html string, opts *ReadabilityOptions) (*ReadabilityArticle, error) {
	r, err := NewFromHTML(html, opts)
	if err != nil {
		return nil, err
	}
	
	return r.Parse()
}

// ToStandardArticle converts a ReadabilityArticle to the standard Article type
func (ra *ReadabilityArticle) ToStandardArticle() *types.Article {
	article := &types.Article{
		Title:        ra.Title,
		Byline:       ra.Byline,
		Content:      ra.Content,
		ContentType:  types.ContentType(ra.ContentType),
	}
	
	// Set publication date if available
	if !ra.Date.IsZero() {
		article.Date = ra.Date
	}
	
	return article
}

// extractTextBlocks creates a slice of Block objects from HTML content
func extractTextBlocks(html string) []types.Block {
	r, err := NewFromHTML(html, nil)
	if err != nil {
		return []types.Block{}
	}

	blocks := []types.Block{}
	r.doc.Find("p, li").Each(func(i int, s *goquery.Selection) {
		text := getInnerText(s, true)
		if text == "" {
			return
		}

		// Create block with text
		block := types.Block{
			Text: text,
		}

		// Add node index if available
		if nodeIndex, exists := s.Attr("data-node-index"); exists {
			block.NodeIndex = nodeIndex
		}

		blocks = append(blocks, block)
	})

	return blocks
}