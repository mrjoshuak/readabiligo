package extractors

import (
	"sort"
)

// ExtractTitle extracts the article title from HTML content
func ExtractTitle(html string) string {
	// List of XPaths for HTML tags that could contain a title
	// Scores reflect confidence in these XPaths and the preference used for extraction
	xpaths := []XPathScore{
		{XPath: "//h1[@class=\"entry-title\"]//text()", Score: 6}, // Highest priority for h1 with class="entry-title"
		{XPath: "//h1[@itemprop=\"headline\"]//text()", Score: 5}, // High priority for h1 with itemprop="headline"
		{XPath: "//header[@class=\"entry-header\"]/h1[@class=\"entry-title\"]//text()", Score: 4},
		{XPath: "//meta[@property=\"og:title\"]/@content", Score: 3},
		{XPath: "//h2[@itemprop=\"headline\"]//text()", Score: 2},
		{XPath: "//meta[contains(@itemprop, \"headline\")]/@content", Score: 2},
		{XPath: "//body/title//text()", Score: 1},
		{XPath: "//div[@class=\"postarea\"]/h2/a//text()", Score: 1},
		{XPath: "//h1[@class=\"post__title\"]//text()", Score: 1},
		{XPath: "//h1[@class=\"title\"]//text()", Score: 1},
		{XPath: "//header/h1//text()", Score: 1},
		{XPath: "//meta[@name=\"dcterms.title\"]/@content", Score: 1},
		{XPath: "//meta[@name=\"fb_title\"]/@content", Score: 1},
		{XPath: "//meta[@name=\"sailthru.title\"]/@content", Score: 1},
		{XPath: "//meta[@name=\"title\"]/@content", Score: 1},
		// Additional XPaths for edge cases
		{XPath: "//h1//text()", Score: 1},
		{XPath: "//h2//text()", Score: 1},
		{XPath: "//h3//text()", Score: 1},
		{XPath: "//meta[@property=\"twitter:title\"]/@content", Score: 2},
		{XPath: "//meta[@name=\"twitter:title\"]/@content", Score: 2},
		{XPath: "//meta[@property=\"article:title\"]/@content", Score: 2},
		{XPath: "//meta[@name=\"article:title\"]/@content", Score: 2},
		{XPath: "//div[@class=\"title\"]//text()", Score: 1},
		{XPath: "//div[@id=\"title\"]//text()", Score: 1},
		{XPath: "//div[contains(@class, \"title\")]//text()", Score: 1},
		{XPath: "//div[contains(@id, \"title\")]//text()", Score: 1},
		// Head title should have the lowest priority
		{XPath: "//head/title//text()", Score: 0},
		{XPath: "//title//text()", Score: 0},
	}

	// Extract titles using the XPaths
	extractedTitles := ExtractElement(html, xpaths, combineSimilarTitles)
	if len(extractedTitles) == 0 {
		return ""
	}

	// Find the title with the highest score
	var bestTitle string
	var highestScore int
	for title, element := range extractedTitles {
		if element.Score > highestScore {
			highestScore = element.Score
			bestTitle = title
		}
	}

	return bestTitle
}

// combineSimilarTitles combines scores for titles that are similar
func combineSimilarTitles(extractedStrings map[string]*ExtractedElement) map[string]*ExtractedElement {
	// Create a slice of titles for permutation
	titles := make([]string, 0, len(extractedStrings))
	for title := range extractedStrings {
		titles = append(titles, title)
	}

	// Sort titles to ensure deterministic processing
	sort.Strings(titles)

	// Check each pair of titles
	for i := 0; i < len(titles); i++ {
		for j := 0; j < len(titles); j++ {
			if i == j {
				continue
			}

			title1 := titles[i]
			title2 := titles[j]

			// If title1 is a subset of title2, combine their scores
			if isSubstring(title1, title2) {
				extractedStrings[title1].Score += extractedStrings[title2].Score
				extractedStrings[title1].XPaths = append(extractedStrings[title1].XPaths, extractedStrings[title2].XPaths...)
				sort.Strings(extractedStrings[title1].XPaths)
			} else if equalIgnoreCase(title1, title2) {
				// If titles are identical ignoring case, combine scores
				// Take the one with more capitals as the key
				if countUppercase(title1) > countUppercase(title2) {
					extractedStrings[title1].Score += extractedStrings[title2].Score
					extractedStrings[title1].XPaths = append(extractedStrings[title1].XPaths, extractedStrings[title2].XPaths...)
					sort.Strings(extractedStrings[title1].XPaths)
				}
			}
		}
	}

	return extractedStrings
}

// isSubstring checks if s1 is a substring of s2
func isSubstring(s1, s2 string) bool {
	return len(s1) < len(s2) && contains(s2, s1)
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// equalIgnoreCase checks if two strings are equal ignoring case
func equalIgnoreCase(s1, s2 string) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := 0; i < len(s1); i++ {
		c1 := s1[i]
		c2 := s2[i]
		if c1 >= 'A' && c1 <= 'Z' {
			c1 += 'a' - 'A'
		}
		if c2 >= 'A' && c2 <= 'Z' {
			c2 += 'a' - 'A'
		}
		if c1 != c2 {
			return false
		}
	}
	return true
}

// countUppercase counts the number of uppercase characters in a string
func countUppercase(s string) int {
	count := 0
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			count++
		}
	}
	return count
}
