package simplifiers

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var (
	whitespaceRegex = regexp.MustCompile(`\s+`)
	retainedChars   = map[rune]bool{
		'\t': true,
		'\n': true,
		'\r': true,
		'\f': true,
	}
)

// NormalizeUnicode normalizes text to NFKC form for consistent character representation
func NormalizeUnicode(text string) string {
	return norm.NFKC.String(text)
}

// NormalizeWhitespace replaces runs of whitespace with a single space and trims
func NormalizeWhitespace(text string) string {
	return whitespaceRegex.ReplaceAllString(strings.TrimSpace(text), " ")
}

// StripControlChars removes Unicode control characters while retaining specific whitespace chars
func StripControlChars(text string) string {
	var b strings.Builder
	b.Grow(len(text))

	for _, r := range text {
		if !unicode.IsControl(r) || retainedChars[r] {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// NormalizeText performs all text normalization steps in the correct order
func NormalizeText(text string) string {
	text = StripControlChars(text)
	text = NormalizeUnicode(text)
	text = NormalizeWhitespace(text)
	return text
}

// StripHTMLWhitespace removes whitespace around HTML tags
func StripHTMLWhitespace(text string) string {
	text = NormalizeText(text)
	// Use regex to handle multiple spaces and different tag formats
	text = regexp.MustCompile(`\s*<\s*(\/?[a-zA-Z][^>]*?)\s*>`).ReplaceAllString(text, "<$1>")
	return text
}

// IsControlCategory checks if a unicode category represents a control character
func IsControlCategory(cat string) bool {
	switch cat {
	case "Cc", "Cf", "Cn", "Co", "Cs":
		return true
	default:
		return false
	}
}
