package readability

// No imports needed here

// ParseHTMLWithReadability parses HTML content using the Readability algorithm
// This is the preferred entry point for parsing HTML content with the internal readability package
func ParseHTMLWithReadability(html string, opts *ReadabilityOptions) (*ReadabilityArticle, error) {
	// Create a new Readability instance from HTML
	r, err := NewFromHTML(html, opts)
	if err != nil {
		return nil, WrapParseError(err, "ParseHTMLWithReadability", "failed to create Readability parser")
	}

	// Run the Readability algorithm
	return r.Parse()
}


// Parse extracts article content from HTML using default options
// This is kept for backwards compatibility. New code should use ParseHTML directly.
func Parse(html string) (*ReadabilityArticle, error) {
	return ParseHTMLWithReadability(html, nil)
}

