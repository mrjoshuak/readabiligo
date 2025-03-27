package readability

import (
	"fmt"
	"math"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// ReadabilityOptions defines configuration options for the Readability parser
type ReadabilityOptions struct {
	Debug                bool     // Debug mode
	MaxElemsToParse      int      // Maximum elements to parse (0 = no limit)
	NbTopCandidates      int      // Number of top candidates to consider
	CharThreshold        int      // Minimum character threshold
	ClassesToPreserve    []string // Classes to preserve
	KeepClasses          bool     // Whether to keep classes
	DisableJSONLD        bool     // Whether to disable JSON-LD processing
	AllowedVideoRegex    *regexp.Regexp // Regex for allowed videos
	PreserveImportantLinks bool     // Whether to preserve important links like "More information..." in cleaned elements
}

// defaultReadabilityOptions returns the default options
func defaultReadabilityOptions() ReadabilityOptions {
	return ReadabilityOptions{
		Debug:                false,
		MaxElemsToParse:      DefaultMaxElemsToParse,
		NbTopCandidates:      DefaultNTopCandidates,
		CharThreshold:        DefaultCharThreshold,
		ClassesToPreserve:    ClassesToPreserve,
		KeepClasses:          false,
		DisableJSONLD:        false,
		AllowedVideoRegex:    RegexpVideos,
		PreserveImportantLinks: false, // Default to false to match ReadabiliPy's behavior
	}
}

// ReadabilityArticle represents the extracted article
type ReadabilityArticle struct {
	Title        string    // Article title
	Byline       string    // Article byline (author)
	Content      string    // Article content (HTML)
	TextContent  string    // Article text content (plain text)
	Length       int       // Length of the text content
	Excerpt      string    // Short excerpt
	SiteName     string    // Site name
	Date         time.Time // Publication date
}

// Readability implements the Readability algorithm
type Readability struct {
	doc              *goquery.Document // The HTML document
	options          ReadabilityOptions // Options for the parser
	articleTitle     string            // Extracted article title
	articleByline    string            // Extracted article byline
	articleDir       string            // Article text direction
	articleSiteName  string            // Site name
	attempts         []int             // Extraction attempts
	flags            int               // Flags controlling the algorithm
}

// createElement is a helper function that creates an element with the given tag name
// This is a workaround for the non-existent CreateElement method in goquery.Document
func (r *Readability) createElement(tagName string) *goquery.Selection {
	node := &html.Node{
		Type: html.ElementNode,
		Data: tagName,
	}
	return goquery.NewDocumentFromNode(node).Find(tagName)
}

// NodeInfo holds information about a node
type NodeInfo struct {
	node         *goquery.Selection // The node
	contentScore float64            // Content score
}

// NewFromDocument creates a new Readability parser from a goquery document
func NewFromDocument(doc *goquery.Document, opts *ReadabilityOptions) *Readability {
	options := defaultReadabilityOptions()
	if opts != nil {
		options = *opts
	}

	r := &Readability{
		doc:     doc,
		options: options,
		flags:   FlagStripUnlikelys | FlagWeightClasses | FlagCleanConditionally,
	}

	return r
}

// NewFromHTML creates a new Readability parser from HTML string
func NewFromHTML(html string, opts *ReadabilityOptions) (*Readability, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	return NewFromDocument(doc, opts), nil
}

// Parse runs the Readability algorithm
func (r *Readability) Parse() (*ReadabilityArticle, error) {
	// Check document
	if r.doc == nil || r.doc.Selection.Length() == 0 {
		return nil, fmt.Errorf("no document to parse")
	}

	// Check if document is too large
	if r.options.MaxElemsToParse > 0 {
		numNodes := r.doc.Find("*").Length()
		if numNodes > r.options.MaxElemsToParse {
			return nil, fmt.Errorf("document too large (%d elements)", numNodes)
		}
	}

	// Unwrap noscript images
	r.unwrapNoscriptImages()

	// Extract JSON-LD metadata (if enabled)
	jsonLd := make(map[string]string)
	if !r.options.DisableJSONLD {
		jsonLd = r.getJSONLD()
	}

	// Remove scripts
	r.removeScripts()

	// Prepare document
	r.prepDocument()

	// Get article metadata
	metadata := r.getArticleMetadata(jsonLd)
	r.articleTitle = metadata["title"]

	// Grab article content
	article := r.grabArticle()
	if article == nil {
		return nil, fmt.Errorf("could not extract article content")
	}

	// Post-process content
	r.postProcessContent(article)

	// If no excerpt in metadata, use the first paragraph
	excerpt := metadata["excerpt"]
	if excerpt == "" {
		article.Find("p").EachWithBreak(func(i int, s *goquery.Selection) bool {
			text := strings.TrimSpace(s.Text())
			if text != "" {
				excerpt = text
				return false // stop iteration
			}
			return true // continue
		})
	}

	// Additional cleanup step: make sure footers are removed
	// This is needed because in some cases, the clean function in prepArticle
	// might not have removed footer elements, especially if grabArticle returned the body
	if r.options.Debug {
		fmt.Printf("DEBUG: Final cleanup pass to remove any remaining footer elements\n")
	}
	
	// Apply the final cleanup to handle footer elements
	r.finalCleanupFooters(article)
	
	// Get text content from the cleaned article
	textContent := getInnerText(article, true)

	// Build the article
	result := &ReadabilityArticle{
		Title:       r.articleTitle,
		Byline:      metadata["byline"],
		Content:     getOuterHTML(article),
		TextContent: textContent,
		Length:      len(textContent),
		Excerpt:     excerpt,
		SiteName:    metadata["siteName"],
	}

	// Try to parse the date
	if date, err := time.Parse(time.RFC3339, metadata["date"]); err == nil {
		result.Date = date
	}

	return result, nil
}

// getArticleMetadata extracts metadata from the document
func (r *Readability) getArticleMetadata(jsonLd map[string]string) map[string]string {
	metadata := make(map[string]string)
	values := make(map[string]string)

	// Process meta tags
	r.doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		elementName, _ := s.Attr("name")
		elementProperty, _ := s.Attr("property")
		content, _ := s.Attr("content")

		if content == "" {
			return
		}

		// Process property attribute (OpenGraph, etc.)
		if elementProperty != "" {
			// Pattern: (dc|dcterm|og|twitter):(author|creator|description|title)
			propertyPattern := `\s*(dc|dcterm|og|twitter)\s*:\s*(author|creator|description|title|site_name)\s*`
			re := regexp.MustCompile(propertyPattern)
			matches := re.FindStringSubmatch(elementProperty)

			if len(matches) > 0 {
				// Convert to lowercase and remove whitespace
				name := strings.ToLower(strings.ReplaceAll(matches[0], " ", ""))
				values[name] = content
			}
		}

		// Process name attribute
		if elementName != "" {
			// Pattern: (dc|dcterm|og|twitter).(author|creator|description|title)
			namePattern := `^\s*(?:(dc|dcterm|og|twitter)\s*[\.:]\s*)?(author|creator|description|title|site_name)\s*$`
			re := regexp.MustCompile(namePattern)
			matches := re.FindStringSubmatch(elementName)

			if len(matches) > 0 {
				// Convert to lowercase, remove whitespace, and convert dots to colons
				name := strings.ToLower(strings.ReplaceAll(elementName, " ", ""))
				name = strings.ReplaceAll(name, ".", ":")
				values[name] = content
			}
		}
	})

	// Extract article title
	metadata["title"] = r.getArticleTitle()

	// Override with JSON-LD title if available
	if jsonLd["title"] != "" {
		metadata["title"] = jsonLd["title"]
	} else if values["dc:title"] != "" {
		metadata["title"] = values["dc:title"]
	} else if values["dcterm:title"] != "" {
		metadata["title"] = values["dcterm:title"]
	} else if values["og:title"] != "" {
		metadata["title"] = values["og:title"]
	} else if values["twitter:title"] != "" {
		metadata["title"] = values["twitter:title"]
	}

	// Extract article byline
	if jsonLd["byline"] != "" {
		metadata["byline"] = jsonLd["byline"]
	} else if values["dc:creator"] != "" {
		metadata["byline"] = values["dc:creator"]
	} else if values["dcterm:creator"] != "" {
		metadata["byline"] = values["dcterm:creator"]
	} else if values["author"] != "" {
		metadata["byline"] = values["author"]
	}

	// Extract article excerpt/description
	if jsonLd["excerpt"] != "" {
		metadata["excerpt"] = jsonLd["excerpt"]
	} else if values["dc:description"] != "" {
		metadata["excerpt"] = values["dc:description"]
	} else if values["dcterm:description"] != "" {
		metadata["excerpt"] = values["dcterm:description"]
	} else if values["og:description"] != "" {
		metadata["excerpt"] = values["og:description"]
	} else if values["description"] != "" {
		metadata["excerpt"] = values["description"]
	} else if values["twitter:description"] != "" {
		metadata["excerpt"] = values["twitter:description"]
	}

	// Extract site name
	if jsonLd["siteName"] != "" {
		metadata["siteName"] = jsonLd["siteName"]
	} else if values["og:site_name"] != "" {
		metadata["siteName"] = values["og:site_name"]
	}

	// Extract date
	if jsonLd["date"] != "" {
		metadata["date"] = jsonLd["date"]
	}

	// Unescape HTML entities
	for key, value := range metadata {
		metadata[key] = unescapeHtmlEntities(value)
	}

	return metadata
}

// getArticleTitle extracts the title from the document
func (r *Readability) getArticleTitle() string {
	// Get title from the document
	docTitle := strings.TrimSpace(r.doc.Find("title").Text())
	origTitle := docTitle

	// If they had an element with id "title" in their HTML
	if docTitle == "" {
		docTitle = origTitle
	}

	// Check for hierarchical separators
	titleHadHierarchicalSeparators := false

	// If there's a separator in the title
	if regexp.MustCompile(` [\|\-\\\/>»] `).MatchString(docTitle) {
		titleHadHierarchicalSeparators = regexp.MustCompile(` [\\\/>»] `).MatchString(docTitle)
		// First remove the final part
		docTitle = regexp.MustCompile(`(.*)[\|\-\\\/>»] .*`).ReplaceAllString(docTitle, "$1")

		// If too short, remove the first part instead
		if wordCount(docTitle) < 3 {
			docTitle = regexp.MustCompile(`[^\|\-\\\/>»]*[\|\-\\\/>»](.*)`).ReplaceAllString(origTitle, "$1")
		}
	} else if strings.Contains(docTitle, ": ") {
		// Check for a colon
		// Check if we have an h1 or h2 with the exact title
		matchFound := false
		r.doc.Find("h1, h2").EachWithBreak(func(i int, s *goquery.Selection) bool {
			if strings.TrimSpace(s.Text()) == docTitle {
				matchFound = true
				return false // stop iteration
			}
			return true // continue
		})

		// If no match, extract the title out of the original string
		if !matchFound {
			// Try the part after the colon
			colonIndex := strings.LastIndex(origTitle, ":")
			if colonIndex != -1 {
				docTitle = strings.TrimSpace(origTitle[colonIndex+1:])

				// If too short, try the part before the colon
				if wordCount(docTitle) < 3 {
					docTitle = strings.TrimSpace(origTitle[:colonIndex])

					// But if we have too many words before the colon, use the original title
					if wordCount(docTitle) > 5 {
						docTitle = origTitle
					}
				}
			}
		}
	} else if docTitle == "" || docTitle == "null" || len(docTitle) > 150 || len(docTitle) < 15 {
		// If the title is empty, too long, or too short, look for h1 elements
		h1s := r.doc.Find("h1")
		if h1s.Length() == 1 {
			docTitle = strings.TrimSpace(h1s.Text())
		}
	}

	// Normalize the title
	docTitle = strings.TrimSpace(RegexpNormalize.ReplaceAllString(docTitle, " "))

	// If title is now very short, use the original title
	if wordCount(docTitle) <= 4 && (!titleHadHierarchicalSeparators || wordCount(docTitle) != wordCount(regexp.MustCompile(`[\|\-\\\/>»]+`).ReplaceAllString(origTitle, ""))-1) {
		docTitle = origTitle
	}

	return docTitle
}

// unwrapNoscriptImages finds all <noscript> that contain <img> and replaces them
func (r *Readability) unwrapNoscriptImages() {
	// First remove any img tags without source or attributes that might contain an image
	r.doc.Find("img").Each(func(i int, img *goquery.Selection) {
		// Check if the img has any attributes that might indicate a valid image
		hasValidAttrs := false
		for _, attr := range []string{"src", "srcset", "data-src", "data-srcset"} {
			if val, exists := img.Attr(attr); exists && val != "" {
				hasValidAttrs = true
				break
			}
		}

		// Check if any attribute has an image extension
		if !hasValidAttrs {
			img.Each(func(i int, s *goquery.Selection) {
				for _, attr := range s.Get(0).Attr {
					if regexp.MustCompile(`\.(jpg|jpeg|png|webp)`).MatchString(attr.Val) {
						hasValidAttrs = true
						break
					}
				}
			})
		}

		// Remove if no valid attributes found
		if !hasValidAttrs {
			img.Remove()
		}
	})

	// Process noscript tags
	r.doc.Find("noscript").Each(func(i int, noscript *goquery.Selection) {
		// Parse content of noscript
		noscriptHTML, err := noscript.Html()
		if err != nil {
			return
		}

		// Create a temporary document to hold the noscript content
		tempDoc, err := goquery.NewDocumentFromReader(strings.NewReader(noscriptHTML))
		if err != nil {
			return
		}

		// Check if it only contains an image
		if !isSingleImage(tempDoc.Selection) {
			return
		}

		// Check if previous sibling is an image
		prevElement := noscript.Prev()
		if prevElement.Length() == 0 || !isSingleImage(prevElement) {
			return
		}

		// Get the previous image
		var prevImg *goquery.Selection
		if getNodeName(prevElement) == "IMG" {
			prevImg = prevElement
		} else {
			prevImg = prevElement.Find("img").First()
		}
		if prevImg.Length() == 0 {
			return
		}

		// Get the new image from noscript
		newImg := tempDoc.Find("img").First()
		if newImg.Length() == 0 {
			return
		}

		// Copy attributes from previous image to new image
		prevImg.Each(func(i int, img *goquery.Selection) {
			for _, attr := range img.Get(0).Attr {
				// Skip empty attributes
				if attr.Val == "" {
					continue
				}

				// Check if it's an image-related attribute
				if attr.Key == "src" || attr.Key == "srcset" || regexp.MustCompile(`\.(jpg|jpeg|png|webp)`).MatchString(attr.Val) {
					// Check if new image already has this attribute with the same value
					newVal, exists := newImg.Attr(attr.Key)
					if exists && newVal == attr.Val {
						return
					}

					// Use a new attribute name if it already exists
					attrName := attr.Key
					if _, exists := newImg.Attr(attrName); exists {
						attrName = "data-old-" + attrName
					}

					newImg.SetAttr(attrName, attr.Val)
				}
			}
		})

		// Replace the previous element with the new image
		newImgHTML, err := goquery.OuterHtml(newImg)
		if err != nil {
			return
		}
		prevElement.ReplaceWithHtml(newImgHTML)
	})
}

// removeScripts removes all script tags from the document
func (r *Readability) removeScripts() {
	// Remove all script and noscript tags
	r.doc.Find("script, noscript").Remove()
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

// cleanClasses removes class attributes except those in classesToPreserve
func (r *Readability) cleanClasses(node *goquery.Selection) {
	if node == nil || node.Length() == 0 {
		return
	}

	// Check if this node's class should be preserved
	if class, exists := node.Attr("class"); exists && class != "" {
		classesToKeep := []string{}
		classes := strings.Fields(class)
		for _, cls := range classes {
			// Check if this class is in the list to preserve
			preserve := false
			for _, preserveClass := range r.options.ClassesToPreserve {
				if cls == preserveClass {
					preserve = true
					break
				}
			}
			if preserve {
				classesToKeep = append(classesToKeep, cls)
			}
		}

		// If we have classes to keep, set them; otherwise remove the attribute
		if len(classesToKeep) > 0 {
			node.SetAttr("class", strings.Join(classesToKeep, " "))
		} else {
			node.RemoveAttr("class")
		}
	}

	// Process all child elements recursively
	node.Children().Each(func(i int, child *goquery.Selection) {
		r.cleanClasses(child)
	})
}

// isImportantLink checks if a link has text matching patterns we consider important
func (r *Readability) isImportantLink(link *goquery.Selection) bool {
	linkText := getInnerText(link, true)
	linkTextLower := strings.ToLower(linkText)
	
	// Check if this is an important link by text pattern
	return strings.Contains(linkTextLower, "more information") || 
	       strings.Contains(linkTextLower, "more info") || 
	       strings.Contains(linkTextLower, "read more") ||
	       strings.Contains(linkTextLower, "continue reading") ||
	       strings.Contains(linkTextLower, "learn more")
}

// finalCleanupFooters handles the final cleanup of footer elements from the article content
// This is needed because in some cases, the clean function in prepArticle might not 
// have removed footer elements, especially if grabArticle returned the body element
func (r *Readability) finalCleanupFooters(article *goquery.Selection) {
	if article.Get(0) == nil {
		return
	}
	
	// Check if it's a body element (for debugging)
	_ = article.Get(0).Data == "body" // isBody
	
	// Find all footers in the article
	footers := article.Find("footer")
	if r.options.Debug {
		fmt.Printf("DEBUG: Found %d footer elements in final article content\n", footers.Length())
	}
	
	// Handle footers based on options and presence
	if footers.Length() > 0 {
		if r.options.PreserveImportantLinks {
			// For preservation mode: We keep the footer content only if it has important links
			// otherwise we remove it
			footers.Each(func(i int, footer *goquery.Selection) {
				// Extract important links
				importantLinks := r.findAndExtractImportantLinks(footer)
				
				if importantLinks != nil && importantLinks.Children().Length() > 0 {
					// If we found important links, preserve the footer text but
					// add the important links to a separate container
					if r.options.Debug {
						fmt.Printf("DEBUG: Found important links in footer, preserving\n")
					}
					
					// Preserve the entire footer if it's in preservation mode
					// DON'T remove it, just add the links separately too for redundancy
					article.AppendSelection(importantLinks)
				} else if !r.options.PreserveImportantLinks {
					// No important links and not in preservation mode, remove the footer
					if r.options.Debug {
						fmt.Printf("DEBUG: Removing footer in final cleanup: %s\n", getOuterHTML(footer))
					}
					footer.Remove()
				}
			})
		} else {
			// Not in preservation mode, remove all footers
			footers.Each(func(i int, footer *goquery.Selection) {
				if r.options.Debug {
					fmt.Printf("DEBUG: Removing footer in final cleanup (preservation disabled): %s\n", getOuterHTML(footer))
				}
				footer.Remove()
			})
		}
	}
}

// findAndExtractImportantLinks extracts important links from the given node
// and returns a container with those links.
// This is a helper function that consolidates the link extraction logic.
func (r *Readability) findAndExtractImportantLinks(node *goquery.Selection) *goquery.Selection {
	if !r.options.PreserveImportantLinks {
		return nil
	}

	// Create a container for important links
	linkContainer := r.createElement("div")
	linkContainer.SetAttr("class", "readability-preserved-links")
	
	// Find links that match our patterns for important links
	node.Find("a").Each(func(j int, link *goquery.Selection) {
		if r.isImportantLink(link) {
			// Clone the link and its attributes
			linkCopy := link.Clone()
			
			// Create paragraph for the link and add it to the container
			p := r.createElement("p")
			p.AppendSelection(linkCopy)
			linkContainer.AppendSelection(p)
		}
	})
	
	// Only return the container if we found any important links
	if linkContainer.Children().Length() > 0 {
		return linkContainer
	}
	
	return nil
}

// hasImportantLinks checks if a node contains any important links
func (r *Readability) hasImportantLinks(node *goquery.Selection) bool {
	hasImportant := false
	node.Find("a").Each(func(i int, a *goquery.Selection) {
		if r.isImportantLink(a) {
			hasImportant = true
			return
		}
	})
	return hasImportant
}

// getJSONLD extracts metadata from JSON-LD objects in the document
func (r *Readability) getJSONLD() map[string]string {
	// Create an empty map to store extracted metadata
	metadata := make(map[string]string)

	// Find all script tags
	r.doc.Find("script[type='application/ld+json']").Each(func(i int, s *goquery.Selection) {
		// Skip if we already found metadata
		if len(metadata) > 0 {
			return
		}

		// Get the script content
		content := s.Text()

		// Strip CDATA markers if present
		content = regexp.MustCompile(`^\s*<!\[CDATA\[|\]\]>\s*$`).ReplaceAllString(content, "")

		// Try to parse as JSON
		// This is a simplification - in a full implementation we'd parse the JSON
		// For now, we'll just use regex to extract key values
		contextRe := regexp.MustCompile(`"@context"\s*:\s*"https?://schema\.org"`)
		if !contextRe.MatchString(content) {
			return
		}

		// Check for article type
		typeRe := regexp.MustCompile(`"@type"\s*:\s*"([^"]+)"`)
		typeMatches := typeRe.FindStringSubmatch(content)
		if len(typeMatches) < 2 || !RegexpJsonLdArticleTypes.MatchString(typeMatches[1]) {
			return
		}

		// Extract title
		titleRe := regexp.MustCompile(`"(?:name|headline)"\s*:\s*"([^"]+)"`)
		titleMatches := titleRe.FindStringSubmatch(content)
		if len(titleMatches) > 1 {
			metadata["title"] = titleMatches[1]
		}

		// Extract author
		authorRe := regexp.MustCompile(`"author"\s*:\s*{\s*"name"\s*:\s*"([^"]+)"`)
		authorMatches := authorRe.FindStringSubmatch(content)
		if len(authorMatches) > 1 {
			metadata["byline"] = authorMatches[1]
		}

		// Extract description
		descRe := regexp.MustCompile(`"description"\s*:\s*"([^"]+)"`)
		descMatches := descRe.FindStringSubmatch(content)
		if len(descMatches) > 1 {
			metadata["excerpt"] = descMatches[1]
		}

		// Extract publisher
		publisherRe := regexp.MustCompile(`"publisher"\s*:\s*{\s*"name"\s*:\s*"([^"]+)"`)
		publisherMatches := publisherRe.FindStringSubmatch(content)
		if len(publisherMatches) > 1 {
			metadata["siteName"] = publisherMatches[1]
		}

		// Extract date
		dateRe := regexp.MustCompile(`"(?:datePublished|dateCreated|dateModified)"\s*:\s*"([^"]+)"`)
		dateMatches := dateRe.FindStringSubmatch(content)
		if len(dateMatches) > 1 {
			metadata["date"] = dateMatches[1]
		}
	})

	return metadata
}

// grabArticle extracts the main content from the document
func (r *Readability) grabArticle() *goquery.Selection {
	// Attempt 1: Using the provided algorithm
	articleContent := r.grabArticleNode()
	if articleContent == nil {
		return nil
	}

	// If enabled, extract and preserve important links from elements that might be removed
	// This is an optional feature that's not part of the original Readability.js algorithm
	if r.options.PreserveImportantLinks {
		// Find links that might be important in elements likely to be removed
		importantLinksContainer := r.findAndExtractImportantLinks(r.doc.Find("footer, aside, nav, .footer"))
		
		// If we found any important links, add them to the article content
		if importantLinksContainer != nil {
			importantLinksContainer.SetAttr("id", "readability-important-links")
			articleContent.AppendSelection(importantLinksContainer)
		}
	}

	// Clean up
	r.prepArticle(articleContent)

	// Check word count and retry with different flags if needed
	textLength := len(getInnerText(articleContent, true))
	if textLength < r.options.CharThreshold {
		// Store the page HTML for reuse
		pageHTML, _ := r.doc.Find("body").Html()

		// Try again with different flags
		if r.flags&FlagStripUnlikelys != 0 {
			r.flags &= ^FlagStripUnlikelys
			r.doc.Find("body").SetHtml(pageHTML)
			articleContent = r.grabArticleNode()
			if articleContent != nil {
				r.prepArticle(articleContent)
				textLength = len(getInnerText(articleContent, true))
			}
		}

		if textLength < r.options.CharThreshold && r.flags&FlagWeightClasses != 0 {
			r.flags &= ^FlagWeightClasses
			r.doc.Find("body").SetHtml(pageHTML)
			articleContent = r.grabArticleNode()
			if articleContent != nil {
				r.prepArticle(articleContent)
				textLength = len(getInnerText(articleContent, true))
			}
		}

		if textLength < r.options.CharThreshold && r.flags&FlagCleanConditionally != 0 {
			r.flags &= ^FlagCleanConditionally
			r.doc.Find("body").SetHtml(pageHTML)
			articleContent = r.grabArticleNode()
			if articleContent != nil {
				r.prepArticle(articleContent)
				textLength = len(getInnerText(articleContent, true))
			}
		}

		// If still too short, use the body
		if textLength < r.options.CharThreshold {
			r.doc.Find("body").SetHtml(pageHTML)
			// Set articleContent to the body element
			articleContent = r.doc.Find("body")
		}
	}

	return articleContent
}

// grabArticleNode finds the main content node in the document
func (r *Readability) grabArticleNode() *goquery.Selection {
	if r.doc == nil {
		return nil
	}
	
	// Start with the document body
	body := r.doc.Find("body")
	if body.Length() == 0 {
		// Create a synthetic body with the document's content
		body = r.createElement("body")
		if body == nil || body.Length() == 0 {
			// If we can't even create a body element, try to return the document itself as a last resort
			return r.doc.Selection
		}
		
		// Make sure the document selection exists before trying to append to it
		if r.doc.Selection != nil && r.doc.Selection.Length() > 0 {
			body.AppendSelection(r.doc.Selection)
		}
		
		// Add this synthetic body to the document
		html := r.doc.Find("html")
		if html.Length() > 0 {
			html.AppendSelection(body)
		} else {
			// If there's no html element either, create that too
			html = r.createElement("html")
			if html == nil || html.Length() == 0 {
				// If we can't create an HTML element, just return the body we created
				return body
			}
			html.AppendSelection(body)
			
			// Make sure the document selection exists before trying to append to it
			if r.doc.Selection != nil && r.doc.Selection.Length() > 0 {
				r.doc.Selection.AppendSelection(html)
			}
		}
	}

	// Initialize variables
	elementsToScore := []*goquery.Selection{}
	shouldRemoveTitleHeader := true

	// First pass: node preparation and scoring
	// Start with either the html element or the body if there's no html
	var node *goquery.Selection
	html := r.doc.Find("html").First()
	if html != nil && html.Length() > 0 {
		node = html
	} else {
		node = body
	}
	
	// Safety check
	if node == nil || node.Length() == 0 {
		// Last resort - just use the document root
		node = r.doc.Selection
		if node == nil || node.Length() == 0 {
			return body // Return whatever we have at this point
		}
	}
	
	for node != nil && node.Length() > 0 {
		nodeTagName := getNodeName(node)

		// Check for HTML lang attribute
		if nodeTagName == "HTML" {
			if lang, exists := node.Attr("lang"); exists {
				r.articleDir = lang
			}
		}

		// Build match string from class and ID
		matchString := ""
		if class, exists := node.Attr("class"); exists {
			matchString += class + " "
		}
		if id, exists := node.Attr("id"); exists {
			matchString += id
		}

		// Skip hidden nodes
		if !isNodeVisible(node.Get(0)) {
			node = removeAndGetNext(node)
			continue
		}

		// Skip elements with aria-modal="true" and role="dialog"
		if ariaModal, exists := node.Attr("aria-modal"); exists && ariaModal == "true" {
			if role, exists := node.Attr("role"); exists && role == "dialog" {
				node = removeAndGetNext(node)
				continue
			}
		}

		// Check for byline and remove if found
		if r.checkByline(node, matchString) {
			node = removeAndGetNext(node)
			continue
		}

		// Remove duplicate title header
		if shouldRemoveTitleHeader && r.headerDuplicatesTitle(node) {
			shouldRemoveTitleHeader = false
			node = removeAndGetNext(node)
			continue
		}

		// Remove unlikely candidates
		if r.flags&FlagStripUnlikelys != 0 {
			if RegexpUnlikelyCandidates.MatchString(matchString) && !RegexpMaybeCandidate.MatchString(matchString) && 
			   !hasAncestorTag(node, "table", -1, nil) && !hasAncestorTag(node, "code", -1, nil) && 
			   nodeTagName != "BODY" && nodeTagName != "A" {
				node = removeAndGetNext(node)
				continue
			}

			// Check for unlikely roles
			if role, exists := node.Attr("role"); exists {
				for _, unlikelyRole := range UnlikelyRoles {
					if role == unlikelyRole {
						node = removeAndGetNext(node)
						continue
					}
				}
			}
		}

		// Remove DIV, SECTION, and HEADER nodes without content
		if (nodeTagName == "DIV" || nodeTagName == "SECTION" || nodeTagName == "HEADER" || 
			nodeTagName == "H1" || nodeTagName == "H2" || nodeTagName == "H3" || 
			nodeTagName == "H4" || nodeTagName == "H5" || nodeTagName == "H6") && 
			isElementWithoutContent(node) {
			node = removeAndGetNext(node)
			continue
		}

		// Add to elements to score
		if contains(DefaultTagsToScore, nodeTagName) {
			elementsToScore = append(elementsToScore, node)
		}

		// Turn DIVs with only non-block level content into Ps
		if nodeTagName == "DIV" {
			// Check if div is actually a paragraph
			if !hasChildBlockElement(node) {
				node = setNodeTag(node, "P")
				elementsToScore = append(elementsToScore, node)
			} else if hasSingleTagInsideElement(node, "P") && getLinkDensity(node) < 0.25 {
				// If it's a div with a single P child and no other content, replace div with the P
				pChild := node.Children().First()
				node.ReplaceWithSelection(pChild)
				node = pChild
				elementsToScore = append(elementsToScore, node)
			}
		}

		// Move to the next node
		node = getNextNode(node, false)
	}

	// Score the candidate elements
	candidates := []*NodeInfo{}
	for _, elem := range elementsToScore {
		// Skip elements with no parent
		parent := elem.Parent()
		if parent.Length() == 0 {
			continue
		}

		// Skip elements with less than 25 characters of text
		innerText := getInnerText(elem, true)
		if len(innerText) < 25 {
			continue
		}

		// Get ancestors up to 5 levels
		ancestors := getNodeAncestors(elem, 5)
		if len(ancestors) == 0 {
			continue
		}

		// Calculate content score for this element
		contentScore := 1.0                      // Base score
		contentScore += float64(getCharCount(elem, ",")) // Bonus for commas
		contentScore += math.Min(float64(len(innerText)/100.0), 3.0) // Bonus for text length

		// Initialize and score ancestors
		for level, ancestor := range ancestors {
			// Skip nodes without tag name or parent
			if getNodeName(ancestor) == "" || ancestor.Parent().Length() == 0 {
				continue
			}

			// Calculate a score divider based on level
			var scoreDivider float64
			if level == 0 {
				scoreDivider = 1.0
			} else if level == 1 {
				scoreDivider = 2.0
			} else {
				scoreDivider = float64(level) * 3.0
			}

			// Initialize node info and add to candidates if new
			found := false
			for i, c := range candidates {
				if isSameNode(c.node.Get(0), ancestor.Get(0)) {
					candidates[i].contentScore += contentScore / scoreDivider
					found = true
					break
				}
			}

			if !found {
				// Initialize score based on tag name
				scoreInitial := 0.0
				switch getNodeName(ancestor) {
				case "DIV":
					scoreInitial = 5.0
				case "PRE", "TD", "BLOCKQUOTE":
					scoreInitial = 3.0
				case "ADDRESS", "OL", "UL", "DL", "DD", "DT", "LI", "FORM":
					scoreInitial = -3.0
				case "H1", "H2", "H3", "H4", "H5", "H6", "TH":
					scoreInitial = -5.0
				}

				// Adjust for class/id weight
				if r.flags&FlagWeightClasses != 0 {
					scoreInitial += float64(getClassWeight(ancestor))
				}

				// Add the new node to candidates
				candidates = append(candidates, &NodeInfo{
					node:         ancestor,
					contentScore: scoreInitial + (contentScore / scoreDivider),
				})
			}
		}
	}

	// If no candidates found, return article with whole body
	if len(candidates) == 0 {
		return r.doc.Find("body")
	}

	// Find the top scoring candidates
	sort.Slice(candidates, func(i, j int) bool {
		// Adjust score based on link density
		scoreI := candidates[i].contentScore * (1.0 - getLinkDensity(candidates[i].node))
		scoreJ := candidates[j].contentScore * (1.0 - getLinkDensity(candidates[j].node))
		return scoreI > scoreJ
	})

	// Get the top candidate
	var topCandidate *NodeInfo
	if len(candidates) > 0 {
		topCandidate = candidates[0]
	}

	// If no top candidate, create one from the body
	if topCandidate == nil || getNodeName(topCandidate.node) == "BODY" {
		// Create a div to hold the content
		topCandidate = &NodeInfo{
			node:         r.doc.Find("body"),
			contentScore: 0,
		}
	}

	// Create a new article element
	article := r.createElement("div")
	article.SetAttr("id", "readability-content")

	// Get the siblings of the top candidate
	var siblingScoreThreshold float64
	if topCandidate.contentScore > 0 {
		siblingScoreThreshold = topCandidate.contentScore * 0.2
	} else {
		siblingScoreThreshold = 10.0
	}

	// Add the top candidate to the article
	article.AppendSelection(topCandidate.node.Clone())

	// Add any siblings that might be related content
	siblings := topCandidate.node.Parent().Children()
	siblings.Each(func(i int, sibling *goquery.Selection) {
		// Skip the node if it's the top candidate
		if isSameNode(sibling.Get(0), topCandidate.node.Get(0)) {
			return
		}

		// Calculate sibling score
		siblingScore := 0.0
		for _, candidate := range candidates {
			if isSameNode(candidate.node.Get(0), sibling.Get(0)) {
				siblingScore = candidate.contentScore
				break
			}
		}

		// Bonus for siblings with the same class
		if sibClass, exists := sibling.Attr("class"); exists {
			if topClass, exists := topCandidate.node.Attr("class"); exists && topClass != "" && sibClass == topClass {
				siblingScore += topCandidate.contentScore * 0.2
			}
		}

		// Add sibling if score is high enough or it's a paragraph with good content
		if siblingScore >= siblingScoreThreshold {
			article.AppendSelection(sibling.Clone())
		} else if getNodeName(sibling) == "P" {
			// Look for paragraphs that might be good content
			linkDensity := getLinkDensity(sibling)
			nodeContent := getInnerText(sibling, true)
			nodeLength := len(nodeContent)

			if nodeLength > 80 && linkDensity < 0.25 {
				article.AppendSelection(sibling.Clone())
			} else if nodeLength < 80 && nodeLength > 0 && linkDensity == 0 &&
				strings.Contains(nodeContent, ". ") {
				article.AppendSelection(sibling.Clone())
			}
		}
	})

	return article
}

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

	// This code was previously here to replace H1 with H2, but we'll comment it out
	// to make our cleanHeaders logic work more effectively
	// articleContent.Find("h1").Each(func(i int, h1 *goquery.Selection) {
	//	setNodeTag(h1, "h2")
	// })

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

// cleanStyles removes style attributes from elements
func (r *Readability) cleanStyles(e *goquery.Selection) {
	if e == nil || e.Length() == 0 {
		return
	}

	// Skip SVG elements
	if getNodeName(e) == "SVG" {
		return
	}

	// Remove presentational attributes
	for _, attr := range PresentationalAttributes {
		e.RemoveAttr(attr)
	}

	// Remove deprecated size attributes
	if contains(DeprecatedSizeAttributeElems, getNodeName(e)) {
		e.RemoveAttr("width")
		e.RemoveAttr("height")
	}

	// Clean styles recursively in children
	e.Children().Each(func(i int, child *goquery.Selection) {
		r.cleanStyles(child)
	})
}

// markDataTables adds a flag to tables that appear to contain data
func (r *Readability) markDataTables(root *goquery.Selection) {
	root.Find("table").Each(func(i int, table *goquery.Selection) {
		// Check for presentation role
		if role, exists := table.Attr("role"); exists && role == "presentation" {
			table.SetAttr("data-readability-table-type", "presentation")
			return
		}

		// Check for datatable attribute
		if datatable, exists := table.Attr("datatable"); exists && datatable == "0" {
			table.SetAttr("data-readability-table-type", "presentation")
			return
		}

		// Check for summary attribute
		if summary, exists := table.Attr("summary"); exists && summary != "" {
			table.SetAttr("data-readability-table-type", "data")
			return
		}

		// Check for caption
		if table.Find("caption").Length() > 0 && table.Find("caption").Text() != "" {
			table.SetAttr("data-readability-table-type", "data")
			return
		}

		// Check for data table descendants
		dataTableDescendants := []string{"col", "colgroup", "tfoot", "thead", "th"}
		for _, tag := range dataTableDescendants {
			if table.Find(tag).Length() > 0 {
				table.SetAttr("data-readability-table-type", "data")
				return
			}
		}

		// Check for nested tables (indicates layout)
		if table.Find("table").Length() > 0 {
			table.SetAttr("data-readability-table-type", "presentation")
			return
		}

		// Count rows and columns
		rows := table.Find("tr").Length()
		columns := 0
		table.Find("tr").Each(func(i int, tr *goquery.Selection) {
			rowCols := 0
			tr.Find("td").Each(func(j int, td *goquery.Selection) {
				colspan, _ := strconv.Atoi(td.AttrOr("colspan", "1"))
				rowCols += colspan
			})
			if rowCols > columns {
				columns = rowCols
			}
		})

		// Mark as data table if large enough
		if rows >= 10 || columns > 4 || rows*columns > 10 {
			table.SetAttr("data-readability-table-type", "data")
		} else {
			table.SetAttr("data-readability-table-type", "presentation")
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

// clean removes all nodes of the specified tag from the element
func (r *Readability) clean(e *goquery.Selection, tag string) {
	// Fixed the critical bug where footer elements weren't being properly removed.
	// The issue was that the footer elements were not in the article content because
	// they were outside the main content area when the article was first extracted.
	// 
	// The solution is to use a two-phase approach:
	// 1. First, look for the specified tag within the article content (as before)
	// 2. If none are found, look in the original document for orphaned elements
	//    and remove them by manipulating the HTML of our article content
	
	if r.options.Debug {
		fmt.Printf("DEBUG: Cleaning tag %s from content\n", tag)
		fmt.Printf("DEBUG: Current content before cleaning: %s\n", getOuterHTML(e))
		foundNodes := e.Find(tag)
		fmt.Printf("DEBUG: Found %d %s elements to clean within article content\n", foundNodes.Length(), tag)
	}
	
	isEmbed := tag == "object" || tag == "embed" || tag == "iframe"
	
	// Phase 1: Clean elements already in our article content
	elementsCleaned := 0
	e.Find(tag).Each(func(i int, node *goquery.Selection) {
		// For debugging
		if r.options.Debug {
			fmt.Printf("DEBUG: Found %s element to clean in article content: %s\n", tag, getOuterHTML(node))
		}
		
		// For footer, aside, and nav elements, check if we should preserve important links
		if r.options.PreserveImportantLinks && (tag == "footer" || tag == "aside" || tag == "nav") {
			// Extract important links before removing the node
			importantLinks := r.findAndExtractImportantLinks(node)
			if importantLinks != nil {
				// Append them to the parent content
				e.AppendSelection(importantLinks)
			}
		}
		
		// Skip allowed videos
		if isEmbed {
			// Check attributes for video URLs
			for _, attr := range node.Get(0).Attr {
				if r.options.AllowedVideoRegex.MatchString(attr.Val) {
					return
				}
			}

			// For object tags, also check inner HTML
			if tag == "object" {
				html, err := node.Html()
				if err == nil && r.options.AllowedVideoRegex.MatchString(html) {
					return
				}
			}
		}

		// Remove the node
		if r.options.Debug {
			fmt.Printf("DEBUG: Removing %s element\n", tag)
		}
		node.Remove()
		elementsCleaned++
		
		if r.options.Debug {
			// Verify removal
			newCount := e.Find(tag).Length()
			fmt.Printf("DEBUG: After removal, found %d %s elements remaining\n", newCount, tag)
		}
	})
	
	// Phase 2: If no elements were found/cleaned in phase 1, and we have a document to work with,
	// look for these elements in the original document and extract important links if needed
	if elementsCleaned == 0 && r.doc != nil && r.doc.Selection != nil && 
	   (tag == "footer" || tag == "aside" || tag == "nav") {
		originalElements := r.doc.Find(tag)
		if r.options.Debug {
			fmt.Printf("DEBUG: Found %d %s elements in original document\n", originalElements.Length(), tag)
		}
		
		// There are two things we need to do here:
		// 1. Extract important links if preservation is enabled
		// 2. Actually remove the elements from the article content
		
		// For important links preservation if enabled
		if r.options.PreserveImportantLinks && originalElements.Length() > 0 {
			// Extract all important links from these elements in the original document
			allImportantLinks := r.findAndExtractImportantLinks(originalElements)
			if allImportantLinks != nil && allImportantLinks.Children().Length() > 0 {
				// Since these aren't in our article content, we need to add them
				if r.options.Debug {
					fmt.Printf("DEBUG: Adding important links from original document: %s\n", 
						getOuterHTML(allImportantLinks))
				}
				e.AppendSelection(allImportantLinks)
			}
		}
		
		// For this specific test case, we need a direct approach
		// The issue is that the article content returned by grabArticle actually contains the entire body element 
		// This is because in some cases, the algorithm falls back to returning the body as a whole
		
		if r.options.Debug {
			fmt.Printf("DEBUG: Attempting to remove %s elements directly from article root\n", tag)
		}
		
		// Get the outer HTML of the current element
		articleHTML, err := goquery.OuterHtml(e)
		if err != nil {
			if r.options.Debug {
				fmt.Printf("DEBUG: Error getting article HTML: %v\n", err)
			}
			return
		}
		
		// Create a completely new document from this HTML
		tempDoc, err := goquery.NewDocumentFromReader(strings.NewReader(articleHTML))
		if err != nil {
			if r.options.Debug {
				fmt.Printf("DEBUG: Error creating temp document: %v\n", err)
			}
			return
		}
		
		// Try both: find the tag directly at the document level
		footerElements := tempDoc.Find(tag)
		if r.options.Debug {
			fmt.Printf("DEBUG: Found %d %s elements at document level\n", footerElements.Length(), tag)
		}
		
		// Remove all instances of the tag
		footerElements.Each(func(i int, element *goquery.Selection) {
			if r.options.Debug {
				eleHTML, _ := goquery.OuterHtml(element)
				fmt.Printf("DEBUG: Removing %s: %s\n", tag, eleHTML)
			}
			element.Remove()
		})
		
		// Get the new article HTML without the tag
		newHTML, err := tempDoc.Html()
		if err != nil {
			if r.options.Debug {
				fmt.Printf("DEBUG: Error getting new HTML: %v\n", err)
			}
			return
		}
		
		// Special handling for body elements in article content
		bodyElements := tempDoc.Find("body")
		if bodyElements.Length() > 0 {
			body := bodyElements.First()
			// Replace the article content with the body's content
			bodyHTML, err := body.Html()
			if err == nil {
				// Set inner HTML to use just the body contents
				e.SetHtml(bodyHTML)
			} else {
				// Fall back to the whole document if we can't extract just the body
				e.SetHtml(newHTML)
			}
		} else {
			// If there's no body, use the whole HTML
			e.SetHtml(newHTML)
		}
		
		// Verify the cleanup worked
		remainingElements := e.Find(tag)
		if r.options.Debug {
			fmt.Printf("DEBUG: After direct removal, found %d %s elements remaining\n", remainingElements.Length(), tag)
			fmt.Printf("DEBUG: New article content: %s\n", getOuterHTML(e))
		}
	}
}

// cleanMatchedNodes removes nodes that match a specific pattern
func (r *Readability) cleanMatchedNodes(e *goquery.Selection, filter func(*goquery.Selection, string) bool) {
	endOfSearchMarker := getNextNode(e, true)
	node := getNextNode(e, false)

	for node != nil && node.Length() > 0 && !isSameNode(node.Get(0), endOfSearchMarker.Get(0)) {
		matchString := ""
		if className, exists := node.Attr("class"); exists {
			matchString += className + " "
		}
		if id, exists := node.Attr("id"); exists {
			matchString += id
		}

		if filter(node, matchString) {
			node = removeAndGetNext(node)
		} else {
			node = getNextNode(node, false)
		}
	}
}

// cleanConditionally removes elements that don't look like content
func (r *Readability) cleanConditionally(e *goquery.Selection, tag string) {
	if r.flags&FlagCleanConditionally == 0 {
		return
	}

	e.Find(tag).Each(func(i int, node *goquery.Selection) {
		// Skip data tables
		if tag == "table" && node.AttrOr("data-readability-table-type", "") == "data" {
			return
		}

		// Skip elements that are inside a data table
		inDataTable := false
		node.ParentsFiltered("table").Each(func(i int, parent *goquery.Selection) {
			if parent.AttrOr("data-readability-table-type", "") == "data" {
				inDataTable = true
				return
			}
		})
		if inDataTable {
			return
		}

		// Skip elements inside code blocks
		if hasAncestorTag(node, "code", -1, nil) {
			return
		}

		// Calculate weight and content score
		weight := getClassWeight(node)
		_ = 0.0 // contentScore placeholder

		// Check if it has enough commas
		if getCharCount(node, ",") < 10 {
			// Count various element types
			p := node.Find("p").Length()
			img := node.Find("img").Length()
			li := node.Find("li").Length() - 100 // Discount list items
			input := node.Find("input").Length()

			// Count headings - get their text to total text ratio
			headingText := 0
			node.Find("h1, h2, h3, h4, h5, h6").Each(func(i int, h *goquery.Selection) {
				headingText += len(getInnerText(h, true))
			})
			totalText := len(getInnerText(node, true))
			headingDensity := 0.0
			if totalText > 0 {
				headingDensity = float64(headingText) / float64(totalText)
			}

			// Count embeds
			embedCount := 0
			node.Find("object, embed, iframe").Each(func(i int, embed *goquery.Selection) {
				// Skip allowed videos
				for _, attr := range embed.Get(0).Attr {
					if r.options.AllowedVideoRegex.MatchString(attr.Val) {
						return
					}
				}
				embedCount++
			})

			// Calculate link density
			linkDensity := getLinkDensity(node)
			contentLength := len(getInnerText(node, true))

			// Figure out if this should be removed
			isList := tag == "ul" || tag == "ol"
			hasListContent := false
			if isList {
				// Check if this is mostly a list of links or actual content
				totalText := 0
				totalLinks := 0
				node.Find("li").Each(func(i int, li *goquery.Selection) {
					text := getInnerText(li, true)
					totalText += len(text)
					
					// Count link text
					linkText := 0
					li.Find("a").Each(func(i int, a *goquery.Selection) {
						linkText += len(getInnerText(a, true))
					})
					totalLinks += linkText
				})

				// If list items are mostly links, it's probably not content
				if totalText > 0 && float64(totalLinks)/float64(totalText) < 0.5 {
					hasListContent = true
				}
			}

			// Check if this contains an important link like "More information..." that we want to preserve
			hasImportantLink := false
			node.Find("a").Each(func(i int, a *goquery.Selection) {
				linkText := getInnerText(a, true)
				// Case-insensitive check for important link patterns
				linkTextLower := strings.ToLower(linkText)
				if strings.Contains(linkTextLower, "more information") || 
				   strings.Contains(linkTextLower, "more info") || 
				   strings.Contains(linkTextLower, "read more") ||
				   strings.Contains(linkTextLower, "continue reading") {
					hasImportantLink = true
					return
				}
			})
			
			// Decision logic for removing the node
			shouldRemove := false
			if !hasImportantLink {
				shouldRemove = (img > 1 && float64(p)/float64(img) < 0.5 && !hasAncestorTag(node, "figure", 3, nil)) ||
					(!isList && li > p) ||
					(float64(input) > math.Floor(float64(p)/3)) ||
					(!isList && headingDensity < 0.9 && contentLength < 25 && (img == 0 || img > 2) && !hasAncestorTag(node, "figure", 3, nil)) ||
					(!isList && weight < 25 && linkDensity > 0.2) ||
					(weight >= 25 && linkDensity > 0.5) ||
					((embedCount == 1 && contentLength < 75) || embedCount > 1)
			}

			// Special handling for lists to keep image galleries
			if isList && shouldRemove && !hasListContent {
				// Count images in the list
				imgInList := node.Find("img").Length()
				liCount := node.Find("li").Length()

				// Allow image galleries (one image per list item)
				if imgInList == liCount {
					shouldRemove = false
				}
			}

			if shouldRemove {
				node.Remove()
			}
		}
	})
}

// cleanHeaders removes headers that don't look like content
// and also removes duplicate headers that match the article title
func (r *Readability) cleanHeaders(e *goquery.Selection) {
	// Track already seen headings by text
	seenHeadings := make(map[string]bool)
	
	// First pass - mark headers as duplicates of the article title
	titleMatches := []*goquery.Selection{}
	
	// First pass - find all headers
	e.Find("h1, h2").Each(func(i int, header *goquery.Selection) {
		// Skip headers with low class weight
		if getClassWeight(header) < 0 {
			return
		}
		
		// Get the header text
		headerText := getInnerText(header, false)
		headingTrimmed := strings.TrimSpace(headerText)
		
		// Check if this is a duplicate of the article title
		if r.headerDuplicatesTitle(header) || 
		   strings.EqualFold(headingTrimmed, strings.TrimSpace(r.articleTitle)) {
			titleMatches = append(titleMatches, header)
		}
	})
	
	// If we found title matches, keep only the first one and remove the rest
	if len(titleMatches) > 0 {
		firstMatch := titleMatches[0]
		headerText := getInnerText(firstMatch, false)
		headingTrimmed := strings.TrimSpace(headerText)
		seenHeadings[headingTrimmed] = true
		
		// Remove all other matches
		for i := 1; i < len(titleMatches); i++ {
			titleMatches[i].Remove()
		}
	}
	
	// Second pass - remove other duplicate headings by text
	e.Find("h1, h2").Each(func(i int, header *goquery.Selection) {
		// Skip headers with low class weight
		if getClassWeight(header) < 0 {
			header.Remove()
			return
		}
		
		// Get the header text
		headerText := getInnerText(header, false)
		headingTrimmed := strings.TrimSpace(headerText)
		
		// Skip if we've already processed it as a title match
		if r.headerDuplicatesTitle(header) || 
		   strings.EqualFold(headingTrimmed, strings.TrimSpace(r.articleTitle)) {
			return
		}
		
		// If we've seen this header text before, remove it
		if seenHeadings[headingTrimmed] {
			header.Remove()
		} else {
			seenHeadings[headingTrimmed] = true
		}
	})
}

// headerDuplicatesTitle checks if this node is an H1 or H2 whose content is mostly the same as the article title
func (r *Readability) headerDuplicatesTitle(node *goquery.Selection) bool {
	if getNodeName(node) != "H1" && getNodeName(node) != "H2" {
		return false
	}

	heading := getInnerText(node, false)
	if heading == "" || r.articleTitle == "" {
		return false
	}

	// First, check for exact match (case-insensitive)
	headingTrimmed := strings.TrimSpace(heading)
	titleTrimmed := strings.TrimSpace(r.articleTitle)
	if strings.EqualFold(headingTrimmed, titleTrimmed) {
		return true
	}

	// Check for similarity if the strings are not identical
	if headingTrimmed != titleTrimmed {
		// If not an exact match, check for similarity
		similarity := textSimilarity(titleTrimmed, headingTrimmed)
		return similarity > 0.75
	}

	return false
}

// checkByline checks if a node is a byline
func (r *Readability) checkByline(node *goquery.Selection, matchString string) bool {
	if r.articleByline != "" {
		return false
	}

	rel, _ := node.Attr("rel")
	itemprop, _ := node.Attr("itemprop")

	if (rel == "author" || (itemprop != "" && strings.Contains(itemprop, "author"))) ||
		RegexpByline.MatchString(matchString) {
		text := getInnerText(node, true)
		if isValidByline(text) {
			r.articleByline = text
			return true
		}
	}

	return false
}

// Helper utilities

// contains checks if a string is in a string slice
func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// isSameNode checks if two nodes are the same
func isSameNode(node1, node2 *html.Node) bool {
	if node1 == nil || node2 == nil {
		return node1 == node2
	}
	return node1 == node2
}