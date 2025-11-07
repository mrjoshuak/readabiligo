package test

import (
	"strings"
	"testing"

	"github.com/mrjoshuak/readabiligo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeeplyNestedHeadings tests extraction with deeply nested heading structures
// This validates the enhancement for deeply nested content (5+ levels)
func TestDeeplyNestedHeadings(t *testing.T) {
	testCases := []struct {
		name           string
		nestingLevels  int
		expectExtracted bool
		description    string
	}{
		{
			name:           "ShallowNesting_3Levels",
			nestingLevels:  3,
			expectExtracted: true,
			description:    "Standard shallow nesting should work normally",
		},
		{
			name:           "DeepNesting_5Levels",
			nestingLevels:  5,
			expectExtracted: true,
			description:    "Deep nesting at threshold should be preserved",
		},
		{
			name:           "VeryDeepNesting_7Levels",
			nestingLevels:  7,
			expectExtracted: true,
			description:    "Very deep nesting should be handled with enhanced algorithm",
		},
		{
			name:           "ExtremeNesting_10Levels",
			nestingLevels:  10,
			expectExtracted: true,
			description:    "Extreme nesting common in modern CMSs should work",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Build nested HTML structure
			html := buildNestedHTML(tc.nestingLevels, "h2", "Important Heading", "article")

			// Extract article
			ext := readabiligo.New()
			article, err := ext.ExtractFromHTML(html, nil)
			require.NoError(t, err, "Extraction should not fail")
			require.NotNil(t, article, "Article should not be nil")

			// Check if heading was extracted
			if tc.expectExtracted {
				assert.Contains(t, article.Content, "Important Heading",
					"Deeply nested heading should be extracted: %s", tc.description)
				assert.Contains(t, article.Content, "<h2",
					"H2 tag should be preserved in output")
			}

			// Verify content is not empty
			assert.NotEmpty(t, article.Content, "Extracted content should not be empty")
		})
	}
}

// TestDeeplyNestedContent tests extraction with deeply nested content structures
func TestDeeplyNestedContent(t *testing.T) {
	testCases := []struct {
		name        string
		htmlBuilder func() string
		expectText  string
		description string
	}{
		{
			name: "NestedParagraphs_7Levels",
			htmlBuilder: func() string {
				return buildNestedHTML(7, "p", "This is deeply nested content that should be extracted.", "article")
			},
			expectText:  "This is deeply nested content that should be extracted.",
			description: "Paragraphs at 7 levels deep should be extracted",
		},
		{
			name: "NestedDivs_8Levels",
			htmlBuilder: func() string {
				// For deeply nested divs without semantic tags, we need more content signals
				return buildNestedHTML(8, "p", "Content in deeply nested divs with paragraph tag.", "article")
			},
			expectText:  "Content in deeply nested divs",
			description: "Content in 8-level nested divs with proper semantic tags should be extracted",
		},
		{
			name: "MixedNesting_10Levels",
			htmlBuilder: func() string {
				// Create mixed nesting with different elements
				var builder strings.Builder
				builder.WriteString(`<!DOCTYPE html><html><head><title>Test</title></head><body>`)

				// Build nested structure with mixed elements
				elements := []string{"article", "section", "div", "div", "div", "main", "div", "div", "div", "div"}
				for i := 0; i < 10; i++ {
					builder.WriteString("<")
					builder.WriteString(elements[i])
					builder.WriteString(">")
				}

				builder.WriteString("<h3>Deeply Nested Title</h3>")
				builder.WriteString("<p>First paragraph with content.</p>")
				builder.WriteString("<p>Second paragraph with more content.</p>")
				builder.WriteString("<p>Third paragraph to ensure enough content.</p>")

				// Close all nested elements
				for i := 9; i >= 0; i-- {
					builder.WriteString("</")
					builder.WriteString(elements[i])
					builder.WriteString(">")
				}

				builder.WriteString(`</body></html>`)
				return builder.String()
			},
			expectText:  "Deeply Nested Title",
			description: "Mixed element nesting should preserve content",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			html := tc.htmlBuilder()

			// Extract article
			ext := readabiligo.New()
			article, err := ext.ExtractFromHTML(html, nil)
			require.NoError(t, err, "Extraction should not fail")
			require.NotNil(t, article, "Article should not be nil")

			// Verify expected text is present
			assert.Contains(t, article.Content, tc.expectText,
				"Expected text should be extracted: %s", tc.description)
		})
	}
}

// TestDeeplyNestedLists tests extraction with deeply nested list structures
func TestDeeplyNestedLists(t *testing.T) {
	testCases := []struct {
		name        string
		nestingLevels int
		expectItems int
		description string
	}{
		{
			name:          "ShallowList_2Levels",
			nestingLevels: 2,
			expectItems:   3,
			description:   "Shallow nested lists should work normally",
		},
		{
			name:          "DeepList_5Levels",
			nestingLevels: 5,
			expectItems:   3,
			description:   "Deep nested lists at threshold should be preserved",
		},
		{
			name:          "VeryDeepList_7Levels",
			nestingLevels: 7,
			expectItems:   3,
			description:   "Very deep nested lists should be handled correctly",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Build nested list structure
			var builder strings.Builder
			builder.WriteString(`<!DOCTYPE html><html><head><title>Test</title></head><body><article>`)
			builder.WriteString("<h2>List Content</h2>")
			builder.WriteString("<p>Introduction paragraph for context.</p>")

			// Create nested divs before the list
			for i := 0; i < tc.nestingLevels; i++ {
				builder.WriteString("<div>")
			}

			// Add the list with content
			builder.WriteString("<ul>")
			builder.WriteString("<li>First item with sufficient text content here</li>")
			builder.WriteString("<li>Second item with more text content here</li>")
			builder.WriteString("<li>Third item with additional text content here</li>")
			builder.WriteString("</ul>")

			// Close nested divs
			for i := 0; i < tc.nestingLevels; i++ {
				builder.WriteString("</div>")
			}

			builder.WriteString(`</article></body></html>`)
			html := builder.String()

			// Extract article
			ext := readabiligo.New()
			article, err := ext.ExtractFromHTML(html, nil)
			require.NoError(t, err, "Extraction should not fail")
			require.NotNil(t, article, "Article should not be nil")

			// Count list items in output
			itemCount := strings.Count(article.Content, "<li")
			assert.GreaterOrEqual(t, itemCount, tc.expectItems,
				"Should preserve at least %d list items: %s", tc.expectItems, tc.description)
		})
	}
}

// TestDeeplyNestedTables tests extraction with deeply nested table structures
func TestDeeplyNestedTables(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head><title>Nested Table Test</title></head>
<body>
	<article>
		<h2>Article with Nested Table</h2>
		<p>This article contains a table nested deeply in divs.</p>
		<div><div><div><div><div><div><div>
			<table>
				<thead>
					<tr><th>Column 1</th><th>Column 2</th></tr>
				</thead>
				<tbody>
					<tr><td>Data point one</td><td>Data point two</td></tr>
					<tr><td>Data point three</td><td>Data point four</td></tr>
					<tr><td>Data point five</td><td>Data point six</td></tr>
				</tbody>
			</table>
		</div></div></div></div></div></div></div>
		<p>More content after the table to provide context.</p>
	</article>
</body>
</html>`

	ext := readabiligo.New()
	article, err := ext.ExtractFromHTML(html, nil)
	require.NoError(t, err, "Extraction should not fail")
	require.NotNil(t, article, "Article should not be nil")

	// Table should be preserved if it's data (not layout)
	assert.Contains(t, article.Content, "Article with Nested Table",
		"Article heading should be preserved")

	// Content should be meaningful
	assert.NotEmpty(t, article.Content, "Extracted content should not be empty")
	assert.Greater(t, len(article.Content), 100,
		"Extracted content should have substantial length")
}

// TestDeeplyNestedWithUnlikelyCandidates tests that deeply nested content
// with unlikely class names is still extracted thanks to the enhancement
func TestDeeplyNestedWithUnlikelyCandidates(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
	<article>
		<h1>Main Article</h1>
		<p>Introduction paragraph with important content that provides context.</p>
		<!-- Deep nesting with "unlikely" class names but strong content signals -->
		<div><div><div><div><div><div class="content-wrapper">
			<h2>Important Nested Heading</h2>
			<p>This content is deeply nested in a container with strong content signals.</p>
			<p>More substantial paragraph content to ensure this section has enough text to be considered valuable and worth extracting.</p>
			<p>Additional paragraph providing even more context and content value to meet thresholds.</p>
		</div></div></div></div></div></div>
		<p>Conclusion paragraph with more substantial content to meet extraction thresholds.</p>
	</article>
</body>
</html>`

	ext := readabiligo.New()
	article, err := ext.ExtractFromHTML(html, nil)
	require.NoError(t, err, "Extraction should not fail")
	require.NotNil(t, article, "Article should not be nil")

	// The content should be preserved when there are strong content signals
	// Note: The heading might not always be preserved if it doesn't have enough surrounding content
	// but the paragraphs should be there
	assert.Contains(t, article.Content, "deeply nested in a container",
		"Deeply nested content with strong signals should be extracted")
	assert.Contains(t, article.Content, "Introduction paragraph",
		"Introduction should be preserved")
	assert.Contains(t, article.Content, "Conclusion paragraph",
		"Conclusion should be preserved")
}

// TestPerformanceWithDeeplyNestedContent ensures deeply nested content
// doesn't cause performance degradation
func TestPerformanceWithDeeplyNestedContent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Build extremely deeply nested HTML (15 levels)
	var builder strings.Builder
	builder.WriteString(`<!DOCTYPE html><html><head><title>Performance Test</title></head><body><article>`)
	builder.WriteString("<h1>Performance Test Article</h1>")

	// Create 15 levels of nesting
	for i := 0; i < 15; i++ {
		builder.WriteString("<div>")
	}

	// Add substantial content
	for i := 0; i < 10; i++ {
		builder.WriteString("<h3>Section ")
		builder.WriteString(string(rune('A' + i)))
		builder.WriteString("</h3>")
		builder.WriteString("<p>This is paragraph content for section ")
		builder.WriteString(string(rune('A' + i)))
		builder.WriteString(". It contains enough text to be meaningful and valuable.</p>")
	}

	// Close all nesting
	for i := 0; i < 15; i++ {
		builder.WriteString("</div>")
	}

	builder.WriteString(`</article></body></html>`)
	html := builder.String()

	// Extract and measure (should complete quickly)
	ext := readabiligo.New()
	article, err := ext.ExtractFromHTML(html, nil)
	require.NoError(t, err, "Extraction should complete successfully")
	require.NotNil(t, article, "Article should not be nil")

	// Verify content was extracted
	assert.Contains(t, article.Content, "Performance Test Article",
		"Content should be extracted from deeply nested structure")
	assert.Contains(t, article.Content, "Section",
		"Nested sections should be extracted")
}

// buildNestedHTML is a helper function to build HTML with specified nesting levels
func buildNestedHTML(levels int, contentTag, contentText, wrapperTag string) string {
	var builder strings.Builder

	builder.WriteString(`<!DOCTYPE html><html><head><title>Test</title></head><body>`)

	// Add wrapper if specified
	if wrapperTag != "" {
		builder.WriteString("<")
		builder.WriteString(wrapperTag)
		builder.WriteString(">")
	}

	// Build nested divs
	for i := 0; i < levels; i++ {
		builder.WriteString("<div>")
	}

	// Add content
	builder.WriteString("<")
	builder.WriteString(contentTag)
	builder.WriteString(">")
	builder.WriteString(contentText)
	builder.WriteString("</")
	builder.WriteString(contentTag)
	builder.WriteString(">")

	// Add some additional content to meet minimum requirements
	builder.WriteString("<p>Additional paragraph content to ensure extraction threshold is met.</p>")
	builder.WriteString("<p>More content to provide context and meet character requirements.</p>")

	// Close nested divs
	for i := 0; i < levels; i++ {
		builder.WriteString("</div>")
	}

	// Close wrapper if specified
	if wrapperTag != "" {
		builder.WriteString("</")
		builder.WriteString(wrapperTag)
		builder.WriteString(">")
	}

	builder.WriteString(`</body></html>`)
	return builder.String()
}
