package extractors

import (
	"testing"
	"time"
)

func TestParseISO8601Format(t *testing.T) {
	tests := []struct {
		name     string
		dateStr  string
		expected time.Time
	}{
		{
			name:     "RFC3339",
			dateStr:  "2023-03-27T15:04:05Z",
			expected: time.Date(2023, 3, 27, 15, 4, 5, 0, time.UTC),
		},
		{
			name:     "ISO8601 without timezone",
			dateStr:  "2023-03-27T15:04:05",
			expected: time.Date(2023, 3, 27, 15, 4, 5, 0, time.UTC),
		},
		{
			name:     "ISO8601 with milliseconds",
			dateStr:  "2023-03-27T15:04:05.123Z",
			expected: time.Date(2023, 3, 27, 15, 4, 5, 0, time.UTC),
		},
		{
			name:     "ISO8601 with timezone offset",
			dateStr:  "2023-03-27T15:04:05+02:00",
			expected: time.Date(2023, 3, 27, 13, 4, 5, 0, time.UTC),
		},
		{
			name:     "ISO8601 date only",
			dateStr:  "2023-03-27",
			expected: time.Date(2023, 3, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "RFC1123",
			dateStr:  "Mon, 27 Mar 2023 15:04:05 GMT",
			expected: time.Date(2023, 3, 27, 15, 4, 5, 0, time.UTC),
		},
		{
			name:     "Invalid format",
			dateStr:  "not a date",
			expected: time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseISO8601Format(tt.dateStr)
			if !result.Equal(tt.expected) {
				t.Errorf("ParseISO8601Format(%q) = %v, want %v", tt.dateStr, result, tt.expected)
			}
		})
	}
}

func TestParseRegionalDateFormats(t *testing.T) {
	tests := []struct {
		name     string
		dateStr  string
		expected time.Time
	}{
		{
			name:     "US Format MM/DD/YYYY",
			dateStr:  "03/27/2023",
			expected: time.Date(2023, 3, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "US Format MM-DD-YYYY",
			dateStr:  "03-27-2023",
			expected: time.Date(2023, 3, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "EU Format DD/MM/YYYY",
			dateStr:  "27/03/2023",
			expected: time.Date(2023, 3, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "EU Format DD-MM-YYYY",
			dateStr:  "27-03-2023",
			expected: time.Date(2023, 3, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "US Format with time",
			dateStr:  "03/27/2023 15:04:05",
			expected: time.Date(2023, 3, 27, 15, 4, 5, 0, time.UTC),
		},
		{
			name:     "2-digit year",
			dateStr:  "03/27/23",
			expected: time.Date(2023, 3, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Year first format",
			dateStr:  "2023/03/27",
			expected: time.Date(2023, 3, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Year first with dashes",
			dateStr:  "2023-03-27",
			expected: time.Date(2023, 3, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Ambiguous date with day > 12",
			dateStr:  "13/02/2023", // Must be day/month
			expected: time.Date(2023, 2, 13, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseRegionalDateFormats(tt.dateStr)
			if !result.Equal(tt.expected) {
				t.Errorf("ParseRegionalDateFormats(%q) = %v, want %v", tt.dateStr, result, tt.expected)
			}
		})
	}
}

func TestParseNaturalLanguageDates(t *testing.T) {
	tests := []struct {
		name     string
		dateStr  string
		expected time.Time
	}{
		{
			name:     "Month Day, Year",
			dateStr:  "March 27, 2023",
			expected: time.Date(2023, 3, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Month Day Year (no comma)",
			dateStr:  "March 27 2023",
			expected: time.Date(2023, 3, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Abbreviated month",
			dateStr:  "Mar 27, 2023",
			expected: time.Date(2023, 3, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Day Month Year",
			dateStr:  "27 March 2023",
			expected: time.Date(2023, 3, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Day Month Year with abbreviated month",
			dateStr:  "27 Mar 2023",
			expected: time.Date(2023, 3, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Day with ordinal",
			dateStr:  "27th March 2023",
			expected: time.Date(2023, 3, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Year Month Day",
			dateStr:  "2023 March 27",
			expected: time.Date(2023, 3, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "2-digit year",
			dateStr:  "March 27, 23",
			expected: time.Date(2023, 3, 27, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseNaturalLanguageDates(tt.dateStr)
			if !result.Equal(tt.expected) {
				t.Errorf("ParseNaturalLanguageDates(%q) = %v, want %v", tt.dateStr, result, tt.expected)
			}
		})
	}
}

func TestParseDateComponents(t *testing.T) {
	tests := []struct {
		name     string
		dateStr  string
		expected time.Time
	}{
		{
			name:     "YYYY-MM-DD",
			dateStr:  "2023-03-27",
			expected: time.Date(2023, 3, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "YYYY/MM/DD",
			dateStr:  "2023/03/27",
			expected: time.Date(2023, 3, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Compact YYYYMMDD",
			dateStr:  "20230327",
			expected: time.Date(2023, 3, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Year and month only",
			dateStr:  "2023-03",
			expected: time.Date(2023, 3, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Year only",
			dateStr:  "2023",
			expected: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Year only in text",
			dateStr:  "Published in 2023",
			expected: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseDateComponents(tt.dateStr)
			if !result.Equal(tt.expected) {
				t.Errorf("ParseDateComponents(%q) = %v, want %v", tt.dateStr, result, tt.expected)
			}
		})
	}
}

func TestCleanupDateString(t *testing.T) {
	tests := []struct {
		name     string
		dateStr  string
		expected string
	}{
		{
			name:     "Remove HTML tags",
			dateStr:  "<span class=\"date\">March 27, 2023</span>",
			expected: "March 27, 2023",
		},
		{
			name:     "Remove extra spaces",
			dateStr:  "  March  27,   2023  ",
			expected: "March 27, 2023",
		},
		{
			name:     "Remove common prefixes",
			dateStr:  "Published: March 27, 2023",
			expected: "March 27, 2023",
		},
		{
			name:     "Remove multiple prefixes",
			dateStr:  "Published on date: March 27, 2023",
			expected: "March 27, 2023",
		},
		{
			name:     "Handle different wordings",
			dateStr:  "Posted on March 27, 2023",
			expected: "March 27, 2023",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanupDateString(tt.dateStr)
			if result != tt.expected {
				t.Errorf("CleanupDateString(%q) = %q, want %q", tt.dateStr, result, tt.expected)
			}
		})
	}
}

// This test verifies that our date extraction works with actual XML/HTML content
func TestManualExtractDate(t *testing.T) {
	html := `<html>
<head>
  <meta property="article:published_time" content="2023-03-27T15:04:05Z">
</head>
<body>
  <span class="date">March 27, 2023</span>
</body>
</html>`

	expected := time.Date(2023, 3, 27, 15, 4, 5, 0, time.UTC)
	result := ExtractDate(html)

	if !result.Equal(expected) {
		t.Errorf("ExtractDate() = %v, want %v", result, expected)
	}
}