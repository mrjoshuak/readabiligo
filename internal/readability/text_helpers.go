// Package readability provides a pure Go implementation of Mozilla's Readability.js
// for extracting the main content from web pages.
package readability

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
)

// wordCount counts the number of words in a string
func wordCount(text string) int {
	return len(strings.Fields(text))
}

// unescapeHtmlEntities converts HTML entities to their corresponding characters
func unescapeHtmlEntities(text string) string {
	if text == "" {
		return text
	}

	// Basic HTML entities
	result := regexp.MustCompile(`&(quot|amp|apos|lt|gt);`).ReplaceAllStringFunc(text, func(match string) string {
		entity := match[1 : len(match)-1]
		if val, ok := HTMLEscapeMap[entity]; ok {
			return val
		}
		return match
	})

	// Numeric entities (&#123; format)
	result = regexp.MustCompile(`&#(?:x([0-9a-f]{1,4})|([0-9]{1,4}));`).ReplaceAllStringFunc(result, func(match string) string {
		if strings.HasPrefix(match, "&#x") {
			// Hex format &#x123;
			hexStr := match[3 : len(match)-1]
			intVal, err := strconv.ParseInt(hexStr, 16, 32)
			if err != nil {
				return match
			}
			return string(rune(intVal))
		} else {
			// Decimal format &#123;
			decStr := match[2 : len(match)-1]
			intVal, err := strconv.Atoi(decStr)
			if err != nil {
				return match
			}
			return string(rune(intVal))
		}
	})

	return result
}

// getInnerText extracts text from a node, optionally normalizing whitespace
// Performance optimized: uses strings.Builder to avoid repeated string concatenation
func getInnerText(s *goquery.Selection, normalize bool) string {
	if s == nil || s.Length() == 0 {
		return ""
	}

	// Use strings.Builder for efficient string concatenation
	// This significantly improves performance for large documents
	var builder strings.Builder

	// Pre-allocate a reasonable buffer size to reduce allocations
	// Average text node is ~100 bytes, estimate based on content count
	contentCount := s.Contents().Length()
	if contentCount > 0 {
		builder.Grow(contentCount * 100)
	}

	// Extract text from all child nodes
	s.Contents().Each(func(i int, el *goquery.Selection) {
		if el.Get(0) != nil {
			switch el.Get(0).Type {
			case TextNode:
				builder.WriteString(el.Text())
			case ElementNode:
				// Handle inline elements that might contain text
				if isPhrasingContent(el.Get(0)) {
					builder.WriteString(getInnerText(el, false))
				} else {
					// For block elements, add space around them
					builder.WriteString(" ")
					builder.WriteString(getInnerText(el, false))
					builder.WriteString(" ")
				}
			}
		}
	})

	text := builder.String()

	if normalize {
		// Replace all whitespace (newlines, tabs, etc.) with a single space
		re := regexp.MustCompile(`\s+`)
		text = re.ReplaceAllString(text, " ")
		// Trim leading and trailing whitespace
		text = strings.TrimSpace(text)
	}

	return text
}

// getText is a legacy wrapper around getInnerText for backwards compatibility
func getText(s *goquery.Selection, normalize bool) string {
	return getInnerText(s, normalize)
}

// textSimilarity measures similarity between two strings
func textSimilarity(textA, textB string) float64 {
	if textA == textB {
		return 1.0
	}
	
	if textA == "" || textB == "" {
		return 0.0
	}
	
	textA = strings.ToLower(textA)
	textB = strings.ToLower(textB)
	
	// Tokenize: split by any non-word characters and filter out empty tokens
	tokenizeAndFilter := func(text string) []string {
		tokens := RegexpTokenize.Split(text, -1)
		filtered := []string{}
		for _, token := range tokens {
			token = strings.TrimSpace(token)
			if token != "" {
				filtered = append(filtered, token)
			}
		}
		return filtered
	}
	
	tokensA := tokenizeAndFilter(textA)
	tokensB := tokenizeAndFilter(textB)
	
	// Count matching tokens
	matches := 0
	for _, tokenA := range tokensA {
		for _, tokenB := range tokensB {
			if tokenA == tokenB {
				matches++
				break
			}
		}
	}
	
	// Calculate Jaccard similarity (size of intersection / size of union)
	lenA := len(tokensA)
	lenB := len(tokensB)
	
	if lenA == 0 || lenB == 0 {
		return 0.0
	}
	
	// Return (matches / (lenA + lenB - matches)) to get similarity score
	return float64(matches) / float64(lenA+lenB-matches)
}

// getCharCount counts occurrences of a specific character in a node's text
func getCharCount(s *goquery.Selection, delimiter string) int {
	if s == nil || s.Length() == 0 {
		return 0
	}

	if delimiter == "" {
		delimiter = ","
	}

	text := getInnerText(s, true)
	return len(strings.Split(text, delimiter)) - 1
}

// getLinkDensity calculates the ratio of link text to total text
func getLinkDensity(s *goquery.Selection) float64 {
	if s == nil || s.Length() == 0 {
		return 0
	}

	// Cache the inner text to avoid recalculating 
	innerText := getInnerText(s, true)
	textLength := len(innerText)
	if textLength == 0 {
		return 0
	}

	// Use a pre-compiled matcher for hash URLs
	hashUrlMatcher := RegexpHashUrl

	// Calculate total link text length in one pass
	var linkLength int
	s.Find("a").Each(func(i int, link *goquery.Selection) {
		// Skip indexterm and noteref links which are just metadata and not real links
		// This matches Mozilla's behavior which doesn't count these in link density
		if dataType, exists := link.Attr("data-type"); exists && (dataType == "indexterm" || dataType == "noteref") {
			return
		}
		
		href, exists := link.Attr("href")
		
		// Apply coefficient for hash links (internal page links)
		coefficient := 1.0
		if exists && hashUrlMatcher.MatchString(href) {
			coefficient = 0.3
		}
		
		// Calculate link text length with coefficient applied
		linkLength += int(float64(len(getInnerText(link, true))) * coefficient)
	})

	// Return the density ratio
	return float64(linkLength) / float64(textLength)
}

// extractMeta extracts metadata from document meta tags
func extractMeta(doc *goquery.Document, field, defaultValue string) string {
	// Metadata naming variations to check for the given field
	// This handles common variations of metadata naming across websites
	metaFieldVariations := map[string][]string{
		"author": {
			"author", "byline", "dc.creator", "article:author", "creator", "og:article:author",
		},
		"date": {
			"date", "created", "article:published_time", "article:modified_time", 
			"publication_date", "sailthru.date", "timestamp", "dc.date", "og:published_time",
			"og:updated_time", "publication-date", "modified-date", "last-modified",
		},
		"sitename": {
			"og:site_name", "application-name", "site_name", "publisher", "dc.publisher", "copyright",
		},
		"description": {
			"description", "og:description", "dc.description", "twitter:description",
		},
		"title": {
			"title", "og:title", "dc.title", "twitter:title",
		},
	}

	// Get the variations to check for this field
	variations, ok := metaFieldVariations[field]
	if !ok {
		// If no variations defined, just check the field as-is
		variations = []string{field}
	}

	// Try each variation, checking multiple meta tag syntaxes for each
	for _, variation := range variations {
		// Check standard "name" attribute
		if value := doc.Find(fmt.Sprintf("meta[name='%s']", variation)).AttrOr("content", ""); value != "" {
			return strings.TrimSpace(value)
		}

		// Check "property" attribute (common for OpenGraph)
		if value := doc.Find(fmt.Sprintf("meta[property='%s']", variation)).AttrOr("content", ""); value != "" {
			return strings.TrimSpace(value)
		}

		// Check "itemprop" attribute (common for schema.org)
		if value := doc.Find(fmt.Sprintf("meta[itemprop='%s']", variation)).AttrOr("content", ""); value != "" {
			return strings.TrimSpace(value)
		}

		// Special case for Twitter cards
		if strings.HasPrefix(variation, "twitter:") {
			if value := doc.Find(fmt.Sprintf("meta[name='%s']", variation)).AttrOr("value", ""); value != "" {
				return strings.TrimSpace(value)
			}
		}
	}

	// If we're looking for title and didn't find it in meta tags, check the <title> element
	if field == "title" {
		if title := doc.Find("title").Text(); title != "" {
			return strings.TrimSpace(title)
		}
	}

	// If no match found, return default value
	return defaultValue
}

// getTextDensity calculates the ratio of text content to total elements
func getTextDensity(s *goquery.Selection, tagNames []string) float64 {
	if s == nil || s.Length() == 0 || len(tagNames) == 0 {
		return 0
	}

	// Get all text
	text := getInnerText(s, true)
	textLength := len(text)
	if textLength == 0 {
		return 0
	}

	// Count tag text
	tagTextLength := 0
	for _, tag := range tagNames {
		s.Find(tag).Each(func(i int, el *goquery.Selection) {
			tagTextLength += len(getInnerText(el, true))
		})
	}

	// If no tags found
	if tagTextLength == 0 {
		return 0
	}

	return float64(tagTextLength) / float64(textLength)
}

// getNormalized returns a normalized string with whitespace trimmed
func getNormalized(text string) string {
	return strings.TrimSpace(RegexpNormalize.ReplaceAllString(text, " "))
}

// isValidByline checks if a string looks like a byline
func isValidByline(text string) bool {
	// Must be short, have at least one character
	if len(text) == 0 || len(text) > 100 {
		return false
	}

	// Check for indicators of a date-only string
	for _, indicator := range []string{"/", "•", "·", "|", "-", "—"} {
		if strings.Contains(text, indicator) {
			// May be a date divider; check if text is mostly numeric
			numericCount := 0
			for _, r := range text {
				if unicode.IsDigit(r) {
					numericCount++
				}
			}
			if float64(numericCount)/float64(len(text)) > 0.3 {
				return false // Too many digits for a byline
			}
		}
	}

	return true
}

// checkAdjacentForByline checks elements adjacent to node for byline data
func checkAdjacentForByline(node *goquery.Selection) string {
	// If the node has siblings, check if one might be a byline
	if node != nil && node.Nodes != nil && len(node.Nodes) > 0 {
		if prevSibling := node.Prev(); prevSibling.Length() > 0 {
			// Check if previous sibling has byline indicators
			if prevSibling.HasClass("byline") || RegexpByline.MatchString(prevSibling.AttrOr("class", "")) {
				byline := getInnerText(prevSibling, true)
				if isValidByline(byline) {
					return byline
				}
			}
		}

		if nextSibling := node.Next(); nextSibling.Length() > 0 {
			// Check if next sibling has byline indicators
			if nextSibling.HasClass("byline") || RegexpByline.MatchString(nextSibling.AttrOr("class", "")) {
				byline := getInnerText(nextSibling, true)
				if isValidByline(byline) {
					return byline
				}
			}
		}
	}

	return ""
}

// getRootDocumentTitle creates an estimated document title
func getRootDocumentTitle(doc *goquery.Document) string {
	if doc == nil {
		return ""
	}

	// First try the <title> element
	title := doc.Find("title").Text()
	title = strings.TrimSpace(title)
	
	// If no title, check for meta title
	if title == "" {
		title = extractMeta(doc, "title", "")
	}
	
	// Fallback to document URL if we have it
	if title == "" {
		// Get URL from base tag, if present
		doc.Find("base").Each(func(i int, s *goquery.Selection) {
			if baseUrl, exists := s.Attr("href"); exists && baseUrl != "" {
				// Extract title from URL as a last resort
				parts := strings.Split(baseUrl, "/")
				if len(parts) > 0 {
					lastPart := parts[len(parts)-1]
					// Remove file extension
					if idx := strings.LastIndex(lastPart, "."); idx > 0 {
						lastPart = lastPart[:idx]
					}
					// Convert dashes to spaces
					lastPart = strings.ReplaceAll(lastPart, "-", " ")
					lastPart = strings.ReplaceAll(lastPart, "_", " ")
					title = strings.Title(lastPart)
				}
			}
		})
	}
	
	return title
}

// cleanTitle cleans and formats article title
func cleanTitle(title string) string {
	// Normalize whitespace
	title = getNormalized(title)
	
	// If there's no title, return empty string
	if title == "" {
		return ""
	}
	
	// Look for separators to identify site name
	separators := []string{" | ", " - ", " :: ", " / ", " » "}
	
	for _, separator := range separators {
		if parts := strings.Split(title, separator); len(parts) > 1 {
			// Check if the last part is the site name
			// Site names are typically short
			if len(parts[len(parts)-1]) < 10 {
				// Return all parts except the last one
				return strings.TrimSpace(strings.Join(parts[:len(parts)-1], separator))
			}
			
			// Check if the first part is the site name
			if len(parts[0]) < 10 {
				// Return all parts except the first one
				return strings.TrimSpace(strings.Join(parts[1:], separator))
			}
		}
	}
	
	// If no separator found or complex patterns, return as-is
	return title
}