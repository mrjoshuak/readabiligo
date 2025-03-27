package readability

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// getInnerText gets the inner text of a node with optional whitespace normalization
func getInnerText(s *goquery.Selection, normalizeSpaces bool) string {
	if s == nil || s.Length() == 0 {
		return ""
	}

	text := strings.TrimSpace(s.Text())
	if normalizeSpaces {
		text = RegexpNormalize.ReplaceAllString(text, " ")
	}
	return text
}

// getNodeText gets the text content of an HTML node
func getNodeText(node *html.Node) string {
	if node == nil {
		return ""
	}

	var text string
	if node.Type == TextNode {
		text = node.Data
	} else {
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			text += getNodeText(child)
		}
	}
	return strings.TrimSpace(text)
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

	textLength := len(getInnerText(s, true))
	if textLength == 0 {
		return 0
	}

	var linkLength int
	s.Find("a").Each(func(i int, link *goquery.Selection) {
		href, exists := link.Attr("href")
		coefficient := 1.0
		if exists && RegexpHashUrl.MatchString(href) {
			coefficient = 0.3
		}
		linkLength += int(float64(len(getInnerText(link, true))) * coefficient)
	})

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

	// Tokenize both texts
	tokensA := strings.FieldsFunc(strings.ToLower(textA), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})
	tokensB := strings.FieldsFunc(strings.ToLower(textB), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})

	if len(tokensA) == 0 || len(tokensB) == 0 {
		return 0
	}

	// Find unique tokens in B
	uniqueTokensB := make([]string, 0)
	for _, token := range tokensB {
		found := false
		for _, tokenA := range tokensA {
			if token == tokenA {
				found = true
				break
			}
		}
		if !found {
			uniqueTokensB = append(uniqueTokensB, token)
		}
	}

	// Calculate the distance
	distanceB := float64(len(strings.Join(uniqueTokensB, " "))) / float64(len(strings.Join(tokensB, " ")))
	return 1 - distanceB
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
func nodeToString(node *html.Node) string {
	if node == nil {
		return "nil"
	}

	if node.Type == TextNode {
		text := strings.TrimSpace(node.Data)
		if len(text) > 20 {
			text = text[:20] + "..."
		}
		return fmt.Sprintf("TextNode (\"%s\")", text)
	}

	var attrPairs []string
	for _, attr := range node.Attr {
		attrPairs = append(attrPairs, fmt.Sprintf("%s=\"%s\"", attr.Key, attr.Val))
	}
	return fmt.Sprintf("<%s %s>", strings.ToLower(node.Data), strings.Join(attrPairs, " "))
}