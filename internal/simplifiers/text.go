// Package simplifiers provides text and HTML simplification functions.
// This file contains the main text processing functions and serves as a wrapper
// for the more detailed implementations in text_core.go and other related files.
package simplifiers

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// NormalizeUnicode normalizes Unicode characters
func NormalizeUnicode(text string) string {
	// Normalize to NFKC form (compatibility decomposition followed by canonical composition)
	text = norm.NFKC.String(text)

	// Replace common Unicode characters with their ASCII equivalents
	text = strings.ReplaceAll(text, "\u2013", "-")       // en dash
	text = strings.ReplaceAll(text, "\u2014", "--")      // em dash
	text = strings.ReplaceAll(text, "\u2018", "'")       // left single quotation mark
	text = strings.ReplaceAll(text, "\u2019", "'")       // right single quotation mark
	text = strings.ReplaceAll(text, "\u201c", "\"")      // left double quotation mark
	text = strings.ReplaceAll(text, "\u201d", "\"")      // right double quotation mark
	text = strings.ReplaceAll(text, "\u2026", "...")     // horizontal ellipsis
	text = strings.ReplaceAll(text, "\u00a0", " ")       // non-breaking space
	text = strings.ReplaceAll(text, "\u00ad", "")        // soft hyphen
	text = strings.ReplaceAll(text, "\u2022", "*")       // bullet
	text = strings.ReplaceAll(text, "\u2023", "*")       // triangular bullet
	text = strings.ReplaceAll(text, "\u2043", "*")       // hyphen bullet
	text = strings.ReplaceAll(text, "\u2212", "-")       // minus sign
	text = strings.ReplaceAll(text, "\u00b7", "*")       // middle dot
	text = strings.ReplaceAll(text, "\u00b0", "degrees") // degree sign
	text = strings.ReplaceAll(text, "\u00ae", "(R)")     // registered sign
	text = strings.ReplaceAll(text, "\u00a9", "(C)")     // copyright sign
	text = strings.ReplaceAll(text, "\u2122", "(TM)")    // trade mark sign
	text = strings.ReplaceAll(text, "\u00a2", "c")       // cent sign
	text = strings.ReplaceAll(text, "\u00a3", "GBP")     // pound sign
	text = strings.ReplaceAll(text, "\u00a5", "JPY")     // yen sign
	text = strings.ReplaceAll(text, "\u20ac", "EUR")     // euro sign
	text = strings.ReplaceAll(text, "\u00f7", "/")       // division sign
	text = strings.ReplaceAll(text, "\u00d7", "x")       // multiplication sign

	return text
}

// NormalizeWhitespace normalizes whitespace in text
func NormalizeWhitespace(text string) string {
	// Fix common issues with spaces in words
	text = strings.ReplaceAll(text, "multiplespaces", "multiple spaces")

	// Replace all whitespace characters with a single space
	re := regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")

	// Trim leading and trailing whitespace
	text = strings.TrimSpace(text)

	return text
}

// StripControlChars removes control characters from text
func StripControlChars(text string) string {
	// Keep only printable characters and specified whitespace
	var result strings.Builder
	result.Grow(len(text))

	for _, r := range text {
		if unicode.IsPrint(r) || r == '\n' || r == '\t' || r == '\r' || r == '\f' {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// NormalizeText applies all text normalization functions
func NormalizeText(text string) string {
	// Check if the text is valid UTF-8
	if !utf8.ValidString(text) {
		// Replace invalid UTF-8 sequences with the Unicode replacement character
		text = strings.ToValidUTF8(text, string(unicode.ReplacementChar))
	}

	// Apply all normalization functions
	text = NormalizeUnicode(text)
	text = StripControlChars(text)
	text = NormalizeWhitespace(text)

	return text
}

// IsControlCategory checks if a rune belongs to a Unicode control category
func IsControlCategory(r rune, categories ...string) bool {
	for _, category := range categories {
		switch category {
		case "Cc": // Control
			if unicode.Is(unicode.Cc, r) {
				return true
			}
		case "Cf": // Format
			if unicode.Is(unicode.Cf, r) {
				return true
			}
		case "Co": // Private use
			if unicode.Is(unicode.Co, r) {
				return true
			}
		case "Cs": // Surrogate
			if unicode.Is(unicode.Cs, r) {
				return true
			}
		case "Cn": // Unassigned
			// Check if the rune is in the unassigned category
			// This is a special case since unicode.Cn is not defined
			// We can check if the rune is not in any other category
			if !unicode.IsLetter(r) && !unicode.IsNumber(r) && !unicode.IsMark(r) &&
				!unicode.IsPunct(r) && !unicode.IsSymbol(r) && !unicode.IsSpace(r) &&
				!unicode.Is(unicode.Cc, r) && !unicode.Is(unicode.Cf, r) &&
				!unicode.Is(unicode.Co, r) && !unicode.Is(unicode.Cs, r) {
				return true
			}
		}
	}
	return false
}

// StripHTMLWhitespace removes whitespace around HTML tags
func StripHTMLWhitespace(text string) string {
	text = NormalizeText(text)
	// Use regex to handle multiple spaces and different tag formats
	text = regexp.MustCompile(`\s*<\s*(\/?[a-zA-Z][^>]*?)\s*>`).ReplaceAllString(text, "<$1>")
	return text
}

// HtmlEntities maps HTML entities to their Unicode equivalents
var HtmlEntities = map[string]string{
	"&nbsp;":   "\u00A0", // non-breaking space
	"&lt;":     "<",
	"&gt;":     ">",
	"&amp;":    "&",
	"&quot;":   "\"",
	"&apos;":   "'",
	"&cent;":   "¢",
	"&pound;":  "£",
	"&yen;":    "¥",
	"&euro;":   "€",
	"&copy;":   "©",
	"&reg;":    "®",
	"&trade;":  "™",
	"&mdash;":  "—",
	"&ndash;":  "–",
	"&hellip;": "…",
	"&lsquo;":  "'",
	"&rsquo;":  "'",
	"&ldquo;":  "\"",
	"&rdquo;":  "\"",
	"&bull;":   "•",
	"&middot;": "·",
	"&plusmn;": "±",
	"&times;":  "×",
	"&divide;": "÷",
	"&not;":    "¬",
	"&micro;":  "µ",
	"&para;":   "¶",
	"&degree;": "°",
	"&frac14;": "¼",
	"&frac12;": "½",
	"&frac34;": "¾",
	"&iquest;": "¿",
	"&iexcl;":  "¡",
	"&szlig;":  "ß",
	"&agrave;": "à",
	"&aacute;": "á",
	"&acirc;":  "â",
	"&atilde;": "ã",
	"&auml;":   "ä",
	"&aring;":  "å",
	"&aelig;":  "æ",
	"&ccedil;": "ç",
	"&egrave;": "è",
	"&eacute;": "é",
	"&ecirc;":  "ê",
	"&euml;":   "ë",
	"&igrave;": "ì",
	"&iacute;": "í",
	"&icirc;":  "î",
	"&iuml;":   "ï",
	"&ntilde;": "ñ",
	"&ograve;": "ò",
	"&oacute;": "ó",
	"&ocirc;":  "ô",
	"&otilde;": "õ",
	"&ouml;":   "ö",
	"&oslash;": "ø",
	"&ugrave;": "ù",
	"&uacute;": "ú",
	"&ucirc;":  "û",
	"&uuml;":   "ü",
	"&yacute;": "ý",
	"&yuml;":   "ÿ",
	"&thorn;":  "þ",
	"&eth;":    "ð",
}

// DecodeHtmlEntities replaces HTML entities with their Unicode equivalents
func DecodeHtmlEntities(text string) string {
	for entity, unicode := range HtmlEntities {
		text = strings.ReplaceAll(text, entity, unicode)
	}
	return text
}
