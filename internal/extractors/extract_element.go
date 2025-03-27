package extractors

import (
	"regexp"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/mrjoshuak/readabiligo/internal/simplifiers"
)

// SelectorScore represents a CSS selector with a confidence score
type SelectorScore struct {
	Selector string
	Score    int
}

// ExtractedElement represents an extracted element with its score and the selectors used to find it
type ExtractedElement struct {
	Score     int
	Selectors []string
}

// ProcessDictFunc is a function that processes the extracted elements dictionary
type ProcessDictFunc func(map[string]*ExtractedElement) map[string]*ExtractedElement

// xpathToCSS helps convert simple XPath expressions to CSS selectors
// This only handles basic cases and may not work for complex XPath expressions
func xpathToCSS(xpath string) (string, bool, string) {
	// Handle quoted attribute values - check for both single and double quotes
	// First try with single quotes
	singleQuoteMatch := regexp.MustCompile(`//([a-zA-Z0-9_-]+)(?:\[@([a-zA-Z0-9_-]+)='([^']+)'\])?(?://@([a-zA-Z0-9_-]+))?`).FindStringSubmatch(xpath)
	if len(singleQuoteMatch) > 0 {
		return processXPathMatch(singleQuoteMatch)
	}
	
	// Try with double quotes
	doubleQuoteMatch := regexp.MustCompile(`//([a-zA-Z0-9_-]+)(?:\[@([a-zA-Z0-9_-]+)="([^"]+)"\])?(?://@([a-zA-Z0-9_-]+))?`).FindStringSubmatch(xpath)
	if len(doubleQuoteMatch) > 0 {
		return processXPathMatch(doubleQuoteMatch)
	}
	
	// Handle XPath with no quoted attributes
	simpleMatch := regexp.MustCompile(`//([a-zA-Z0-9_-]+)(?://@([a-zA-Z0-9_-]+))?`).FindStringSubmatch(xpath)
	if len(simpleMatch) > 0 {
		tag := simpleMatch[1]
		if tag == "*" {
			tag = "" // Universal selector in CSS
		}
		
		// Simple tag with no attribute condition
		if len(simpleMatch) == 2 || simpleMatch[2] == "" {
			return tag, false, ""
		}
		
		// Tag with attribute extraction
		return tag, true, simpleMatch[2]
	}
	
	// For more complex XPaths, provide a default that will fail gracefully
	return xpath, false, ""
}

// processXPathMatch handles the matched components from the XPath regular expression
func processXPathMatch(match []string) (string, bool, string) {
	tag := match[1]
	if tag == "*" {
		tag = "" // Universal selector in CSS
	}
	
	// Case 1: Simple tag selector like //div
	if match[2] == "" && (len(match) <= 4 || match[4] == "") {
		return tag, false, ""
	}
	
	// Case 2: Attribute condition like //div[@class='content']
	if match[2] != "" && (len(match) <= 4 || match[4] == "") {
		return tag + "[" + match[2] + "='" + match[3] + "']", false, ""
	}
	
	// Case 3: Attribute extraction like //meta[@property='og:title']//@content
	if match[2] != "" && len(match) > 4 && match[4] != "" {
		return tag + "[" + match[2] + "='" + match[3] + "']", true, match[4]
	}
	
	// Case 4: Just attribute extraction like //meta//@content
	if (match[2] == "" || len(match) <= 2) && len(match) > 4 && match[4] != "" {
		return tag, true, match[4]
	}
	
	return tag, false, ""
}

// ExtractElement extracts elements from HTML using a list of selectors with confidence scores
func ExtractElement(htmlContent string, selectors []SelectorScore, processDictFn ProcessDictFunc) map[string]*ExtractedElement {
	// Parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil
	}

	// Extract elements using selectors
	extractedStrings := make(map[string]*ExtractedElement)
	for _, selectorScore := range selectors {
		// Special case for common date selectors that have issues with the conversion
		if selectorScore.Selector == "//meta[@property='article:published_time']/@content" {
			// Direct approach for article published time
			doc.Find("meta[property='article:published_time']").Each(func(i int, s *goquery.Selection) {
				if content, exists := s.Attr("content"); exists && content != "" {
					extractedStrings[content] = &ExtractedElement{
						Score:     selectorScore.Score,
						Selectors: []string{selectorScore.Selector},
					}
				}
			})
			continue
		}
		
		// Handle XPath to CSS selector conversion for backward compatibility
		cssSelector, isAttrSelector, attrName := xpathToCSS(selectorScore.Selector)
		
		// Use Find with the CSS selector
		doc.Find(cssSelector).Each(func(i int, s *goquery.Selection) {
			var element string
			
			if isAttrSelector {
				// Extract the attribute value
				element, _ = s.Attr(attrName)
			} else {
				// Get the text content
				element = s.Text()
			}
			
			// Normalize whitespace
			element = simplifiers.NormalizeWhitespace(element)
			if element == "" {
				return
			}
			
			// Add or update the element in the map
			if _, exists := extractedStrings[element]; !exists {
				extractedStrings[element] = &ExtractedElement{
					Score:     selectorScore.Score,
					Selectors: []string{selectorScore.Selector},
				}
			} else {
				extractedStrings[element].Score += selectorScore.Score
				extractedStrings[element].Selectors = append(extractedStrings[element].Selectors, selectorScore.Selector)
				sort.Strings(extractedStrings[element].Selectors)
			}
		})
	}

	// Process the dictionary if a processing function is provided
	if processDictFn != nil {
		extractedStrings = processDictFn(extractedStrings)
	}

	return extractedStrings
}
