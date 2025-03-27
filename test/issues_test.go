package test

import (
	"strings"
	"testing"

	"github.com/mrjoshuak/readabiligo/extractor"
	"github.com/mrjoshuak/readabiligo/internal/readability"
	"github.com/stretchr/testify/assert"
)

// TestExampleDomainMoreInfoLink tests that the "More information..." link is preserved
func TestExampleDomainMoreInfoLink(t *testing.T) {
	html := `<\!DOCTYPE html>
	<html>
	<head>
		<title>Example Domain</title>
	</head>
	<body>
		<div>
			<h1>Example Domain</h1>
			<p>This domain is for use in illustrative examples in documents. You may use this domain in literature without prior coordination or asking for permission.</p>
			<p><a href="https://www.iana.org/domains/example">More information...</a></p>
		</div>
	</body>
	</html>`

	// Create a new extractor with pure Go implementation
	ex := extractor.New(
		extractor.WithReadability(false),
		extractor.WithContentDigests(false),
	)

	// Extract the article
	article, err := ex.ExtractFromHTML(html, nil)
	assert.NoError(t, err)
	assert.NotNil(t, article)

	// Check that the content contains the "More information..." link
	assert.Contains(t, article.Content, "More information...")
	assert.Contains(t, article.Content, "https://www.iana.org/domains/example")
}

// TestDuplicateHeadings tests that duplicate headings are handled properly
func TestDuplicateHeadings(t *testing.T) {
	html := `<\!DOCTYPE html>
	<html>
	<head>
		<title>Test Page</title>
	</head>
	<body>
		<div>
			<h1>Example Domain</h1>
			<div>
				<h1>Example Domain</h1>
				<p>This domain is for use in illustrative examples in documents.</p>
			</div>
		</div>
	</body>
	</html>`

	// Parse directly with our implementation
	article, err := readability.Parse(html)
	assert.NoError(t, err)
	assert.NotNil(t, article)

	// Count the number of "Example Domain" headings
	// There should be only one
	count := strings.Count(article.Content, "Example Domain")
	assert.Equal(t, 1, count, "Expected only one heading with 'Example Domain'")
}

// TestNavigationRemoval tests that navigation elements are removed
func TestNavigationRemoval(t *testing.T) {
	html := `<\!DOCTYPE html>
	<html>
	<head>
		<title>Test Page</title>
	</head>
	<body>
		<nav>
			<ul>
				<li><a href="/">Home</a></li>
				<li><a href="/about">About</a></li>
				<li><a href="/contact">Contact</a></li>
			</ul>
		</nav>
		<main>
			<h1>Main Content</h1>
			<p>This is the main content of the page.</p>
			<a href="https://example.com/important-link">Important Link</a>
		</main>
		<footer>
			<p>Copyright 2025</p>
		</footer>
	</body>
	</html>`

	// Create a new extractor with pure Go implementation
	ex := extractor.New(
		extractor.WithReadability(false),
		extractor.WithContentDigests(false),
	)

	// Extract the article
	article, err := ex.ExtractFromHTML(html, nil)
	assert.NoError(t, err)
	assert.NotNil(t, article)

	// The content should contain the main content but not the navigation or footer
	assert.Contains(t, article.Content, "Main Content")
	assert.Contains(t, article.Content, "This is the main content of the page.")
	assert.Contains(t, article.Content, "Important Link")
	assert.NotContains(t, article.Content, "Home")
	assert.NotContains(t, article.Content, "About")
	assert.NotContains(t, article.Content, "Contact")
	assert.NotContains(t, article.Content, "Copyright 2025")
}

// TestImportantLinkPreservation tests that important links are preserved
func TestImportantLinkPreservation(t *testing.T) {
	html := `<\!DOCTYPE html>
	<html>
	<head>
		<title>Test Page</title>
	</head>
	<body>
		<div class="content">
			<h1>Main Content</h1>
			<p>This is the main content of the page.</p>
		</div>
		<footer>
			<p><a href="https://example.com/important-info">More information...</a></p>
			<p>Copyright 2025</p>
		</footer>
	</body>
	</html>`

	// Create a new extractor with pure Go implementation
	ex := extractor.New(
		extractor.WithReadability(false),
		extractor.WithContentDigests(false),
	)

	// Extract the article
	article, err := ex.ExtractFromHTML(html, nil)
	assert.NoError(t, err)
	assert.NotNil(t, article)

	// The content should contain the "More information..." link despite being in the footer
	assert.Contains(t, article.Content, "Main Content")
	assert.Contains(t, article.Content, "More information...")
	assert.Contains(t, article.Content, "https://example.com/important-info")
	// But not other footer content
	assert.NotContains(t, article.Content, "Copyright 2025")
}

// TestSpecialCaseCompareWithJavaScript tests that our pure Go implementation
// gives the same results as the JavaScript implementation for special cases
func TestSpecialCaseCompareWithJavaScript(t *testing.T) {
	// Skip this test if JavaScript is not available
	if \!hasNodeJS() {
		t.Skip("Node.js not available, skipping JavaScript comparison test")
	}

	html := `<\!DOCTYPE html>
	<html>
	<head>
		<title>Example Domain</title>
	</head>
	<body>
		<div>
			<h1>Example Domain</h1>
			<p>This domain is for use in illustrative examples in documents. You may use this domain in literature without prior coordination or asking for permission.</p>
			<p><a href="https://www.iana.org/domains/example">More information...</a></p>
		</div>
	</body>
	</html>`

	// Create a new extractor with JavaScript implementation
	jsExtractor := extractor.New(
		extractor.WithReadability(true),
		extractor.WithContentDigests(false),
	)

	// Create a new extractor with pure Go implementation
	goExtractor := extractor.New(
		extractor.WithReadability(false),
		extractor.WithContentDigests(false),
	)

	// Extract using both implementations
	jsArticle, err := jsExtractor.ExtractFromHTML(html, nil)
	assert.NoError(t, err)
	assert.NotNil(t, jsArticle)

	goArticle, err := goExtractor.ExtractFromHTML(html, nil)
	assert.NoError(t, err)
	assert.NotNil(t, goArticle)

	// The title should be the same
	assert.Equal(t, jsArticle.Title, goArticle.Title)

	// Both should preserve the "More information..." link
	assert.Contains(t, jsArticle.Content, "More information...")
	assert.Contains(t, goArticle.Content, "More information...")
	assert.Contains(t, jsArticle.Content, "https://www.iana.org/domains/example")
	assert.Contains(t, goArticle.Content, "https://www.iana.org/domains/example")
}

// hasNodeJS checks if Node.js is available
func hasNodeJS() bool {
	// This is a simplified version of the check in the JavaScript package
	return true // Assume it's available for now
}
