# ReadabiliGo

ReadabiliGo is a Go implementation of Mozilla's [Readability.js](https://github.com/mozilla/readability) library, designed to extract readable content from HTML pages. It provides both a command-line interface and a Go library for article extraction.

This package is a Go port of [ReadabiliPy](https://github.com/alan-turing-institute/ReadabiliPy) by [The Alan Turing Institute](https://github.com/alan-turing-institute), Ed Chalstrey, James Robinson, Martin O'Reilly, Gertjan van den Burg, Nelson Liu, and many other valuable contributors. ReadabiliGo maintains compatibility with ReadabiliPy's output format and features while providing a native Go implementation.

## Features

- Extract article content, title, byline, and date from HTML
- Content-type awareness with specialized extraction for different content types
- Superior structure preservation for reference content and documentation
- Better heading hierarchy and list element retention
- Improved link preservation for sources and citations
- Output in JSON, HTML, or plain text formats
- Support for content digests and node indexes for tracking HTML structure
- 100% Pure Go implementation, no JavaScript dependencies
- Comprehensive test suite with real-world examples

## Installation

### Prerequisites

No external dependencies are required! ReadabiliGo uses a pure Go implementation of the Readability algorithm.

### Installing the Command-Line Tool

```bash
go install github.com/mrjoshuak/readabiligo/cmd/readabiligo@latest
```

### Installing the Library

```bash
go get github.com/mrjoshuak/readabiligo
```

This will install the latest version of the library, which can be imported directly:

```go
import "github.com/mrjoshuak/readabiligo"
```

## Command-Line Usage

The `readabiligo` command-line tool can extract articles from HTML files or standard input.

### Basic Usage

Extract an article from an HTML file and output as JSON:

```bash
readabiligo -input article.html -output article.json
```

Extract an article and output as HTML:

```bash
readabiligo -input article.html -format html -output article.html
```

Extract an article and output as plain text:

```bash
readabiligo -input article.html -format text -output article.txt
```

Process multiple files at once:

```bash
readabiligo -input article1.html,article2.html -output-dir ./extracted
```

Read from standard input:

```bash
cat article.html | readabiligo -input - > article.json
```

### Command-Line Options

```
Usage: readabiligo [options]

Options:
  -input string
        Input HTML file path(s) (comma-separated, use '-' for stdin)
  -output string
        Output file path (default: stdout)
  -output-dir string
        Output directory for batch processing (default: same as input)
  -format string
        Output format: json, html, or text (default "json")
  -digests
        Add content digest attributes
  -indexes
        Add node index attributes
  -compact
        Output compact JSON without indentation
  -timeout duration
        Timeout for extraction (default 30s)
  -detect-content-type
        Enable content type detection (default true)
  -content-type string
        Force content type: reference, article, technical, error, minimal (bypasses detection)
  -preserve-links
        Preserve important links in cleanup
  -js
        DEPRECATED: No effect - JavaScript implementation has been removed
  -version
        Show version information
  -help
        Show help information
```

## Library Usage

ReadabiliGo can be used as a Go library in your projects.

### Basic Usage

```go
package main

import (
	"fmt"
	"os"

	"github.com/mrjoshuak/readabiligo"
)

func main() {
	// Create a new extractor
	ext := readabiligo.New()

	// Open an HTML file
	file, err := os.Open("article.html")
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	// Extract the article
	article, err := ext.ExtractFromReader(file, nil)
	if err != nil {
		fmt.Printf("Error extracting article: %v\n", err)
		return
	}

	// Access article data
	fmt.Printf("Title: %s\n", article.Title)
	fmt.Printf("Byline: %s\n", article.Byline)
	fmt.Printf("Date: %s\n", article.Date)
	fmt.Printf("Content length: %d bytes\n", len(article.Content))
	fmt.Printf("Number of paragraphs: %d\n", len(article.PlainText))
}
```

### Advanced Usage

```go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/mrjoshuak/readabiligo"
)

func main() {
	// Create a new extractor with custom options
	ext := readabiligo.New(
		readabiligo.WithContentDigests(true),    // Add content digest attributes
		readabiligo.WithNodeIndexes(true),       // Add node index attributes
		readabiligo.WithTimeout(time.Second*60), // Set a 60-second timeout
		readabiligo.WithDetectContentType(true), // Enable content type detection
		readabiligo.WithPreserveImportantLinks(true), // Preserve "Read more" links
	)

	// Extract from an HTML string
	html := `<html><body><h1>Article Title</h1><p>This is a paragraph.</p></body></html>`
	article, err := ext.ExtractFromHTML(html, nil)
	if err != nil {
		fmt.Printf("Error extracting article: %v\n", err)
		return
	}

	// Access article data
	fmt.Printf("Title: %s\n", article.Title)
	fmt.Printf("Content Type: %s\n", article.ContentType)

	// Access plain text paragraphs
	for i, block := range article.PlainText {
		fmt.Printf("Paragraph %d: %s\n", i+1, block.Text)
		if block.NodeIndex != "" {
			fmt.Printf("  Node index: %s\n", block.NodeIndex)
		}
	}
}

// For specific content types, specify them directly
func extractReferenceContent() {
	ext := readabiligo.New(
		// Specify content type directly (bypasses auto-detection)
		readabiligo.WithContentType(readabiligo.ContentTypeReference),
		// Disable auto-detection when using explicit type
		readabiligo.WithDetectContentType(false),
	)
	
	// Now the extractor will use reference-optimized extraction rules
}
```

> **Note**: For backward compatibility, the library can still be imported from `github.com/mrjoshuak/readabiligo/extractor` as in previous versions.

## Output Format

The extractor returns an `Article` struct with the following fields:

- `Title`: The article title
- `Byline`: Author information
- `Date`: Publication date
- `Content`: A simplified HTML representation of the article
- `PlainContent`: A "plain" version of the simplified HTML, preserving structure
- `PlainText`: A slice of text blocks, each representing a paragraph or list
- `ContentType`: The detected content type (Reference, Article, Technical, Error, Minimal)

Additional notes:

- All text is Unicode normalized using the NFKC normal form
- When content digests are enabled, each HTML element in `PlainContent` has a `data-content-digest` attribute containing a SHA256 hash of its content
- When node indexes are enabled, each HTML element in `PlainContent` has a `data-node-index` attribute describing its position in the HTML structure

## Differences from ReadabiliPy

ReadabiliGo is designed to be compatible with ReadabiliPy, with the following differences:

- Implemented in Go instead of Python
- Uses a pure Go implementation of Readability.js with no JavaScript dependencies
- Enhanced structure preservation, particularly beneficial for reference content
- Comprehensive content extraction that maintains document hierarchy and organization
- Improved link preservation for sources and references 
- Better preservation of headings and lists for more navigable extracted content
- Potentially better performance due to Go's efficiency compared to Python
- Concurrent extraction with timeout support
- Enhanced command-line interface with batch processing capabilities
- Multiple output format options (JSON, HTML, text)

### Content Extraction Philosophy

ReadabiliGo takes a different approach to content extraction compared to ReadabiliPy and other Readability implementations:

- **Content-Type Awareness**: ReadabiliGo automatically detects the type of content (Reference, Article, Technical, Error, Minimal) and applies specialized extraction rules for optimal results with each type.

- **Structure Preservation**: ReadabiliGo preserves more of the document's original structure, including heading hierarchy, lists, and reference links. This is particularly valuable for reference content like Wikipedia articles, technical documentation, and educational materials.

- **Content-Rich Extraction**: While other implementations focus on aggressive cleaning and simplification, ReadabiliGo maintains more of the content's context and related elements. This approach provides a richer reading experience, especially for complex, structured content.

- **Reference Link Preservation**: ReadabiliGo is more likely to preserve functional links to sources, citations, and related content, making the extracted content more useful for research and fact-checking.

- **Intelligent Cleaning**: For error pages and minimal content, ReadabiliGo applies more aggressive cleaning to focus on the essential content and remove navigation and other non-content elements.

This approach makes ReadabiliGo particularly well-suited for:

- Reference material and documentation
- Technical content with code examples
- Educational content with structured information
- Research articles with citations and references
- Any content where structure and organization are important to understanding

For simpler content like news articles, both approaches produce similar results, but ReadabiliGo may provide additional context and structure that enhances the reading experience.

#### Content Types

ReadabiliGo detects and optimizes extraction for five content types:

1. **Reference** (Wikipedia, documentation): Preserves more structure, headings, lists, and citations
2. **Article** (News, blog posts): Standard extraction with balanced cleaning
3. **Technical** (Code examples, tutorials): Preserves code blocks and technical details
4. **Error** (404, error pages): Aggressive cleaning to focus on error messages
5. **Minimal** (Login pages, simple forms): Focuses on core content only

## License

MIT License, see the `LICENSE` file.

Copyright (c) 2025, Joshua Kolden

If you encounter any issues or have suggestions for improvement, please open an issue on GitHub.
