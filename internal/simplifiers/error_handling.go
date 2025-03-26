package simplifiers

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ErrorHandlingOptions defines options for error handling
type ErrorHandlingOptions struct {
	TimeoutMs      int
	MaxRetries     int
	LogErrors      bool
	EnableFallback bool
}

// DefaultErrorHandlingOptions provides default options for error handling
var DefaultErrorHandlingOptions = ErrorHandlingOptions{
	TimeoutMs:      5000,
	MaxRetries:     3,
	LogErrors:      true,
	EnableFallback: true,
}

// ErrorWithFallback represents an error that occurred but a fallback was executed
type ErrorWithFallback struct {
	Err              error
	FallbackExecuted bool
}

// Error implements the error interface
func (e *ErrorWithFallback) Error() string {
	if e.FallbackExecuted {
		return fmt.Sprintf("error occurred but fallback was executed: %v", e.Err)
	}
	return e.Err.Error()
}

// Unwrap returns the underlying error
func (e *ErrorWithFallback) Unwrap() error {
	return e.Err
}

// WithTimeout executes a function with a timeout
func WithTimeout(timeoutMs int, fn func() (interface{}, error)) (interface{}, error) {
	resultCh := make(chan struct {
		result interface{}
		err    error
	})

	go func() {
		result, err := fn()
		resultCh <- struct {
			result interface{}
			err    error
		}{result, err}
	}()

	select {
	case res := <-resultCh:
		return res.result, res.err
	case <-time.After(time.Duration(timeoutMs) * time.Millisecond):
		return nil, fmt.Errorf("operation timed out after %d ms", timeoutMs)
	}
}

// WithRetry executes a function with retries
func WithRetry(maxRetries int, fn func() (interface{}, error)) (interface{}, error) {
	var lastErr error
	for i := 0; i <= maxRetries; i++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		lastErr = err
		if i < maxRetries {
			// Exponential backoff
			time.Sleep(time.Duration(100*(1<<i)) * time.Millisecond)
		}
	}
	return nil, fmt.Errorf("operation failed after %d retries: %w", maxRetries, lastErr)
}

// WithFallback executes a function with a fallback
func WithFallback(fn func() (interface{}, error), fallbackFn func() interface{}) (interface{}, error) {
	result, err := fn()
	if err == nil {
		return result, nil
	}
	return fallbackFn(), err
}

// LogError logs an error if logging is enabled
func LogError(err error, opts ErrorHandlingOptions) {
	if opts.LogErrors {
		log.Printf("Error: %v", err)
	}
}

// ExtractWithErrorHandling extracts content with error handling
func ExtractWithErrorHandling(html string, opts ErrorHandlingOptions) (*goquery.Document, error) {
	// Try the primary extraction method with timeout
	doc, err := WithTimeout(opts.TimeoutMs, func() (interface{}, error) {
		return goquery.NewDocumentFromReader(strings.NewReader(html))
	})

	if err == nil {
		return doc.(*goquery.Document), nil
	}

	// Log the error
	LogError(err, opts)

	if !opts.EnableFallback {
		return nil, err
	}

	// Try fallback method: repair invalid HTML
	doc, err = WithFallback(
		func() (interface{}, error) {
			return repairInvalidHTML(html)
		},
		func() interface{} {
			// Last resort: create a basic document
			doc, _ := goquery.NewDocumentFromReader(strings.NewReader("<html><body></body></html>"))
			return doc
		},
	)

	if err != nil {
		return doc.(*goquery.Document), &ErrorWithFallback{
			Err:              err,
			FallbackExecuted: true,
		}
	}

	return doc.(*goquery.Document), nil
}

// repairInvalidHTML attempts to repair invalid HTML
func repairInvalidHTML(html string) (*goquery.Document, error) {
	// Simple HTML repair: add missing tags
	if !strings.Contains(html, "<html") {
		html = "<html>" + html + "</html>"
	}
	if !strings.Contains(html, "<body") {
		html = strings.Replace(html, "<html>", "<html><body>", 1)
		html = strings.Replace(html, "</html>", "</body></html>", 1)
	}
	if !strings.Contains(html, "<head") {
		html = strings.Replace(html, "<html>", "<html><head></head>", 1)
	}

	// Try to parse the repaired HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, errors.New("failed to repair invalid HTML: " + err.Error())
	}

	return doc, nil
}

// FindMainContentWithFallback finds the main content with fallbacks
func FindMainContentWithFallback(doc *goquery.Document) *goquery.Selection {
	// Try to find the main content using the primary method
	mainContent := FindMainContentNode(doc)
	if mainContent.Length() > 0 && len(mainContent.Text()) > 100 {
		return mainContent
	}

	// Fallback 1: Try to find article or main elements
	mainContent = doc.Find("article, main")
	if mainContent.Length() > 0 && len(mainContent.Text()) > 100 {
		return mainContent
	}

	// Fallback 2: Try to find elements with content-related classes/IDs
	mainContent = doc.Find("[id*='content'], [class*='content'], [id*='article'], [class*='article'], [id*='main'], [class*='main']")
	if mainContent.Length() > 0 && len(mainContent.Text()) > 100 {
		return mainContent
	}

	// Fallback 3: Try to find the element with the most text
	var bestElement *goquery.Selection
	var maxTextLength int
	doc.Find("div, section").Each(func(i int, s *goquery.Selection) {
		textLength := len(s.Text())
		if textLength > maxTextLength {
			maxTextLength = textLength
			bestElement = s
		}
	})

	if bestElement != nil && maxTextLength > 100 {
		return bestElement
	}

	// Last resort: return the body
	return doc.Find("body")
}

// ExtractTextWithFallback extracts text with fallbacks
func ExtractTextWithFallback(s *goquery.Selection) string {
	// Try to extract text using the primary method
	text := s.Text()
	if text != "" {
		return text
	}

	// Fallback 1: Try to extract text from paragraphs
	var paragraphText strings.Builder
	s.Find("p").Each(func(i int, p *goquery.Selection) {
		paragraphText.WriteString(p.Text())
		paragraphText.WriteString("\n\n")
	})

	if paragraphText.Len() > 0 {
		return paragraphText.String()
	}

	// Fallback 2: Try to extract text from divs
	var divText strings.Builder
	s.Find("div").Each(func(i int, div *goquery.Selection) {
		divText.WriteString(div.Text())
		divText.WriteString("\n")
	})

	if divText.Len() > 0 {
		return divText.String()
	}

	// Last resort: return any text we can find
	return s.Find("*").Text()
}
