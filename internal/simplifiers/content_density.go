package simplifiers

import (
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

// Content-related patterns for scoring
// These maps allow for O(1) lookups instead of iterating through arrays
var (
	contentPatternMap = map[string]bool{
		"article":   true,
		"content":   true,
		"entry":     true,
		"hentry":    true,
		"main":      true,
		"page":      true,
		"pagination": true,
		"post":      true,
		"text":      true,
		"blog":      true,
		"story":     true,
		"body":      true,
		"section":   true,
		"readable":  true,
	}

	nonContentPatternMap = map[string]bool{
		"combx":         true,
		"comment":       true,
		"com-":          true,
		"contact":       true,
		"foot":          true,
		"footer":        true,
		"footnote":      true,
		"masthead":      true,
		"media":         true,
		"meta":          true,
		"outbrain":      true,
		"promo":         true,
		"related":       true,
		"scroll":        true,
		"shoutbox":      true,
		"sidebar":       true,
		"sponsor":       true,
		"shopping":      true,
		"tags":          true,
		"tool":          true,
		"widget":        true,
		"nav":           true,
		"menu":          true,
		"header":        true,
		"ad":            true,
		"advertisement": true,
		"banner":        true,
		"social":        true,
		"share":         true,
		"sharing":       true,
		"login":         true,
		"signup":        true,
	}

	// Maintain these slices for backward compatibility but they're no longer used for lookups
	contentPatterns = []string{
		"article", "content", "entry", "hentry", "main", "page", "pagination", "post",
		"text", "blog", "story", "body", "section", "readable",
	}

	nonContentPatterns = []string{
		"combx", "comment", "com-", "contact", "foot", "footer", "footnote", "masthead",
		"media", "meta", "outbrain", "promo", "related", "scroll", "shoutbox", "sidebar",
		"sponsor", "shopping", "tags", "tool", "widget", "nav", "menu", "header", "ad",
		"advertisement", "banner", "social", "share", "sharing", "login", "signup",
	}
)

// nodeDataCache stores cached data for nodes to avoid recalculation
// ThreadSafe cache for node data to avoid redundant calculations
type nodeDataCache struct {
	sync.RWMutex
	textContent map[*goquery.Selection]string
	htmlContent map[*goquery.Selection]string
	linkText    map[*goquery.Selection]string
	wordCount   map[*goquery.Selection]int
	sentCount   map[*goquery.Selection]int
}

// newNodeDataCache creates a new node data cache
func newNodeDataCache() *nodeDataCache {
	return &nodeDataCache{
		textContent: make(map[*goquery.Selection]string),
		htmlContent: make(map[*goquery.Selection]string),
		linkText:    make(map[*goquery.Selection]string),
		wordCount:   make(map[*goquery.Selection]int),
		sentCount:   make(map[*goquery.Selection]int),
	}
}

// Global cache instance
var globalNodeDataCache = newNodeDataCache()

// clearGlobalCache clears the global cache - useful for testing or when memory needs to be freed
func clearGlobalCache() {
	globalNodeDataCache.Lock()
	defer globalNodeDataCache.Unlock()
	
	globalNodeDataCache.textContent = make(map[*goquery.Selection]string)
	globalNodeDataCache.htmlContent = make(map[*goquery.Selection]string)
	globalNodeDataCache.linkText = make(map[*goquery.Selection]string)
	globalNodeDataCache.wordCount = make(map[*goquery.Selection]int)
	globalNodeDataCache.sentCount = make(map[*goquery.Selection]int)
}

// getNodeText gets cached text or calculates and caches it
func getNodeText(s *goquery.Selection) string {
	// Check cache first
	globalNodeDataCache.RLock()
	text, found := globalNodeDataCache.textContent[s]
	globalNodeDataCache.RUnlock()
	
	if found {
		return text
	}
	
	// Calculate and cache
	text = s.Text()
	
	globalNodeDataCache.Lock()
	globalNodeDataCache.textContent[s] = text
	globalNodeDataCache.Unlock()
	
	return text
}

// getNodeHTML gets cached HTML or calculates and caches it
func getNodeHTML(s *goquery.Selection) string {
	// Check cache first
	globalNodeDataCache.RLock()
	html, found := globalNodeDataCache.htmlContent[s]
	globalNodeDataCache.RUnlock()
	
	if found {
		return html
	}
	
	// Calculate and cache
	html, err := goquery.OuterHtml(s)
	if err != nil {
		return ""
	}
	
	globalNodeDataCache.Lock()
	globalNodeDataCache.htmlContent[s] = html
	globalNodeDataCache.Unlock()
	
	return html
}

// getNodeLinkText gets cached link text or calculates and caches it
func getNodeLinkText(s *goquery.Selection) string {
	// Check cache first
	globalNodeDataCache.RLock()
	linkText, found := globalNodeDataCache.linkText[s]
	globalNodeDataCache.RUnlock()
	
	if found {
		return linkText
	}
	
	// Calculate and cache
	linkText = s.Find("a").Text()
	
	globalNodeDataCache.Lock()
	globalNodeDataCache.linkText[s] = linkText
	globalNodeDataCache.Unlock()
	
	return linkText
}

// getNodeWordCount gets cached word count or calculates and caches it
func getNodeWordCount(s *goquery.Selection) int {
	// Check cache first
	globalNodeDataCache.RLock()
	count, found := globalNodeDataCache.wordCount[s]
	globalNodeDataCache.RUnlock()
	
	if found {
		return count
	}
	
	// Calculate and cache
	text := getNodeText(s)
	count = CountWords(text)
	
	globalNodeDataCache.Lock()
	globalNodeDataCache.wordCount[s] = count
	globalNodeDataCache.Unlock()
	
	return count
}

// getNodeSentenceCount gets cached sentence count or calculates and caches it
func getNodeSentenceCount(s *goquery.Selection) int {
	// Check cache first
	globalNodeDataCache.RLock()
	count, found := globalNodeDataCache.sentCount[s]
	globalNodeDataCache.RUnlock()
	
	if found {
		return count
	}
	
	// Calculate and cache
	text := getNodeText(s)
	count = CountSentences(text)
	
	globalNodeDataCache.Lock()
	globalNodeDataCache.sentCount[s] = count
	globalNodeDataCache.Unlock()
	
	return count
}

// Optimized implementation using map lookup instead of iterating through slices
// containsContentPattern checks if a string contains any content-related patterns
func containsContentPattern(s string) bool {
	s = strings.ToLower(s)
	
	// Fast path: direct matching of pattern
	if _, exists := contentPatternMap[s]; exists {
		return true
	}
	
	// Slower path: substring matching
	for pattern := range contentPatternMap {
		if strings.Contains(s, pattern) {
			return true
		}
	}
	return false
}

// containsNonContentPattern checks if a string contains any non-content-related patterns
func containsNonContentPattern(s string) bool {
	s = strings.ToLower(s)
	
	// Fast path: direct matching of pattern
	if _, exists := nonContentPatternMap[s]; exists {
		return true
	}
	
	// Slower path: substring matching
	for pattern := range nonContentPatternMap {
		if strings.Contains(s, pattern) {
			return true
		}
	}
	return false
}

// CalculateLinkDensity calculates the ratio of link text to total text
func CalculateLinkDensity(s *goquery.Selection) float64 {
	text := getNodeText(s)
	textLength := len(text)
	if textLength == 0 {
		return 0.0
	}

	linkText := getNodeLinkText(s)
	return float64(len(linkText)) / float64(textLength)
}

// AnalyzeContentDensity calculates the content density of an element
func AnalyzeContentDensity(s *goquery.Selection) float64 {
	// Get the cached HTML content and text content of the node
	html := getNodeHTML(s)
	text := getNodeText(s)

	// Calculate the ratio of text to HTML
	if len(html) == 0 {
		return 0.0
	}

	// Calculate text density
	textDensity := float64(len(text)) / float64(len(html))

	// Calculate paragraph density - cache paragraph count for future use
	paragraphCount := s.Find("p").Length()
	paragraphDensity := 0.0
	if len(text) > 0 {
		paragraphDensity = float64(paragraphCount) / float64(len(text)) * 1000 // Scale for readability
	}

	// Calculate sentence density using cached sentence count
	sentenceCount := getNodeSentenceCount(s)
	sentenceDensity := 0.0
	if len(text) > 0 {
		sentenceDensity = float64(sentenceCount) / float64(len(text)) * 1000 // Scale for readability
	}

	// Calculate word density using cached word count
	wordCount := getNodeWordCount(s)
	wordDensity := 0.0
	if len(text) > 0 {
		wordDensity = float64(wordCount) / float64(len(text)) * 100 // Scale for readability
	}

	// Check for content-related patterns in ID and class attributes
	id, hasID := s.Attr("id")
	class, hasClass := s.Attr("class")

	contentBoost := 1.0
	if hasID && containsContentPattern(id) {
		contentBoost += 5.0
	}
	if hasClass && containsContentPattern(class) {
		contentBoost += 3.0
	}

	// Penalize non-content patterns
	if (hasID && containsNonContentPattern(id)) || (hasClass && containsNonContentPattern(class)) {
		contentBoost *= 0.5
	}

	// Combine all factors
	return (textDensity*50.0 + paragraphDensity*20.0 + sentenceDensity*15.0 + wordDensity*15.0) * contentBoost
}

// CalculateTextToCodeRatio calculates the ratio of text to code in an element
func CalculateTextToCodeRatio(s *goquery.Selection) float64 {
	html := getNodeHTML(s)
	text := getNodeText(s)

	// Calculate the ratio of text to HTML
	if len(html) == 0 {
		return 0.0
	}

	return float64(len(text)) / float64(len(html))
}

// CalculateLinkDensityScore calculates a score based on link density
func CalculateLinkDensityScore(s *goquery.Selection) float64 {
	text := getNodeText(s)
	if len(text) == 0 {
		return 0.0
	}

	linkText := getNodeLinkText(s)
	linkDensity := float64(len(linkText)) / float64(len(text))

	// Penalize high link density
	return 1.0 - linkDensity
}

// CalculateHeadingDensity calculates the density of headings in an element
func CalculateHeadingDensity(s *goquery.Selection) float64 {
	text := getNodeText(s)
	if len(text) == 0 {
		return 0.0
	}

	// Count headings
	headingCount := s.Find("h1, h2, h3, h4, h5, h6").Length()

	// Calculate the ratio of headings to text
	return float64(headingCount) / float64(len(text)) * 1000 // Scale for readability
}

// CalculateListDensity calculates the density of list items in an element
func CalculateListDensity(s *goquery.Selection) float64 {
	text := getNodeText(s)
	if len(text) == 0 {
		return 0.0
	}

	// Count list items
	listItemCount := s.Find("li").Length()

	// Calculate the ratio of list items to text
	return float64(listItemCount) / float64(len(text)) * 1000 // Scale for readability
}

// CalculateImageDensity calculates the density of images in an element
func CalculateImageDensity(s *goquery.Selection) float64 {
	html := getNodeHTML(s)
	
	// Count images
	imageCount := s.Find("img").Length()

	// Calculate the ratio of images to HTML
	if len(html) == 0 {
		return 0.0
	}

	return float64(imageCount) / float64(len(html)) * 1000 // Scale for readability
}

// CalculateContentScore calculates a comprehensive content score
func CalculateContentScore(s *goquery.Selection) float64 {
	// Base score from content density
	score := AnalyzeContentDensity(s)

	// Adjust based on link density
	score *= CalculateLinkDensityScore(s)

	// Boost based on heading density
	score += CalculateHeadingDensity(s) * 10.0

	// Boost based on list density
	score += CalculateListDensity(s) * 5.0

	// Boost based on image density
	score += CalculateImageDensity(s) * 3.0

	// Boost for certain tag types
	tagName := goquery.NodeName(s)
	switch tagName {
	case "article", "section", "div", "main":
		score += 10.0
	case "p", "pre", "td":
		score += 5.0
	case "blockquote":
		score += 3.0
	case "address", "ol", "ul", "dl", "dd", "dt", "li":
		score += 3.0
	case "form", "aside", "footer", "header", "nav":
		score -= 10.0
	}

	// Boost for paragraphs and text nodes
	paragraphCount := s.Find("p").Length()
	score += float64(paragraphCount) * 5.0

	// Boost for images with captions
	figureBoost := 0.0
	s.Find("figure").Each(func(_ int, figure *goquery.Selection) {
		if figure.Find("figcaption").Length() > 0 && figure.Find("img").Length() > 0 {
			figureBoost += 10.0
		}
	})
	score += figureBoost

	return score
}

// contentNodeCandidates returns a batch-optimized array of content node candidates
// by executing a single query against the document instead of multiple separate ones
func contentNodeCandidates(doc *goquery.Document) []*goquery.Selection {
	candidates := []*goquery.Selection{}
	
	// Consolidated selector for all potential content nodes with a single DOM traversal
	selectorList := []string{
		"*[id*='content']", 
		"*[id*='article']", 
		"*[id*='main']", 
		"*[id*='body']", 
		"*[id*='entry']",
		"*[class*='content']", 
		"*[class*='article']", 
		"*[class*='main']", 
		"*[class*='body']", 
		"*[class*='entry']",
		"article", 
		"main", 
		".post", 
		".hentry",
		"*[data-content-focus='true']",
	}
	
	// Combine all selectors
	combinedSelector := strings.Join(selectorList, ", ")
	
	// Find all matching elements with a single DOM traversal
	doc.Find(combinedSelector).Each(func(_ int, s *goquery.Selection) {
		// Check if this candidate is already in our list
		// This avoids duplicate processing of elements that match multiple selectors
		alreadyIncluded := false
		for _, candidate := range candidates {
			if selectionEquals(s, candidate) {
				alreadyIncluded = true
				break
			}
		}
		
		if !alreadyIncluded {
			candidates = append(candidates, s)
		}
	})
	
	return candidates
}

// selectionEquals checks if two selections point to the same underlying element
func selectionEquals(a, b *goquery.Selection) bool {
	if a.Length() != b.Length() || a.Length() == 0 {
		return false
	}
	
	// Compare the first nodes (assuming single element selections)
	return a.Nodes[0] == b.Nodes[0]
}

// FindMainContentNode identifies the main content node in a document using
// an optimized algorithm with reduced DOM traversals
func FindMainContentNode(doc *goquery.Document) *goquery.Selection {
	// Get candidates with a single DOM traversal
	candidates := contentNodeCandidates(doc)

	// If we found candidates, score them and return the best one
	if len(candidates) > 0 {
		bestScore := -1.0
		var bestCandidate *goquery.Selection

		// Use cached scoring to avoid redundant calculations
		for _, candidate := range candidates {
			score := CalculateContentScore(candidate)
			if score > bestScore {
				bestScore = score
				bestCandidate = candidate
			}
		}

		if bestCandidate != nil {
			return bestCandidate
		}
	}

	// If no good candidates found, score div and section elements
	// Create a single selector to find all potential candidates in one pass
	bestScore := -1.0
	var bestCandidate *goquery.Selection

	// Using batch processing: combine multiple operations into a single DOM traversal
	operations := map[string]func(*goquery.Selection){
		"div, section": func(s *goquery.Selection) {
			// Skip tiny elements
			text := getNodeText(s)
			if len(text) < 100 {
				return
			}

			score := CalculateContentScore(s)
			if score > bestScore {
				bestScore = score
				bestCandidate = s
			}
		},
	}

	// Execute the batch processing
	for selector, operation := range operations {
		doc.Find(selector).Each(func(_ int, s *goquery.Selection) {
			operation(s)
		})
	}

	if bestCandidate != nil {
		return bestCandidate
	}

	// Last resort: use the body
	return doc.Find("body")
}