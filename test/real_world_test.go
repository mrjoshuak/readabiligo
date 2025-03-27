package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mrjoshuak/readabiligo/extractor"
	"github.com/stretchr/testify/assert"
)

// TestRealWorldExamples tests article extraction from real-world HTML examples
// that have been downloaded using the download_real_world_examples.sh script.
// Note: These real-world examples are not included in the repository due to potential
// copyright issues. You need to run the download script before running these tests.
// The downloaded files are in .gitignore to prevent accidental commits.
func TestRealWorldExamples(t *testing.T) {
	// Skip this test if running in short mode
	if testing.Short() {
		t.Skip("Skipping real-world examples test in short mode")
	}

	// Get the list of HTML files in the real_world directory
	realWorldDir := filepath.Join("data", "real_world")
	files, err := os.ReadDir(realWorldDir)
	if err != nil {
		t.Skipf("Failed to read real_world directory: %v", err)
		return
	}

	// Skip the test if no files were found
	if len(files) == 0 {
		t.Skip("No real-world examples found. Run download_real_world_examples.sh first.")
		return
	}

	// Create the extractor with default options
	ext := extractor.New()

	// Test each real-world example
	for _, file := range files {
		// Skip non-HTML files
		if !strings.HasSuffix(file.Name(), ".html") {
			continue
		}

		t.Run(file.Name(), func(t *testing.T) {
			// Get the full path to the file
			filePath := filepath.Join(realWorldDir, file.Name())

			// Open the file
			f, err := os.Open(filePath)
			if err != nil {
				t.Fatalf("Failed to open file %s: %v", filePath, err)
			}
			defer f.Close()

			// Extract the article
			article, err := ext.ExtractFromReader(f, nil)
			if err != nil {
				t.Fatalf("Failed to extract article from %s: %v", filePath, err)
			}

			// Basic validation
			if article.Title == "" {
				t.Logf("Warning: Article title is empty for %s", filePath)
			} else {
				t.Logf("Title: %s", article.Title)
			}

			if article.Content == "" {
				t.Errorf("Article content is empty for %s", filePath)
			}

			if article.PlainContent == "" {
				t.Errorf("Article plain content is empty for %s", filePath)
			}

			if len(article.PlainText) == 0 {
				t.Errorf("Article plain text is empty for %s", filePath)
			} else {
				t.Logf("Extracted %d paragraphs", len(article.PlainText))
			}

			// Print the first paragraph for verification
			if len(article.PlainText) > 0 {
				firstPara := article.PlainText[0]
				if len(firstPara.Text) > 100 {
					t.Logf("First paragraph: %s...", firstPara.Text[:100])
				} else {
					t.Logf("First paragraph: %s", firstPara.Text)
				}
			}
		})
	}
}

// TestMultipleExtractionOptions tests article extraction with different options
func TestMultipleExtractionOptions(t *testing.T) {
	// Skip this test if running in short mode
	if testing.Short() {
		t.Skip("Skipping options comparison test in short mode")
	}

	// Get the list of HTML files in the real_world directory
	realWorldDir := filepath.Join("data", "real_world")
	files, err := os.ReadDir(realWorldDir)
	if err != nil {
		t.Skipf("Failed to read real_world directory: %v", err)
		return
	}

	// Skip the test if no files were found
	if len(files) == 0 {
		t.Skip("No real-world examples found. Run download_real_world_examples.sh first.")
		return
	}

	// Create the extractors with different options
	defaultExt := extractor.New()
	preserveLinksExt := extractor.New(
		extractor.WithPreserveImportantLinks(true),
	)

	// Test each real-world example
	for _, file := range files {
		// Skip non-HTML files
		if !strings.HasSuffix(file.Name(), ".html") {
			continue
		}

		t.Run(file.Name(), func(t *testing.T) {
			// Get the full path to the file
			filePath := filepath.Join(realWorldDir, file.Name())

			// Read the file content
			htmlBytes, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read file %s: %v", filePath, err)
			}
			htmlContent := string(htmlBytes)

			// Extract the article using the default extractor
			defaultArticle, err := defaultExt.ExtractFromHTML(htmlContent, nil)
			if err != nil {
				t.Fatalf("Failed to extract article from %s using default extractor: %v", filePath, err)
			}

			// Extract the article using the preserveLinks extractor
			preserveLinksArticle, err := preserveLinksExt.ExtractFromHTML(htmlContent, nil)
			if err != nil {
				t.Fatalf("Failed to extract article from %s using preserveLinks extractor: %v", filePath, err)
			}

			// Basic validation
			t.Logf("Default extractor title: %s", defaultArticle.Title)
			t.Logf("PreserveLinks extractor title: %s", preserveLinksArticle.Title)

			// Compare the number of paragraphs
			t.Logf("Default extractor paragraphs: %d", len(defaultArticle.PlainText))
			t.Logf("PreserveLinks extractor paragraphs: %d", len(preserveLinksArticle.PlainText))

			// The titles should match
			assert.Equal(t, defaultArticle.Title, preserveLinksArticle.Title, "Titles should match")
		})
	}
}

// TestBenchmarkRealWorld benchmarks article extraction from real-world HTML examples
func TestBenchmarkRealWorld(t *testing.T) {
	// Skip this test if running in short mode
	if testing.Short() {
		t.Skip("Skipping real-world benchmark test in short mode")
	}

	// Get the list of HTML files in the real_world directory
	realWorldDir := filepath.Join("data", "real_world")
	files, err := os.ReadDir(realWorldDir)
	if err != nil {
		t.Skipf("Failed to read real_world directory: %v", err)
		return
	}

	// Skip the test if no files were found
	if len(files) == 0 {
		t.Skip("No real-world examples found. Run download_real_world_examples.sh first.")
		return
	}

	// Create the extractors with different options
	defaultExt := extractor.New()
	digestExt := extractor.New(
		extractor.WithContentDigests(true),
	)

	// Test each real-world example
	for _, file := range files {
		// Skip non-HTML files
		if !strings.HasSuffix(file.Name(), ".html") {
			continue
		}

		t.Run(file.Name(), func(t *testing.T) {
			// Get the full path to the file
			filePath := filepath.Join(realWorldDir, file.Name())

			// Read the file content
			htmlBytes, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read file %s: %v", filePath, err)
			}
			htmlContent := string(htmlBytes)

			// Benchmark the default extractor
			start := time.Now()
			_, err = defaultExt.ExtractFromHTML(htmlContent, nil)
			if err != nil {
				t.Fatalf("Failed to extract article from %s using default extractor: %v", filePath, err)
			}
			defaultDuration := time.Since(start)

			// Benchmark the extractor with content digests
			start = time.Now()
			_, err = digestExt.ExtractFromHTML(htmlContent, nil)
			if err != nil {
				t.Fatalf("Failed to extract article from %s using digest extractor: %v", filePath, err)
			}
			digestDuration := time.Since(start)

			// Compare the results
			t.Logf("Default extractor: %v", defaultDuration)
			t.Logf("With digests: %v", digestDuration)
			t.Logf("Difference: %.2fx", float64(digestDuration)/float64(defaultDuration))
		})
	}
}
