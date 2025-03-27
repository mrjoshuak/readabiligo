// Package readability provides a pure Go implementation of Mozilla's Readability.js
// for extracting the main content from web pages.
package readability

import (
	"fmt"
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
	// Enhanced list of error-related phrases
	errorPhrases := []string{
		"404", "not found", "page not found", "page doesn't exist", "error", 
		"page missing", "no longer available", "page unavailable", "cannot be found",
		"couldn't be found", "could not be found", "doesn't exist", "does not exist",
		"broken link", "page deleted", "no longer exists", "500", "server error",
		"internal error", "service unavailable", "unavailable", "temporarily unavailable",
		"maintenance", "we're sorry", "we are sorry", "gone", "bad request", "forbidden",
		"access denied", "403",
	}
	
	// Extended HTTP error status codes
	errorStatusCodes := []string{"400", "401", "403", "404", "405", "406", "408", "409", "410", "429", "500", "501", "502", "503", "504"}
	
	// Check title for error indicators
	title := strings.ToLower(doc.Find("title").Text())
	for _, phrase := range errorPhrases {
		if strings.Contains(title, phrase) {
			return true
		}
	}
	
	// Check for error classes and IDs on body or main content containers
	errorClasses := []string{"error", "not-found", "404", "500", "missing", "unavailable"}
	for _, class := range errorClasses {
		if doc.Find(fmt.Sprintf("body.%s, div.%s, main.%s, #%s, .error-page, .not-found-page", 
			class, class, class, class)).Length() > 0 {
			return true
		}
	}

	// Check for error status code in meta tags (expanded to include Open Graph)
	statusCodeSelectors := []string{
		"meta[name='status-code']", 
		"meta[http-equiv='Status']",
		"meta[property='og:title']", // Check for error in Open Graph title
	}
	
	for _, selector := range statusCodeSelectors {
		content := strings.ToLower(doc.Find(selector).AttrOr("content", ""))
		
		// Check actual status codes
		for _, code := range errorStatusCodes {
			if strings.Contains(content, code) {
				return true
			}
		}
		
		// Also check error phrases in meta content
		for _, phrase := range errorPhrases {
			if strings.Contains(content, phrase) {
				return true
			}
		}
	}

	// Check for error indicators in headings and main content blocks
	errorMatchCount := 0
	doc.Find("h1, h2, h3, .error-title, .error-message, .error-description, .message, .alert").Each(func(_ int, s *goquery.Selection) {
		text := strings.ToLower(s.Text())
		for _, phrase := range errorPhrases {
			if strings.Contains(text, phrase) {
				errorMatchCount++
			}
		}
	})
	
	if errorMatchCount >= 1 {
		return true
	}
	
	// Check for standard error page images
	errorImagePatterns := []string{
		"404", "error", "not-found", "not_found", "missing", "broken", "unavailable",
	}
	
	errorImageCount := 0
	doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists {
			return
		}
		
		src = strings.ToLower(src)
		alt := strings.ToLower(s.AttrOr("alt", ""))
		
		for _, pattern := range errorImagePatterns {
			if strings.Contains(src, pattern) || strings.Contains(alt, pattern) {
				errorImageCount++
				break
			}
		}
	})
	
	if errorImageCount > 0 {
		return true
	}
	
	// Check for sparse content - error pages often have very little content
	// but with prominent error messages
	if doc.Find("body *").Length() < 30 { // Sparse DOM
		// Only consider it an error page if there's at least one error-like phrase in the text
		bodyText := strings.ToLower(doc.Find("body").Text())
		for _, phrase := range errorPhrases {
			if strings.Contains(bodyText, phrase) {
				return true
			}
		}
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
	// Calculate form-to-content ratio (a key indicator of login/signup pages)
	formElements := doc.Find("form").Length()
	formInputs := doc.Find("input[type='text'], input[type='email'], input[type='password'], input[type='tel'], textarea").Length()
	formButtons := doc.Find("button[type='submit'], input[type='submit'], .btn-submit, .submit-button").Length()
	totalFormElements := formElements + formInputs + formButtons
	
	// Enhanced content counting - look for paragraph and article elements
	paragraphs := doc.Find("p").Length()
	articles := doc.Find("article, .article, .post, .content-area").Length()
	headings := doc.Find("h1, h2, h3").Length()
	contentElements := paragraphs + articles + headings
	
	// Enhanced text content analysis
	bodyText := strings.TrimSpace(doc.Find("body").Text())
	mainText := ""
	doc.Find("main, #main, .main-content, article, .content").Each(func(i int, s *goquery.Selection) {
		mainText += s.Text() + " "
	})
	mainText = strings.TrimSpace(mainText)
	
	// Calculate text length without considering form labels and button text
	contentTextLength := len(bodyText)
	
	// Look for authentication keywords in headings and content
	authKeywords := []string{
		"login", "sign in", "signin", "log in", "username", "password", 
		"register", "sign up", "signup", "create account", "join", 
		"forgot password", "reset password", "authentication", "access",
		"member", "membership",
	}
	
	authKeywordCount := 0
	doc.Find("h1, h2, h3, h4, label, button, .form-heading").Each(func(i int, s *goquery.Selection) {
		text := strings.ToLower(s.Text())
		for _, keyword := range authKeywords {
			if strings.Contains(text, keyword) {
				authKeywordCount++
				break
			}
		}
	})
	
	// Check for login-specific field names
	hasLoginFields := doc.Find("input#username, input#password, input#email, input[name='username'], input[name='password'], input[name='email']").Length() > 0
	
	// Check for a common login form structure
	hasLoginForm := doc.Find("form input[type='password']").Length() > 0
	
	// Check for restricted access messages
	restrictedAccessMessages := []string{
		"must be logged in", "login required", "please log in", "sign in to", 
		"sign in required", "members only", "restricted access", "please sign in",
		"account required", "login to continue", "access denied", "unauthorized",
	}
	
	hasRestrictedMessage := false
	doc.Find("p, div, h1, h2, h3, .message, .alert").Each(func(i int, s *goquery.Selection) {
		text := strings.ToLower(s.Text())
		for _, message := range restrictedAccessMessages {
			if strings.Contains(text, message) {
				hasRestrictedMessage = true
				break
			}
		}
	})
	
	// Main decision logic for minimal content detection
	
	// Strong indicators - if ANY of these are true, it's likely minimal content
	if hasRestrictedMessage {
		return true
	}
	
	if hasLoginForm && authKeywordCount >= 2 {
		return true
	}
	
	if formElements > 0 && hasLoginFields && contentElements < 5 {
		return true
	}
	
	// Moderate indicators - combinations suggest minimal content
	formContentRatio := 0.0
	if contentElements > 0 {
		formContentRatio = float64(totalFormElements) / float64(contentElements)
	} else if totalFormElements > 0 {
		formContentRatio = float64(totalFormElements) // If no content, just use form count
	}
	
	// High form-to-content ratio is typical of login/signup pages
	if formContentRatio > 0.8 && contentTextLength < 2000 && authKeywordCount >= 1 {
		return true
	}
	
	// Low content amount with form elements present
	if contentElements < 10 && totalFormElements > 0 && contentTextLength < 1500 {
		return true
	}
	
	// Additional heuristic for single form pages
	if formElements == 1 && paragraphs < 5 && contentTextLength < 2000 {
		// Check if the form contains typical login fields
		if hasLoginFields {
			return true
		}
	}
	
	// Default case
	return false
}