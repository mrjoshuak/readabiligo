package simplifiers

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// SimpleTree represents a simplified HTML tree
type SimpleTree struct {
	doc *goquery.Document
}

// String returns the HTML representation of the SimpleTree
func (st *SimpleTree) String() string {
	if st == nil || st.doc == nil {
		return ""
	}

	html, err := st.doc.Html()
	if err != nil {
		return ""
	}

	return StripHTMLWhitespace(html)
}

// SimpleTreeFromHTMLString creates a SimpleTree from an HTML string
func SimpleTreeFromHTMLString(html string) (*SimpleTree, error) {
	// Insert space into non-spaced comments so that html5lib can interpret them correctly
	html = strings.ReplaceAll(html, "<!---->", "<!-- -->")

	// Parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	// Apply the simplification steps
	opts := ContentOptions{
		RemoveBlacklist:   true,
		UnwrapElements:    true,
		ProcessSpecial:    true,
		ConsolidateText:   true,
		RemoveEmpty:       true,
		UnnestParagraphs:  true,
		InsertBreaks:      true,
		WrapBareText:      true,
		AddContentDigests: false,
		AddNodeIndexes:    false,
	}

	// Remove comments, CDATA (which is converted to comments) and DOCTYPE
	removeMetadata(doc)

	// Strip tag attributes apart from 'class' and 'style'
	stripAttributes(doc)

	// Remove blacklisted elements
	if opts.RemoveBlacklist {
		removeBlacklist(doc)
	}

	// Unwrap elements where we want to keep the text but drop the containing tag
	if opts.UnwrapElements {
		unwrapElements(doc)
	}

	// Process elements with special innerText handling
	if opts.ProcessSpecial {
		processSpecialElements(doc)
	}

	// Process unknown elements
	processUnknownElements(doc)

	// Consolidate text, joining any consecutive NavigableStrings together.
	// Must come before any whitespace operations (eg. remove_empty_strings_and_elements or normalise_strings)
	if opts.ConsolidateText {
		consolidateText(doc)
	}

	// Remove empty string elements
	if opts.RemoveEmpty {
		removeEmptyStringsAndElements(doc)
	}

	// Split out block-level elements illegally contained inside paragraphs
	if opts.UnnestParagraphs {
		unnestParagraphs(doc)
	}

	// Replace <br> and <hr> elements with paragraph breaks
	// Must come after remove_empty_strings_and_elements so that consecutive <br>s can be identified
	// Re-consolidates strings at the end, so must come before normalise_strings
	if opts.InsertBreaks {
		insertParagraphBreaks(doc)
	}

	// Wrap any remaining bare text in a suitable block level element
	// Must come after consolidate_text and identify_and_replace_break_elements
	// otherwise there may be multiple strings inside a <p> tag which would create nested <p>s
	if opts.WrapBareText {
		wrapBareText(doc)
	}

	// Normalize all strings, removing whitespace and fixing unicode issues
	// Must come after consolidate_text and insert_paragraph_breaks which join
	// strings with semantic whitespace
	normalizeStrings(doc)

	// Recursively replace any elements which have no children or only zero-length children
	recursivelyPruneElements(doc)

	// Finally ensure that the whole tree is wrapped in a div
	// Strip out enclosing elements that cannot live inside a div
	body := doc.Find("body")
	if body.Length() > 0 {
		// If the body has multiple children, wrap them in a div
		if body.Children().Length() > 1 {
			// Create a new div
			div := doc.Find("body").AppendHtml("<div></div>").Find("div").Last()

			// Move all children to the div
			body.Children().Each(func(i int, s *goquery.Selection) {
				// Skip the div we just created
				if i == body.Children().Length()-1 {
					return
				}

				// Get the HTML of the child
				html, err := goquery.OuterHtml(s)
				if err != nil {
					return
				}

				// Append to the div
				div.AppendHtml(html)

				// Remove the original
				s.Remove()
			})
		} else if body.Children().Length() == 1 && body.Children().First().Is("div") {
			// If the body has a single div child, we're good
		} else if body.Children().Length() == 1 {
			// If the body has a single non-div child, wrap it in a div
			child := body.Children().First()
			html, err := goquery.OuterHtml(child)
			if err == nil {
				body.SetHtml("<div>" + html + "</div>")
			}
		} else {
			// If the body has no children, add an empty div
			body.SetHtml("<div></div>")
		}
	}

	return &SimpleTree{doc: doc}, nil
}

// PlainContent generates plain content from HTML with optional content digests and node indexes
func PlainContent(html string, addContentDigests, addNodeIndexes bool) (string, error) {
	// Parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", fmt.Errorf("parsing HTML: %w", err)
	}

	// Apply the simplification steps
	opts := ContentOptions{
		AddContentDigests: addContentDigests,
		AddNodeIndexes:    addNodeIndexes,
	}

	// Make all elements plain
	body := doc.Find("body")
	if body.Length() > 0 {
		el := NewPlainElement(body)

		// Add node indexes if requested
		if opts.AddNodeIndexes {
			// Use the function from html.go
			el.SetAttr("data-node-index", "0")

			// Add indexes to child elements recursively
			addChildNodeIndexes(el, "0")
		}

		// Add content digests if requested
		if opts.AddContentDigests {
			// Process all paragraph and list item elements
			doc.Find("p, li").Each(func(i int, s *goquery.Selection) {
				pel := NewPlainElement(s)
				digest := calculateContentDigest(pel)
				if digest != "" {
					pel.SetAttr("data-content-digest", digest)
				}
			})
		}
	}

	// Generate output
	renderedHTML, err := doc.Html()
	if err != nil {
		return "", fmt.Errorf("rendering HTML: %w", err)
	}

	renderedHTML = StripHTMLWhitespace(renderedHTML)

	// Fix HTML entities in the output
	renderedHTML = strings.ReplaceAll(renderedHTML, "&#34;", "\"")

	return renderedHTML, nil
}

// normalizeStrings normalizes all text nodes in the document
func normalizeStrings(doc *goquery.Document) {
	// Find all text nodes
	var textNodes []*goquery.Selection
	doc.Find("*").Each(func(_ int, s *goquery.Selection) {
		s.Contents().Each(func(_ int, c *goquery.Selection) {
			if c.Get(0) != nil && c.Get(0).Type == 3 { // TextNode
				textNodes = append(textNodes, c)
			}
		})
	})

	// Normalize each text node
	for _, node := range textNodes {
		text := node.Text()
		normalized := NormalizeText(text)
		if text != normalized {
			node.Get(0).Data = normalized
		}
	}
}

// recursivelyPruneElements removes empty elements
func recursivelyPruneElements(doc *goquery.Document) {
	// Keep pruning until no more elements are removed
	for {
		removed := false
		doc.Find("*").Each(func(_ int, s *goquery.Selection) {
			// Skip if this is a structural element
			name := goquery.NodeName(s)
			if name == "html" || name == "head" || name == "body" {
				return
			}

			// Check if this element has no children or only empty text nodes
			isEmpty := true
			s.Contents().Each(func(_ int, c *goquery.Selection) {
				if c.Get(0) != nil && c.Get(0).Type == 3 { // TextNode
					if NormalizeText(c.Text()) != "" {
						isEmpty = false
					}
				} else {
					isEmpty = false
				}
			})

			if isEmpty {
				s.Remove()
				removed = true
			}
		})

		if !removed {
			break
		}
	}
}

// addChildNodeIndexes adds node indexes to child elements recursively
func addChildNodeIndexes(el *PlainElement, parentIndex string) {
	if el == nil || el.Selection == nil {
		return
	}

	el.Children().Each(func(i int, s *goquery.Selection) {
		childEl := NewPlainElement(s)
		childIndex := fmt.Sprintf("%s.%d", parentIndex, i+1)
		childEl.SetAttr("data-node-index", childIndex)

		// Recursively process children
		addChildNodeIndexes(childEl, childIndex)
	})
}
