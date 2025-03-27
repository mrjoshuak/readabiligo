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
		if children.Length() > 0 {
			return children.First()
		}
	}

	// Then for siblings - more efficient by checking length directly
	nextSibling := s.Next()
	if nextSibling.Length() > 0 {
		return nextSibling
	}

	// Finally, move up the parent chain and find a sibling
	// This is a frequently traversed code path, so we optimize it
	parent := s.Parent()
	for parent.Length() > 0 {
		nextParentSibling := parent.Next()
		if nextParentSibling.Length() > 0 {
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
// Using a more efficient approach that doesn't create a new document for each tag change
func setNodeTag(s *goquery.Selection, tagName string) *goquery.Selection {
	if s == nil || s.Length() == 0 {
		return nil
	}

	// Get the original node content and attributes
	html, err := s.Html()
	if err != nil {
		return nil
	}
	
	// Store all attributes
	attrs := make(map[string]string)
	node := s.Get(0)
	if node != nil {
		for _, attr := range node.Attr {
			attrs[attr.Key] = attr.Val
		}
	}
	
	// Create a new element directly in the current document context
	// This is more efficient than creating a new document
	parentNode := s.Parent()
	if parentNode.Length() == 0 {
		// Fallback if we don't have a parent
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(fmt.Sprintf("<%s></%s>", tagName, tagName)))
		if err != nil {
			return nil
		}
		newElement := doc.Find(tagName)
		
		// Copy attributes
		for key, val := range attrs {
			newElement.SetAttr(key, val)
		}
		
		// Copy content
		newElement.SetHtml(html)
		
		// Replace the original node
		s.ReplaceWithSelection(newElement)
		return newElement
	}
	
	// Create an empty element with the right tag
	newElement := s.AppendSelection(s.Parent().AppendHtml(fmt.Sprintf("<%s></%s>", tagName, tagName)).Children().Last().Remove())
	
	// Copy attributes
	for key, val := range attrs {
		newElement.SetAttr(key, val)
	}
	
	// Copy content
	newElement.SetHtml(html)
	
	// Replace the original node
	s.ReplaceWithSelection(newElement)
	return newElement
}