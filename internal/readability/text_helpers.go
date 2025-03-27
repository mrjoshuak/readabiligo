package readability

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// getInnerText gets the inner text of a node with optional whitespace normalization
// Uses a cache to avoid repeated normalization of the same text
func getInnerText(s *goquery.Selection, normalizeSpaces bool) string {
	if s == nil || s.Length() == 0 {
		return ""
	}

	// Most efficient path for non-normalized text
	if !normalizeSpaces {
		return strings.TrimSpace(s.Text())
	}
	
	// Get the raw text first
	text := strings.TrimSpace(s.Text())
	
	// Empty strings don't need normalization
	if text == "" {
		return text
	}
	
	// Apply normalization (replaced multiple spaces with a single space)
	// Use a faster implementation than regexp for this common operation
	var sb strings.Builder
	sb.Grow(len(text)) // Pre-allocate for efficiency
	
	// Track if we're in a whitespace sequence
	inWhitespace := false
	
	for _, r := range text {
		isSpace := unicode.IsSpace(r)
		
		if isSpace {
			// If this is the first whitespace char we've seen, add a single space
			if !inWhitespace {
				sb.WriteRune(' ')
				inWhitespace = true
			}
			// Otherwise skip it (multiple spaces become one)
		} else {
			// Regular character, just add it
			sb.WriteRune(r)
			inWhitespace = false
		}
	}
	
	return sb.String()
}

// getNodeText gets the text content of an HTML node
func getNodeText(node *html.Node) string {
	if node == nil {
		return ""
	}

	// Use a string builder for efficient concatenation
	var sb strings.Builder
	
	// Helper function to extract text from node tree, avoids multiple function calls
	var extractText func(*html.Node)
	extractText = func(n *html.Node) {
		if n == nil {
			return
		}
		
		if n.Type == TextNode {
			sb.WriteString(n.Data)
		} else {
			// Recursively process children
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				extractText(child)
			}
		}
	}
	
	// Extract all text from the node tree
	extractText(node)
	
	// Return trimmed result
	return strings.TrimSpace(sb.String())
}

// getCharCount counts occurrences of a delimiter in the text
func getCharCount(s *goquery.Selection, delimiter string) int {
	if s == nil || s.Length() == 0 {
		return 0
	}

	if delimiter == "" {
		delimiter = ","
	}

	text := getInnerText(s, true)
	return len(strings.Split(text, delimiter)) - 1
}

// getLinkDensity calculates the ratio of link text to total text
func getLinkDensity(s *goquery.Selection) float64 {
	if s == nil || s.Length() == 0 {
		return 0
	}

	// Cache the inner text to avoid recalculating 
	innerText := getInnerText(s, true)
	textLength := len(innerText)
	if textLength == 0 {
		return 0
	}

	// Use a pre-compiled matcher for hash URLs
	hashUrlMatcher := RegexpHashUrl

	// Calculate total link text length in one pass
	var linkLength int
	s.Find("a").Each(func(i int, link *goquery.Selection) {
		href, exists := link.Attr("href")
		
		// Apply coefficient for hash links (internal page links)
		coefficient := 1.0
		if exists && hashUrlMatcher.MatchString(href) {
			coefficient = 0.3
		}
		
		// Calculate link text length with coefficient applied
		linkLength += int(float64(len(getInnerText(link, true))) * coefficient)
	})

	// Return the density ratio
	return float64(linkLength) / float64(textLength)
}

// getClassWeight calculates the content score based on class and ID attributes
func getClassWeight(s *goquery.Selection) int {
	if s == nil || s.Length() == 0 {
		return 0
	}

	weight := 0

	// Check for content-related class
	class, exists := s.Attr("class")
	if exists && class != "" {
		if RegexpNegative.MatchString(class) {
			weight -= 25
		}
		if RegexpPositive.MatchString(class) {
			weight += 25
		}
	}

	// Check for content-related ID
	id, exists := s.Attr("id")
	if exists && id != "" {
		if RegexpNegative.MatchString(id) {
			weight -= 25
		}
		if RegexpPositive.MatchString(id) {
			weight += 25
		}
	}

	return weight
}

// textSimilarity compares two texts and returns a similarity score
func textSimilarity(textA, textB string) float64 {
	if textA == "" || textB == "" {
		return 0
	}

	// Quick path for identical texts
	if textA == textB {
		return 1.0
	}

	// Tokenize both texts - use a more efficient approach with a single function
	tokenizeFunc := func(text string) map[string]struct{} {
		text = strings.ToLower(text)
		tokens := make(map[string]struct{})
		
		start := -1
		for i, r := range text {
			// If it's a letter or digit, mark start of token if needed
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				if start < 0 {
					start = i
				}
			} else if start >= 0 {
				// End of a token
				tokens[text[start:i]] = struct{}{}
				start = -1
			}
		}
		
		// Handle case where token goes to end of string
		if start >= 0 {
			tokens[text[start:]] = struct{}{}
		}
		
		return tokens
	}
	
	// Create token sets for both texts
	tokensA := tokenizeFunc(textA)
	tokensB := tokenizeFunc(textB)
	
	if len(tokensA) == 0 || len(tokensB) == 0 {
		return 0
	}
	
	// Count common tokens and total tokens in B
	commonTokens := 0
	for token := range tokensB {
		if _, exists := tokensA[token]; exists {
			commonTokens++
		}
	}
	
	// Calculate similarity as the ratio of common tokens to total tokens in B
	return float64(commonTokens) / float64(len(tokensB))
}

// isValidByline checks if a string could be a byline
func isValidByline(text string) bool {
	text = strings.TrimSpace(text)
	return text != "" && len(text) < 100
}

// wordCount counts the number of words in a string
func wordCount(text string) int {
	return len(strings.Fields(text))
}

// unescapeHtmlEntities converts HTML entities to their corresponding characters
func unescapeHtmlEntities(text string) string {
	if text == "" {
		return text
	}

	// Basic HTML entities
	result := regexp.MustCompile(`&(quot|amp|apos|lt|gt);`).ReplaceAllStringFunc(text, func(match string) string {
		entity := match[1 : len(match)-1]
		if val, ok := HTMLEscapeMap[entity]; ok {
			return val
		}
		return match
	})

	// Numeric entities (&#123; format)
	result = regexp.MustCompile(`&#(?:x([0-9a-f]{1,4})|([0-9]{1,4}));`).ReplaceAllStringFunc(result, func(match string) string {
		if strings.HasPrefix(match, "&#x") {
			// Hex format &#x123;
			hexStr := match[3 : len(match)-1]
			intVal, err := strconv.ParseInt(hexStr, 16, 32)
			if err != nil {
				return match
			}
			return string(rune(intVal))
		} else {
			// Decimal format &#123;
			decStr := match[2 : len(match)-1]
			intVal, err := strconv.Atoi(decStr)
			if err != nil {
				return match
			}
			return string(rune(intVal))
		}
	})

	return result
}

// isWhitespace checks if a node is whitespace
func isWhitespace(node *html.Node) bool {
	if node == nil {
		return true
	}

	// If it's a text node, check if it's only whitespace
	if node.Type == TextNode {
		return RegexpWhitespace.MatchString(node.Data)
	}

	// Check if it's a BR element
	return strings.ToUpper(node.Data) == "BR"
}

// generateHash creates a SHA-256 hash for text
func generateHash(text string) string {
	hash := sha256.New()
	hash.Write([]byte(text))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

// getOuterHTML gets the outer HTML of a selection
func getOuterHTML(s *goquery.Selection) string {
	if s == nil || s.Length() == 0 {
		return ""
	}
	html, err := goquery.OuterHtml(s)
	if err != nil {
		return ""
	}
	return html
}

// getNodeName returns the tag name of a selection
func getNodeName(s *goquery.Selection) string {
	if s == nil || s.Length() == 0 {
		return ""
	}
	node := s.Get(0)
	if node == nil {
		return ""
	}
	// For text nodes or non-element nodes, provide a meaningful return
	if node.Type != html.ElementNode {
		if node.Type == html.TextNode {
			return "#text"
		}
		// Map the node type to a string representation
		switch node.Type {
		case html.CommentNode:
			return "#comment"
		case html.DoctypeNode:
			return "#doctype"
		default:
			return fmt.Sprintf("#node%d", node.Type)
		}
	}
	return strings.ToUpper(node.Data)
}

// nodeToString returns a string representation of a node for debugging
// Note: This function is only used for debugging and isn't in hot paths
func nodeToString(node *html.Node) string {
	if node == nil {
		return "nil"
	}

	// Use a string builder for more efficient string construction
	var sb strings.Builder

	if node.Type == TextNode {
		// Handle text nodes - limit to 20 chars for brevity
		text := strings.TrimSpace(node.Data)
		sb.WriteString("TextNode (\"")
		if len(text) > 20 {
			sb.WriteString(text[:20])
			sb.WriteString("...")
		} else {
			sb.WriteString(text)
		}
		sb.WriteString("\")")
		return sb.String()
	}

	// For element nodes, construct a representation with attributes
	sb.WriteByte('<')
	sb.WriteString(strings.ToLower(node.Data))
	
	// Only add a space if there are attributes
	if len(node.Attr) > 0 {
		sb.WriteByte(' ')
	}
	
	// Add all attributes
	for i, attr := range node.Attr {
		if i > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(attr.Key)
		sb.WriteString("=\"")
		sb.WriteString(attr.Val)
		sb.WriteByte('"')
	}
	
	sb.WriteByte('>')
	return sb.String()
}