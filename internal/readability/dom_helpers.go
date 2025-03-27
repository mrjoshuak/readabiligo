package readability

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// isSameNode checks if two nodes are the same
func isSameNode(node1, node2 *html.Node) bool {
	if node1 == nil || node2 == nil {
		return node1 == node2
	}
	return node1 == node2
}

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

// contains checks if a string is in a string slice
func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// hasAncestorTag checks if the node has an ancestor with the given tag
func hasAncestorTag(s *goquery.Selection, tagName string, maxDepth int, filterFn func(*goquery.Selection) bool) bool {
	if s == nil || s.Length() == 0 {
		return false
	}

	tagName = strings.ToUpper(tagName)
	depth := 0

	parent := s.Parent()
	for parent != nil && parent.Length() > 0 {
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

	children := s.Children()
	if children.Length() != 1 {
		return false
	}
	
	firstChild := children.First()
	if firstChild.Length() == 0 || strings.ToUpper(goquery.NodeName(firstChild)) != strings.ToUpper(tag) {
		return false
	}

	// Check if there are non-empty text nodes
	hasTextNode := false
	s.Contents().Each(func(i int, c *goquery.Selection) {
		if c.Length() > 0 && c.Get(0) != nil && c.Get(0).Type == TextNode {
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