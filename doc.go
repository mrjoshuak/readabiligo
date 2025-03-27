/*
Package readabiligo is a Go implementation of Mozilla's Readability.js library for extracting
the main content from HTML pages. It is designed to remove navigation, advertisements,
and other distractions, leaving only the article content.

This package is a Go port of ReadabiliPy by The Alan Turing Institute with a 100% pure
Go implementation that requires no external dependencies.

Basic Usage:

    import "github.com/mrjoshuak/readabiligo"

    // Create a new extractor
    ext := readabiligo.New()

    // Extract from HTML string
    article, err := ext.ExtractFromHTML(htmlString, nil)
    if err != nil {
        // Handle error
    }

    // Access article data
    fmt.Printf("Title: %s\n", article.Title)
    fmt.Printf("Byline: %s\n", article.Byline)
    fmt.Printf("Date: %s\n", article.Date)
    fmt.Printf("Content: %s\n", article.Content)

    // Access plain text paragraphs
    for i, block := range article.PlainText {
        fmt.Printf("Paragraph %d: %s\n", i+1, block.Text)
    }

Advanced Usage with Options:

    // Create a new extractor with custom options
    ext := readabiligo.New(
        readabiligo.WithContentDigests(true),
        readabiligo.WithNodeIndexes(true),
        readabiligo.WithTimeout(time.Second*60),
    )

    // Extract from a reader (like a file or HTTP response)
    article, err := ext.ExtractFromReader(reader, nil)

Features:

- Extract article content, title, byline, and date from HTML
- Pure Go implementation with no external dependencies
- Support for content digests and node indexes for tracking HTML structure
- Configurable timeout for extraction process
- Compatible with ReadabiliPy output format
*/
package readabiligo