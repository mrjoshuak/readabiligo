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

// ParseWithOptions extracts article content from HTML using custom options
// This function is deprecated. Use ParseHTML with a properly configured ReadabilityOptions instead.
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
	return ParseHTMLWithReadability(html, &opts)
}

// Parse extracts article content from HTML using default options
// This is kept for backwards compatibility. New code should use ParseHTML directly.
func Parse(html string) (*ReadabilityArticle, error) {
	return ParseHTMLWithReadability(html, nil)
}

// ToStandardArticleV2 is an alias for ToStandardArticle in adapter.go
// This method exists only to avoid compilation errors after refactoring
func (r *ReadabilityArticle) ToStandardArticleV2() *Article {
	return r.ToStandardArticle()
}