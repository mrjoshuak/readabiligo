package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/mrjoshuak/readabiligo/extractor"
	"github.com/mrjoshuak/readabiligo/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContentTypeOptimizedExtraction tests content type-specific extraction on real-world examples
func TestContentTypeOptimizedExtraction(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping content type optimized test in short mode")
	}

	// Real-world test cases with expected content types
	testCases := []struct {
		name         string
		htmlFile     string
		contentType  types.ContentType
		description  string
	}{
		{"Wikipedia", "wikipedia_go.html", types.ContentTypeReference, "Go programming language Wikipedia page"},
		{"TechBlog", "go_blog.html", types.ContentTypeTechnical, "Go blog with technical content"},
		{"ErrorPage", "guardian.html", types.ContentTypeError, "Guardian 404 page"},
		{"NewsArticle", "nytimes.html", types.ContentTypeArticle, "News article from NYTimes"},
	}

	// Get the list of HTML files in the real_world directory
	realWorldDir := filepath.Join("data", "real_world")
	
	// For each test case
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get the full path to the file
			filePath := filepath.Join(realWorldDir, tc.htmlFile)
			
			// Skip if file doesn't exist
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Skipf("HTML file %s does not exist. Run download_real_world_examples.sh first.", filePath)
				return
			}
			
			// Read the HTML file
			htmlContent, err := os.ReadFile(filePath)
			require.NoError(t, err)
			
			// 1. Auto-detect content type
			autoDetectExt := extractor.New(
				extractor.WithDetectContentType(true),
			)
			autoDetectArticle, err := autoDetectExt.ExtractFromHTML(string(htmlContent), nil)
			require.NoError(t, err)
			
			// 2. Force specific content type
			forcedTypeExt := extractor.New(
				extractor.WithDetectContentType(false),
				extractor.WithContentType(tc.contentType),
			)
			forcedTypeArticle, err := forcedTypeExt.ExtractFromHTML(string(htmlContent), nil)
			require.NoError(t, err)
			
			// 3. Use opposite content type for comparison
			var oppositeType types.ContentType
			switch tc.contentType {
			case types.ContentTypeReference:
				oppositeType = types.ContentTypeArticle
			case types.ContentTypeTechnical:
				oppositeType = types.ContentTypeArticle
			case types.ContentTypeError:
				oppositeType = types.ContentTypeArticle
			case types.ContentTypeArticle:
				oppositeType = types.ContentTypeReference
			default:
				oppositeType = types.ContentTypeArticle
			}
			
			oppositeTypeExt := extractor.New(
				extractor.WithDetectContentType(false),
				extractor.WithContentType(oppositeType),
			)
			oppositeTypeArticle, err := oppositeTypeExt.ExtractFromHTML(string(htmlContent), nil)
			require.NoError(t, err)
			
			// Log detected and forced content types
			t.Logf("Auto-detected content type: %s", autoDetectArticle.ContentType.String())
			t.Logf("Forced content type: %s", forcedTypeArticle.ContentType.String())
			t.Logf("Opposite content type: %s", oppositeTypeArticle.ContentType.String())
			
			// Compare content between optimal and opposite content types
			optimalContent := forcedTypeArticle.Content
			oppositeContent := oppositeTypeArticle.Content
			
			// Compare DOM structure using goquery
			optimalDoc, err := goquery.NewDocumentFromReader(strings.NewReader(optimalContent))
			require.NoError(t, err)
			
			oppositeDoc, err := goquery.NewDocumentFromReader(strings.NewReader(oppositeContent))
			require.NoError(t, err)
			
			// Count HTML elements to compare structure preservation differences
			countElements := func(doc *goquery.Document, selector string, description string) int {
				count := doc.Find(selector).Length()
				return count
			}
			
			// Check for key elements
			optimalPCount := countElements(optimalDoc, "p", "Paragraph")
			oppositePCount := countElements(oppositeDoc, "p", "Paragraph")
			
			optimalLinkCount := countElements(optimalDoc, "a", "Link")
			oppositeLinkCount := countElements(oppositeDoc, "a", "Link")
			
			optimalHeadingCount := countElements(optimalDoc, "h1, h2, h3, h4, h5, h6", "Heading")
			oppositeHeadingCount := countElements(oppositeDoc, "h1, h2, h3, h4, h5, h6", "Heading")
			
			optimalListCount := countElements(optimalDoc, "ul, ol", "List")
			oppositeListCount := countElements(oppositeDoc, "ul, ol", "List")
			
			optimalListItemCount := countElements(optimalDoc, "li", "List item")
			oppositeListItemCount := countElements(oppositeDoc, "li", "List item")
			
			optimalNavCount := countElements(optimalDoc, "nav, .nav, .navigation, .menu", "Navigation")
			oppositeNavCount := countElements(oppositeDoc, "nav, .nav, .navigation, .menu", "Navigation")
			
			// Log element counts for comparison
			t.Logf("Paragraphs - Optimal: %d, Opposite: %d, Ratio: %.2f", 
				optimalPCount, oppositePCount, safeRatio(optimalPCount, oppositePCount))
			
			t.Logf("Links - Optimal: %d, Opposite: %d, Ratio: %.2f", 
				optimalLinkCount, oppositeLinkCount, safeRatio(optimalLinkCount, oppositeLinkCount))
			
			t.Logf("Headings - Optimal: %d, Opposite: %d, Ratio: %.2f", 
				optimalHeadingCount, oppositeHeadingCount, safeRatio(optimalHeadingCount, oppositeHeadingCount))
			
			t.Logf("Lists - Optimal: %d, Opposite: %d, Ratio: %.2f", 
				optimalListCount, oppositeListCount, safeRatio(optimalListCount, oppositeListCount))
			
			t.Logf("List items - Optimal: %d, Opposite: %d, Ratio: %.2f", 
				optimalListItemCount, oppositeListItemCount, safeRatio(optimalListItemCount, oppositeListItemCount))
			
			t.Logf("Navigation elements - Optimal: %d, Opposite: %d, Ratio: %.2f", 
				optimalNavCount, oppositeNavCount, safeRatio(optimalNavCount, oppositeNavCount))
			
			// Check text content length
			optimalText := optimalDoc.Text()
			oppositeText := oppositeDoc.Text()
			
			t.Logf("Text length - Optimal: %d, Opposite: %d, Ratio: %.2f", 
				len(optimalText), len(oppositeText), safeRatio(len(optimalText), len(oppositeText)))
			
			// Perform content-type specific assertions
			switch tc.contentType {
			case types.ContentTypeReference:
				// Reference content should preserve more structure
				assert.True(t, optimalHeadingCount >= oppositeHeadingCount, 
					"Reference content should preserve headings better")
				assert.True(t, optimalListItemCount >= oppositeListItemCount, 
					"Reference content should preserve list items better")
				
			case types.ContentTypeError:
				// Error pages should have less navigation
				assert.True(t, optimalNavCount <= oppositeNavCount, 
					"Error page mode should remove more navigation elements")
				
			case types.ContentTypeTechnical:
				// Technical content should preserve code blocks (not easily testable)
				// but should have better structure preservation than generic article
				assert.True(t, optimalHeadingCount >= oppositeHeadingCount, 
					"Technical content should preserve heading structure")
			}
		})
	}
}

// safeRatio calculates a ratio between two numbers, handling divide by zero
func safeRatio(a, b int) float64 {
	if b == 0 {
		if a == 0 {
			return 1.0 // Both zero means equal
		}
		return float64(a) // Avoid division by zero
	}
	return float64(a) / float64(b)
}