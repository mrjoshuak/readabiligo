package test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mrjoshuak/readabiligo/extractor"
	"github.com/mrjoshuak/readabiligo/types"
)

// BenchmarkDefaultExtraction benchmarks the entire extraction process with default options
// This provides a baseline for comparing various optimizations
func BenchmarkDefaultExtraction(b *testing.B) {
	benchCases := []struct {
		name     string
		filename string
		category string
	}{
		// Common file sizes
		{"Small", "addictinginfo.com-1_full_page.html", "size"},
		{"Medium", "davidwolfe.com-1_full_page.html", "size"},
		{"Large", "benchmarkinghuge.html", "size"},

		// Content types
		{"Reference", "real_world/wikipedia_go.html", "type"},
		{"Article", "real_world/bbc_news.html", "type"},
		{"Technical", "real_world/mdn_js_intro.html", "type"},
		
		// Edge cases
		{"NestedContent", "edge_cases/nested_content_test.html", "edge"},
		{"TableLayout", "edge_cases/table_layout_test.html", "edge"},
		{"FooterHandling", "edge_cases/footer_test.html", "edge"},
		{"MinimalContent", "edge_cases/minimal_content_test.html", "edge"},
		{"PaywallContent", "edge_cases/paywall_content_test.html", "edge"},
	}

	for _, bc := range benchCases {
		b.Run(bc.name, func(b *testing.B) {
			// Get the test data file
			testFile := filepath.Join("data", bc.filename)

			// Check if file exists
			if _, err := os.Stat(testFile); os.IsNotExist(err) {
				b.Skipf("Test file %s does not exist, skipping", testFile)
				return
			}

			// Read the file content once
			htmlBytes, err := os.ReadFile(testFile)
			if err != nil {
				b.Fatalf("Failed to read test file: %v", err)
			}
			htmlContent := string(htmlBytes)

			// Create the extractor with default options
			ext := extractor.New()

			// Reset the timer before the loop
			b.ResetTimer()

			// Run the benchmark
			for i := 0; i < b.N; i++ {
				_, err := ext.ExtractFromHTML(htmlContent, nil)
				if err != nil {
					b.Fatalf("Failed to extract article: %v", err)
				}
			}
		})
	}
}

// BenchmarkContentTypeAwareExtraction benchmarks extraction with content type awareness
// to measure the performance impact of auto-detection and specialized extraction
func BenchmarkContentTypeAwareExtraction(b *testing.B) {
	benchCases := []struct {
		name        string
		filename    string
		contentType types.ContentType
	}{
		{"Reference", "real_world/wikipedia_go.html", types.ContentTypeReference},
		{"Article", "real_world/bbc_news.html", types.ContentTypeArticle},
		{"Technical", "real_world/mdn_js_intro.html", types.ContentTypeTechnical},
		{"Minimal", "edge_cases/minimal_content_test.html", types.ContentTypeMinimal},
		{"Error", "non_article_full_page.html", types.ContentTypeError},
	}

	for _, bc := range benchCases {
		// Create two sub-benchmarks for each case:
		// 1. With auto-detection
		// 2. With explicit type
		
		b.Run(bc.name+"_AutoDetect", func(b *testing.B) {
			// Get the test data file
			testFile := filepath.Join("data", bc.filename)

			// Check if file exists
			if _, err := os.Stat(testFile); os.IsNotExist(err) {
				b.Skipf("Test file %s does not exist, skipping", testFile)
				return
			}

			// Read the file content once
			htmlBytes, err := os.ReadFile(testFile)
			if err != nil {
				b.Fatalf("Failed to read test file: %v", err)
			}
			htmlContent := string(htmlBytes)

			// Create the extractor with content type detection enabled
			ext := extractor.New(
				extractor.WithDetectContentType(true),
			)

			// Reset the timer before the loop
			b.ResetTimer()

			// Run the benchmark
			for i := 0; i < b.N; i++ {
				_, err := ext.ExtractFromHTML(htmlContent, nil)
				if err != nil {
					b.Fatalf("Failed to extract article: %v", err)
				}
			}
		})

		b.Run(bc.name+"_ExplicitType", func(b *testing.B) {
			// Get the test data file
			testFile := filepath.Join("data", bc.filename)

			// Check if file exists
			if _, err := os.Stat(testFile); os.IsNotExist(err) {
				b.Skipf("Test file %s does not exist, skipping", testFile)
				return
			}

			// Read the file content once
			htmlBytes, err := os.ReadFile(testFile)
			if err != nil {
				b.Fatalf("Failed to read test file: %v", err)
			}
			htmlContent := string(htmlBytes)

			// Create the extractor with explicit content type
			ext := extractor.New(
				extractor.WithDetectContentType(false),
				extractor.WithContentType(bc.contentType),
			)

			// Reset the timer before the loop
			b.ResetTimer()

			// Run the benchmark
			for i := 0; i < b.N; i++ {
				_, err := ext.ExtractFromHTML(htmlContent, nil)
				if err != nil {
					b.Fatalf("Failed to extract article: %v", err)
				}
			}
		})
	}
}

// BenchmarkFeatureOverhead measures the performance impact of optional features
func BenchmarkFeatureOverhead(b *testing.B) {
	// Define feature combinations to test
	features := []struct {
		name                 string
		contentDigests      bool
		nodeIndexes         bool
		preserveLinks       bool
	}{
		{"NoFeatures", false, false, false},
		{"ContentDigests", true, false, false},
		{"NodeIndexes", false, true, false},
		{"PreserveLinks", false, false, true},
		{"AllFeatures", true, true, true},
	}

	// Use a medium-sized test file
	testFile := filepath.Join("data", "davidwolfe.com-1_full_page.html")

	// Check if file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		b.Skipf("Test file %s does not exist, skipping", testFile)
		return
	}

	// Read the file content once
	htmlBytes, err := os.ReadFile(testFile)
	if err != nil {
		b.Fatalf("Failed to read test file: %v", err)
	}
	htmlContent := string(htmlBytes)

	for _, feature := range features {
		b.Run(feature.name, func(b *testing.B) {
			// Create the extractor with specified features
			ext := extractor.New(
				extractor.WithContentDigests(feature.contentDigests),
				extractor.WithNodeIndexes(feature.nodeIndexes),
				extractor.WithPreserveImportantLinks(feature.preserveLinks),
			)

			// Reset the timer before the loop
			b.ResetTimer()

			// Run the benchmark
			for i := 0; i < b.N; i++ {
				_, err := ext.ExtractFromHTML(htmlContent, nil)
				if err != nil {
					b.Fatalf("Failed to extract article: %v", err)
				}
			}
		})
	}
}

// BenchmarkDOMOperations focuses on specific DOM operations that are performance-critical
func BenchmarkDOMOperations(b *testing.B) {
	// Define test cases for different HTML complexities
	testCases := []struct {
		name     string
		filename string
	}{
		{"Simple", "addictinginfo.com-1_full_page.html"},
		{"Complex", "real_world/wikipedia_go.html"},
		{"DeepNesting", "edge_cases/nested_content_test.html"},
		{"TableHeavy", "edge_cases/table_layout_test.html"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Get the test data file
			testFile := filepath.Join("data", tc.filename)

			// Check if file exists
			if _, err := os.Stat(testFile); os.IsNotExist(err) {
				b.Skipf("Test file %s does not exist, skipping", testFile)
				return
			}

			// Read the file content once
			htmlBytes, err := os.ReadFile(testFile)
			if err != nil {
				b.Fatalf("Failed to read test file: %v", err)
			}
			htmlContent := string(htmlBytes)

			// Create the extractor with default options
			ext := extractor.New()

			// Extract just once for benchmark preparation
			article, err := ext.ExtractFromHTML(htmlContent, nil)
			if err != nil {
				b.Fatalf("Failed to extract article: %v", err)
			}

			// Reset the timer
			b.ResetTimer()

			// Benchmark the DOM operations by extracting again with a timeout
			// This forces the operation to run each time but without completing it
			for i := 0; i < b.N; i++ {
				options := types.DefaultOptions()
				options.Timeout = 100 * time.Millisecond
				_, _ = ext.ExtractFromHTML(htmlContent, &options)
			}

			// Stop the timer to prevent cleanup from affecting measurement
			b.StopTimer()

			// Make sure to use article to prevent compile error
			_ = article
		})
	}
}

// BenchmarkTimeoutImpact measures how different timeout values affect performance
func BenchmarkTimeoutImpact(b *testing.B) {
	// Define timeout values to test
	timeouts := []struct {
		name    string
		timeout time.Duration
	}{
		{"NoTimeout", 0},
		{"ShortTimeout", 100 * time.Millisecond},
		{"MediumTimeout", 1 * time.Second},
		{"LongTimeout", 30 * time.Second},
	}

	// Use a large benchmark file to ensure timeout has an impact
	testFile := filepath.Join("data", "benchmarkinghuge.html")

	// Check if file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		b.Skipf("Test file %s does not exist, skipping", testFile)
		return
	}

	// Read the file content once
	htmlBytes, err := os.ReadFile(testFile)
	if err != nil {
		b.Fatalf("Failed to read test file: %v", err)
	}
	htmlContent := string(htmlBytes)

	for _, timeout := range timeouts {
		b.Run(timeout.name, func(b *testing.B) {
			// Create the extractor with the specified timeout
			ext := extractor.New(
				extractor.WithTimeout(timeout.timeout),
			)

			// Reset the timer before the loop
			b.ResetTimer()

			// Run the benchmark, ignoring timeout errors
			for i := 0; i < b.N; i++ {
				_, _ = ext.ExtractFromHTML(htmlContent, nil)
			}
		})
	}
}

// BenchmarkMemoryUsage measures the impact of different buffer sizes on performance
func BenchmarkMemoryUsage(b *testing.B) {
	// Define buffer sizes to test
	bufferSizes := []struct {
		name       string
		bufferSize int
	}{
		{"SmallBuffer", 64 * 1024},       // 64KB
		{"DefaultBuffer", 1024 * 1024},   // 1MB (default)
		{"LargeBuffer", 10 * 1024 * 1024}, // 10MB
	}

	// Define files of different sizes to test
	fileTests := []struct {
		name     string
		filename string
	}{
		{"SmallDocument", "addictinginfo.com-1_full_page.html"},
		{"LargeDocument", "benchmarkinghuge.html"},
	}

	for _, fileTest := range fileTests {
		for _, bufferSize := range bufferSizes {
			b.Run(fileTest.name+"_"+bufferSize.name, func(b *testing.B) {
				// Get the test data file
				testFile := filepath.Join("data", fileTest.filename)

				// Check if file exists
				if _, err := os.Stat(testFile); os.IsNotExist(err) {
					b.Skipf("Test file %s does not exist, skipping", testFile)
					return
				}

				// Read the file content once
				htmlBytes, err := os.ReadFile(testFile)
				if err != nil {
					b.Fatalf("Failed to read test file: %v", err)
				}
				htmlContent := string(htmlBytes)

				// Create the extractor with specified buffer size
				ext := extractor.New(
					extractor.WithMaxBufferSize(bufferSize.bufferSize),
				)

				// Reset the timer before the loop
				b.ResetTimer()

				// Run the benchmark
				for i := 0; i < b.N; i++ {
					_, err := ext.ExtractFromHTML(htmlContent, nil)
					if err != nil {
						b.Fatalf("Failed to extract article: %v", err)
					}
				}
			})
		}
	}
}