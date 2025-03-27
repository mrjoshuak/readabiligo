// Package readabiligo provides a Go implementation of Mozilla's Readability.js library
// for extracting the main content from HTML pages.
package readabiligo

import (
	"github.com/mrjoshuak/readabiligo/types"
)

// Block represents a block of text with optional metadata.
// It is used to store paragraphs of plain text extracted from an article,
// with optional node index information for tracking the source HTML elements.
type Block = types.Block

// Article represents the extracted content and metadata from a webpage.
// It contains the article title, byline, publication date, HTML content,
// simplified HTML content, and plain text paragraphs.
type Article = types.Article

// ExtractionOptions configures the article extraction process.
// It controls whether to use Readability.js, whether to include content digests
// and node indexes, and sets limits on buffer size and extraction timeout.
type ExtractionOptions = types.ExtractionOptions

// DefaultOptions returns the default extraction options.
// By default, Readability.js is used if available, content digests and node indexes
// are disabled, buffer size is limited to 1MB, and timeout is set to 30 seconds.
func DefaultOptions() ExtractionOptions {
	return types.DefaultOptions()
}

// BuildInfo contains version and build information for the ReadabiliGo library.
type BuildInfo = types.BuildInfo

// GetBuildInfo returns the current version information for the ReadabiliGo library.
func GetBuildInfo() BuildInfo {
	return types.GetBuildInfo()
}

// Version is the current version of the ReadabiliGo library.
var Version = types.Version

// Name is the name of the ReadabiliGo library.
var Name = types.Name