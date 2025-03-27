package extractors

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ExtractDate extracts the article date from HTML content
func ExtractDate(html string) time.Time {
	// ---- STEP 1: Extract dates from metadata tags ----
	// List of selectors for HTML tags that could contain a date
	// Scores reflect confidence in these selectors and the preference used for extraction
	selectors := []SelectorScore{
		{Selector: "//meta[@property='article:published_time']/@content", Score: 13},
		{Selector: "//meta[@property='og:updated_time']/@content", Score: 10},
		{Selector: "//meta[@property='og:article:published_time']/@content", Score: 10},
		{Selector: "//meta[@property='og:article:modified_time']/@content", Score: 10},
		{Selector: "//meta[@name='pubdate']/@content", Score: 10},
		{Selector: "//meta[@name='publishdate']/@content", Score: 10},
		{Selector: "//meta[@name='date']/@content", Score: 9},
		{Selector: "//meta[@property='article:published']/@content", Score: 7},
		{Selector: "//meta[@itemprop='datePublished']/@content", Score: 3},
		{Selector: "//time/@datetime", Score: 3},
		{Selector: "//meta[@itemprop='dateModified']/@content", Score: 2},
		{Selector: "//meta[@property='article:modified_time']/@content", Score: 2},
		{Selector: "//meta[@name='DC.date.issued']/@content", Score: 2},
		{Selector: "//meta[@name='DC.date.created']/@content", Score: 2},
		{Selector: "//meta[@name='DC.date.modified']/@content", Score: 1},
		{Selector: "//meta[@name='dcterms.modified']/@content", Score: 1},
		{Selector: "//meta[@name='dcterms.created']/@content", Score: 1},
	}

	// Extract dates from metadata using the selectors
	extractedDates := ExtractElement(html, selectors, nil)
	
	// ---- STEP 2: Extract dates from visible elements ----
	// Additional selectors for visible dates in common article structures
	visibleDateSelectors := []SelectorScore{
		{Selector: "//span[@class='date']", Score: 3},
		{Selector: "//span[@class='time']", Score: 3},
		{Selector: "//span[@class='timestamp']", Score: 3},
		{Selector: "//span[@class='published']", Score: 3},
		{Selector: "//time", Score: 2},
		{Selector: "//span[contains(@class, 'date')]", Score: 2},
		{Selector: "//div[contains(@class, 'date')]", Score: 2},
		{Selector: "//p[contains(@class, 'date')]", Score: 2},
		{Selector: "//p[contains(@class, 'time')]", Score: 2},
		{Selector: "//div[contains(@class, 'byline')]", Score: 1}, // Often contains date with byline
		{Selector: "//p[contains(@class, 'byline')]", Score: 1},
		{Selector: "//*[contains(@class, 'dateline')]", Score: 1},
	}
	
	// Extract visible dates
	visibleDates := ExtractElement(html, visibleDateSelectors, nil)
	
	// Combine all extracted dates
	allDates := make([]dateEntry, 0)
	
	// Process metadata dates
	for dateStr, element := range extractedDates {
		allDates = append(allDates, dateEntry{
			dateStr: dateStr,
			score:   element.Score,
			source:  "metadata",
		})
	}
	
	// Process visible dates
	for dateStr, element := range visibleDates {
		allDates = append(allDates, dateEntry{
			dateStr: dateStr,
			score:   element.Score,
			source:  "visible",
		})
	}
	
	// If we have no dates after extraction, return empty time
	if len(allDates) == 0 {
		return time.Time{}
	}

	// Sort by score in descending order
	sort.Slice(allDates, func(i, j int) bool {
		return allDates[i].score > allDates[j].score
	})

	// Try to parse each date string in order of score
	var firstFoundDate time.Time
	
	for _, entry := range allDates {
		var parsedTime time.Time
		
		// For metadata sources, try ISO format parsing first (higher priority)
		if entry.source == "metadata" {
			parsedTime = ParseISO8601Format(entry.dateStr)
			if !parsedTime.IsZero() {
				// Return ISO dates directly since they often include time information
				return parsedTime
			}
		}
		
		// Then try comprehensive format parsing for all sources
		parsedTime = ParseFlexibleDateFormat(entry.dateStr)
		if !parsedTime.IsZero() {
			// For regular date parsing, check if we have time information
			if parsedTime.Hour() != 0 || parsedTime.Minute() != 0 || parsedTime.Second() != 0 {
				// Return immediately with time information
				return parsedTime
			} else if firstFoundDate.IsZero() {
				// Store the first date without time info
				// We'll return this later if no date with time info is found
				firstFoundDate = parsedTime
			}
		}
	}
	
	// Return the first found date if we have one
	if !firstFoundDate.IsZero() {
		return firstFoundDate
	}

	// If we still have no valid date, try extracting relative dates
	if relativeDate := ExtractRelativeDate(html); !relativeDate.IsZero() {
		return relativeDate
	}

	return time.Time{}
}

// dateEntry represents a date string with its score and source
type dateEntry struct {
	dateStr string
	score   int
	source  string
}

// ParseISO8601Format parses a date string in various ISO formats
func ParseISO8601Format(dateStr string) time.Time {
	// Cleanup the string
	dateStr = strings.TrimSpace(dateStr)
	
	// Supported ISO8601 date formats
	formats := []string{
		time.RFC3339,                 // "2006-01-02T15:04:05Z07:00"
		"2006-01-02T15:04:05",        // "2014-10-24T17:32:46"
		"2006-01-02T15:04:05Z",       // "2014-10-24T17:32:46Z"
		"2006-01-02T15:04:05.999Z",   // "2014-10-24T17:32:46.000Z"
		"2006-01-02T15:04:05.999999", // "2014-10-24T17:32:46.493"
		"2006-01-02T15:04:05-0700",   // Without colon in timezone
		"2006-01-02T15:04:05+0000",   // Without colon in timezone
		"2006-01-02",                 // Just date
		"2006-01-02Z",                // Just date with Z
		"20060102T150405Z",           // Compact format
		time.RFC1123,                 // HTTP format
		time.RFC1123Z,                // HTTP format with numeric zone
		time.RFC822,                  // Old RFC822
		time.RFC822Z,                 // Old RFC822 with numeric zone
		time.RFC850,                  // Old RFC850
	}

	for _, format := range formats {
		parsedTime, err := time.Parse(format, dateStr)
		if err == nil {
			// Set timezone to UTC and remove microseconds for consistency
			return parsedTime.UTC().Truncate(time.Second)
		}
	}

	// Special case for timezone with colon
	if len(dateStr) > 3 && (dateStr[len(dateStr)-3] == ':' && (dateStr[len(dateStr)-6] == '+' || dateStr[len(dateStr)-6] == '-')) {
		// Try to parse by removing the colon in the timezone
		modifiedDateStr := dateStr[:len(dateStr)-3] + dateStr[len(dateStr)-2:]
		parsedTime, err := time.Parse(time.RFC3339, modifiedDateStr)
		if err == nil {
			return parsedTime.UTC().Truncate(time.Second)
		}
	}

	return time.Time{}
}

// ParseFlexibleDateFormat handles a wide variety of date formats
func ParseFlexibleDateFormat(dateStr string) time.Time {
	// Cleanup and normalize
	dateStr = CleanupDateString(dateStr)
	if dateStr == "" {
		return time.Time{}
	}
	
	// Try various date formats
	
	// First try ISO formats again (after cleanup)
	if parsed := ParseISO8601Format(dateStr); !parsed.IsZero() {
		return parsed
	}
	
	// Try standard regional formats
	if parsed := ParseRegionalDateFormats(dateStr); !parsed.IsZero() {
		return parsed
	}
	
	// Try natural language date formats
	if parsed := ParseNaturalLanguageDates(dateStr); !parsed.IsZero() {
		return parsed
	}
	
	// Try extracting date components from a variety of formats
	if parsed := ParseDateComponents(dateStr); !parsed.IsZero() {
		return parsed
	}
	
	return time.Time{}
}

// CleanupDateString sanitizes date strings for parsing
func CleanupDateString(dateStr string) string {
	// Convert to lowercase for easier pattern matching
	dateStr = strings.TrimSpace(dateStr)
	
	// Remove HTML tags if present
	reHTML := regexp.MustCompile("<[^>]*>")
	dateStr = reHTML.ReplaceAllString(dateStr, " ")
	
	// Remove multiple spaces
	reSpaces := regexp.MustCompile(`\s+`)
	dateStr = reSpaces.ReplaceAllString(dateStr, " ")
	
	// Remove strings commonly surrounding dates
	removePatterns := []string{
		"published:", "published ", "updated:", "updated ",
		"date:", "date ", "on ", "posted on ", "written on ",
		"on date ", "as of ", "posted ", "written ",
	}
	
	// Convert the string to lowercase for case-insensitive matching
	lowerDateStr := strings.ToLower(dateStr)
	
	// Try each removal pattern
	for _, pattern := range removePatterns {
		// Find the index of the pattern in the lowercase string
		index := strings.Index(lowerDateStr, pattern)
		if index >= 0 {
			// If found, remove that portion from the original string
			// This preserves case in the remaining part
			prefix := dateStr[:index]
			suffix := dateStr[index+len(pattern):]
			dateStr = prefix + suffix
			
			// Update the lowercase version for next iteration
			lowerDateStr = strings.ToLower(dateStr)
		}
	}
	
	return strings.TrimSpace(dateStr)
}

// ParseRegionalDateFormats tries common regional date formats
func ParseRegionalDateFormats(dateStr string) time.Time {
	formats := []string{
		// U.S. formats (MM/DD/YYYY)
		"01/02/2006", "01-02-2006", "01.02.2006",
		
		// European formats (DD/MM/YYYY)
		"02/01/2006", "02-01-2006", "02.01.2006",
		
		// Variations with time
		"01/02/2006 15:04:05", "01/02/2006 15:04", "01/02/2006 3:04 PM",
		"02/01/2006 15:04:05", "02/01/2006 15:04", "02/01/2006 3:04 PM",
		
		// Variations with 2-digit year
		"01/02/06", "02/01/06", "01-02-06", "02-01-06",
		
		// Month name formats
		"January 2, 2006", "2 January 2006", "Jan 2, 2006", "2 Jan 2006",
		"January 2, 2006 15:04", "2 January 2006 15:04",
		"January 2, 2006 3:04 PM", "2 January 2006 3:04 PM",
		
		// Year first formats
		"2006/01/02", "2006-01-02", "2006.01.02",
		"2006/02/01", "2006-02-01", "2006.02.01",
	}
	
	// Try each format
	for _, format := range formats {
		parsedTime, err := time.Parse(format, dateStr)
		if err == nil {
			// For ambiguous formats (could be MM/DD or DD/MM),
			// apply sanity check on the month and day values
			if strings.Contains(format, "01/02") || strings.Contains(format, "01-02") || 
			   strings.Contains(format, "01.02") {
				// If day value in parsed time > 12, it's not a valid month, so must be DD/MM format
				if parsedTime.Day() > 12 && parsedTime.Month() <= 12 {
					// Re-parse with the opposite format
					reverseFormat := strings.Replace(format, "01/02", "02/01", 1)
					reverseFormat = strings.Replace(reverseFormat, "01-02", "02-01", 1)
					reverseFormat = strings.Replace(reverseFormat, "01.02", "02.01", 1)
					reverseParsedTime, reverseErr := time.Parse(reverseFormat, dateStr)
					if reverseErr == nil {
						return reverseParsedTime.UTC().Truncate(time.Second)
					}
				}
			}
			
			return parsedTime.UTC().Truncate(time.Second)
		}
	}
	
	return time.Time{}
}

// ParseNaturalLanguageDates handles common textual date formats
func ParseNaturalLanguageDates(dateStr string) time.Time {
	// Handle the specific test case for "Year Month Day" format
	yearMonthDayRegex := regexp.MustCompile(`^(\d{4})\s+(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{1,2})$`)
	if matches := yearMonthDayRegex.FindStringSubmatch(dateStr); len(matches) == 4 {
		year, _ := strconv.Atoi(matches[1])
		monthStr := matches[2]
		day, _ := strconv.Atoi(matches[3])
		
		// Convert month name to number
		var month time.Month
		switch strings.ToLower(monthStr) {
		case "january": month = time.January
		case "february": month = time.February
		case "march": month = time.March
		case "april": month = time.April
		case "may": month = time.May
		case "june": month = time.June
		case "july": month = time.July
		case "august": month = time.August
		case "september": month = time.September
		case "october": month = time.October
		case "november": month = time.November
		case "december": month = time.December
		}
		
		if month > 0 && day >= 1 && day <= 31 {
			return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		}
	}
	
	// Clean up string for consistent parsing
	dateStr = strings.ToLower(dateStr)
	dateStr = strings.TrimSpace(dateStr)
	
	// Month name mapping
	months := map[string]int{
		"january": 1, "jan": 1,
		"february": 2, "feb": 2,
		"march": 3, "mar": 3,
		"april": 4, "apr": 4,
		"may": 5,
		"june": 6, "jun": 6,
		"july": 7, "jul": 7,
		"august": 8, "aug": 8,
		"september": 9, "sep": 9, "sept": 9,
		"october": 10, "oct": 10,
		"november": 11, "nov": 11,
		"december": 12, "dec": 12,
	}
	
	// Patterns for month, day, year
	monthPattern := `(january|february|march|april|may|june|july|august|september|october|november|december|jan|feb|mar|apr|jun|jul|aug|sep|sept|oct|nov|dec)`
	dayPattern := `(\d{1,2})(st|nd|rd|th)?`
	yearPattern := `(\d{4}|\d{2})`
	
	// Try various date patterns
	
	// Pattern: Month Day, Year
	re1 := regexp.MustCompile(monthPattern + `\s+` + dayPattern + `(\s*,\s*|\s+)` + yearPattern)
	if matches := re1.FindStringSubmatch(dateStr); len(matches) >= 4 {
		month := months[matches[1]]
		day, _ := strconv.Atoi(matches[2])
		year, _ := strconv.Atoi(matches[5])
		
		// Handle 2-digit years
		if year < 100 {
			if year < 50 {
				year += 2000
			} else {
				year += 1900
			}
		}
		
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	}
	
	// Pattern: Day Month Year
	re2 := regexp.MustCompile(dayPattern + `\s+` + monthPattern + `(\s*,\s*|\s+)` + yearPattern)
	if matches := re2.FindStringSubmatch(dateStr); len(matches) >= 4 {
		day, _ := strconv.Atoi(matches[1])
		month := months[matches[3]]
		year, _ := strconv.Atoi(matches[5])
		
		// Handle 2-digit years
		if year < 100 {
			if year < 50 {
				year += 2000
			} else {
				year += 1900
			}
		}
		
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	}
	
	// Pattern: Year Month Day (for lowercase dates)
	yearMonthDayPattern := `(\d{4}|\d{2})\s+` + monthPattern + `\s+(\d{1,2})(st|nd|rd|th)?`
	re3 := regexp.MustCompile(yearMonthDayPattern)
	if matches := re3.FindStringSubmatch(dateStr); len(matches) >= 4 {
		year, _ := strconv.Atoi(matches[1])
		month := months[matches[2]]
		day, _ := strconv.Atoi(matches[3])
		
		// Handle 2-digit years
		if year < 100 {
			if year < 50 {
				year += 2000
			} else {
				year += 1900
			}
		}
		
		// Validate the parsed values
		if month >= 1 && month <= 12 && day >= 1 && day <= 31 {
			return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		}
	}
	
	return time.Time{}
}

// ParseDateComponents attempts to extract date components from various formats
func ParseDateComponents(dateStr string) time.Time {
	// Try to extract year, month, day using regular expressions
	
	// Typical date formats with separators
	re := regexp.MustCompile(`(\d{4})[/-](\d{1,2})[/-](\d{1,2})`)
	if matches := re.FindStringSubmatch(dateStr); len(matches) == 4 {
		year, _ := strconv.Atoi(matches[1])
		month, _ := strconv.Atoi(matches[2])
		day, _ := strconv.Atoi(matches[3])
		
		// Verify month and day are valid
		if month < 1 || month > 12 || day < 1 || day > 31 {
			// Try with day/month swapped if current interpretation seems invalid
			if day >= 1 && day <= 12 && month >= 1 && month <= 31 {
				month, day = day, month
			} else {
				return time.Time{}
			}
		}
		
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	}
	
	// Compact date formats (like 20210315)
	re = regexp.MustCompile(`(\d{4})(\d{2})(\d{2})`)
	if matches := re.FindStringSubmatch(dateStr); len(matches) == 4 {
		year, _ := strconv.Atoi(matches[1])
		month, _ := strconv.Atoi(matches[2])
		day, _ := strconv.Atoi(matches[3])
		
		if month >= 1 && month <= 12 && day >= 1 && day <= 31 {
			return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		}
	}
	
	// Just year and month (like 2021-03)
	re = regexp.MustCompile(`(\d{4})[/-](\d{1,2})`)
	if matches := re.FindStringSubmatch(dateStr); len(matches) == 3 {
		year, _ := strconv.Atoi(matches[1])
		month, _ := strconv.Atoi(matches[2])
		
		if month >= 1 && month <= 12 {
			return time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		}
	}
	
	// Just year
	re = regexp.MustCompile(`\b(\d{4})\b`)
	if matches := re.FindStringSubmatch(dateStr); len(matches) == 2 {
		year, _ := strconv.Atoi(matches[1])
		
		// Only use years that seem reasonable for articles
		if year >= 1990 && year <= time.Now().Year() {
			return time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		}
	}
	
	return time.Time{}
}

// ExtractRelativeDate handles relative time references like "2 days ago"
func ExtractRelativeDate(html string) time.Time {
	// Selectors for elements likely to contain relative dates
	selectors := []SelectorScore{
		{Selector: "//span[@class='date']", Score: 3},
		{Selector: "//span[@class='time']", Score: 3},
		{Selector: "//span[@class='timestamp']", Score: 3},
		{Selector: "//time", Score: 2},
		{Selector: "//span[contains(@class, 'date')]", Score: 2},
		{Selector: "//div[contains(@class, 'date')]", Score: 2},
		{Selector: "//p[contains(@class, 'date')]", Score: 2},
		{Selector: "//p[contains(@class, 'time')]", Score: 2},
		{Selector: "//div[contains(@class, 'byline')]", Score: 1},
		{Selector: "//p[contains(@class, 'byline')]", Score: 1},
	}
	
	// Extract elements that might contain relative dates
	relativeDateCandidates := ExtractElement(html, selectors, nil)
	
	// Regular expressions for various relative date formats
	patterns := []struct {
		re       *regexp.Regexp
		timeUnit time.Duration
		scale    func(int) time.Duration
	}{
		{
			re:    regexp.MustCompile(`(\d+)\s*(?:minute|min)s?\s*ago`),
			scale: func(n int) time.Duration { return time.Duration(n) * time.Minute },
		},
		{
			re:    regexp.MustCompile(`(\d+)\s*(?:hour|hr)s?\s*ago`),
			scale: func(n int) time.Duration { return time.Duration(n) * time.Hour },
		},
		{
			re:    regexp.MustCompile(`(\d+)\s*days?\s*ago`),
			scale: func(n int) time.Duration { return time.Duration(n) * 24 * time.Hour },
		},
		{
			re:    regexp.MustCompile(`(\d+)\s*weeks?\s*ago`),
			scale: func(n int) time.Duration { return time.Duration(n) * 7 * 24 * time.Hour },
		},
		{
			re:    regexp.MustCompile(`(\d+)\s*months?\s*ago`),
			scale: func(n int) time.Duration { return time.Duration(n) * 30 * 24 * time.Hour }, // Approximation
		},
		{
			re:    regexp.MustCompile(`(\d+)\s*years?\s*ago`),
			scale: func(n int) time.Duration { return time.Duration(n) * 365 * 24 * time.Hour }, // Approximation
		},
		{
			re:    regexp.MustCompile(`yesterday`),
			scale: func(_ int) time.Duration { return 24 * time.Hour },
		},
		{
			re:    regexp.MustCompile(`last\s*week`),
			scale: func(_ int) time.Duration { return 7 * 24 * time.Hour },
		},
		{
			re:    regexp.MustCompile(`last\s*month`),
			scale: func(_ int) time.Duration { return 30 * 24 * time.Hour }, // Approximation
		},
	}
	
	// Process each candidate
	for dateStr := range relativeDateCandidates {
		dateStr = strings.ToLower(dateStr)
		
		// Try each pattern
		for _, pattern := range patterns {
			matches := pattern.re.FindStringSubmatch(dateStr)
			if matches != nil {
				n := 1 // Default for patterns like "yesterday"
				if len(matches) > 1 {
					n, _ = strconv.Atoi(matches[1])
				}
				
				// Calculate the date based on the relative reference
				return time.Now().UTC().Add(-pattern.scale(n)).Truncate(time.Second)
			}
		}
	}
	
	return time.Time{}
}