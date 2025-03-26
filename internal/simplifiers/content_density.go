package simplifiers

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Content-related patterns for scoring
var (
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

// containsContentPattern checks if a string contains any content-related patterns
func containsContentPattern(s string) bool {
	s = strings.ToLower(s)
	for _, pattern := range contentPatterns {
		if strings.Contains(s, pattern) {
			return true
		}
	}
	return false
}

// containsNonContentPattern checks if a string contains any non-content-related patterns
func containsNonContentPattern(s string) bool {
	s = strings.ToLower(s)
	for _, pattern := range nonContentPatterns {
		if strings.Contains(s, pattern) {
			return true
		}
	}
	return false
}

// CalculateLinkDensity calculates the ratio of link text to total text
func CalculateLinkDensity(s *goquery.Selection) float64 {
	textLength := len(s.Text())
	if textLength == 0 {
		return 0.0
	}

	linkText := s.Find("a").Text()
	return float64(len(linkText)) / float64(textLength)
}

// AnalyzeContentDensity calculates the content density of an element
func AnalyzeContentDensity(s *goquery.Selection) float64 {
	// Get the HTML content of the node
	html, err := goquery.OuterHtml(s)
	if err != nil {
		return 0.0
	}

	// Get the text content of the node
	text := s.Text()

	// Calculate the ratio of text to HTML
	if len(html) == 0 {
		return 0.0
	}

	// Calculate text density
	textDensity := float64(len(text)) / float64(len(html))

	// Calculate paragraph density
	paragraphCount := s.Find("p").Length()
	paragraphDensity := 0.0
	if len(text) > 0 {
		paragraphDensity = float64(paragraphCount) / float64(len(text)) * 1000 // Scale for readability
	}

	// Calculate sentence density
	sentenceCount := CountSentences(text)
	sentenceDensity := 0.0
	if len(text) > 0 {
		sentenceDensity = float64(sentenceCount) / float64(len(text)) * 1000 // Scale for readability
	}

	// Calculate word density
	wordCount := CountWords(text)
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
	// Get the HTML content of the node
	html, err := goquery.OuterHtml(s)
	if err != nil {
		return 0.0
	}

	// Get the text content of the node
	text := s.Text()

	// Calculate the ratio of text to HTML
	if len(html) == 0 {
		return 0.0
	}

	return float64(len(text)) / float64(len(html))
}

// CalculateLinkDensityScore calculates a score based on link density
func CalculateLinkDensityScore(s *goquery.Selection) float64 {
	// Get the text content of the node
	text := s.Text()
	if len(text) == 0 {
		return 0.0
	}

	// Get the text content of all links
	linkText := s.Find("a").Text()

	// Calculate the ratio of link text to total text
	linkDensity := float64(len(linkText)) / float64(len(text))

	// Penalize high link density
	return 1.0 - linkDensity
}

// CalculateHeadingDensity calculates the density of headings in an element
func CalculateHeadingDensity(s *goquery.Selection) float64 {
	// Get the text content of the node
	text := s.Text()
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
	// Get the text content of the node
	text := s.Text()
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
	// Get the HTML content of the node
	html, err := goquery.OuterHtml(s)
	if err != nil {
		return 0.0
	}

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
	s.Find("figure").Each(func(_ int, figure *goquery.Selection) {
		if figure.Find("figcaption").Length() > 0 && figure.Find("img").Length() > 0 {
			score += 10.0
		}
	})

	return score
}

// FindMainContentNode identifies the main content node in a document
func FindMainContentNode(doc *goquery.Document) *goquery.Selection {
	// First try to find elements with content-related IDs or classes
	candidates := []*goquery.Selection{}

	// Look for elements with content-related IDs
	doc.Find("[id*='content'], [id*='article'], [id*='main'], [id*='body'], [id*='entry']").Each(func(_ int, s *goquery.Selection) {
		candidates = append(candidates, s)
	})

	// Look for elements with content-related classes
	doc.Find("[class*='content'], [class*='article'], [class*='main'], [class*='body'], [class*='entry']").Each(func(_ int, s *goquery.Selection) {
		candidates = append(candidates, s)
	})

	// Look for common content elements
	doc.Find("article, main, .post, .hentry").Each(func(_ int, s *goquery.Selection) {
		candidates = append(candidates, s)
	})

	// Look for elements with data-content-focus attribute
	doc.Find("[data-content-focus='true']").Each(func(_ int, s *goquery.Selection) {
		candidates = append(candidates, s)
	})

	// If we found candidates, score them and return the best one
	if len(candidates) > 0 {
		bestScore := -1.0
		var bestCandidate *goquery.Selection

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

	// If no good candidates found, score all divs and sections
	bestScore := -1.0
	var bestCandidate *goquery.Selection

	doc.Find("div, section").Each(func(_ int, s *goquery.Selection) {
		// Skip tiny elements
		if len(s.Text()) < 100 {
			return
		}

		score := CalculateContentScore(s)
		if score > bestScore {
			bestScore = score
			bestCandidate = s
		}
	})

	if bestCandidate != nil {
		return bestCandidate
	}

	// Last resort: use the body
	return doc.Find("body")
}
