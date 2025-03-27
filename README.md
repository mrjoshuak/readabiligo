# ReadabiliGo

ReadabiliGo is a Go implementation of Mozilla's [Readability.js](https://github.com/mozilla/readability) library, designed to extract readable content from HTML pages. It provides both a command-line interface and a Go library for article extraction.

This package is a Go port of [ReadabiliPy](https://github.com/alan-turing-institute/ReadabiliPy) by [The Alan Turing Institute](https://github.com/alan-turing-institute), Ed Chalstrey, James Robinson, Martin O'Reilly, Gertjan van den Burg, Nelson Liu, and many other valuable contributors. ReadabiliGo maintains compatibility with ReadabiliPy's output format and features while providing a native Go implementation.

## Features

- Extract article content, title, byline, and date from HTML
- Output in JSON, HTML, or plain text formats
- Support for content digests and node indexes for tracking HTML structure
- 100% Pure Go implementation, no JavaScript dependencies
- Comprehensive test suite with real-world examples

## Installation

### Prerequisites

No external dependencies are required! ReadabiliGo now uses a pure Go implementation of the Readability algorithm.

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

	// Access plain text paragraphs
	for i, block := range article.PlainText {
		fmt.Printf("Paragraph %d: %s\n", i+1, block.Text)
		if block.NodeIndex != "" {
			fmt.Printf("  Node index: %s\n", block.NodeIndex)
		}
	}
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

Additional notes:

- All text is Unicode normalized using the NFKC normal form
- When content digests are enabled, each HTML element in `PlainContent` has a `data-content-digest` attribute containing a SHA256 hash of its content
- When node indexes are enabled, each HTML element in `PlainContent` has a `data-node-index` attribute describing its position in the HTML structure

## Differences from ReadabiliPy

ReadabiliGo is designed to be compatible with ReadabiliPy, with the following differences:

- Implemented in Go instead of Python
- Uses a pure Go implementation of Readability.js with no JavaScript dependencies
- Potentially better performance due to Go's efficiency compared to Python
- Concurrent extraction with timeout support
- Enhanced command-line interface with batch processing capabilities
- Multiple output format options (JSON, HTML, text)

## License

MIT License, see the `LICENSE` file.

Copyright (c) 2025, Joshua Kolden

If you encounter any issues or have suggestions for improvement, please open an issue on GitHub.
