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

// Helper functions for the Readability implementation

// isNodeVisible checks if a node is visible to the user
func isNodeVisible(node *html.Node) bool {
	if node == nil {
		return false
	}

	// Check for display:none
	for _, attr := range node.Attr {
		if attr.Key == "style" && strings.Contains(attr.Val, "display:none") {
			return false
		}
	}

	// Check for hidden attribute
	hasHidden := false
	for _, attr := range node.Attr {
		if attr.Key == "hidden" {
			hasHidden = true
			break
		}
	}
	if hasHidden {
		return false
	}

	// Check for aria-hidden attribute
	for _, attr := range node.Attr {
		if attr.Key == "aria-hidden" && attr.Val == "true" {
			// Check for fallback-image class exception
			hasExceptionClass := false
			for _, attr := range node.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "fallback-image") {
					hasExceptionClass = true
					break
				}
			}
			if !hasExceptionClass {
				return false
			}
		}
	}

	return true
}

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

// hasAncestorTag checks if the node has an ancestor with the given tag
func hasAncestorTag(s *goquery.Selection, tagName string, maxDepth int, filterFn func(*goquery.Selection) bool) bool {
	if s == nil || s.Length() == 0 {
		return false
	}

	tagName = strings.ToUpper(tagName)
	depth := 0

	parent := s.Parent()
	for parent.Length() > 0 {
		if maxDepth > 0 && depth > maxDepth {
			return false
		}

		if strings.ToUpper(goquery.NodeName(parent)) == tagName {
			if filterFn == nil || filterFn(parent) {
				return true
			}
		}

		parent = parent.Parent()
		depth++
	}

	return false
}

// isElementWithoutContent checks if a node has no content
func isElementWithoutContent(s *goquery.Selection) bool {
	if s == nil || s.Length() == 0 {
		return true
	}

	text := strings.TrimSpace(s.Text())
	if text != "" {
		return false
	}

	children := s.Children()
	brCount := s.Find("br").Length()
	hrCount := s.Find("hr").Length()

	// Only br and hr elements, or no children at all
	return children.Length() == 0 || children.Length() == brCount+hrCount
}

// hasSingleTagInsideElement checks if the node contains only a single tag of the given type
func hasSingleTagInsideElement(s *goquery.Selection, tag string) bool {
	if s == nil || s.Length() == 0 {
		return false
	}

	// There should be exactly 1 element child with the given tag
	if s.Children().Length() != 1 || strings.ToUpper(goquery.NodeName(s.Children())) != strings.ToUpper(tag) {
		return false
	}

	// Check if there are non-empty text nodes
	hasTextNode := false
	s.Contents().Each(func(i int, c *goquery.Selection) {
		if c.Get(0).Type == TextNode {
			text := strings.TrimSpace(c.Text())
			if text != "" {
				hasTextNode = true
			}
		}
	})

	return !hasTextNode
}

// hasChildBlockElement checks if an element has any block level children
func hasChildBlockElement(s *goquery.Selection) bool {
	if s == nil || s.Length() == 0 {
		return false
	}

	// Check if the node has any children that are block-level elements
	for _, elem := range DivToPElems {
		if s.Find(elem).Length() > 0 {
			return true
		}
	}

	return false
}

// isPhrasingContent checks if a node is phrasing content
func isPhrasingContent(node *html.Node) bool {
	if node == nil {
		return false
	}

	// Text nodes are phrasing content
	if node.Type == TextNode {
		return true
	}

	// Check if it's in the phrasing elements list
	tag := strings.ToUpper(node.Data)
	for _, elem := range PhrasingElems {
		if tag == elem {
			return true
		}
	}

	// Special handling for A, DEL, and INS elements
	if tag == "A" || tag == "DEL" || tag == "INS" {
		// Check if all its children are phrasing content
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if !isPhrasingContent(child) {
				return false
			}
		}
		return true
	}

	return false
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

// getNodeAncestors gets a list of ancestors for a node, optionally limited by depth
func getNodeAncestors(s *goquery.Selection, maxDepth int) []*goquery.Selection {
	ancestors := []*goquery.Selection{}
	parent := s.Parent()

	i := 0
	for parent.Length() > 0 {
		ancestors = append(ancestors, parent)
		if maxDepth > 0 && i == maxDepth {
			break
		}
		parent = parent.Parent()
		i++
	}

	return ancestors
}

// generateHash creates a SHA-256 hash for text
func generateHash(text string) string {
	hash := sha256.New()
	hash.Write([]byte(text))
	return fmt.Sprintf("%x", hash.Sum(nil))
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

// getNodeName returns the tag name of a selection
func getNodeName(s *goquery.Selection) string {
	if s == nil || s.Length() == 0 {
		return ""
	}
	node := s.Get(0)
	if node == nil {
		return ""
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
		return fmt.Sprintf("%s (\"%s\")", node.Type, text)
	}

	var attrPairs []string
	for _, attr := range node.Attr {
		attrPairs = append(attrPairs, fmt.Sprintf("%s=\"%s\"", attr.Key, attr.Val))
	}
	return fmt.Sprintf("<%s %s>", strings.ToLower(node.Data), strings.Join(attrPairs, " "))
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

// getNextNode gets the next node in the DOM in depth-first order
func getNextNode(s *goquery.Selection, ignoreSelfAndKids bool) *goquery.Selection {
	if s == nil || s.Length() == 0 {
		return nil
	}

	// First check for kids if not ignoring
	if !ignoreSelfAndKids && s.Children().Length() > 0 {
		return s.Children().First()
	}

	// Then for siblings
	if s.Next().Length() > 0 {
		return s.Next()
	}

	// Finally, move up the parent chain and find a sibling
	parent := s.Parent()
	for parent.Length() > 0 && parent.Next().Length() == 0 {
		parent = parent.Parent()
	}

	if parent.Length() == 0 {
		return nil
	}
	return parent.Next()
}

// removeNode removes a node from the DOM and returns the next node
func removeAndGetNext(s *goquery.Selection) *goquery.Selection {
	next := getNextNode(s, true)
	if s.Length() > 0 {
		s.Remove()
	}
	return next
}

// forEachNode applies a function to each node in a selection
func forEachNode(selection *goquery.Selection, fn func(int, *goquery.Selection)) {
	if selection == nil || selection.Length() == 0 {
		return
	}
	selection.Each(fn)
}

// findNode finds a node in a selection that matches a function
func findNode(selection *goquery.Selection, fn func(int, *goquery.Selection) bool) *goquery.Selection {
	if selection == nil || selection.Length() == 0 {
		return nil
	}

	var result *goquery.Selection
	selection.Each(func(i int, s *goquery.Selection) {
		if result != nil {
			return
		}
		if fn(i, s) {
			result = s
		}
	})
	return result
}

// someNode checks if any node in a selection matches a function
func someNode(selection *goquery.Selection, fn func(int, *goquery.Selection) bool) bool {
	if selection == nil || selection.Length() == 0 {
		return false
	}

	matches := false
	selection.EachWithBreak(func(i int, s *goquery.Selection) bool {
		if fn(i, s) {
			matches = true
			return false // stop iterating
		}
		return true // continue
	})
	return matches
}

// everyNode checks if all nodes in a selection match a function
func everyNode(selection *goquery.Selection, fn func(int, *goquery.Selection) bool) bool {
	if selection == nil || selection.Length() == 0 {
		return true
	}

	allMatch := true
	selection.EachWithBreak(func(i int, s *goquery.Selection) bool {
		if !fn(i, s) {
			allMatch = false
			return false // stop iterating
		}
		return true // continue
	})
	return allMatch
}

// getAllNodesWithTag gets all nodes with the specified tag names
func getAllNodesWithTag(s *goquery.Selection, tagNames []string) *goquery.Selection {
	if s == nil || s.Length() == 0 || len(tagNames) == 0 {
		return nil
	}

	// Create a selector with all tag names
	selector := strings.Join(tagNames, ",")
	return s.Find(selector)
}

// setNodeTag changes the tag name of a node
func setNodeTag(s *goquery.Selection, tagName string) *goquery.Selection {
	if s == nil || s.Length() == 0 {
		return nil
	}

	// Create a new element with the desired tag
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(fmt.Sprintf("<%s></%s>", tagName, tagName)))
	if err != nil {
		return nil
	}
	newElement := doc.Find(tagName)

	// Copy attributes
	for _, attr := range s.Get(0).Attr {
		newElement.SetAttr(attr.Key, attr.Val)
	}

	// Copy content
	html, err := s.Html()
	if err == nil {
		newElement.SetHtml(html)
	}

	// Replace the original node
	s.ReplaceWithSelection(newElement)
	return newElement
}

// isSingleImage checks if node is an image or contains exactly one image
func isSingleImage(s *goquery.Selection) bool {
	if s == nil || s.Length() == 0 {
		return false
	}

	// If it's an image tag itself
	if getNodeName(s) == "IMG" {
		return true
	}

	// If it has exactly one child and no text
	if s.Children().Length() != 1 || strings.TrimSpace(s.Text()) != "" {
		return false
	}

	// Recursively check the single child
	return isSingleImage(s.Children())
}