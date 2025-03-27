package readability

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// getNextNode gets the next node in the DOM in depth-first order
func getNextNode(s *goquery.Selection, ignoreSelfAndKids bool) *goquery.Selection {
	if s == nil || s.Length() == 0 {
		return nil
	}

	// First check for kids if not ignoring
	if !ignoreSelfAndKids {
		children := s.Children()
		if children != nil && children.Length() > 0 {
			firstChild := children.First()
			if firstChild != nil && firstChild.Length() > 0 {
				return firstChild
			}
		}
	}

	// Then for siblings
	nextSibling := s.Next()
	if nextSibling != nil && nextSibling.Length() > 0 {
		return nextSibling
	}

	// Finally, move up the parent chain and find a sibling
	parent := s.Parent()
	for parent != nil && parent.Length() > 0 {
		nextParentSibling := parent.Next()
		if nextParentSibling != nil && nextParentSibling.Length() > 0 {
			return nextParentSibling
		}
		parent = parent.Parent()
	}

	return nil
}

// removeNode removes a node from the DOM and returns the next node
func removeAndGetNext(s *goquery.Selection) *goquery.Selection {
	next := getNextNode(s, true)
	if s.Length() > 0 {
		s.Remove()
	}
	return next
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