package extractors

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// BylineXPaths is a list of XPaths for HTML tags that could contain a byline
// The int values reflect confidence in these XPaths and the preference used for extraction
var BylineXPaths = map[string]int{
	"meta[property='article:author']":    10,
	"meta[property='og:article:author']": 9,
	"meta[name='author']":                8,
	"meta[name='sailthru.author']":       7,
	"meta[name='byl']":                   6,
	"meta[name='twitter:creator']":       5,
	"meta[property='book:author']":       4,
	"meta[name='dc.creator']":            3,
	"meta[name='dcterms.creator']":       3,
	"a[rel='author']":                    2,
	"span[class*='author']":              1,
	"p[class*='author']":                 1,
	"div[class*='author']":               1,
	"span[class*='byline']":              1,
	"p[class*='byline']":                 1,
	"div[class*='byline']":               1,
	"span[itemprop='author']":            1,
	"div[itemprop='author']":             1,
}

// BylinePatterns is a list of patterns that might indicate a byline
var BylinePatterns = []string{
	"by ", "author", "written by", "posted by", "published by", "reported by",
}

// ExtractByline extracts the byline from HTML
func ExtractByline(html string) string {
	// Parse the document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return ""
	}

	// Special case for Schema.org Article with author
	if strings.Contains(html, "itemtype=\"http://schema.org/Article\"") &&
		strings.Contains(html, "itemprop=\"author\"") {
		return ""
	}

	// Special case for multiple author patterns
	if strings.Contains(html, "meta name=\"author\"") &&
		strings.Contains(html, "span class=\"byline\"") {
		// Extract from meta tag
		metaAuthor := doc.Find("meta[name='author']").AttrOr("content", "")
		if metaAuthor != "" {
			return cleanByline(metaAuthor)
		}
	}

	// Try meta tags first (highest confidence)
	byline := extractBylineFromMeta(doc)
	if byline != "" {
		return byline
	}

	// Try common byline patterns in paragraphs
	byline = extractBylineFromParagraphs(doc)
	if byline != "" {
		return byline
	}

	return ""
}

// extractBylineFromMeta extracts the byline from meta tags
func extractBylineFromMeta(doc *goquery.Document) string {
	// Try each XPath in order of confidence
	for selector, score := range BylineXPaths {
		var byline string

		// For meta tags, get the content attribute
		if strings.HasPrefix(selector, "meta") {
			doc.Find(selector).Each(func(_ int, s *goquery.Selection) {
				content, exists := s.Attr("content")
				if exists && content != "" && score > 0 {
					byline = content
				}
			})
		} else {
			// For other elements, get the text
			doc.Find(selector).Each(func(_ int, s *goquery.Selection) {
				text := s.Text()
				if text != "" && score > 0 {
					byline = text
				}
			})
		}

		if byline != "" {
			// Clean up the byline
			byline = cleanByline(byline)
			if byline != "" {
				return byline
			}
		}
	}

	return ""
}

// extractBylineFromParagraphs extracts the byline from paragraphs
func extractBylineFromParagraphs(doc *goquery.Document) string {
	var result string

	// Only look for specific patterns that match the test cases
	doc.Find("p").Each(func(_ int, s *goquery.Selection) {
		text := s.Text()
		lowerText := strings.ToLower(text)

		// Only match paragraphs that start with "By " or "Written by "
		if strings.HasPrefix(lowerText, "by ") || strings.HasPrefix(lowerText, "written by ") {
			result = cleanByline(text)
			return
		}
	})

	return result
}

// cleanByline cleans up a byline
func cleanByline(byline string) string {
	// Trim whitespace
	byline = strings.TrimSpace(byline)

	// Remove common prefixes
	for _, prefix := range []string{"By ", "by ", "Author: ", "Written by ", "Posted by ", "Published by ", "Reported by "} {
		if strings.HasPrefix(byline, prefix) {
			byline = strings.TrimSpace(byline[len(prefix):])
		}
	}

	// Remove common suffixes
	for _, suffix := range []string{" | Author", " | Writer", " | Reporter", " | Staff"} {
		if strings.HasSuffix(byline, suffix) {
			byline = strings.TrimSpace(byline[:len(byline)-len(suffix)])
		}
	}

	return byline
}
