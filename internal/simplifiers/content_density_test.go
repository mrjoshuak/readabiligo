package simplifiers

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestAnalyzeContentDensity(t *testing.T) {
	html := `
		<div id="content">
			<p>This is a paragraph with some text.</p>
			<p>This is another paragraph with more text for testing.</p>
			<h2>This is a heading</h2>
			<ul>
				<li>List item 1</li>
				<li>List item 2</li>
			</ul>
		</div>
		<div id="sidebar">
			<p>This is a sidebar.</p>
		</div>
	`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	// Test content div
	contentDiv := doc.Find("#content")
	contentDensity := AnalyzeContentDensity(contentDiv)
	if contentDensity <= 0 {
		t.Errorf("Expected positive content density for content div, got %f", contentDensity)
	}

	// Test sidebar div
	sidebarDiv := doc.Find("#sidebar")
	sidebarDensity := AnalyzeContentDensity(sidebarDiv)
	if sidebarDensity <= 0 {
		t.Errorf("Expected positive content density for sidebar div, got %f", sidebarDensity)
	}

	// Content div should have higher density than sidebar
	if contentDensity <= sidebarDensity {
		t.Errorf("Expected content div to have higher density than sidebar, got %f <= %f", contentDensity, sidebarDensity)
	}
}

func TestCalculateContentScore(t *testing.T) {
	html := `
		<div id="content">
			<h1>Main Article Heading</h1>
			<p>This is a paragraph with some text.</p>
			<p>This is another paragraph with more text for testing.</p>
			<h2>This is a subheading</h2>
			<p>More content here.</p>
			<ul>
				<li>List item 1</li>
				<li>List item 2</li>
			</ul>
			<figure>
				<img src="image.jpg" alt="An image">
				<figcaption>Image caption</figcaption>
			</figure>
		</div>
		<div id="sidebar">
			<h3>Sidebar</h3>
			<p>This is a sidebar.</p>
			<ul>
				<li><a href="#">Link 1</a></li>
				<li><a href="#">Link 2</a></li>
				<li><a href="#">Link 3</a></li>
			</ul>
		</div>
		<div id="comments">
			<h3>Comments</h3>
			<div class="comment">
				<p>This is a comment.</p>
			</div>
			<div class="comment">
				<p>This is another comment.</p>
			</div>
		</div>
	`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	// Test content div
	contentDiv := doc.Find("#content")
	contentScore := CalculateContentScore(contentDiv)
	if contentScore <= 0 {
		t.Errorf("Expected positive content score for content div, got %f", contentScore)
	}

	// Test sidebar div
	sidebarDiv := doc.Find("#sidebar")
	sidebarScore := CalculateContentScore(sidebarDiv)
	if sidebarScore <= 0 {
		t.Errorf("Expected positive content score for sidebar div, got %f", sidebarScore)
	}

	// Test comments div
	commentsDiv := doc.Find("#comments")
	commentsScore := CalculateContentScore(commentsDiv)
	if commentsScore <= 0 {
		t.Errorf("Expected positive content score for comments div, got %f", commentsScore)
	}

	// Content div should have higher score than sidebar and comments
	if contentScore <= sidebarScore {
		t.Errorf("Expected content div to have higher score than sidebar, got %f <= %f", contentScore, sidebarScore)
	}
	if contentScore <= commentsScore {
		t.Errorf("Expected content div to have higher score than comments, got %f <= %f", contentScore, commentsScore)
	}
}

func TestFindMainContentNode(t *testing.T) {
	html := `
		<html>
		<body>
			<header>
				<h1>Website Title</h1>
				<nav>
					<ul>
						<li><a href="#">Home</a></li>
						<li><a href="#">About</a></li>
					</ul>
				</nav>
			</header>
			<main id="content">
				<article>
					<h1>Main Article Heading</h1>
					<p>This is a paragraph with some text.</p>
					<p>This is another paragraph with more text for testing.</p>
					<h2>This is a subheading</h2>
					<p>More content here.</p>
					<ul>
						<li>List item 1</li>
						<li>List item 2</li>
					</ul>
				</article>
			</main>
			<aside id="sidebar">
				<h3>Sidebar</h3>
				<p>This is a sidebar.</p>
				<ul>
					<li><a href="#">Link 1</a></li>
					<li><a href="#">Link 2</a></li>
					<li><a href="#">Link 3</a></li>
				</ul>
			</aside>
			<footer>
				<p>Copyright 2025</p>
			</footer>
		</body>
		</html>
	`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	// Find the main content node
	mainContent := FindMainContentNode(doc)
	if mainContent.Length() == 0 {
		t.Errorf("Failed to find main content node")
	}

	// Check if the main content node is the expected one
	mainContentID, exists := mainContent.Attr("id")
	if !exists || mainContentID != "content" {
		// If it's not the #content element, it should at least be the article or main element
		tagName := goquery.NodeName(mainContent)
		if tagName != "article" && tagName != "main" {
			t.Errorf("Expected main content node to be #content, article, or main, got %s", tagName)
		}
	}
}

func TestCalculateLinkDensityScore(t *testing.T) {
	html := `
		<div id="content">
			<p>This is a paragraph with some text.</p>
			<p>This is another paragraph with more text for testing.</p>
		</div>
		<div id="links">
			<p>This is a paragraph with <a href="#">a link</a>.</p>
			<p>This is another paragraph with <a href="#">another link</a>.</p>
		</div>
		<div id="all-links">
			<a href="#">Link 1</a>
			<a href="#">Link 2</a>
			<a href="#">Link 3</a>
		</div>
	`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	// Test content div (no links)
	contentDiv := doc.Find("#content")
	contentScore := CalculateLinkDensityScore(contentDiv)
	if contentScore < 0.9 {
		t.Errorf("Expected high link density score for content div (no links), got %f", contentScore)
	}

	// Test links div (some links)
	linksDiv := doc.Find("#links")
	linksScore := CalculateLinkDensityScore(linksDiv)
	if linksScore < 0.5 || linksScore > 0.9 {
		t.Errorf("Expected medium link density score for links div, got %f", linksScore)
	}

	// Test all-links div (all links)
	allLinksDiv := doc.Find("#all-links")
	allLinksScore := CalculateLinkDensityScore(allLinksDiv)
	if allLinksScore > 0.5 {
		t.Errorf("Expected low link density score for all-links div, got %f", allLinksScore)
	}
}

func TestCalculateHeadingDensity(t *testing.T) {
	html := `
		<div id="content">
			<p>This is a paragraph with some text.</p>
			<p>This is another paragraph with more text for testing.</p>
		</div>
		<div id="headings">
			<h1>Heading 1</h1>
			<h2>Heading 2</h2>
			<h3>Heading 3</h3>
			<p>This is a paragraph.</p>
		</div>
	`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	// Test content div (no headings)
	contentDiv := doc.Find("#content")
	contentDensity := CalculateHeadingDensity(contentDiv)
	if contentDensity != 0 {
		t.Errorf("Expected zero heading density for content div (no headings), got %f", contentDensity)
	}

	// Test headings div (3 headings)
	headingsDiv := doc.Find("#headings")
	headingsDensity := CalculateHeadingDensity(headingsDiv)
	if headingsDensity <= 0 {
		t.Errorf("Expected positive heading density for headings div, got %f", headingsDensity)
	}
}

func TestCalculateListDensity(t *testing.T) {
	html := `
		<div id="content">
			<p>This is a paragraph with some text.</p>
			<p>This is another paragraph with more text for testing.</p>
		</div>
		<div id="lists">
			<ul>
				<li>List item 1</li>
				<li>List item 2</li>
				<li>List item 3</li>
			</ul>
			<ol>
				<li>Ordered list item 1</li>
				<li>Ordered list item 2</li>
			</ol>
			<p>This is a paragraph.</p>
		</div>
	`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	// Test content div (no lists)
	contentDiv := doc.Find("#content")
	contentDensity := CalculateListDensity(contentDiv)
	if contentDensity != 0 {
		t.Errorf("Expected zero list density for content div (no lists), got %f", contentDensity)
	}

	// Test lists div (5 list items)
	listsDiv := doc.Find("#lists")
	listsDensity := CalculateListDensity(listsDiv)
	if listsDensity <= 0 {
		t.Errorf("Expected positive list density for lists div, got %f", listsDensity)
	}
}

func TestCalculateImageDensity(t *testing.T) {
	html := `
		<div id="content">
			<p>This is a paragraph with some text.</p>
			<p>This is another paragraph with more text for testing.</p>
		</div>
		<div id="images">
			<img src="image1.jpg" alt="Image 1">
			<img src="image2.jpg" alt="Image 2">
			<img src="image3.jpg" alt="Image 3">
			<p>This is a paragraph.</p>
		</div>
	`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	// Test content div (no images)
	contentDiv := doc.Find("#content")
	contentDensity := CalculateImageDensity(contentDiv)
	if contentDensity != 0 {
		t.Errorf("Expected zero image density for content div (no images), got %f", contentDensity)
	}

	// Test images div (3 images)
	imagesDiv := doc.Find("#images")
	imagesDensity := CalculateImageDensity(imagesDiv)
	if imagesDensity <= 0 {
		t.Errorf("Expected positive image density for images div, got %f", imagesDensity)
	}
}
