// Package main provides the command-line interface for ReadabiliGo.
// It allows extracting readable content from HTML files or standard input
// and outputting the results in various formats.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mrjoshuak/readabiligo"
)

// OutputFormat represents the supported output formats for the extracted content.
// The available formats are JSON, HTML, and plain text.
type OutputFormat string

const (
	FormatJSON OutputFormat = "json"
	FormatHTML OutputFormat = "html"
	FormatText OutputFormat = "text"
)

func main() {
	// Define command-line flags
	inputFiles := flag.String("input", "", "Input HTML file path(s) (comma-separated, use '-' for stdin)")
	outputDir := flag.String("output-dir", "", "Output directory for batch processing (default: same as input)")
	outputFile := flag.String("output", "", "Output file path (default: stdout)")
	formatStr := flag.String("format", "json", "Output format: json, html, or text")
	useReadability := flag.Bool("js", false, "DEPRECATED: No effect - JavaScript implementation has been removed")
	contentDigests := flag.Bool("digests", false, "Add content digest attributes")
	nodeIndexes := flag.Bool("indexes", false, "Add node index attributes")
	compact := flag.Bool("compact", false, "Output compact JSON without indentation")
	timeout := flag.Duration("timeout", 30*time.Second, "Timeout for extraction")
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show help information")

	// Customize usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "ReadabiliGo - Extract readable content from HTML\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -input article.html -output article.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -input article.html -format html -output article.html\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -input article1.html,article2.html -output-dir ./extracted\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  cat article.html | %s -input - > article.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -input article.html -digests -indexes\n", os.Args[0])
	}

	flag.Parse()

	// Show help if requested
	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}

	// Show version information if requested
	if *showVersion {
		fmt.Printf("ReadabiliGo version %s\n", readabiligo.Version)
		os.Exit(0)
	}

	// Validate output format
	format := OutputFormat(strings.ToLower(*formatStr))
	if format != FormatJSON && format != FormatHTML && format != FormatText {
		fmt.Printf("Invalid output format: %s. Must be one of: json, html, text\n", *formatStr)
		os.Exit(1)
	}

	// Parse input files
	var inputs []string
	if *inputFiles == "" || *inputFiles == "-" {
		// Read from stdin
		inputs = []string{"-"}
	} else {
		// Split comma-separated input files
		inputs = strings.Split(*inputFiles, ",")
	}

	// Check for deprecated flag
	if *useReadability {
		fmt.Println("Warning: The -js flag is deprecated and has no effect. The JavaScript implementation has been removed and only the pure Go implementation is available.")
	}

	// Create the extractor with options
	ext := readabiligo.New(
		readabiligo.WithReadability(*useReadability),
		readabiligo.WithContentDigests(*contentDigests),
		readabiligo.WithNodeIndexes(*nodeIndexes),
		readabiligo.WithTimeout(*timeout),
	)

	// Process each input file
	for _, inputPath := range inputs {
		var input io.ReadCloser
		var outputPath string

		// Determine input source
		if inputPath == "-" {
			// Read from stdin
			input = os.Stdin
			outputPath = *outputFile // Use the specified output file
		} else {
			// Open the input file
			file, err := os.Open(inputPath)
			if err != nil {
				fmt.Printf("Error opening input file %s: %v\n", inputPath, err)
				continue
			}
			defer file.Close()
			input = file

			// Determine output path
			if *outputDir != "" {
				// Create output directory if it doesn't exist
				err = os.MkdirAll(*outputDir, 0755)
				if err != nil {
					fmt.Printf("Error creating output directory: %v\n", err)
					os.Exit(1)
				}

				// Use input filename with appropriate extension in output directory
				baseName := filepath.Base(inputPath)
				ext := filepath.Ext(baseName)
				nameWithoutExt := strings.TrimSuffix(baseName, ext)

				var outputExt string
				switch format {
				case FormatJSON:
					outputExt = ".json"
				case FormatHTML:
					outputExt = ".html"
				case FormatText:
					outputExt = ".txt"
				}

				outputPath = filepath.Join(*outputDir, nameWithoutExt+outputExt)
			} else if *outputFile != "" && len(inputs) == 1 {
				// Use specified output file only if processing a single input
				outputPath = *outputFile
			} else if *outputFile == "" {
				// Default: use stdout
				outputPath = ""
			} else {
				// Multiple inputs with single output file - use stdout and warn
				fmt.Println("Warning: Multiple input files with single output file specified. Using stdout.")
				outputPath = ""
			}
		}

		// Extract the article
		article, err := ext.ExtractFromReader(input, nil)
		if err != nil {
			fmt.Printf("Error extracting article from %s: %v\n", inputPath, err)
			continue
		}

		// Generate output based on format
		var outputData []byte
		switch format {
		case FormatJSON:
			if *compact {
				outputData, err = json.Marshal(article)
			} else {
				outputData, err = json.MarshalIndent(article, "", "  ")
			}
			if err != nil {
				fmt.Printf("Error converting article to JSON: %v\n", err)
				continue
			}
		case FormatHTML:
			outputData = []byte(article.Content)
		case FormatText:
			// Concatenate all plain text blocks
			var textBuilder strings.Builder
			for _, block := range article.PlainText {
				textBuilder.WriteString(block.Text)
				textBuilder.WriteString("\n\n")
			}
			outputData = []byte(textBuilder.String())
		}

		// Write the output
		var output io.Writer = os.Stdout
		if outputPath != "" {
			file, err := os.Create(outputPath)
			if err != nil {
				fmt.Printf("Error creating output file %s: %v\n", outputPath, err)
				continue
			}
			defer file.Close()
			output = file
			fmt.Printf("Processed %s -> %s\n", inputPath, outputPath)
		}

		// Write the data to the output
		_, err = output.Write(outputData)
		if err != nil {
			fmt.Printf("Error writing output: %v\n", err)
			continue
		}

		// Add a newline if writing to stdout
		if output == os.Stdout {
			fmt.Println()
		}
	}
}
