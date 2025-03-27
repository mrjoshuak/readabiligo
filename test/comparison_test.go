package test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mrjoshuak/readabiligo/extractor"
	"github.com/mrjoshuak/readabiligo/internal/readability"
	"github.com/stretchr/testify/assert"
)

// TestComparisonWithPython tests that our Go implementation produces
// identical results to the Python implementation
func TestComparisonWithPython(t *testing.T) {
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
	tempDir, err := ioutil.TempDir("", "readabiligo-comparison")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Run comparisons for each test case
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test HTML file for this case
			htmlContent := createTestHTML(t, tc.name)
			htmlPath := filepath.Join(tempDir, tc.htmlFile)
			err := ioutil.WriteFile(htmlPath, []byte(htmlContent), 0644)
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
	outputFile, err := ioutil.TempFile("", "readabilipy-output-*.json")
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
	scriptFile, err := ioutil.TempFile("", "readabilipy-script-*.py")
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
	outputData, err := ioutil.ReadFile(outputFile.Name())
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
		extractor.WithReadability(false),
		extractor.WithContentDigests(false),
	)

	// Extract the article
	article, err := ex.ExtractFromHTML(html, nil)
	if err != nil {
		return nil, err
	}

	// Convert to map for comparison
	result := map[string]interface{}{
		"title":   article.Title,
		"byline":  article.Byline,
		"content": article.Content,
	}

	return result, nil
}

// assertEqualOutput compares the Python and Go outputs
func assertEqualOutput(t *testing.T, pythonOutput, goOutput map[string]interface{}) {
	// Compare titles
	assert.Equal(t, pythonOutput["title"], goOutput["title"], "Titles should match")

	// Compare bylines if both are present
	if pythonByline, ok := pythonOutput["byline"].(string); ok && pythonByline != "" {
		assert.Equal(t, pythonByline, goOutput["byline"], "Bylines should match")
	}

	// For content, we need a more flexible comparison since HTML formatting might differ slightly
	// while still being semantically equivalent. Here we'll just check for key elements.
	pythonContent := pythonOutput["content"].(string)
	goContent := goOutput["content"].(string)
	
	// Make sure key content elements are present in both
	// This is a simplification - a real test would need to parse and compare the DOM trees
	assert.Contains(t, goContent, "<h1>", "Go output should contain h1 headers")
	assert.Contains(t, pythonContent, "<h1>", "Python output should contain h1 headers")
	
	// Check for paragraphs
	assert.Contains(t, goContent, "<p>", "Go output should contain paragraphs")
	assert.Contains(t, pythonContent, "<p>", "Python output should contain paragraphs")
	
	// Check for links
	if pythonContent != "" && goContent != "" {
		assert.Contains(t, goContent, "<a ", "Go output should contain links")
		assert.Contains(t, pythonContent, "<a ", "Python output should contain links")
	}
	
	// Check content length is roughly similar
	// Allow for some variation due to whitespace differences, etc.
	pythonLen := len(pythonContent)
	goLen := len(goContent)
	ratio := float64(goLen) / float64(pythonLen)
	assert.True(t, ratio > 0.8 && ratio < 1.2, "Content length should be similar (ratio: %f)", ratio)
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