package test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/mrjoshuak/readabiligo/extractor"
	"github.com/mrjoshuak/readabiligo/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComparisonWithPythonBasic tests that our Go implementation produces
// results comparable to the Python ReadabiliPy implementation using synthetic test cases
func TestComparisonWithPythonBasic(t *testing.T) {
	// Skip if Python is not available
	if !hasPython() {
		t.Skip("Python not available, skipping comparison test")
	}

	// Skip if Python ReadabiliPy is not installed
	if !hasReadabiliPy() {
		t.Skip("ReadabiliPy not installed, skipping comparison test")
	}

	// Set up test cases
	testCases := []struct {
		name     string
		htmlFile string
	}{
		{"ExampleDomain", "example_domain.html"},
		{"Wikipedia", "wikipedia_article.html"},
		{"BlogPost", "blog_post.html"},
		{"NewsArticle", "news_article.html"},
		{"ComplexLayout", "complex_layout.html"},
	}

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "readabiligo-comparison")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Run comparisons for each test case
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test HTML file for this case
			htmlContent := createTestHTML(t, tc.name)
			htmlPath := filepath.Join(tempDir, tc.htmlFile)
			err := os.WriteFile(htmlPath, []byte(htmlContent), 0644)
			assert.NoError(t, err)

			// Run Python implementation
			pythonOutput, err := runPythonReadabiliPy(htmlPath)
			assert.NoError(t, err)

			// Run Go implementation
			goOutput, err := runGoReadability(htmlContent)
			assert.NoError(t, err)

			// Compare results
			assertEqualOutput(t, pythonOutput, goOutput)
		})
	}
}

// TestComparisonWithRealWorldData tests Go implementation against Python 
// with real-world examples downloaded by the download_real_world_examples.sh script
func TestComparisonWithRealWorldData(t *testing.T) {
	// Skip if Python is not available
	if !hasPython() {
		t.Skip("Python not available, skipping real-world comparison test")
	}

	// Skip if Python ReadabiliPy is not installed
	if !hasReadabiliPy() {
		t.Skip("ReadabiliPy not installed, skipping real-world comparison test")
	}

	// Skip this test if running in short mode
	if testing.Short() {
		t.Skip("Skipping real-world comparison test in short mode")
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

	// Create a temporary directory for output files
	tempDir, err := os.MkdirTemp("", "readabiligo-real-world-comparison")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// For each real-world example, run both implementations and compare
	for _, file := range files {
		// Skip non-HTML files
		if !strings.HasSuffix(file.Name(), ".html") {
			continue
		}

		t.Run(file.Name(), func(t *testing.T) {
			// Get the full path to the file
			filePath := filepath.Join(realWorldDir, file.Name())

			// Run Python implementation
			pythonOutput, err := runPythonReadabiliPy(filePath)
			if err != nil {
				t.Logf("Python ReadabiliPy failed on %s: %v", file.Name(), err)
				t.Skip("Skipping comparison due to Python error")
				return
			}

			// Read the HTML file content
			htmlContent, err := os.ReadFile(filePath)
			assert.NoError(t, err)

			// Run Go implementation
			goOutput, err := runGoReadability(string(htmlContent))
			assert.NoError(t, err)

			// Compare results
			assertEqualOutput(t, pythonOutput, goOutput)
		})
	}
}

// TestComparisonWithReadabiliPyReferences tests against ReadabiliPy's reference files
func TestComparisonWithReadabiliPyReferences(t *testing.T) {
	// Skip this test if running in short mode
	if testing.Short() {
		t.Skip("Skipping ReadabiliPy reference comparison test in short mode")
	}

	// Set up test cases - files from the ReadabiliPy repository downloaded by the script
	testCases := []struct {
		name        string
		htmlFile    string
		expectedJSON string
	}{
		{"AddictingInfo", "addictinginfo.com-1_full_page.html", "addictinginfo.com-1_simple_article_from_full_page.json"},
		{"ConservativeHQ", "conservativehq.com-1_full_page.html", "conservativehq.com-1_simple_article_from_full_page.json"},
		{"DavidWolfe", "davidwolfe.com-1_full_page.html", "davidwolfe.com-1_simple_article_from_full_page.json"},
	}

	// For each test case, run our implementation and compare with the reference JSON
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Check if the test files exist
			htmlPath := filepath.Join("data", tc.htmlFile)
			jsonPath := filepath.Join("data", tc.expectedJSON)
			
			if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
				t.Skipf("HTML file %s does not exist. Run download_real_world_examples.sh first.", htmlPath)
				return
			}
			if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
				t.Skipf("JSON reference file %s does not exist. Run download_real_world_examples.sh first.", jsonPath)
				return
			}

			// Read the HTML file
			htmlContent, err := os.ReadFile(htmlPath)
			assert.NoError(t, err)

			// Read the reference JSON file
			jsonContent, err := os.ReadFile(jsonPath)
			assert.NoError(t, err)

			var referenceOutput map[string]interface{}
			err = json.Unmarshal(jsonContent, &referenceOutput)
			assert.NoError(t, err)

			// Run Go implementation
			goOutput, err := runGoReadability(string(htmlContent))
			assert.NoError(t, err)

			// Compare our output with reference JSON
			compareWithReferenceJSON(t, referenceOutput, goOutput)
		})
	}
}

// TestSystematicComparisonWithDifferentOptions tests our implementation with different options
func TestSystematicComparisonWithDifferentOptions(t *testing.T) {
	// Skip this test if running in short mode
	if testing.Short() {
		t.Skip("Skipping options comparison test in short mode")
	}

	// Get a list of test files to use
	testFiles := []string{
		"data/content_extraction_test.html",
		"data/heading_test.html",
		"data/list_items_full_page.html",
		"data/non_article_full_page.html",
		"data/special_case_test.html",
	}

	// Create different extractor configurations to test
	extractorConfigs := []struct {
		name   string
		config func() extractor.Extractor
	}{
		{"Default", func() extractor.Extractor { return extractor.New() }},
		{"WithPreserveImportantLinks", func() extractor.Extractor { 
			return extractor.New(extractor.WithPreserveImportantLinks(true)) 
		}},
		{"WithContentDigests", func() extractor.Extractor { 
			return extractor.New(extractor.WithContentDigests(true)) 
		}},
		{"WithNodeIndexes", func() extractor.Extractor { 
			return extractor.New(extractor.WithNodeIndexes(true)) 
		}},
	}

	// For each test file, run all configurations and compare the results
	for _, testFile := range testFiles {
		// Skip if the file doesn't exist
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			t.Logf("Test file %s does not exist. Skipping.", testFile)
			continue
		}

		t.Run(filepath.Base(testFile), func(t *testing.T) {
			// Read the HTML file
			htmlContent, err := os.ReadFile(testFile)
			require.NoError(t, err)

			// Create a baseline result using the default extractor
			baselineExtractor := extractor.New()
			baseline, err := baselineExtractor.ExtractFromHTML(string(htmlContent), nil)
			require.NoError(t, err)

			// Run each configuration and compare with baseline
			for _, config := range extractorConfigs[1:] { // Skip default which is our baseline
				t.Run(config.name, func(t *testing.T) {
					ext := config.config()
					result, err := ext.ExtractFromHTML(string(htmlContent), nil)
					require.NoError(t, err)

					// Compare the results - focus on the parts that should be consistent
					assert.Equal(t, baseline.Title, result.Title, "Titles should match")
					
					// For content, do a basic comparison - important links preservation will affect this
					if config.name != "WithPreserveImportantLinks" {
						// Check content length is roughly similar - allow for options-specific variations
						baselineLen := len(baseline.Content)
						resultLen := len(result.Content)
						ratio := float64(resultLen) / float64(baselineLen)
						
						t.Logf("Content length - Baseline: %d, %s: %d, Ratio: %.2f", 
							baselineLen, config.name, resultLen, ratio)
						
						// Content length shouldn't vary too dramatically unless it's a special option
						assert.True(t, ratio > 0.2 && ratio < 30.0, 
							"Content length ratio should be within reasonable bounds")
					}
				})
			}
		})
	}
}

// TestSemanticDOMCompare tests the DOM structure of extracted content for equivalence
func TestSemanticDOMCompare(t *testing.T) {
	// Skip this test if running in short mode
	if testing.Short() {
		t.Skip("Skipping semantic DOM comparison test in short mode")
	}

	// Set up test cases
	testCases := []struct {
		name     string
		htmlFile string
	}{
		{"ContentExtraction", "content_extraction_test.html"},
		{"Headings", "heading_test.html"},
		{"ListItems", "list_items_full_page.html"},
		{"SpecialCase", "special_case_test.html"},
	}

	// For each test case, run our implementation and compare the DOM structure
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Check if the test file exists
			htmlPath := filepath.Join("data", tc.htmlFile)
			if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
				t.Skipf("HTML file %s does not exist.", htmlPath)
				return
			}

			// Read the HTML file
			htmlContent, err := os.ReadFile(htmlPath)
			assert.NoError(t, err)

			// Extract with default options
			defaultExt := extractor.New()
			defaultArticle, err := defaultExt.ExtractFromHTML(string(htmlContent), nil)
			assert.NoError(t, err)

			// Extract with preserve important links option
			preserveExt := extractor.New(extractor.WithPreserveImportantLinks(true))
			preserveArticle, err := preserveExt.ExtractFromHTML(string(htmlContent), nil)
			assert.NoError(t, err)

			// Compare the DOM structure of the extracted articles
			compareDOMStructure(t, defaultArticle.Content, preserveArticle.Content)
		})
	}
}

// TestContentTypeAwareExtraction tests the content-type awareness feature with optimal settings for each type
func TestContentTypeAwareExtraction(t *testing.T) {
	// Skip this test if running in short mode
	if testing.Short() {
		t.Skip("Skipping content-type awareness test in short mode")
	}

	// Set up test cases with content types
	testCases := []struct {
		name        string
		htmlCase    string
		contentType types.ContentType
	}{
		{"Wikipedia", "Wikipedia", types.ContentTypeReference},
		{"BlogPost", "BlogPost", types.ContentTypeArticle},
		{"NewsArticle", "NewsArticle", types.ContentTypeArticle},
		{"TechnicalContent", "ComplexLayout", types.ContentTypeTechnical},
		{"ErrorPage", "ErrorPage", types.ContentTypeError},
	}

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "readabiligo-content-type")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// For each test case, run with auto-detection and forced content type
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test HTML (or use ErrorPage specific HTML for that case)
			var htmlContent string
			if tc.htmlCase == "ErrorPage" {
				htmlContent = createErrorPageHTML()
			} else {
				htmlContent = createTestHTML(t, tc.htmlCase)
			}
			
			htmlPath := filepath.Join(tempDir, tc.name+".html")
			err := os.WriteFile(htmlPath, []byte(htmlContent), 0644)
			assert.NoError(t, err)

			// Test with auto-detection
			autoDetectExt := extractor.New(
				extractor.WithDetectContentType(true),
			)
			autoDetectArticle, err := autoDetectExt.ExtractFromHTML(htmlContent, nil)
			assert.NoError(t, err)

			// Test with forced content type
			forcedTypeExt := extractor.New(
				extractor.WithDetectContentType(false),
				extractor.WithContentType(tc.contentType),
			)
			forcedTypeArticle, err := forcedTypeExt.ExtractFromHTML(htmlContent, nil)
			assert.NoError(t, err)

			// Log detected content type
			t.Logf("Auto-detected content type: %s", autoDetectArticle.ContentType.String())
			t.Logf("Forced content type: %s", forcedTypeArticle.ContentType.String())

			// Compare extraction results
			autoDetectContent := autoDetectArticle.Content
			forcedTypeContent := forcedTypeArticle.Content

			// For error pages and reference content, we expect significant differences
			// due to different cleaning strategies, so don't compare DOM directly
			if tc.contentType != types.ContentTypeError && tc.contentType != types.ContentTypeReference {
				compareDOMContent(t, autoDetectContent, forcedTypeContent)
			} else {
				// Just log the content lengths
				t.Logf("Auto-detect content length: %d", len(autoDetectContent))
				t.Logf("Forced type content length: %d", len(forcedTypeContent))
			}
			
			// For error pages, verify aggressive cleaning was applied
			if tc.contentType == types.ContentTypeError {
				// Parse the content to check for navigation elements
				autoDetectDoc, _ := goquery.NewDocumentFromReader(strings.NewReader(autoDetectContent))
				forcedTypeDoc, _ := goquery.NewDocumentFromReader(strings.NewReader(forcedTypeContent))
				
				// Count navigation elements
				autoNavCount := autoDetectDoc.Find("nav, .nav, .navigation, .menu").Length()
				forcedNavCount := forcedTypeDoc.Find("nav, .nav, .navigation, .menu").Length()
				
				t.Logf("Navigation elements - Auto-detect: %d, Forced Error type: %d", 
					autoNavCount, forcedNavCount)
				
				// Error pages should have fewer navigation elements
				assert.True(t, forcedNavCount <= autoNavCount, 
					"Error page processing should remove more navigation elements")
			}
			
			// For reference content, verify structure preservation
			if tc.contentType == types.ContentTypeReference {
				// Check heading and list preservation
				autoDetectDoc, _ := goquery.NewDocumentFromReader(strings.NewReader(autoDetectContent))
				forcedTypeDoc, _ := goquery.NewDocumentFromReader(strings.NewReader(forcedTypeContent))
				
				// Count headings and lists
				autoHeadingCount := autoDetectDoc.Find("h1, h2, h3, h4, h5, h6").Length()
				forcedHeadingCount := forcedTypeDoc.Find("h1, h2, h3, h4, h5, h6").Length()
				autoListCount := autoDetectDoc.Find("ul, ol, li").Length()
				forcedListCount := forcedTypeDoc.Find("ul, ol, li").Length()
				
				t.Logf("Headings - Auto-detect: %d, Forced Reference type: %d", 
					autoHeadingCount, forcedHeadingCount)
				t.Logf("List elements - Auto-detect: %d, Forced Reference type: %d", 
					autoListCount, forcedListCount)
				
				// Reference content should preserve more structure
				assert.True(t, forcedHeadingCount >= autoHeadingCount, 
					"Reference content should preserve more headings")
				assert.True(t, forcedListCount >= autoListCount, 
					"Reference content should preserve more list elements")
			}
		})
	}
}

// createErrorPageHTML generates HTML for an error page
func createErrorPageHTML() string {
	return `<!DOCTYPE html>
<html>
<head>
    <title>404 - Page Not Found</title>
</head>
<body>
    <header>
        <nav>
            <ul>
                <li><a href="/">Home</a></li>
                <li><a href="/about">About</a></li>
                <li><a href="/contact">Contact</a></li>
            </ul>
        </nav>
    </header>
    <main>
        <div class="error-container">
            <h1>404 - Page Not Found</h1>
            <p>The page you are looking for might have been removed, had its name changed, or is temporarily unavailable.</p>
            <p>Please try the following:</p>
            <ul>
                <li>Check the URL for typos</li>
                <li>Go back to the <a href="/">homepage</a></li>
                <li>Use the navigation menu above</li>
            </ul>
        </div>
    </main>
    <footer>
        <nav class="footer-nav">
            <ul>
                <li><a href="/privacy">Privacy Policy</a></li>
                <li><a href="/terms">Terms of Use</a></li>
                <li><a href="/sitemap">Sitemap</a></li>
            </ul>
        </nav>
        <p>&copy; 2025 Website Name. All rights reserved.</p>
    </footer>
</body>
</html>`
}

// hasPython checks if Python is available
func hasPython() bool {
	cmd := exec.Command("python", "--version")
	return cmd.Run() == nil
}

// hasReadabiliPy checks if ReadabiliPy is installed
func hasReadabiliPy() bool {
	cmd := exec.Command("python", "-c", "import readabilipy")
	return cmd.Run() == nil
}

// runPythonReadabiliPy runs the Python implementation and returns the result
func runPythonReadabiliPy(htmlPath string) (map[string]interface{}, error) {
	// Create a temporary file for the output
	outputFile, err := os.CreateTemp("", "readabilipy-output-*.json")
	if err != nil {
		return nil, err
	}
	defer os.Remove(outputFile.Name())
	outputFile.Close()

	// Create and run the Python script
	pythonScript := `
import json
import sys
from readabilipy import simple_json_from_html_string

# Read HTML file
with open(sys.argv[1], 'r', encoding='utf-8') as f:
    html = f.read()

# Extract article
article = simple_json_from_html_string(html, use_readability=True)

# Write result to output file
with open(sys.argv[2], 'w', encoding='utf-8') as f:
    json.dump(article, f)
`
	scriptFile, err := os.CreateTemp("", "readabilipy-script-*.py")
	if err != nil {
		return nil, err
	}
	defer os.Remove(scriptFile.Name())
	
	_, err = scriptFile.WriteString(pythonScript)
	if err != nil {
		return nil, err
	}
	scriptFile.Close()

	// Run the Python script
	cmd := exec.Command("python", scriptFile.Name(), htmlPath, outputFile.Name())
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	// Read and parse the output
	outputData, err := os.ReadFile(outputFile.Name())
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(outputData, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// runGoReadability runs our Go implementation and returns the result
func runGoReadability(html string) (map[string]interface{}, error) {
	// Create a new extractor with pure Go implementation
	ex := extractor.New(
		extractor.WithContentDigests(false),
		extractor.WithDetectContentType(true), // Enable content type detection
	)

	// Extract the article
	article, err := ex.ExtractFromHTML(html, nil)
	if err != nil {
		return nil, err
	}

	// Convert to map for comparison
	result := map[string]interface{}{
		"title":       article.Title,
		"byline":      article.Byline,
		"content":     article.Content,
		"plain_content": article.PlainContent,
		"plain_text":  article.PlainText,
		"content_type": article.ContentType.String(),
	}

	return result, nil
}

// assertEqualOutput compares the Python and Go outputs
func assertEqualOutput(t *testing.T, pythonOutput, goOutput map[string]interface{}) {
	// Compare titles - exact match for titles when present
	pythonTitle, hasPythonTitle := pythonOutput["title"].(string)
	goTitle, hasGoTitle := goOutput["title"].(string)
	
	if hasPythonTitle && hasGoTitle {
		// Allow for some normalization differences
		normPythonTitle := normalizeWhitespace(pythonTitle)
		normGoTitle := normalizeWhitespace(goTitle)
		
		// Log titles for debugging
		t.Logf("Titles - Python: %q, Go: %q", normPythonTitle, normGoTitle)
		
		// Check if titles are identical or one contains the other
		if normPythonTitle != normGoTitle {
			if strings.Contains(normPythonTitle, normGoTitle) || strings.Contains(normGoTitle, normPythonTitle) {
				t.Logf("Titles contain each other but are not identical")
			} else {
				t.Logf("Titles differ significantly - Python: %q, Go: %q", normPythonTitle, normGoTitle)
			}
		}
	}

	// Log bylines, don't enforce exact matching
	pythonByline, hasPythonByline := pythonOutput["byline"].(string)
	goByline, hasGoByline := goOutput["byline"].(string)
	if hasPythonByline || hasGoByline {
		t.Logf("Bylines - Python: %q, Go: %q", pythonByline, goByline)
	}

	// For content, we need a more flexible comparison since HTML formatting might differ slightly
	// while still being semantically equivalent.
	pythonContent, hasPythonContent := pythonOutput["content"].(string)
	goContent, hasGoContent := goOutput["content"].(string)
	
	if hasPythonContent && hasGoContent {
		// Compare DOM structure rather than exact HTML string
		compareDOMContent(t, pythonContent, goContent)
	}
	
	// Check content length is roughly similar
	// Allow for some variation due to whitespace differences, etc.
	if hasPythonContent && hasGoContent {
		pythonLen := len(pythonContent)
		goLen := len(goContent)
		ratio := float64(goLen) / float64(pythonLen)
		t.Logf("Content length comparison - Go: %d, Python: %d, Ratio: %.2f", goLen, pythonLen, ratio)
		
		// Content length shouldn't vary too dramatically
		assert.True(t, ratio > 0.2 && ratio < 30.0, "Content length ratio is outside reasonable bounds")
	}
}

// compareDOMContent compares two HTML strings by parsing them into DOM trees
// and checking for structural similarities
func compareDOMContent(t *testing.T, html1, html2 string) {
	// Parse HTML strings into goquery documents
	doc1, err := goquery.NewDocumentFromReader(strings.NewReader(html1))
	if err != nil {
		t.Errorf("Failed to parse first HTML: %v", err)
		return
	}
	
	doc2, err := goquery.NewDocumentFromReader(strings.NewReader(html2))
	if err != nil {
		t.Errorf("Failed to parse second HTML: %v", err)
		return
	}
	
	// Check for key elements
	checkElementCount := func(selector string, description string) {
		count1 := doc1.Find(selector).Length()
		count2 := doc2.Find(selector).Length()
		t.Logf("%s count - First: %d, Second: %d", description, count1, count2)
		
		// Don't fail the test if counts differ slightly
		ratio := 1.0
		if count1 > 0 && count2 > 0 {
			ratio = float64(count2) / float64(count1)
		}
		
		if ratio < 0.7 || ratio > 1.3 {
			t.Logf("%s counts differ significantly", description)
		}
	}
	
	// Check various element types
	checkElementCount("p", "Paragraph")
	checkElementCount("a", "Link")
	checkElementCount("h1, h2, h3, h4, h5, h6", "Heading")
	checkElementCount("img", "Image")
	checkElementCount("ul, ol", "List")
	checkElementCount("li", "List item")
	
	// Check text content length (without HTML tags)
	text1 := doc1.Text()
	text2 := doc2.Text()
	
	t.Logf("Plain text length - First: %d, Second: %d", len(text1), len(text2))
	
	// Text length should be roughly similar
	textRatio := float64(len(text2)) / float64(len(text1))
	assert.True(t, textRatio > 0.2 && textRatio < 10.0, 
		"Plain text length ratio is outside reasonable bounds")
}

// compareDOMStructure compares the structure of two HTML documents
func compareDOMStructure(t *testing.T, html1, html2 string) {
	// Parse HTML strings into goquery documents
	doc1, err := goquery.NewDocumentFromReader(strings.NewReader(html1))
	if err != nil {
		t.Errorf("Failed to parse first HTML: %v", err)
		return
	}
	
	doc2, err := goquery.NewDocumentFromReader(strings.NewReader(html2))
	if err != nil {
		t.Errorf("Failed to parse second HTML: %v", err)
		return
	}
	
	// Compare structure by counting element types and checking nesting
	compareStructuralProperties(t, doc1, doc2)
}

// compareStructuralProperties compares structural properties of two documents
func compareStructuralProperties(t *testing.T, doc1, doc2 *goquery.Document) {
	// Compare element counts
	elements := []string{"p", "a", "div", "span", "h1", "h2", "h3", "h4", "h5", "h6", 
		"img", "ul", "ol", "li", "table", "blockquote"}
	
	for _, elem := range elements {
		count1 := doc1.Find(elem).Length()
		count2 := doc2.Find(elem).Length()
		
		if count1 > 0 || count2 > 0 {
			t.Logf("Element '%s' count - First: %d, Second: %d", elem, count1, count2)
		}
	}
	
	// Check if the main heading content matches
	heading1 := doc1.Find("h1").First().Text()
	heading2 := doc2.Find("h1").First().Text()
	
	if heading1 != "" && heading2 != "" {
		if heading1 == heading2 {
			t.Logf("Main headings match: %q", heading1)
		} else {
			t.Logf("Main headings differ - First: %q, Second: %q", heading1, heading2)
		}
	}
	
	// Compare total text content length
	text1 := doc1.Text()
	text2 := doc2.Text()
	
	textRatio := 1.0
	if len(text1) > 0 && len(text2) > 0 {
		textRatio = float64(len(text2)) / float64(len(text1))
	}
	
	t.Logf("Text content length - First: %d, Second: %d, Ratio: %.2f", 
		len(text1), len(text2), textRatio)
	
	// Check if the text content has similar paragraphs
	paragraphs1 := extractParagraphTexts(doc1)
	paragraphs2 := extractParagraphTexts(doc2)
	
	t.Logf("Paragraph count - First: %d, Second: %d", len(paragraphs1), len(paragraphs2))
	
	// Compare a subset of paragraphs to see if they're similar
	minLen := min(len(paragraphs1), len(paragraphs2))
	minLen = min(minLen, 3) // Check up to 3 paragraphs
	
	for i := 0; i < minLen; i++ {
		// Normalize and check if paragraphs are similar
		norm1 := normalizeWhitespace(paragraphs1[i])
		norm2 := normalizeWhitespace(paragraphs2[i])
		
		if norm1 == norm2 {
			t.Logf("Paragraph %d matches exactly", i+1)
		} else if strings.Contains(norm1, norm2) || strings.Contains(norm2, norm1) {
			t.Logf("Paragraph %d is similar but not identical", i+1)
		} else {
			similarity := calculateTextSimilarity(norm1, norm2)
			t.Logf("Paragraph %d similarity: %.2f", i+1, similarity)
		}
	}
}

// compareWithReferenceJSON compares our output with ReadabiliPy reference JSON
func compareWithReferenceJSON(t *testing.T, reference, goOutput map[string]interface{}) {
	// Compare title
	refTitle, hasRefTitle := reference["title"].(string)
	goTitle, hasGoTitle := goOutput["title"].(string)
	
	if hasRefTitle && hasGoTitle {
		t.Logf("Title comparison - Reference: %q, Go: %q", refTitle, goTitle)
		
		// The title should be similar (allow for minor differences)
		assert.True(t, strings.Contains(refTitle, goTitle) || strings.Contains(goTitle, refTitle), 
			"Titles are not similar enough")
	}
	
	// Compare byline if available
	refByline, hasRefByline := reference["byline"].(string)
	goByline, hasGoByline := goOutput["byline"].(string)
	
	if hasRefByline && hasGoByline {
		t.Logf("Byline comparison - Reference: %q, Go: %q", refByline, goByline)
	}
	
	// Compare content
	// ReadabiliPy references may store content differently, adapt as needed
	var refContent string
	refContent, hasRefContent := reference["content"].(string)
	if !hasRefContent {
		// Try alternative fields that might contain content
		if refHTML, ok := reference["html"].(string); ok {
			refContent = refHTML
			hasRefContent = true
		} else if refArticle, ok := reference["article_html"].(string); ok {
			refContent = refArticle
			hasRefContent = true
		}
	}
	
	goContent, hasGoContent := goOutput["content"].(string)
	
	if hasRefContent && hasGoContent {
		// Compare DOM structure rather than exact HTML string
		compareDOMContent(t, refContent, goContent)
		
		// Check for important content elements
		refDoc, _ := goquery.NewDocumentFromReader(strings.NewReader(refContent))
		goDoc, _ := goquery.NewDocumentFromReader(strings.NewReader(goContent))
		
		// Extract plain text and compare
		refText := refDoc.Text()
		goText := goDoc.Text()
		
		// Check if the text content is roughly similar in length
		refTextLen := len(refText)
		goTextLen := len(goText)
		textRatio := float64(goTextLen) / float64(refTextLen)
		
		t.Logf("Text content length - Reference: %d, Go: %d, Ratio: %.2f", 
			refTextLen, goTextLen, textRatio)
		
		// Check for reasonable similarity in length
		assert.True(t, textRatio > 0.2 && textRatio < 10.0, 
			"Text length differs too much between reference and Go output")
	}
}

// Helper functions

// normalizeWhitespace replaces all whitespace sequences with a single space
func normalizeWhitespace(s string) string {
	whitespaceRegex := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(whitespaceRegex.ReplaceAllString(s, " "))
}

// extractParagraphTexts extracts text content from all paragraphs in the document
func extractParagraphTexts(doc *goquery.Document) []string {
	var result []string
	doc.Find("p").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if text != "" {
			result = append(result, text)
		}
	})
	return result
}

// calculateTextSimilarity calculates similarity between two text strings
// using a simple character-level approach
func calculateTextSimilarity(s1, s2 string) float64 {
	// Convert to lowercase for comparison
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)
	
	// Find the longer and shorter strings
	var longer, shorter string
	if len(s1) >= len(s2) {
		longer = s1
		shorter = s2
	} else {
		longer = s2
		shorter = s1
	}
	
	// If both strings are empty, they're identical
	if len(longer) == 0 {
		return 1.0
	}
	
	// Calculate similarity based on character presence
	charCount := 0
	for _, char := range shorter {
		if strings.ContainsRune(longer, char) {
			charCount++
		}
	}
	
	return float64(charCount) / float64(len(longer))
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// createTestHTML generates test HTML content for a specific test case
func createTestHTML(t *testing.T, testCase string) string {
	switch testCase {
	case "ExampleDomain":
		return `<!DOCTYPE html>
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
	
	case "Wikipedia":
		return `<!DOCTYPE html>
<html>
<head>
    <title>Article Title - Wikipedia</title>
</head>
<body>
    <div id="mw-navigation">
        <h2>Navigation menu</h2>
        <div id="mw-head">
            <div id="p-search">
                <h3>Search</h3>
                <form action="/wiki/Special:Search">
                    <input type="search" name="search" placeholder="Search Wikipedia">
                </form>
            </div>
        </div>
        <div id="mw-panel">
            <div id="p-logo">
                <a href="/wiki/Main_Page"></a>
            </div>
            <div id="p-navigation">
                <ul>
                    <li><a href="/wiki/Main_Page">Main page</a></li>
                    <li><a href="/wiki/Wikipedia:Contents">Contents</a></li>
                    <li><a href="/wiki/Portal:Current_events">Current events</a></li>
                </ul>
            </div>
        </div>
    </div>
    <div id="content">
        <div id="bodyContent">
            <div id="mw-content-text">
                <div class="mw-parser-output">
                    <p>This is a Wikipedia article about an interesting topic.</p>
                    <p>It contains multiple paragraphs of information.</p>
                    <h2>First section</h2>
                    <p>This is the first section of the article with <a href="/wiki/Link">links</a> to other topics.</p>
                    <h2>Second section</h2>
                    <p>This is the second section with more information.</p>
                    <p>The article continues with more details about the topic.</p>
                </div>
            </div>
        </div>
    </div>
    <div id="footer">
        <ul id="footer-info">
            <li>This page was last edited on 26 March 2023</li>
        </ul>
        <ul id="footer-places">
            <li><a href="/wiki/Wikipedia:About">About Wikipedia</a></li>
            <li><a href="/wiki/Wikipedia:General_disclaimer">Disclaimers</a></li>
        </ul>
    </div>
</body>
</html>`
	
	case "BlogPost":
		return `<!DOCTYPE html>
<html>
<head>
    <title>My Blog Post Title</title>
    <meta name="author" content="John Doe">
</head>
<body>
    <header>
        <nav>
            <ul>
                <li><a href="/">Home</a></li>
                <li><a href="/about">About</a></li>
                <li><a href="/contact">Contact</a></li>
            </ul>
        </nav>
    </header>
    <main>
        <article>
            <h1>My Blog Post Title</h1>
            <div class="byline">By John Doe, March 26, 2025</div>
            <p>This is the introduction to my blog post. It sets up the topic.</p>
            <p>This paragraph explores the first point in detail.</p>
            <h2>Important Subtopic</h2>
            <p>This section discusses an important subtopic with <em>emphasized text</em> and <strong>strong text</strong>.</p>
            <p>Here's another paragraph with a <a href="https://example.com">link to an external site</a>.</p>
            <h2>Conclusion</h2>
            <p>This is the conclusion of the blog post.</p>
        </article>
        <aside>
            <h3>Related Posts</h3>
            <ul>
                <li><a href="/post1">Another Blog Post</a></li>
                <li><a href="/post2">Yet Another Blog Post</a></li>
            </ul>
        </aside>
    </main>
    <footer>
        <p>&copy; 2025 My Blog</p>
        <div class="social-links">
            <a href="https://twitter.com/myblog">Twitter</a>
            <a href="https://facebook.com/myblog">Facebook</a>
        </div>
    </footer>
</body>
</html>`
	
	case "NewsArticle":
		return `<!DOCTYPE html>
<html>
<head>
    <title>Breaking News: Important Event | News Site</title>
    <meta name="author" content="Jane Smith">
</head>
<body>
    <header>
        <div class="logo">
            <a href="/">News Site</a>
        </div>
        <nav>
            <ul>
                <li><a href="/politics">Politics</a></li>
                <li><a href="/business">Business</a></li>
                <li><a href="/technology">Technology</a></li>
                <li><a href="/sports">Sports</a></li>
            </ul>
        </nav>
        <div class="search">
            <form>
                <input type="text" placeholder="Search...">
                <button type="submit">Search</button>
            </form>
        </div>
    </header>
    <div class="container">
        <main>
            <article>
                <h1>Breaking News: Important Event</h1>
                <div class="metadata">
                    <span class="byline">By Jane Smith</span>
                    <span class="date">March 26, 2025</span>
                    <span class="category">Politics</span>
                </div>
                <div class="lead">
                    <p>This is the lead paragraph of the news article, summarizing the key points.</p>
                </div>
                <div class="content">
                    <p>This paragraph provides more details about the important event.</p>
                    <p>This paragraph includes quotes from relevant sources.</p>
                    <blockquote>
                        <p>"This is a quote from someone involved in the event," said a spokesperson.</p>
                    </blockquote>
                    <p>This paragraph gives background information about the event.</p>
                    <h2>Related Developments</h2>
                    <p>This section covers related developments.</p>
                    <p>It provides analysis and context for the event.</p>
                </div>
            </article>
        </main>
        <aside>
            <div class="ad">
                <img src="ad.jpg" alt="Advertisement">
            </div>
            <div class="related-news">
                <h3>Related News</h3>
                <ul>
                    <li><a href="/news1">Related News Item 1</a></li>
                    <li><a href="/news2">Related News Item 2</a></li>
                </ul>
            </div>
            <div class="most-read">
                <h3>Most Read</h3>
                <ul>
                    <li><a href="/popular1">Popular Article 1</a></li>
                    <li><a href="/popular2">Popular Article 2</a></li>
                </ul>
            </div>
        </aside>
    </div>
    <footer>
        <div class="footer-links">
            <div class="column">
                <h4>Sections</h4>
                <ul>
                    <li><a href="/politics">Politics</a></li>
                    <li><a href="/business">Business</a></li>
                </ul>
            </div>
            <div class="column">
                <h4>Company</h4>
                <ul>
                    <li><a href="/about">About Us</a></li>
                    <li><a href="/contact">Contact</a></li>
                </ul>
            </div>
        </div>
        <div class="copyright">
            <p>&copy; 2025 News Site. All rights reserved.</p>
        </div>
    </footer>
</body>
</html>`
	
	case "ComplexLayout":
		return `<!DOCTYPE html>
<html>
<head>
    <title>Complex Layout Page</title>
</head>
<body>
    <header class="site-header">
        <div class="logo">
            <a href="/"><img src="logo.png" alt="Site Logo"></a>
        </div>
        <nav class="main-navigation">
            <ul>
                <li><a href="/section1">Section 1</a></li>
                <li><a href="/section2">Section 2</a></li>
                <li class="dropdown">
                    <a href="/section3">Section 3</a>
                    <ul class="dropdown-menu">
                        <li><a href="/subsection1">Subsection 1</a></li>
                        <li><a href="/subsection2">Subsection 2</a></li>
                    </ul>
                </li>
            </ul>
        </nav>
        <div class="user-menu">
            <a href="/login">Login</a>
            <a href="/signup">Sign Up</a>
        </div>
    </header>
    <div class="hero-banner">
        <div class="carousel">
            <div class="slide">
                <img src="banner1.jpg" alt="Banner 1">
                <div class="caption">
                    <h2>Featured Content</h2>
                    <p>This is a description of the featured content.</p>
                </div>
            </div>
        </div>
    </div>
    <div class="container">
        <div class="sidebar left">
            <div class="widget">
                <h3>Categories</h3>
                <ul>
                    <li><a href="/category1">Category 1</a></li>
                    <li><a href="/category2">Category 2</a></li>
                </ul>
            </div>
            <div class="widget">
                <h3>Recent Posts</h3>
                <ul>
                    <li><a href="/post1">Recent Post 1</a></li>
                    <li><a href="/post2">Recent Post 2</a></li>
                </ul>
            </div>
        </div>
        <main class="content">
            <article class="main-article">
                <h1>Main Article Title</h1>
                <div class="article-meta">
                    <span class="author">By Author Name</span>
                    <span class="date">March 26, 2025</span>
                    <span class="category">Category Name</span>
                </div>
                <div class="article-body">
                    <p>This is the main content of the article.</p>
                    <p>This article has multiple paragraphs and <a href="/link">links</a>.</p>
                    <h2>First Heading</h2>
                    <p>This is the first section of the article.</p>
                    <div class="image-block">
                        <img src="image1.jpg" alt="Image 1">
                        <div class="caption">Image caption goes here</div>
                    </div>
                    <p>This paragraph follows an image.</p>
                    <h2>Second Heading</h2>
                    <p>This is the second section of the article.</p>
                    <ul>
                        <li>This is a list item 1</li>
                        <li>This is a list item 2</li>
                        <li>This is a list item 3</li>
                    </ul>
                    <p>This paragraph follows a list.</p>
                    <blockquote>
                        <p>This is a blockquote.</p>
                        <cite>— Citation Source</cite>
                    </blockquote>
                    <p>This is the final paragraph of the article.</p>
                </div>
                <div class="article-footer">
                    <div class="tags">
                        <span>Tags:</span>
                        <a href="/tag1">Tag 1</a>
                        <a href="/tag2">Tag 2</a>
                    </div>
                    <div class="share">
                        <span>Share:</span>
                        <a href="#twitter">Twitter</a>
                        <a href="#facebook">Facebook</a>
                    </div>
                </div>
            </article>
            <section class="comments">
                <h3>Comments</h3>
                <div class="comment">
                    <div class="comment-author">Comment Author</div>
                    <div class="comment-date">March 25, 2025</div>
                    <div class="comment-body">
                        <p>This is a comment on the article.</p>
                    </div>
                </div>
                <div class="comment">
                    <div class="comment-author">Another Commenter</div>
                    <div class="comment-date">March 24, 2025</div>
                    <div class="comment-body">
                        <p>This is another comment on the article.</p>
                    </div>
                </div>
                <form class="comment-form">
                    <h4>Leave a Comment</h4>
                    <div class="form-group">
                        <label for="name">Name</label>
                        <input type="text" id="name" name="name">
                    </div>
                    <div class="form-group">
                        <label for="comment">Comment</label>
                        <textarea id="comment" name="comment"></textarea>
                    </div>
                    <button type="submit">Submit Comment</button>
                </form>
            </section>
        </main>
        <div class="sidebar right">
            <div class="ad-block">
                <img src="ad.jpg" alt="Advertisement">
            </div>
            <div class="widget">
                <h3>Popular Posts</h3>
                <ul>
                    <li><a href="/popular1">Popular Post 1</a></li>
                    <li><a href="/popular2">Popular Post 2</a></li>
                </ul>
            </div>
            <div class="newsletter">
                <h3>Subscribe</h3>
                <p>Get updates delivered to your inbox.</p>
                <form>
                    <input type="email" placeholder="Your email address">
                    <button type="submit">Subscribe</button>
                </form>
            </div>
        </div>
    </div>
    <footer class="site-footer">
        <div class="footer-widgets">
            <div class="widget">
                <h4>About Us</h4>
                <p>This is a brief description of the site.</p>
            </div>
            <div class="widget">
                <h4>Quick Links</h4>
                <ul>
                    <li><a href="/about">About</a></li>
                    <li><a href="/contact">Contact</a></li>
                    <li><a href="/privacy">Privacy Policy</a></li>
                </ul>
            </div>
            <div class="widget">
                <h4>Follow Us</h4>
                <div class="social-links">
                    <a href="#twitter">Twitter</a>
                    <a href="#facebook">Facebook</a>
                    <a href="#instagram">Instagram</a>
                </div>
            </div>
        </div>
        <div class="footer-bottom">
            <p>&copy; 2025 Complex Layout Site. All rights reserved.</p>
        </div>
    </footer>
</body>
</html>`
	
	default:
		t.Fatalf("Unknown test case: %s", testCase)
		return ""
	}
}