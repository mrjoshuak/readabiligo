// Package readability provides a pure Go implementation of Mozilla's Readability.js
// for extracting the main content from web pages.
package readability

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ContentType defines the type of content in a document,
// which helps optimize the extraction algorithm.
type ContentType int

// Content type constants
const (
	ContentTypeUnknown ContentType = iota
	ContentTypeReference  // Wikipedia, documentation
	ContentTypeArticle    // News, blog posts
	ContentTypeTechnical  // Code examples, tech blogs
	ContentTypeError      // Error pages, 404s
	ContentTypeMinimal    // Login pages, etc.
)

// String returns a string representation of the content type
func (ct ContentType) String() string {
	switch ct {
	case ContentTypeReference:
		return "Reference"
	case ContentTypeArticle:
		return "Article"
	case ContentTypeTechnical:
		return "Technical"
	case ContentTypeError:
		return "Error"
	case ContentTypeMinimal:
		return "Minimal"
	default:
		return "Unknown"
	}
}

// DetectContentType analyzes a document to determine its content type
func DetectContentType(doc *goquery.Document) ContentType {
	// Check for error page indicators first
	if hasErrorPageIndicators(doc) {
		return ContentTypeError
	}

	// Check for Wikipedia/reference structure
	if hasReferenceStructure(doc) {
		return ContentTypeReference
	}

	// Check for technical content with code blocks
	if hasTechnicalContent(doc) {
		return ContentTypeTechnical
	}

	// Check for article structure (news, blog posts)
	if hasArticleStructure(doc) {
		return ContentTypeArticle
	}

	// Check for minimal pages
	if hasMinimalContent(doc) {
		return ContentTypeMinimal
	}

	// Default to article if no specific type is detected
	return ContentTypeArticle
}

// hasErrorPageIndicators checks for signs that a page is an error page
func hasErrorPageIndicators(doc *goquery.Document) bool {
	// Look for common error page indicators in title
	title := strings.ToLower(doc.Find("title").Text())
	errorPhrases := []string{"404", "not found", "error", "page doesn't exist", "page not found"}
	for _, phrase := range errorPhrases {
		if strings.Contains(title, phrase) {
			return true
		}
	}

	// Check for error indicators in headings
	errorCount := 0
	doc.Find("h1, h2, h3").Each(func(_ int, s *goquery.Selection) {
		text := strings.ToLower(s.Text())
		for _, phrase := range errorPhrases {
			if strings.Contains(text, phrase) {
				errorCount++
			}
		}
	})
	if errorCount >= 1 {
		return true
	}

	// Check for error status code in meta tags
	statusCode := doc.Find("meta[name='status-code']").AttrOr("content", "")
	if statusCode == "404" || statusCode == "500" {
		return true
	}

	return false
}

// hasReferenceStructure checks if a document has Wikipedia-like reference structure
func hasReferenceStructure(doc *goquery.Document) bool {
	// Look for table of contents
	hasTOC := doc.Find("div#toc, div.toc, div#mw-content-text, div.mw-content-text, div#wiki-content").Length() > 0

	// Look for infoboxes (common in Wikipedia)
	hasInfobox := doc.Find("table.infobox, div.infobox, table.wikitable").Length() > 0

	// Look for citation notes
	hasCitations := doc.Find("div.reflist, div.references, ol.references").Length() > 0

	// Look for edit links (common in wikis)
	hasEditLinks := doc.Find("a[title*='edit'], a.edit, span.mw-editsection").Length() > 0

	// Combine signals
	referenceScore := 0
	if hasTOC {
		referenceScore += 2
	}
	if hasInfobox {
		referenceScore += 2
	}
	if hasCitations {
		referenceScore += 1
	}
	if hasEditLinks {
		referenceScore += 1
	}

	return referenceScore >= 3
}

// hasTechnicalContent checks if a document contains technical content like code examples
func hasTechnicalContent(doc *goquery.Document) bool {
	// Count code blocks
	codeElements := doc.Find("pre, code, pre code, div.code, div.highlight").Length()

	// Look for function signatures or programming language indicators
	techPatterns := []string{"function", "class", "def ", "var ", "const ", "import ", "public ", "private "}
	techCount := 0

	doc.Find("pre, code, .code, .syntax, .highlight").Each(func(_ int, s *goquery.Selection) {
		text := s.Text()
		for _, pattern := range techPatterns {
			if strings.Contains(text, pattern) {
				techCount++
				break
			}
		}
	})

	// Combined check
	return codeElements >= 3 || techCount >= 2
}

// hasArticleStructure checks if a document has a standard article structure
func hasArticleStructure(doc *goquery.Document) bool {
	// Look for article tags
	hasArticleTag := doc.Find("article").Length() > 0

	// Look for bylines
	hasByline := doc.Find("[class*='byline'], [class*='author'], [rel*='author']").Length() > 0

	// Look for dates
	hasDate := doc.Find("time, [class*='date'], [class*='time'], [property*='datePublished']").Length() > 0
	
	// Look for article containers
	hasArticleContainer := doc.Find(".article, .post, .story, .entry, .blog-entry, #article, #post").Length() > 0

	// Combined check
	return hasArticleTag || (hasByline && hasDate) || hasArticleContainer
}

// hasMinimalContent checks if a document has minimal content (login, signup, etc.)
func hasMinimalContent(doc *goquery.Document) bool {
	// Count form elements
	formElements := doc.Find("form, input, button[type='submit'], button[type='button']").Length()
	
	// Count paragraphs and content
	contentElements := doc.Find("p, article").Length()
	
	// Estimate total text content length
	totalText := strings.TrimSpace(doc.Find("body").Text())
	
	// If page has more forms than content and little text, it's likely minimal
	return (formElements > contentElements) && len(totalText) < 1000
}