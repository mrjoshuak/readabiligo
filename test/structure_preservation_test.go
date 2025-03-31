package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/mrjoshuak/readabiligo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStructurePreservation verifies that our structure preservation improvements work
func TestStructurePreservation(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping structure preservation test in short mode")
	}

	// Create list of test files - we'll use the reference test files
	testFiles := []string{
		"addictinginfo.html",
		"conservativehq.html",
		"davidwolfe.html",
		"list_items.html",
	}

	refDir := filepath.Join("data", "reference")
	htmlDir := filepath.Join(refDir, "html")

	// Skip if reference directories don't exist
	if _, err := os.Stat(htmlDir); os.IsNotExist(err) {
		t.Skipf("Reference HTML directory %s does not exist. Run download_test_data.sh first.", htmlDir)
	}

	// Run tests for each file
	for _, htmlFile := range testFiles {
		htmlPath := filepath.Join(htmlDir, htmlFile)

		// Skip if file doesn't exist
		if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
			t.Logf("Skipping test for %s: HTML file not found", htmlFile)
			continue
		}

		baseName := strings.TrimSuffix(htmlFile, ".html")
		t.Run(baseName, func(t *testing.T) {
			// Read the HTML file content
			htmlContent, err := os.ReadFile(htmlPath)
			require.NoError(t, err)

			// Run the extractor with and without structure preservation
			t.Log("Extracting with current structure preservation")
			article := extractWithCurrentSettings(t, string(htmlContent))

			// Measure the elements
			articleDoc, err := goquery.NewDocumentFromReader(strings.NewReader(article.Content))
			require.NoError(t, err)
			
			// Count structure elements
			paragraphCount := articleDoc.Find("p").Length()
			headingCount := articleDoc.Find("h1, h2, h3, h4, h5, h6").Length()
			listCount := articleDoc.Find("ul, ol").Length()
			listItemCount := articleDoc.Find("li").Length()

			t.Logf("Structure elements: %d paragraphs, %d headings, %d lists, %d list items", 
				paragraphCount, headingCount, listCount, listItemCount)

			// Verify minimum expectations
			assert.True(t, paragraphCount > 0, "Should have at least one paragraph")
			
			if listItemCount > 0 {
				assert.True(t, listCount > 0, "Should have lists if list items are present")
			}
		})
	}
}

// extractWithCurrentSettings extracts content with the current settings
func extractWithCurrentSettings(t *testing.T, htmlContent string) *readabiligo.Article {
	ex := readabiligo.New(
		readabiligo.WithContentDigests(false),
		readabiligo.WithDetectContentType(true),
	)
	article, err := ex.ExtractFromHTML(htmlContent, nil)
	require.NoError(t, err)
	return article
}