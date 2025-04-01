# ReadabiliGo

ReadabiliGo is a Go implementation of Mozilla's [Readability.js](https://github.com/mozilla/readability) library, designed to extract readable content from HTML pages. It provides both a command-line interface and a Go library for article extraction.

This package is a Go port of [ReadabiliPy](https://github.com/alan-turing-institute/ReadabiliPy) by [The Alan Turing Institute](https://github.com/alan-turing-institute), Ed Chalstrey, James Robinson, Martin O'Reilly, Gertjan van den Burg, Nelson Liu, and many other valuable contributors. ReadabiliGo aims for compatibility with ReadabiliPy's output format while providing a pure Go implementation with some enhancements.

## Features

- Extract article content, title, byline, and date from HTML
- Faithful implementation of Mozilla's Readability.js algorithm
- Consistent content extraction for all types of documents
- Good structure preservation and heading hierarchy
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
        Deprecated: No longer has any effect, maintained for backward compatibility
  -content-type string
        Deprecated: No longer has any effect, maintained for backward compatibility
  -preserve-links
        Preserve important links in cleanup
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
```


## Output Format

The extractor returns an `Article` struct with the following fields:

- `Title`: The article title
- `Byline`: Author information
- `Date`: Publication date
- `Content`: A simplified HTML representation of the article
- `PlainContent`: A "plain" version of the simplified HTML, preserving structure
- `PlainText`: A slice of text blocks, each representing a paragraph or list
- `ContentType`: The content type field (maintained for backward compatibility, always set to "Article")

Additional notes:

- All text is Unicode normalized using the NFKC normal form
- When content digests are enabled, each HTML element in `PlainContent` has a `data-content-digest` attribute containing a SHA256 hash of its content
- When node indexes are enabled, each HTML element in `PlainContent` has a `data-node-index` attribute describing its position in the HTML structure

## Differences from ReadabiliPy

ReadabiliGo aims to be compatible with ReadabiliPy's output format, with these key differences:

- Implemented in Go instead of Python for better performance and portability
- Uses a pure Go implementation with no external dependencies
- Enhanced structure preservation, particularly beneficial for reference content
- Title extraction prioritizes h1 elements with itemprop="headline" matching Python's behavior
- Comprehensive content extraction that maintains document hierarchy and organization
- Improved link preservation for sources and references 
- Better preservation of headings and lists for more navigable extracted content
- Superior performance due to Go's efficiency compared to Python
- Concurrent extraction with configurable timeout support
- Enhanced command-line interface with batch processing capabilities
- Multiple output format options (JSON, HTML, text)

### Content Extraction Philosophy

ReadabiliGo aims to provide a faithful implementation of Mozilla's Readability.js algorithm in pure Go. Key aspects of the implementation:

- **Mozilla Compatible**: ReadabiliGo follows the same core algorithm as Mozilla's Readability.js, ensuring consistent results across implementations.

- **Structure Preservation**: The algorithm preserves appropriate document structure, including heading hierarchy, lists, and reference links, while removing non-content elements.

- **Balanced Extraction**: Following Mozilla's approach, ReadabiliGo balances between aggressive cleaning and preserving important context and related elements.

- **Reference Link Preservation**: The implementation preserves functional links to sources, citations, and related content where appropriate.

- **Unified Approach**: ReadabiliGo uses a single, unified algorithm for all content types, following Mozilla's implementation philosophy.

ReadabiliGo works well with various content types:

- News articles and blog posts
- Reference material and documentation
- Technical content with code examples
- Educational content
- Research articles with citations

The algorithm analyzes the content structure and applies appropriate extraction rules based on the content characteristics, without relying on predefined content type categories.

## License

MIT License, see the `LICENSE` file.

Copyright (c) 2025, Joshua Kolden

If you encounter any issues or have suggestions for improvement, please open an issue on GitHub.
