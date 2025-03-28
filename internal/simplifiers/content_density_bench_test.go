package simplifiers

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

// Benchmark HTML
const benchmarkHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>Benchmark Document</title>
</head>
<body>
    <header>
        <h1>Website Title</h1>
        <nav>
            <ul>
                <li><a href="#">Home</a></li>
                <li><a href="#">About</a></li>
                <li><a href="#">Contact</a></li>
            </ul>
        </nav>
    </header>
    <main id="content">
        <article>
            <h1>Main Article Heading</h1>
            <p>This is the first paragraph with some text. It contains information that might be interesting to readers.</p>
            <p>This is another paragraph with more text for testing. This paragraph is longer and contains more detailed information about the topic at hand. It also includes some keywords that might be relevant to the content.</p>
            <h2>This is a subheading</h2>
            <p>More content here. This section provides additional details and explores some related concepts that readers might find useful or interesting.</p>
            <ul>
                <li>List item 1 with some descriptive text.</li>
                <li>List item 2 with more information.</li>
                <li>List item 3 with even more content.</li>
            </ul>
            <p>Another paragraph after the list. This continues the discussion and elaborates on the points mentioned earlier.</p>
            <figure>
                <img src="image.jpg" alt="An image related to the article topic">
                <figcaption>Image caption with a brief description of what the image shows.</figcaption>
            </figure>
            <p>Final paragraph with concluding remarks and some summary information to wrap up the article nicely.</p>
        </article>
    </main>
    <aside id="sidebar">
        <h3>Sidebar</h3>
        <p>This is a sidebar with some related content. It might contain links to other articles or additional resources.</p>
        <ul>
            <li><a href="#">Related Link 1</a></li>
            <li><a href="#">Related Link 2</a></li>
            <li><a href="#">Related Link 3</a></li>
            <li><a href="#">Related Link 4</a></li>
            <li><a href="#">Related Link 5</a></li>
        </ul>
        <div class="advertisement">
            <p>Advertisement content goes here. This might be a promotional message or an actual ad placement.</p>
        </div>
    </aside>
    <div id="comments">
        <h3>Comments</h3>
        <div class="comment">
            <p>This is a comment from a user. It contains their thoughts and opinions about the article.</p>
            <p class="author">- Comment Author 1</p>
        </div>
        <div class="comment">
            <p>This is another comment from a different user, presenting a different perspective or asking a question.</p>
            <p class="author">- Comment Author 2</p>
        </div>
        <div class="comment">
            <p>A third comment with more user feedback or discussion about the topic. This helps to show engagement with the content.</p>
            <p class="author">- Comment Author 3</p>
        </div>
    </div>
    <footer>
        <p>Copyright 2025 | Website Name | All Rights Reserved</p>
        <nav>
            <ul>
                <li><a href="#">Privacy Policy</a></li>
                <li><a href="#">Terms of Service</a></li>
                <li><a href="#">Contact Us</a></li>
            </ul>
        </nav>
    </footer>
</body>
</html>
`

// setupBenchmarkDoc creates a document for benchmarking
func setupBenchmarkDoc() (*goquery.Document, error) {
	return goquery.NewDocumentFromReader(strings.NewReader(benchmarkHTML))
}

// BenchmarkContentPatternChecking benchmarks the content pattern checking functions
func BenchmarkContentPatternChecking(b *testing.B) {
	testCases := []struct {
		name     string
		id       string
		class    string
		isContent bool
	}{
		{"ContentID", "main-content", "", true},
		{"ContentClass", "", "article-body", true},
		{"NonContentID", "sidebar", "", false},
		{"NonContentClass", "", "comment-section", false}, // Fixed: comments-section actually contains "comment" which is a nonContent pattern
		{"MixedCase", "ArticleContent", "mainBody", true},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			var result bool
			for i := 0; i < b.N; i++ {
				if tc.id != "" {
					result = containsContentPattern(tc.id)
				} else {
					result = containsContentPattern(tc.class)
				}
			}
			// Prevent compiler from optimizing away the benchmark
			if result != tc.isContent {
				b.Fatalf("Unexpected result: got %v, expected %v", result, tc.isContent)
			}
		})
	}
}

// BenchmarkNodeTextExtraction benchmarks the text extraction operations
func BenchmarkNodeTextExtraction(b *testing.B) {
	doc, err := setupBenchmarkDoc()
	if err != nil {
		b.Fatalf("Failed to create document: %v", err)
	}

	elements := []struct {
		name     string
		selector string
	}{
		{"MainContent", "#content"},
		{"Article", "article"},
		{"Paragraph", "p"},
		{"List", "ul"},
		{"Comments", "#comments"},
	}

	// Run uncached first to establish baseline
	b.Run("Uncached", func(b *testing.B) {
		for _, elem := range elements {
			b.Run(elem.name, func(b *testing.B) {
				selection := doc.Find(elem.selector)
				// Clear cache before each test
				clearGlobalCache()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					_ = selection.Text()
				}
			})
		}
	})

	// Then run with caching enabled
	b.Run("Cached", func(b *testing.B) {
		for _, elem := range elements {
			b.Run(elem.name, func(b *testing.B) {
				selection := doc.Find(elem.selector)
				// Clear cache before each test
				clearGlobalCache()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					// First call will cache, subsequent calls will use cache
					_ = getNodeText(selection)
				}
			})
		}
	})
}

// BenchmarkAnalyzeContentDensity benchmarks the content density analysis
func BenchmarkAnalyzeContentDensity(b *testing.B) {
	doc, err := setupBenchmarkDoc()
	if err != nil {
		b.Fatalf("Failed to create document: %v", err)
	}

	elements := []struct {
		name     string
		selector string
	}{
		{"Article", "article"},
		{"ContentDiv", "#content"},
		{"Sidebar", "#sidebar"},
		{"Comments", "#comments"},
	}

	for _, elem := range elements {
		b.Run(elem.name, func(b *testing.B) {
			selection := doc.Find(elem.selector)
			// Clear cache before each benchmark
			clearGlobalCache()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				AnalyzeContentDensity(selection)
			}
		})
	}
}

// BenchmarkCalculateContentScore benchmarks the content scoring algorithm
func BenchmarkCalculateContentScore(b *testing.B) {
	doc, err := setupBenchmarkDoc()
	if err != nil {
		b.Fatalf("Failed to create document: %v", err)
	}

	elements := []struct {
		name     string
		selector string
	}{
		{"Article", "article"},
		{"ContentDiv", "#content"},
		{"Sidebar", "#sidebar"},
		{"Comments", "#comments"},
	}

	// First run without caching
	b.Run("Original", func(b *testing.B) {
		for _, elem := range elements {
			b.Run(elem.name, func(b *testing.B) {
				selection := doc.Find(elem.selector)
				// Clear cache to ensure fair comparison
				clearGlobalCache()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					CalculateContentScore(selection)
				}
			})
		}
	})

	// Then run with repeated calls to show caching benefit
	b.Run("Repeated", func(b *testing.B) {
		for _, elem := range elements {
			b.Run(elem.name, func(b *testing.B) {
				selection := doc.Find(elem.selector)
				// Clear cache to start fresh
				clearGlobalCache()
				
				// Warm up cache with first call
				CalculateContentScore(selection)
				
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					// Subsequent calls should be faster due to caching
					CalculateContentScore(selection)
				}
			})
		}
	})
}

// BenchmarkFindMainContentNode benchmarks the main content node detection
func BenchmarkFindMainContentNode(b *testing.B) {
	doc, err := setupBenchmarkDoc()
	if err != nil {
		b.Fatalf("Failed to create document: %v", err)
	}

	// Run the benchmark
	b.Run("FindMainContentNode", func(b *testing.B) {
		// Reset timer for accurate measurement
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Clear cache for each iteration to ensure fair testing
			clearGlobalCache()
			FindMainContentNode(doc)
		}
	})
}