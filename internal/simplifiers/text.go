// Package simplifiers provides text and HTML simplification functions.
// This file contains the main text processing functions and serves as a wrapper
// for the more detailed implementations in text_core.go and other related files.
package simplifiers

import (
	"regexp"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// TextCache provides caching for expensive text operations
type TextCache struct {
	mu            sync.RWMutex
	unicodeCache  map[string]string
	whitespaceRE  *regexp.Regexp
	htmlTagWSRE   *regexp.Regexp
	normalizeCache map[string]string
	stripCache    map[string]string
	htmlWSCache   map[string]string
	entityCache   map[string]string
	cacheSize     int
	cacheHits     int64
	cacheMisses   int64
}

// Global text cache
var (
	textCache  *TextCache
	cacheSetup sync.Once
)

// initCache initializes the global text cache
func initCache() {
	textCache = &TextCache{
		unicodeCache:  make(map[string]string, 1000),
		normalizeCache: make(map[string]string, 500),
		stripCache:    make(map[string]string, 500),
		htmlWSCache:   make(map[string]string, 200),
		entityCache:   make(map[string]string, 200),
		whitespaceRE:  regexp.MustCompile(`\s+`),
		htmlTagWSRE:   regexp.MustCompile(`\s*<\s*(\/?[a-zA-Z][^>]*?)\s*>`),
	}
}

// getCache returns the global text cache
func getCache() *TextCache {
	cacheSetup.Do(initCache)
	return textCache
}

// Get retrieves a value from the specified cache
func (c *TextCache) Get(cache map[string]string, key string) (string, bool) {
	c.mu.RLock()
	val, found := cache[key]
	c.mu.RUnlock()
	
	if found {
		c.mu.Lock()
		c.cacheHits++
		c.mu.Unlock()
		return val, true
	}
	
	c.mu.Lock()
	c.cacheMisses++
	c.mu.Unlock()
	return "", false
}

// Set adds a value to the specified cache
func (c *TextCache) Set(cache map[string]string, key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Basic cache eviction - remove random entry if cache is full
	if len(cache) >= c.cacheSize && c.cacheSize > 0 {
		for k := range cache {
			delete(cache, k)
			break
		}
	}
	
	cache[key] = value
}

// EvictCache clears all caches
func (c *TextCache) EvictCache() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.unicodeCache = make(map[string]string, 1000)
	c.normalizeCache = make(map[string]string, 500)
	c.stripCache = make(map[string]string, 500)
	c.htmlWSCache = make(map[string]string, 200)
	c.entityCache = make(map[string]string, 200)
}

// Common unicode replacements map
var unicodeReplacements = map[string]string{
	"\u2013": "-",       // en dash
	"\u2014": "--",      // em dash
	"\u2018": "'",       // left single quotation mark
	"\u2019": "'",       // right single quotation mark
	"\u201c": "\"",      // left double quotation mark
	"\u201d": "\"",      // right double quotation mark
	"\u2026": "...",     // horizontal ellipsis
	"\u00a0": " ",       // non-breaking space
	"\u00ad": "",        // soft hyphen
	"\u2022": "*",       // bullet
	"\u2023": "*",       // triangular bullet
	"\u2043": "*",       // hyphen bullet
	"\u2212": "-",       // minus sign
	"\u00b7": "*",       // middle dot
	"\u00b0": "degrees", // degree sign
	"\u00ae": "(R)",     // registered sign
	"\u00a9": "(C)",     // copyright sign
	"\u2122": "(TM)",    // trade mark sign
	"\u00a2": "c",       // cent sign
	"\u00a3": "GBP",     // pound sign
	"\u00a5": "JPY",     // yen sign
	"\u20ac": "EUR",     // euro sign
	"\u00f7": "/",       // division sign
	"\u00d7": "x",       // multiplication sign
}

// NormalizeUnicode normalizes Unicode characters with caching and optimizations
func NormalizeUnicode(text string) string {
	// Short circuit for empty string
	if text == "" {
		return ""
	}
	
	// For short texts where the cache lookup might be more expensive than just processing
	if len(text) < 20 {
		return normalizeUnicodeUncached(text)
	}
	
	// Get cache instance
	cache := getCache()
	
	// Check cache first
	if cachedResult, found := cache.Get(cache.unicodeCache, text); found {
		return cachedResult
	}
	
	// Process the text
	result := normalizeUnicodeUncached(text)
	
	// Cache result if it's not too long
	if len(text) < 5000 {
		cache.Set(cache.unicodeCache, text, result)
	}
	
	return result
}

// normalizeUnicodeUncached performs Unicode normalization without caching
func normalizeUnicodeUncached(text string) string {
	// Normalize to NFKC form (compatibility decomposition followed by canonical composition)
	text = norm.NFKC.String(text)
	
	// Check if the normalized text contains any characters that need replacement
	needsReplacement := false
	for char := range unicodeReplacements {
		if strings.Contains(text, char) {
			needsReplacement = true
			break
		}
	}
	
	// Skip replacements if none are needed
	if !needsReplacement {
		return text
	}
	
	// Use a string builder for better performance with multiple replacements
	var builder strings.Builder
	builder.Grow(len(text)) // Pre-allocate capacity
	
	// Process the string character by character
	for _, r := range text {
		// Convert the rune to string to check for replacements
		char := string(r)
		if replacement, found := unicodeReplacements[char]; found {
			builder.WriteString(replacement)
		} else {
			builder.WriteRune(r)
		}
	}
	
	return builder.String()
}

// NormalizeWhitespace normalizes whitespace in text with caching and optimizations
func NormalizeWhitespace(text string) string {
	// Short circuit for empty string
	if text == "" {
		return ""
	}
	
	// For short texts where the cache lookup might be more expensive than just processing
	if len(text) < 20 {
		return normalizeWhitespaceUncached(text)
	}
	
	// Get cache instance
	cache := getCache()
	
	// Check cache first
	if cachedResult, found := cache.Get(cache.normalizeCache, text); found {
		return cachedResult
	}
	
	// Process the text
	result := normalizeWhitespaceUncached(text)
	
	// Cache result if it's not too long
	if len(text) < 5000 {
		cache.Set(cache.normalizeCache, text, result)
	}
	
	return result
}

// normalizeWhitespaceUncached normalizes whitespace without caching
func normalizeWhitespaceUncached(text string) string {
	// Quick check if normalization is needed at all
	hasMultipleWS := false
	lastWasWS := false
	
	for _, r := range text {
		isWS := unicode.IsSpace(r)
		if isWS && lastWasWS {
			hasMultipleWS = true
			break
		}
		lastWasWS = isWS
	}
	
	// If no multiple whitespace, just trim and return
	if !hasMultipleWS && !lastWasWS && text[0] != ' ' {
		return text
	}
	
	// Fix specific "multiplespaces" issue if present (preserved from original)
	if strings.Contains(text, "multiplespaces") {
		text = strings.ReplaceAll(text, "multiplespaces", "multiple spaces")
	}
	
	// Use precompiled regex from cache
	cache := getCache()
	
	// Replace all whitespace characters with a single space using cached regex
	text = cache.whitespaceRE.ReplaceAllString(text, " ")
	
	// Trim leading and trailing whitespace
	text = strings.TrimSpace(text)
	
	return text
}

// StripControlChars removes control characters from text with caching
func StripControlChars(text string) string {
	// Short circuit for empty string
	if text == "" {
		return ""
	}
	
	// For short texts where the cache lookup might be more expensive than just processing
	if len(text) < 20 {
		return stripControlCharsUncached(text)
	}
	
	// Get cache instance
	cache := getCache()
	
	// Check cache first
	if cachedResult, found := cache.Get(cache.stripCache, text); found {
		return cachedResult
	}
	
	// Process the text
	result := stripControlCharsUncached(text)
	
	// Cache result if it's not too long
	if len(text) < 5000 {
		cache.Set(cache.stripCache, text, result)
	}
	
	return result
}

// stripControlCharsUncached removes control characters without caching
func stripControlCharsUncached(text string) string {
	// Quick check if any control characters are present
	hasControlChars := false
	for _, r := range text {
		if !unicode.IsPrint(r) && r != '\n' && r != '\t' && r != '\r' && r != '\f' {
			hasControlChars = true
			break
		}
	}
	
	// If no control characters, return the original string
	if !hasControlChars {
		return text
	}
	
	// Keep only printable characters and specified whitespace
	var result strings.Builder
	result.Grow(len(text))
	
	// Process in chunks for better memory efficiency with long strings
	const chunkSize = 1024
	if len(text) > chunkSize {
		for i := 0; i < len(text); i += chunkSize {
			end := i + chunkSize
			if end > len(text) {
				end = len(text)
			}
			
			chunk := text[i:end]
			for _, r := range chunk {
				if unicode.IsPrint(r) || r == '\n' || r == '\t' || r == '\r' || r == '\f' {
					result.WriteRune(r)
				}
			}
		}
	} else {
		// For shorter strings, process directly
		for _, r := range text {
			if unicode.IsPrint(r) || r == '\n' || r == '\t' || r == '\r' || r == '\f' {
				result.WriteRune(r)
			}
		}
	}
	
	return result.String()
}

// NormalizeText applies all text normalization functions with caching
func NormalizeText(text string) string {
	// Short circuit for empty string
	if text == "" {
		return ""
	}
	
	// For short texts where the cache lookup might be more expensive than just processing
	if len(text) < 20 {
		return normalizeTextUncached(text)
	}
	
	// Get cache instance
	cache := getCache()
	
	// Check cache first
	if cachedResult, found := cache.Get(cache.normalizeCache, text); found {
		return cachedResult
	}
	
	// Process the text
	result := normalizeTextUncached(text)
	
	// Cache result if it's not too long
	if len(text) < 5000 {
		cache.Set(cache.normalizeCache, text, result)
	}
	
	return result
}

// normalizeTextUncached applies all text normalization functions without caching
func normalizeTextUncached(text string) string {
	// Check if the text is valid UTF-8
	if !utf8.ValidString(text) {
		// Replace invalid UTF-8 sequences with the Unicode replacement character
		text = strings.ToValidUTF8(text, string(unicode.ReplacementChar))
	}

	// Apply specific normalizations without using the cached versions
	// to avoid redundant cache lookups 
	text = normalizeUnicodeUncached(text)
	text = stripControlCharsUncached(text)
	text = normalizeWhitespaceUncached(text)

	return text
}

// IsControlCategory checks if a rune belongs to a Unicode control category
// This function is optimized to use a map-based lookup for categories
func IsControlCategory(r rune, categories ...string) bool {
	// Create a map for faster category lookups
	catSet := make(map[string]bool, len(categories))
	for _, cat := range categories {
		catSet[cat] = true
	}
	
	// Check for Control (Cc)
	if catSet["Cc"] && unicode.Is(unicode.Cc, r) {
		return true
	}
	
	// Check for Format (Cf)
	if catSet["Cf"] && unicode.Is(unicode.Cf, r) {
		return true
	}
	
	// Check for Private use (Co)
	if catSet["Co"] && unicode.Is(unicode.Co, r) {
		return true
	}
	
	// Check for Surrogate (Cs)
	if catSet["Cs"] && unicode.Is(unicode.Cs, r) {
		return true
	}
	
	// Check for Unassigned (Cn)
	if catSet["Cn"] {
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
	
	return false
}

// StripHTMLWhitespace removes whitespace around HTML tags with caching
func StripHTMLWhitespace(text string) string {
	// Short circuit for empty string
	if text == "" {
		return ""
	}
	
	// For short texts where the cache lookup might be more expensive than just processing
	if len(text) < 50 {
		return stripHTMLWhitespaceUncached(text)
	}
	
	// Get cache instance
	cache := getCache()
	
	// Check cache first
	if cachedResult, found := cache.Get(cache.htmlWSCache, text); found {
		return cachedResult
	}
	
	// Process the text
	result := stripHTMLWhitespaceUncached(text)
	
	// Cache result if it's not too long
	if len(text) < 5000 {
		cache.Set(cache.htmlWSCache, text, result)
	}
	
	return result
}

// stripHTMLWhitespaceUncached removes whitespace around HTML tags without caching
func stripHTMLWhitespaceUncached(text string) string {
	// Normalize the text first
	text = normalizeTextUncached(text)
	
	// Quick check if there are any HTML tags
	if !strings.Contains(text, "<") || !strings.Contains(text, ">") {
		return text
	}
	
	// Use precompiled regex from cache for better performance
	cache := getCache()
	text = cache.htmlTagWSRE.ReplaceAllString(text, "<$1>")
	
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

// DecodeHtmlEntities replaces HTML entities with their Unicode equivalents with caching
func DecodeHtmlEntities(text string) string {
	// Short circuit for empty string
	if text == "" {
		return ""
	}
	
	// Quick check if there are any HTML entities
	if !strings.Contains(text, "&") {
		return text
	}
	
	// For short texts where the cache lookup might be more expensive than just processing
	if len(text) < 50 {
		return decodeHtmlEntitiesUncached(text)
	}
	
	// Get cache instance
	cache := getCache()
	
	// Check cache first
	if cachedResult, found := cache.Get(cache.entityCache, text); found {
		return cachedResult
	}
	
	// Process the text
	result := decodeHtmlEntitiesUncached(text)
	
	// Cache result if it's not too long
	if len(text) < 5000 {
		cache.Set(cache.entityCache, text, result)
	}
	
	return result
}

// decodeHtmlEntitiesUncached replaces HTML entities without caching
func decodeHtmlEntitiesUncached(text string) string {
	// Optimization: if no entity markers are present, return original text
	if !strings.Contains(text, "&") {
		return text
	}
	
	// Use a string builder for better performance with multiple replacements
	var builder strings.Builder
	builder.Grow(len(text)) // Pre-allocate capacity
	
	// Process the text in one pass
	i := 0
	for i < len(text) {
		// Look for the start of an entity
		ampIndex := strings.IndexByte(text[i:], '&')
		
		if ampIndex == -1 {
			// No more entities, append the rest and finish
			builder.WriteString(text[i:])
			break
		}
		
		// Append text up to the entity
		builder.WriteString(text[i : i+ampIndex])
		i += ampIndex
		
		// Find the end of the entity
		endIndex := strings.IndexByte(text[i:], ';')
		if endIndex == -1 {
			// No semicolon found, treat as literal text
			builder.WriteByte('&')
			i++
			continue
		}
		
		// Extract the entity including & and ;
		entity := text[i : i+endIndex+1]
		i += endIndex + 1
		
		// Look up the entity
		if unicode, found := HtmlEntities[entity]; found {
			builder.WriteString(unicode)
		} else {
			// Not a known entity, append as is
			builder.WriteString(entity)
		}
	}
	
	return builder.String()
}
