package readability

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// prepArticle prepares the article node for display
func (r *Readability) prepArticle(articleContent *goquery.Selection) {
	// Clean styles
	r.cleanStyles(articleContent)

	// Mark data tables
	r.markDataTables(articleContent)

	// Fix lazy-loaded images
	r.fixLazyImages(articleContent)

	// Clean bad elements
	r.cleanConditionally(articleContent, "form")
	r.cleanConditionally(articleContent, "fieldset")
	r.clean(articleContent, "object")
	r.clean(articleContent, "embed")
	
	// Now safe to remove these container elements
	r.clean(articleContent, "footer")
	r.clean(articleContent, "link")
	r.clean(articleContent, "aside")
	r.clean(articleContent, "nav") // Explicitly remove navigation elements
	
	// Clean duplicate headings early
	r.cleanHeaders(articleContent)

	// Clean elements with share buttons
	articleContent.Children().Each(func(i int, child *goquery.Selection) {
		r.cleanMatchedNodes(child, func(node *goquery.Selection, matchString string) bool {
			return RegexpShareElements.MatchString(matchString) &&
				len(getInnerText(node, true)) < r.options.CharThreshold
		})
	})

	// Clean other elements
	r.clean(articleContent, "iframe")
	r.clean(articleContent, "input")
	r.clean(articleContent, "textarea")
	r.clean(articleContent, "select")
	r.clean(articleContent, "button")
	
	// Run cleanHeaders a second time to catch any headers we might have missed
	r.cleanHeaders(articleContent)

	// Clean tables and other elements conditionally
	r.cleanConditionally(articleContent, "table")
	r.cleanConditionally(articleContent, "ul")
	r.cleanConditionally(articleContent, "div")

	// Remove empty paragraphs
	articleContent.Find("p").Each(func(i int, p *goquery.Selection) {
		// Count embedded elements
		imgCount := p.Find("img").Length()
		embedCount := p.Find("embed").Length()
		objectCount := p.Find("object").Length()
		iframeCount := p.Find("iframe").Length()
		totalCount := imgCount + embedCount + objectCount + iframeCount

		// Remove if empty and has no embeds
		if totalCount == 0 && getInnerText(p, false) == "" {
			p.Remove()
		}
	})

	// Remove BR elements before paragraphs
	articleContent.Find("br").Each(func(i int, br *goquery.Selection) {
		next := br.Next()
		if next.Length() > 0 && getNodeName(next) == "P" {
			br.Remove()
		}
	})

	// Replace single-cell tables with their content
	articleContent.Find("table").Each(func(i int, table *goquery.Selection) {
		tbody := table.Find("tbody").First()
		if tbody.Length() == 0 {
			tbody = table
		}

		// Check if table has a single row
		rows := tbody.Find("tr")
		if rows.Length() == 1 {
			// Check if row has a single cell
			cells := rows.First().Find("td")
			if cells.Length() == 1 {
				cell := cells.First()
				// Replace table with cell content wrapped in div or p
				if everyNode(cell.Contents(), func(i int, s *goquery.Selection) bool {
					return s.Get(0) != nil && isPhrasingContent(s.Get(0))
				}) {
					cell = setNodeTag(cell, "p")
				} else {
					cell = setNodeTag(cell, "div")
				}
				table.ReplaceWithSelection(cell)
			}
		}
	})
}

// prepDocument prepares the document for readability to scrape it
func (r *Readability) prepDocument() {
	// Remove all style tags in head
	r.doc.Find("style").Remove()

	// Replace <br> tags with proper <p> elements
	if body := r.doc.Find("body"); body.Length() > 0 {
		r.replaceBrs(body)
	}

	// Replace font tags with spans
	r.doc.Find("font").Each(func(i int, s *goquery.Selection) {
		setNodeTag(s, "SPAN")
	})
}

// replaceBrs replaces 2 or more successive <br> elements with a single <p>
func (r *Readability) replaceBrs(elem *goquery.Selection) {
	elem.Find("br").Each(func(i int, br *goquery.Selection) {
		next := br.Next()

		// Whether 2 or more <br> elements have been found and replaced
		replaced := false

		// Find <br> chain and remove all but the first one
		for next.Length() > 0 && getNodeName(next) == "BR" {
			replaced = true
			nextSibling := next.Next()
			next.Remove()
			next = nextSibling
		}

		// If we removed a <br> chain, replace the remaining <br> with a <p>
		if replaced {
			p := r.createElement("p")
			br.ReplaceWithSelection(p)

			// Move all siblings until the next <br><br> into the new paragraph
			next = p.Next()
			for next.Length() > 0 {
				// If we've hit another <br><br>, we're done
				if getNodeName(next) == "BR" {
					nextElem := next.Next()
					if nextElem.Length() > 0 && getNodeName(nextElem) == "BR" {
						break
					}
				}

				// If not phrasing content, break
				if next.Get(0) != nil && !isPhrasingContent(next.Get(0)) {
					break
				}

				// Move the node into the paragraph
				nextSibling := next.Next()
				next.Remove()
				p.AppendSelection(next)
				next = nextSibling
			}

			// Remove any trailing whitespace
			p.Contents().Each(func(i int, c *goquery.Selection) {
				if c.Get(0) != nil && c.Get(0).Type == TextNode && c.Get(0).Data == " " {
					c.Remove()
				}
			})

			// If the paragraph's parent is also a paragraph, change it to a div
			if getNodeName(p.Parent()) == "P" {
				setNodeTag(p.Parent(), "DIV")
			}
		}
	})
}

// fixLazyImages fixes lazy-loaded images
func (r *Readability) fixLazyImages(root *goquery.Selection) {
	root.Find("img, picture, figure").Each(func(i int, elem *goquery.Selection) {
		// Check for lazy-loaded images
		src, hasSrc := elem.Attr("src")
		_, hasSrcset := elem.Attr("srcset")
		className, _ := elem.Attr("class")

		// Skip non-lazy images
		if (hasSrc || hasSrcset) && !strings.Contains(strings.ToLower(className), "lazy") {
			return
		}

		// Check for base64 data URIs that might be placeholders
		if hasSrc && RegexpB64DataUrl.MatchString(src) {
			parts := RegexpB64DataUrl.FindStringSubmatch(src)
			if len(parts) > 1 && parts[1] == "image/svg+xml" {
				// SVG can be meaningful even in small size, so keep it
				return
			}

			// Look for other attributes with image URLs
			hasImageAttribute := false
			for _, attr := range elem.Get(0).Attr {
				if attr.Key == "src" {
					continue
				}
				if regexp.MustCompile(`\.(jpg|jpeg|png|webp)`).MatchString(attr.Val) {
					hasImageAttribute = true
					break
				}
			}

			// If we found other image attributes, remove the src
			if hasImageAttribute {
				// Calculate the size of the base64 data
				b64starts := strings.Index(src, "base64,") + 7
				if b64starts >= 7 && len(src)-b64starts < 133 {
					// Less than 100 bytes (133 in base64), likely a placeholder
					elem.RemoveAttr("src")
				}
			}
		}

		// Look for alternative image URLs in other attributes
		for _, attr := range elem.Get(0).Attr {
			if attr.Key == "src" || attr.Key == "srcset" || attr.Key == "alt" {
				continue
			}

			// Check for attributes with image URLs
			value := attr.Val
			if regexp.MustCompile(`\.(jpg|jpeg|png|webp)\s+\d`).MatchString(value) {
				// Add as srcset if it has dimension info
				elem.SetAttr("srcset", value)
			} else if regexp.MustCompile(`^\s*\S+\.(jpg|jpeg|png|webp)\S*\s*$`).MatchString(value) {
				// Add as src if it's a clean image URL
				elem.SetAttr("src", value)
			}
		}
	})
}

// postProcessContent runs any post-process modifications to article content
func (r *Readability) postProcessContent(articleContent *goquery.Selection) {
	// Fix relative URIs
	r.fixRelativeUris(articleContent)

	// Simplify nested elements
	r.simplifyNestedElements(articleContent)

	// Clean classes if not keeping them
	if !r.options.KeepClasses {
		r.cleanClasses(articleContent)
	}
}

// fixRelativeUris converts relative URIs to absolute ones
func (r *Readability) fixRelativeUris(articleContent *goquery.Selection) {
	// Get base URI
	baseURI := ""
	documentURI := ""

	// Try to get base URI from the document
	r.doc.Find("base").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			baseURI = href
		}
	})

	// If no base URI found, use document.location
	if baseURI == "" {
		// Try to extract from URL in head
		r.doc.Find("head meta[property='og:url']").Each(func(i int, s *goquery.Selection) {
			if content, exists := s.Attr("content"); exists {
				documentURI = content
			}
		})
	}

	// If still no URI, we'll have to leave relative links as is
	if baseURI == "" && documentURI == "" {
		return
	}

	// Function to convert a relative URI to absolute
	toAbsoluteURI := func(uri string) string {
		// Leave hash links alone if baseURI equals documentURI
		if baseURI == documentURI && strings.HasPrefix(uri, "#") {
			return uri
		}

		// Otherwise, resolve against base URI
		base, err := url.Parse(baseURI)
		if err != nil {
			return uri
		}

		relative, err := url.Parse(uri)
		if err != nil {
			return uri
		}

		return base.ResolveReference(relative).String()
	}

	// Fix links
	articleContent.Find("a").Each(func(i int, link *goquery.Selection) {
		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		// Remove javascript: URIs since they won't work after scripts are removed
		if strings.HasPrefix(href, "javascript:") {
			// If the link has only text content, convert to text node
			if link.Children().Length() == 0 {
				link.ReplaceWithHtml(link.Text())
			} else {
				// Replace with a <span> to preserve children
				span := r.createElement("span")
				link.ReplaceWithSelection(span)
				link.Children().Each(func(i int, child *goquery.Selection) {
					span.AppendSelection(child)
				})
			}
		} else {
			// Convert to absolute URI
			link.SetAttr("href", toAbsoluteURI(href))
		}
	})

	// Fix media references
	articleContent.Find("img, picture, figure, video, audio, source").Each(func(i int, media *goquery.Selection) {
		// Fix src attribute
		if src, exists := media.Attr("src"); exists && src != "" {
			media.SetAttr("src", toAbsoluteURI(src))
		}

		// Fix poster attribute (for videos)
		if poster, exists := media.Attr("poster"); exists && poster != "" {
			media.SetAttr("poster", toAbsoluteURI(poster))
		}

		// Fix srcset attribute
		if srcset, exists := media.Attr("srcset"); exists && srcset != "" {
			// Pattern: URL, optional size, optional comma
			// Example: "image.jpg 1x, image2.jpg 2x"
			newSrcset := RegexpSrcsetUrl.ReplaceAllStringFunc(srcset, func(match string) string {
				parts := RegexpSrcsetUrl.FindStringSubmatch(match)
				if len(parts) < 4 {
					return match
				}
				url := parts[1]
				size := parts[2]
				separator := parts[3]
				return toAbsoluteURI(url) + size + separator
			})
			media.SetAttr("srcset", newSrcset)
		}
	})
}

// simplifyNestedElements simplifies unnecessarily nested elements
func (r *Readability) simplifyNestedElements(articleContent *goquery.Selection) {
	if articleContent == nil || articleContent.Length() == 0 {
		return
	}

	node := articleContent

	for node != nil && node.Length() > 0 {
		if getNodeName(node) == "DIV" || getNodeName(node) == "SECTION" {
			// Skip elements with readability ID
			id, exists := node.Attr("id")
			if exists && strings.HasPrefix(id, "readability") {
				node = getNextNode(node, false)
				continue
			}

			// Check if it's an element without content
			if isElementWithoutContent(node) {
				node = removeAndGetNext(node)
				continue
			}

			// Check if it has a single child of the same type
			if hasSingleTagInsideElement(node, "DIV") || hasSingleTagInsideElement(node, "SECTION") {
				// Replace node with its child, preserving attributes
				child := node.Children().First()
				for _, attr := range node.Get(0).Attr {
					// Add the attribute to the child if it doesn't already have it
					if _, exists := child.Attr(attr.Key); !exists {
						child.SetAttr(attr.Key, attr.Val)
					}
				}
				node.ReplaceWithSelection(child)
				node = child
				continue
			}
		}

		nextNode := getNextNode(node, false)
		// Ensure we don't get stuck in an infinite loop if getNextNode returns nil
		if nextNode == nil || nextNode.Length() == 0 {
			break
		}
		node = nextNode
	}
}