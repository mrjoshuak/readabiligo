package readability

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected ContentType
	}{
		{
			name: "Error Page Detection",
			html: `
				<!DOCTYPE html>
				<html>
				<head>
					<title>404 - Page Not Found</title>
				</head>
				<body>
					<h1>Page Not Found</h1>
					<p>The page you were looking for does not exist.</p>
					<a href="/">Go back to homepage</a>
				</body>
				</html>
			`,
			expected: ContentTypeError,
		},
		{
			name: "Wikipedia Reference Content",
			html: `
				<!DOCTYPE html>
				<html>
				<head>
					<title>Article Title - Wikipedia</title>
				</head>
				<body>
					<div id="mw-content-text">
						<div class="infobox">Info content</div>
						<div id="toc">Table of Contents</div>
						<p>Article content with <a class="citation">citation</a>.</p>
						<div class="references">
							<ol>
								<li>Reference 1</li>
								<li>Reference 2</li>
							</ol>
						</div>
						<span class="mw-editsection">[edit]</span>
					</div>
				</body>
				</html>
			`,
			expected: ContentTypeReference,
		},
		{
			name: "Technical Blog Content",
			html: `
				<!DOCTYPE html>
				<html>
				<head>
					<title>Technical Blog Post</title>
				</head>
				<body>
					<article>
						<h1>How to Use Go Modules</h1>
						<div class="byline">By Author - January 1, 2023</div>
						<p>This is a technical article about Go modules.</p>
						<pre>
							<code>
								func main() {
									fmt.Println("Hello, world!")
								}
							</code>
						</pre>
						<p>More content with code examples.</p>
						<div class="code">
								import "fmt"
								
								func Example() {
									// This is a code example
								}
						</div>
					</article>
				</body>
				</html>
			`,
			expected: ContentTypeTechnical,
		},
		{
			name: "Standard Article",
			html: `
				<!DOCTYPE html>
				<html>
				<head>
					<title>News Article</title>
				</head>
				<body>
					<article>
						<h1>Breaking News</h1>
						<div class="byline">By Reporter - January 1, 2023</div>
						<p>This is a news article with content.</p>
						<p>Second paragraph with more details.</p>
						<p>Third paragraph with conclusion.</p>
					</article>
				</body>
				</html>
			`,
			expected: ContentTypeArticle,
		},
		{
			name: "Minimal Login Page",
			html: `
				<!DOCTYPE html>
				<html>
				<head>
					<title>Login</title>
				</head>
				<body>
					<div class="login-container">
						<h1>Login</h1>
						<form action="/login" method="post">
							<input type="text" name="username" placeholder="Username">
							<input type="password" name="password" placeholder="Password">
							<button type="submit">Login</button>
						</form>
						<a href="/forgot-password">Forgot password?</a>
					</div>
				</body>
				</html>
			`,
			expected: ContentTypeMinimal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.html))
			if err != nil {
				t.Fatalf("Error creating document: %v", err)
			}

			contentType := DetectContentType(doc)
			if contentType != tt.expected {
				t.Errorf("Expected content type %s, got %s", tt.expected, contentType)
			}
		})
	}
}

func TestContentTypeAwareExtraction(t *testing.T) {
	tests := []struct {
		name             string
		html             string
		forceContentType ContentType
		detectEnabled    bool
	}{
		{
			name: "Error Page With Detection",
			html: `
				<!DOCTYPE html>
				<html>
				<head>
					<title>404 - Page Not Found</title>
				</head>
				<body>
					<h1>Page Not Found</h1>
					<p>The page you were looking for does not exist.</p>
					<nav>
						<ul>
							<li><a href="/">Home</a></li>
							<li><a href="/about">About</a></li>
							<li><a href="/contact">Contact</a></li>
						</ul>
					</nav>
				</body>
				</html>
			`,
			detectEnabled: true,
		},
		{
			name: "Error Page With Forced Type",
			html: `
				<!DOCTYPE html>
				<html>
				<head>
					<title>404 - Page Not Found</title>
				</head>
				<body>
					<h1>Page Not Found</h1>
					<p>The page you were looking for does not exist.</p>
					<nav>
						<ul>
							<li><a href="/">Home</a></li>
							<li><a href="/about">About</a></li>
							<li><a href="/contact">Contact</a></li>
						</ul>
					</nav>
				</body>
				</html>
			`,
			forceContentType: ContentTypeError,
			detectEnabled:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := defaultReadabilityOptions()
			opts.DetectContentType = tt.detectEnabled
			if !tt.detectEnabled {
				opts.ContentType = tt.forceContentType
			}

			article, err := ParseHTML(tt.html, &opts)
			if err != nil {
				t.Fatalf("Error parsing HTML: %v", err)
			}

			// Just verify the article was extracted and the content type is set
			if article == nil {
				t.Fatal("Expected article to be extracted, got nil")
			}

			if !tt.detectEnabled && article.ContentType != tt.forceContentType {
				t.Errorf("Expected content type %s, got %s", tt.forceContentType, article.ContentType)
			}
		})
	}
}