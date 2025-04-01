package readability

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// ReadabilityOptions defines configuration options for the Readability parser
type ReadabilityOptions struct {
	Debug                bool     // Debug mode
	MaxElemsToParse      int      // Maximum elements to parse (0 = no limit)
	NbTopCandidates      int      // Number of top candidates to consider
	CharThreshold        int      // Minimum character threshold
	ClassesToPreserve    []string // Classes to preserve
	KeepClasses          bool     // Whether to keep classes
	DisableJSONLD        bool     // Whether to disable JSON-LD processing
	AllowedVideoRegex    *regexp.Regexp // Regex for allowed videos
	PreserveImportantLinks bool     // Whether to preserve important links like "More information..." in cleaned elements
	DetectContentType    bool     // Whether to enable content type detection
	ContentType          ContentType // Content type to use for extraction (or auto-detected if DetectContentType is true)
}

// defaultReadabilityOptions returns the default options
func defaultReadabilityOptions() ReadabilityOptions {
	return ReadabilityOptions{
		Debug:                false,
		MaxElemsToParse:      DefaultMaxElemsToParse,
		NbTopCandidates:      DefaultNTopCandidates,
		CharThreshold:        DefaultCharThreshold,
		ClassesToPreserve:    ClassesToPreserve,
		KeepClasses:          false,
		DisableJSONLD:        false,
		AllowedVideoRegex:    RegexpVideos,
		PreserveImportantLinks: false, // Default to false to match ReadabiliPy's behavior
		DetectContentType:    true,    // Enable content type detection by default
		ContentType:          ContentTypeUnknown, // Auto-detect by default
	}
}

// ReadabilityArticle represents the extracted article
type ReadabilityArticle struct {
	Title        string      // Article title
	Byline       string      // Article byline (author)
	Content      string      // Article content (HTML)
	TextContent  string      // Article text content (plain text)
	Length       int         // Length of the text content
	Excerpt      string      // Short excerpt
	SiteName     string      // Site name
	Date         time.Time   // Publication date
	ContentType  ContentType // Detected content type
}

// Readability implements the Readability algorithm
type Readability struct {
	doc              *goquery.Document // The HTML document
	options          ReadabilityOptions // Options for the parser
	articleTitle     string            // Extracted article title
	articleByline    string            // Extracted article byline
	articleDir       string            // Article text direction
	articleSiteName  string            // Site name
	attempts         []int             // Extraction attempts
	flags            int               // Flags controlling the algorithm
	contentType      ContentType       // Detected or specified content type
}

// NodeInfo holds information about a node
type NodeInfo struct {
	node         *goquery.Selection // The node
	contentScore float64            // Content score
}

// createElement is a helper function that creates an element with the given tag name
// This is a workaround for the non-existent CreateElement method in goquery.Document
func (r *Readability) createElement(tagName string) *goquery.Selection {
	node := &html.Node{
		Type: html.ElementNode,
		Data: tagName,
	}
	return goquery.NewDocumentFromNode(node).Find(tagName)
}

// NewFromDocument creates a new Readability parser from a goquery document
func NewFromDocument(doc *goquery.Document, opts *ReadabilityOptions) *Readability {
	options := defaultReadabilityOptions()
	if opts != nil {
		options = *opts
	}

	r := &Readability{
		doc:     doc,
		options: options,
		flags:   FlagStripUnlikelys | FlagWeightClasses | FlagCleanConditionally,
		contentType: options.ContentType,
	}

	return r
}

// NewFromHTML creates a new Readability parser from HTML string
func NewFromHTML(html string, opts *ReadabilityOptions) (*Readability, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, WrapParseError(err, "NewFromHTML", "failed to parse HTML document")
	}

	return NewFromDocument(doc, opts), nil
}

// Parse runs the Readability algorithm
func (r *Readability) Parse() (*ReadabilityArticle, error) {
	// Check document
	if r.doc == nil || r.doc.Selection.Length() == 0 {
		return nil, WrapValidationError(ErrNoDocument, "Parse", "")
	}

	// Check if document is too large
	if r.options.MaxElemsToParse > 0 {
		numNodes := r.doc.Find("*").Length()
		if numNodes > r.options.MaxElemsToParse {
			err := WrapValidationError(ErrDocumentLarge, "Parse", "")
			// Add extra context information to the error message
			err = fmt.Errorf("%w: %d elements (exceeds limit of %d)", 
				err, numNodes, r.options.MaxElemsToParse)
			return nil, err
		}
	}

	// For compatibility, still set a content type but use standard algorithm
	if r.contentType == ContentTypeUnknown {
		// Default to Article type for all content
		r.contentType = ContentTypeArticle
		if r.options.Debug {
			fmt.Printf("DEBUG: Using standard Mozilla algorithm for all content types\n")
		}
	}
	
	// Set standard flags for all content types (consistent with Mozilla's implementation)
	r.flags = FlagStripUnlikelys | FlagWeightClasses | FlagCleanConditionally

	// Unwrap noscript images
	r.unwrapNoscriptImages()

	// Extract JSON-LD metadata (if enabled)
	jsonLd := make(map[string]string)
	if !r.options.DisableJSONLD {
		jsonLd = r.getJSONLD()
	}

	// Remove scripts
	r.removeScripts()

	// Prepare document
	r.prepDocument()

	// Get article metadata
	metadata := r.getArticleMetadata(jsonLd)
	r.articleTitle = metadata["title"]

	// Grab article content
	article := r.grabArticle()
	if article == nil {
		return nil, WrapExtractionError(ErrNoContent, "Parse", "")
	}

	// Post-process content
	r.postProcessContent(article)

	// If no excerpt in metadata, use the first paragraph
	excerpt := metadata["excerpt"]
	if excerpt == "" {
		article.Find("p").EachWithBreak(func(i int, s *goquery.Selection) bool {
			text := strings.TrimSpace(s.Text())
			if text != "" {
				excerpt = text
				return false // stop iteration
			}
			return true // continue
		})
	}

	// Apply standard content cleanup (ignoring content type)
	// Keep the function call for compatibility, but it now uses a standard approach
	r.applyContentTypeCleanup(article)

	// Additional cleanup step: make sure footers are removed
	// This is needed because in some cases, the clean function in prepArticle
	// might not have removed footer elements, especially if grabArticle returned the body
	if r.options.Debug {
		fmt.Printf("DEBUG: Final cleanup pass to remove any remaining footer elements\n")
	}
	
	// Apply the final cleanup to handle footer elements and important links
	if r.options.PreserveImportantLinks {
		// Explicitly search for important links anywhere in the article before final cleanup
		r.preserveImportantLinksAnywhere(article)
	}
	
	// Apply the final cleanup to handle footer elements
	r.finalCleanupFooters(article)
	
	// Get text content from the cleaned article
	textContent := getInnerText(article, true)

	// Build the article
	result := &ReadabilityArticle{
		Title:       r.articleTitle,
		Byline:      metadata["byline"],
		Content:     getOuterHTML(article),
		TextContent: textContent,
		Length:      len(textContent),
		Excerpt:     excerpt,
		SiteName:    metadata["siteName"],
		ContentType: r.contentType,
	}

	// Try to parse the date
	if date, err := time.Parse(time.RFC3339, metadata["date"]); err == nil {
		result.Date = date
	}

	return result, nil
}

// removeScripts removes all script tags from the document
func (r *Readability) removeScripts() {
	// Remove all script and noscript tags
	r.doc.Find("script, noscript").Remove()
}