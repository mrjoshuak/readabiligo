package readability

import (
	"fmt"
	"time"
)

// ParseHTML parses HTML content using the Readability algorithm
func ParseHTML(html string, opts *ReadabilityOptions) (*ReadabilityArticle, error) {
	// Create a new Readability instance from HTML
	r, err := NewFromHTML(html, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create Readability parser: %w", err)
	}

	// Run the Readability algorithm
	return r.Parse()
}

// ParseWithOptions extracts article content from HTML using custom options
func ParseWithOptions(html string, debug bool, maxElems int, charThreshold int) (*ReadabilityArticle, error) {
	// Create custom options
	opts := defaultReadabilityOptions()
	opts.Debug = debug
	if maxElems > 0 {
		opts.MaxElemsToParse = maxElems
	}
	if charThreshold > 0 {
		opts.CharThreshold = charThreshold
	}

	// Parse with custom options
	return ParseHTML(html, &opts)
}

// Parse extracts article content from HTML using default options
func Parse(html string) (*ReadabilityArticle, error) {
	return ParseHTML(html, nil)
}

// ToArticle converts a ReadabilityArticle to a standard Article format
// This allows compatibility with existing code that expects the Article type
type Article struct {
	Title        string    `json:"title"`
	Byline       string    `json:"byline"`
	Date         time.Time `json:"date"`
	Content      string    `json:"content"`
	PlainContent string    `json:"plain_content"`
	TextContent  string    `json:"text_content"`
	Excerpt      string    `json:"excerpt"`
	SiteName     string    `json:"site_name"`
	Length       int       `json:"length"`
}

// ToArticle converts a ReadabilityArticle to a standard Article format
func (r *ReadabilityArticle) ToArticle() *Article {
	return &Article{
		Title:       r.Title,
		Byline:      r.Byline,
		Date:        r.Date,
		Content:     r.Content,
		PlainContent: r.Content, // Will be processed by caller
		TextContent: r.TextContent,
		Excerpt:     r.Excerpt,
		SiteName:    r.SiteName,
		Length:      r.Length,
	}
}