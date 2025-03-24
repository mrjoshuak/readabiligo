package extractors

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/mrjoshuak/readabiligo/internal/simplifiers"
	"github.com/mrjoshuak/readabiligo/types"
)

// ExtractTextBlocks extracts plain text blocks from HTML content
func ExtractTextBlocks(html string, useReadability bool) []types.Block {
	if useReadability {
		return extractTextBlocksJS(html)
	}
	return extractTextBlocksAsPlainText(html)
}

// extractTextBlocksJS extracts text blocks from HTML content using JavaScript-like approach
func extractTextBlocksJS(html string) []types.Block {
	// Load article as DOM
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil
	}

	// Select all text blocks
	var textBlocks []types.Block
	doc.Find("*").Each(func(_ int, s *goquery.Selection) {
		// Skip elements with no text
		if s.Text() == "" {
			return
		}

		// Get the text content
		text := s.Text()
		text = simplifiers.NormalizeText(text)
		if text == "" {
			return
		}

		// Create a block with the text
		block := types.Block{
			Text: text,
		}

		// Add node index if available
		if nodeIndex, exists := s.Attr("data-node-index"); exists {
			block.NodeIndex = nodeIndex
		}

		textBlocks = append(textBlocks, block)
	})

	return textBlocks
}

// extractTextBlocksAsPlainText extracts text blocks from HTML content as plain text
func extractTextBlocksAsPlainText(html string) []types.Block {
	// Load article as DOM
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil
	}

	// Process lists - prefix text in all list items with "* " and make lists paragraphs
	doc.Find("ul, ol").Each(func(_ int, list *goquery.Selection) {
		var plainItems strings.Builder
		list.Find("li").Each(func(_ int, li *goquery.Selection) {
			text := simplifiers.NormalizeText(li.Text())
			if text != "" {
				plainItems.WriteString("* ")
				plainItems.WriteString(text)
				plainItems.WriteString(", ")
			}
		})

		// Replace the list with a paragraph containing the plain items
		list.ReplaceWithHtml("<p>" + plainItems.String() + "</p>")
	})

	// Select all text blocks
	var textBlocks []types.Block
	doc.Find("*").Each(func(_ int, s *goquery.Selection) {
		// Skip elements with no text
		if s.Text() == "" {
			return
		}

		// Get the text content
		text := s.Text()
		text = simplifiers.NormalizeText(text)
		if text == "" {
			return
		}

		// Create a block with the text
		block := types.Block{
			Text: text,
		}

		// Add node index if available
		if nodeIndex, exists := s.Attr("data-node-index"); exists {
			block.NodeIndex = nodeIndex
		}

		textBlocks = append(textBlocks, block)
	})

	return textBlocks
}
