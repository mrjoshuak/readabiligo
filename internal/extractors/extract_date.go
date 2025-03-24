package extractors

import (
	"sort"
	"time"
)

// ExtractDate extracts the article date from HTML content
func ExtractDate(html string) time.Time {
	// List of XPaths for HTML tags that could contain a date
	// Scores reflect confidence in these XPaths and the preference used for extraction
	xpaths := []XPathScore{
		{XPath: "//meta[@property=\"article:published_time\"]/@content", Score: 13},
		{XPath: "//meta[@property=\"og:updated_time\"]/@content", Score: 10},
		{XPath: "//meta[@property=\"og:article:published_time\"]/@content", Score: 10},
		{XPath: "//meta[@property=\"og:article:modified_time\"]/@content", Score: 10},
		{XPath: "//meta[@property=\"article:published\"]/@content", Score: 7},
		{XPath: "//meta[@itemprop=\"datePublished\"]/@content", Score: 3},
		{XPath: "//time/@datetime", Score: 3},
		{XPath: "//meta[@itemprop=\"dateModified\"]/@content", Score: 2},
		{XPath: "//meta[@property=\"article:modified_time\"]/@content", Score: 2},
	}

	// Extract dates using the XPaths
	extractedDates := ExtractElement(html, xpaths, nil)
	if len(extractedDates) == 0 {
		return time.Time{}
	}

	// Create a slice of date strings sorted by score
	type dateScore struct {
		dateStr string
		score   int
	}
	dateScores := make([]dateScore, 0, len(extractedDates))
	for dateStr, element := range extractedDates {
		dateScores = append(dateScores, dateScore{dateStr, element.Score})
	}

	// Sort by score in descending order
	sort.Slice(dateScores, func(i, j int) bool {
		return dateScores[i].score > dateScores[j].score
	})

	// Try to parse each date string in order of score
	for _, ds := range dateScores {
		parsedTime := ensureISODateFormat(ds.dateStr)
		if !parsedTime.IsZero() {
			return parsedTime
		}
	}

	return time.Time{}
}

// ensureISODateFormat parses a date string in various ISO formats
func ensureISODateFormat(dateStr string) time.Time {
	// Supported date formats
	formats := []string{
		time.RFC3339,                 // "2006-01-02T15:04:05Z07:00"
		"2006-01-02T15:04:05",        // "2014-10-24T17:32:46"
		"2006-01-02T15:04:05Z",       // "2014-10-24T17:32:46Z"
		"2006-01-02T15:04:05.999Z",   // "2014-10-24T17:32:46.000Z"
		"2006-01-02T15:04:05.999999", // "2014-10-24T17:32:46.493"
	}

	for _, format := range formats {
		parsedTime, err := time.Parse(format, dateStr)
		if err == nil {
			// Set timezone to UTC and remove microseconds for consistency
			return parsedTime.UTC().Truncate(time.Second)
		}
	}

	// Special case for timezone with colon
	if len(dateStr) > 3 && dateStr[len(dateStr)-3] == ':' {
		// Try to parse by removing the colon in the timezone
		modifiedDateStr := dateStr[:len(dateStr)-3] + dateStr[len(dateStr)-2:]
		parsedTime, err := time.Parse(time.RFC3339, modifiedDateStr)
		if err == nil {
			return parsedTime.UTC().Truncate(time.Second)
		}
	}

	return time.Time{}
}
