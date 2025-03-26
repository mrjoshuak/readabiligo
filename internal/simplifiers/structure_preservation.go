package simplifiers

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// PreserveHeadingStructure ensures proper nesting of headings (h1 > h2 > h3, etc.)
func PreserveHeadingStructure(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return html
	}

	// Find all headings
	headings := doc.Find("h1, h2, h3, h4, h5, h6")

	// Ensure proper nesting of headings (h1 > h2 > h3, etc.)
	var lastLevel int
	headings.Each(func(i int, s *goquery.Selection) {
		// Get the heading level
		tagName := goquery.NodeName(s)
		level := int(tagName[1] - '0')

		// If this is the first heading, set the last level
		if i == 0 {
			lastLevel = level
			return
		}

		// If the level jumps by more than 1, adjust it
		if level > lastLevel+1 {
			// Create a new tag name with the correct level
			newTagName := fmt.Sprintf("h%d", lastLevel+1)

			// Get the HTML content
			content, err := s.Html()
			if err == nil {
				// Replace the element with the correct level
				s.ReplaceWithHtml(fmt.Sprintf("<%s>%s</%s>", newTagName, content, newTagName))
				level = lastLevel + 1
			}
		}

		lastLevel = level
	})

	html, err = doc.Html()
	if err != nil {
		return html
	}

	return html
}

// PreserveListStructure ensures proper nesting of lists
func PreserveListStructure(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return html
	}

	// Find all list items that are not inside a list
	doc.Find("li").Each(func(i int, s *goquery.Selection) {
		if s.ParentsFiltered("ul, ol").Length() == 0 {
			// Wrap the list item in an unordered list
			content, err := s.Html()
			if err == nil {
				s.ReplaceWithHtml(fmt.Sprintf("<ul><li>%s</li></ul>", content))
			}
		}
	})

	html, err = doc.Html()
	if err != nil {
		return html
	}

	return html
}

// PreserveTableStructure ensures proper table structure
func PreserveTableStructure(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return html
	}

	// Find all table cells that are not inside a table
	doc.Find("td, th").Each(func(i int, s *goquery.Selection) {
		if s.ParentsFiltered("table").Length() == 0 {
			// Wrap the cell in a table
			content, err := s.Html()
			if err == nil {
				s.ReplaceWithHtml(fmt.Sprintf("<table><tr><td>%s</td></tr></table>", content))
			}
		}
	})

	// Find all table rows that are not inside a table
	doc.Find("tr").Each(func(i int, s *goquery.Selection) {
		if s.ParentsFiltered("table").Length() == 0 {
			// Wrap the row in a table
			content, err := s.Html()
			if err == nil {
				s.ReplaceWithHtml(fmt.Sprintf("<table>%s</table>", content))
			}
		}
	})

	html, err = doc.Html()
	if err != nil {
		return html
	}

	return html
}

// EstimateReadingTime estimates the reading time in minutes
func EstimateReadingTime(text string) int {
	// Average reading speed is about 200-250 words per minute
	// We'll use 225 words per minute as a middle ground
	wordCount := CountWords(text)
	readingTimeMinutes := wordCount / 225

	// Ensure at least 1 minute
	if readingTimeMinutes < 1 {
		readingTimeMinutes = 1
	}

	return readingTimeMinutes
}

// PreserveStructure applies all structure preservation functions
func PreserveStructure(html string) string {
	html = PreserveHeadingStructure(html)
	html = PreserveListStructure(html)
	html = PreserveTableStructure(html)
	return html
}
