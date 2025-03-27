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
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mrjoshuak/readabiligo/internal/simplifiers"
	"github.com/mrjoshuak/readabiligo/types"
)

// ExtractFromHTML extracts readable content from HTML using pure Go Readability
// This function adapts our implementation to match the expected interface
func ExtractFromHTML(html string, options *types.ExtractionOptions) (*types.Article, error) {
	// Parse HTML using Readability algorithm
	article, err := Parse(html)
	if err != nil {
		return nil, err
	}

	// Create the article structure
	result := &types.Article{
		Title:        article.Title,
		Byline:       article.Byline,
		Date:         article.Date,
		Content:      article.Content,
		PlainContent: "",
		PlainText:    []types.Block{},
	}

	// Set default date if needed
	if result.Date.IsZero() {
		result.Date = time.Now()
	}

	// Generate plain content with content digests and node indexes if requested
	plainContent, err := simplifiers.PlainContent(article.Content, options.ContentDigests, options.NodeIndexes)
	if err != nil {
		return nil, err
	}
	result.PlainContent = plainContent

	// Extract plain text blocks
	result.PlainText = extractTextBlocks(result.PlainContent)

	return result, nil
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