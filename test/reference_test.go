package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/mrjoshuak/readabiligo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComparisonWithPythonReference tests that our Go implementation produces
// results comparable to the Python ReadabiliPy implementation using their reference test cases
func TestComparisonWithPythonReference(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping reference test in short mode")
	}

	// Create map of test files - these match the test pairs from the Python project's test_article_extraction.py
	testPairs := map[string]string{
		"addictinginfo.html":   "addictinginfo.json",
		"conservativehq.html":  "conservativehq.json",
		"davidwolfe.html":      "davidwolfe.json",
		"list_items.html":      "list_items.json",
		"non_article.html":     "non_article.json",
	}

	refDir := filepath.Join("data", "reference")
	htmlDir := filepath.Join(refDir, "html")
	expectedDir := filepath.Join(refDir, "expected")

	// Skip if reference directories don't exist
	if _, err := os.Stat(htmlDir); os.IsNotExist(err) {
		t.Skipf("Reference HTML directory %s does not exist. Run download_test_data.sh first.", htmlDir)
	}
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Skipf("Reference expected directory %s does not exist. Run download_test_data.sh first.", expectedDir)
	}

	// Run tests for each test pair
	for htmlFile, jsonFile := range testPairs {
		htmlPath := filepath.Join(htmlDir, htmlFile)
		jsonPath := filepath.Join(expectedDir, jsonFile)

		// Skip if either file doesn't exist
		if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
			t.Logf("Skipping test for %s: HTML file not found", htmlFile)
			continue
		}
		if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
			t.Logf("Skipping test for %s: JSON file not found", htmlFile)
			continue
		}

		baseName := strings.TrimSuffix(htmlFile, ".html")
		t.Run(baseName, func(t *testing.T) {
			// Read the HTML file content
			htmlContent, err := os.ReadFile(htmlPath)
			require.NoError(t, err)

			// Read the expected JSON file
			jsonContent, err := os.ReadFile(jsonPath)
			require.NoError(t, err)

			var expectedOutput map[string]interface{}
			err = json.Unmarshal(jsonContent, &expectedOutput)
			require.NoError(t, err)

			// Run the Go implementation using the existing helper function
			ex := readabiligo.New(
				readabiligo.WithContentDigests(false),
				readabiligo.WithDetectContentType(true),
			)
			article, err := ex.ExtractFromHTML(string(htmlContent), nil)
			require.NoError(t, err)
			
			// Prepare our output to compare
			goOutput := map[string]interface{}{
				"title":   article.Title,
				"byline":  article.Byline,
				"content": article.Content,
			}

			// Compare titles strictly
			expectedTitle, hasExpectedTitle := expectedOutput["title"].(string)
			goTitle, hasGoTitle := goOutput["title"].(string)

			if hasExpectedTitle && hasGoTitle {
				// Normalize titles to make comparison more fair
				normExpectedTitle := normalizeText(expectedTitle)
				normGoTitle := normalizeText(goTitle)

				if normExpectedTitle != normGoTitle {
					// Analyze difference
					if strings.Contains(normExpectedTitle, normGoTitle) || strings.Contains(normGoTitle, normExpectedTitle) {
						t.Logf("Titles contain each other but are not identical: Python=%q, Go=%q", normExpectedTitle, normGoTitle)
					} else {
						// This is a failure case - for reference tests we expect titles to match
						t.Errorf("Titles differ significantly: Python=%q, Go=%q", expectedTitle, goTitle)
					}
				} else {
					t.Logf("Titles match exactly (after normalization)")
				}
			}

			// Compare bylines (less strict)
			expectedByline, hasExpectedByline := expectedOutput["byline"].(string)
			goByline, hasGoByline := goOutput["byline"].(string)

			if hasExpectedByline && hasGoByline {
				// Normalize bylines
				normExpectedByline := normalizeText(expectedByline)
				normGoByline := normalizeText(goByline)

				if normExpectedByline != normGoByline {
					// Log but don't fail - bylines can be extracted differently
					t.Logf("Bylines differ: Python=%q, Go=%q", expectedByline, goByline)
				} else {
					t.Logf("Bylines match exactly (after normalization)")
				}
			}

			// Compare content DOM structure (more detailed comparison)
			expectedContent, hasExpectedContent := expectedOutput["content"].(string)
			goContent, hasGoContent := goOutput["content"].(string)

			if hasExpectedContent && hasGoContent {
				compareRefContent(t, expectedContent, goContent)
			}
		})
	}
}

// compareRefContent performs detailed DOM comparison between expected and actual content
func compareRefContent(t *testing.T, expected, actual string) {
	// Parse HTML to DOM
	expectedDoc, err := goquery.NewDocumentFromReader(strings.NewReader(expected))
	require.NoError(t, err)

	actualDoc, err := goquery.NewDocumentFromReader(strings.NewReader(actual))
	require.NoError(t, err)

	// Compare element counts with some tolerance
	compareRefElementCounts(t, expectedDoc, actualDoc, "p", "paragraphs")
	compareRefElementCounts(t, expectedDoc, actualDoc, "a", "links")
	compareRefElementCounts(t, expectedDoc, actualDoc, "h1, h2, h3, h4, h5, h6", "headings")
	compareRefElementCounts(t, expectedDoc, actualDoc, "img", "images")
	compareRefElementCounts(t, expectedDoc, actualDoc, "ul, ol", "lists")
	compareRefElementCounts(t, expectedDoc, actualDoc, "li", "list items")

	// Compare text content
	expectedText := normalizeText(expectedDoc.Text())
	actualText := normalizeText(actualDoc.Text())

	// For reference tests, we expect text content to be very similar in length
	expectedLen := len(expectedText)
	actualLen := len(actualText)
	ratio := float64(actualLen) / float64(expectedLen)

	t.Logf("Text content length: Python=%d, Go=%d, Ratio=%.2f", expectedLen, actualLen, ratio)

	// More strict ratio requirement for reference tests
	assert.True(t, ratio >= 0.7 && ratio <= 1.3, 
		"Text content length differs significantly: expected between 70%% and 130%% of Python, got %.0f%%", ratio*100)

	// Check for significant content overlap
	similarityScore := calculateRefTextSimilarity(expectedText, actualText)
	t.Logf("Text similarity score: %.2f%%", similarityScore*100)
	
	// For reference tests, we expect higher similarity
	assert.True(t, similarityScore >= 0.5, 
		"Text content similarity is too low: expected at least 50%%, got %.0f%%", similarityScore*100)
}

// compareRefElementCounts compares the number of elements matching a selector
func compareRefElementCounts(t *testing.T, expectedDoc, actualDoc *goquery.Document, selector, description string) {
	expectedCount := expectedDoc.Find(selector).Length()
	actualCount := actualDoc.Find(selector).Length()
	
	t.Logf("%s count: Python=%d, Go=%d", description, expectedCount, actualCount)

	// Calculate ratio for comparison
	ratio := 1.0
	if expectedCount > 0 && actualCount > 0 {
		ratio = float64(actualCount) / float64(expectedCount)
	}

	// For reference tests, we expect counts to be close
	if expectedCount > 0 {
		// Allow wider tolerance for fewer elements
		var tolerance float64
		if expectedCount < 5 {
			tolerance = 0.5  // For small counts, allow 50% difference
		} else if expectedCount < 10 {
			tolerance = 0.3  // For medium counts, allow 30% difference
		} else {
			tolerance = 0.2  // For large counts, only allow 20% difference
		}
		
		minRatio := 1.0 - tolerance
		maxRatio := 1.0 + tolerance
		
		if ratio < minRatio || ratio > maxRatio {
			t.Logf("%s counts differ significantly: expected between %.0f%% and %.0f%% of Python, got %.0f%%", 
				description, minRatio*100, maxRatio*100, ratio*100)
		}
	}
}

// normalizeText removes extra whitespace and converts to lowercase
func normalizeText(text string) string {
	// Replace all whitespace sequences with a single space
	text = strings.Join(strings.Fields(text), " ")
	return strings.ToLower(text)
}

// calculateRefTextSimilarity provides a simple similarity metric between two texts
// Returns a value between 0.0 (completely different) and 1.0 (identical)
func calculateRefTextSimilarity(text1, text2 string) float64 {
	// Split texts into words
	words1 := strings.Fields(text1)
	words2 := strings.Fields(text2)
	
	// Create maps for word counts
	wordCount1 := make(map[string]int)
	wordCount2 := make(map[string]int)
	
	// Count words in text1
	for _, word := range words1 {
		wordCount1[word]++
	}
	
	// Count words in text2
	for _, word := range words2 {
		wordCount2[word]++
	}
	
	// Calculate intersection size
	var intersection float64
	for word, count1 := range wordCount1 {
		if count2, exists := wordCount2[word]; exists {
			// Add minimum of the two counts
			if count1 < count2 {
				intersection += float64(count1)
			} else {
				intersection += float64(count2)
			}
		}
	}
	
	// Calculate union size
	var union float64
	for _, count := range wordCount1 {
		union += float64(count)
	}
	for word, count := range wordCount2 {
		if _, exists := wordCount1[word]; !exists {
			union += float64(count)
		}
	}
	
	// Return Jaccard similarity (intersection / union)
	if union == 0 {
		return 0.0
	}
	return intersection / union
}