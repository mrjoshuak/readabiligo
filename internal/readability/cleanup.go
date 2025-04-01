package readability

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

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
	
	// Phase 1: Clean elements already in our article content
	elementsCleaned := r.cleanElementsInArticle(e, tag)
	
	// Phase 2: If no elements were found/cleaned in phase 1, and we have a document to work with,
	// look for these elements in the original document and extract important links if needed
	if elementsCleaned == 0 && r.doc != nil && r.doc.Selection != nil && 
	   (tag == "footer" || tag == "aside" || tag == "nav") {
		r.cleanElementsFromOriginalDocument(e, tag)
	}
}

// cleanElementsInArticle removes elements of the specified tag that are within the article content
func (r *Readability) cleanElementsInArticle(e *goquery.Selection, tag string) int {
	isEmbed := tag == "object" || tag == "embed" || tag == "iframe"
	elementsCleaned := 0
	
	e.Find(tag).Each(func(i int, node *goquery.Selection) {
		// For debugging
		if r.options.Debug {
			fmt.Printf("DEBUG: Found %s element to clean in article content: %s\n", tag, getOuterHTML(node))
		}
		
		// For footer, aside, and nav elements, check if we should preserve important links
		if r.options.PreserveImportantLinks && (tag == "footer" || tag == "aside" || tag == "nav") {
			r.preserveImportantLinksIfNeeded(e, node)
		}
		
		// Skip allowed videos
		if isEmbed && r.isAllowedVideo(node, tag) {
			return
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
	
	return elementsCleaned
}

// isAllowedVideo checks if a video embed should be preserved
func (r *Readability) isAllowedVideo(node *goquery.Selection, tag string) bool {
	// Check attributes for video URLs
	for _, attr := range node.Get(0).Attr {
		if r.options.AllowedVideoRegex.MatchString(attr.Val) {
			return true
		}
	}

	// For object tags, also check inner HTML
	if tag == "object" {
		html, err := node.Html()
		if err == nil && r.options.AllowedVideoRegex.MatchString(html) {
			return true
		}
	}
	
	return false
}

// preserveImportantLinksIfNeeded extracts and preserves important links from a node being removed
func (r *Readability) preserveImportantLinksIfNeeded(article *goquery.Selection, node *goquery.Selection) {
	// Extract important links before removing the node
	importantLinks := r.findAndExtractImportantLinks(node)
	if importantLinks != nil {
		// Append them to the parent content
		article.AppendSelection(importantLinks)
	}
}

// cleanElementsFromOriginalDocument handles orphaned elements from the original document
func (r *Readability) cleanElementsFromOriginalDocument(e *goquery.Selection, tag string) {
	originalElements := r.doc.Find(tag)
	if r.options.Debug {
		fmt.Printf("DEBUG: Found %d %s elements in original document\n", originalElements.Length(), tag)
	}
	
	// For important links preservation if enabled
	r.preserveImportantLinksFromOriginal(e, originalElements)
	
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
	
	// Find the tag directly at the document level
	elementsToRemove := tempDoc.Find(tag)
	if r.options.Debug {
		fmt.Printf("DEBUG: Found %d %s elements at document level\n", elementsToRemove.Length(), tag)
	}
	
	// Remove all instances of the tag
	elementsToRemove.Each(func(i int, element *goquery.Selection) {
		if r.options.Debug {
			eleHTML, _ := goquery.OuterHtml(element)
			fmt.Printf("DEBUG: Removing %s: %s\n", tag, eleHTML)
		}
		element.Remove()
	})
	
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
			newHTML, _ := tempDoc.Html()
			e.SetHtml(newHTML)
		}
	} else {
		// If there's no body, use the whole HTML
		newHTML, _ := tempDoc.Html()
		e.SetHtml(newHTML)
	}
	
	// Verify the cleanup worked
	if r.options.Debug {
		remainingElements := e.Find(tag)
		fmt.Printf("DEBUG: After direct removal, found %d %s elements remaining\n", remainingElements.Length(), tag)
		fmt.Printf("DEBUG: New article content: %s\n", getOuterHTML(e))
	}
}


// preserveImportantLinksFromOriginal preserves important links from original document elements
func (r *Readability) preserveImportantLinksFromOriginal(article *goquery.Selection, elements *goquery.Selection) {
	if !r.options.PreserveImportantLinks || elements.Length() == 0 {
		return
	}
	
	// Extract all important links from these elements in the original document
	allImportantLinks := r.findAndExtractImportantLinks(elements)
	if allImportantLinks != nil && allImportantLinks.Children().Length() > 0 {
		// Since these aren't in our article content, we need to add them
		if r.options.Debug {
			fmt.Printf("DEBUG: Adding important links from original document: %s\n", 
				getOuterHTML(allImportantLinks))
		}
		article.AppendSelection(allImportantLinks)
	}
}

// removeElementsFromHTML removes elements from the article HTML
// Uses direct DOM manipulation instead of creating temporary documents
func (r *Readability) removeElementsFromHTML(e *goquery.Selection, tag string) {
	if r.options.Debug {
		fmt.Printf("DEBUG: Attempting to remove %s elements directly from article root\n", tag)
	}
	
	// Find elements that match the tag
	elementsToRemove := e.Find(tag)
	count := elementsToRemove.Length()
	
	if r.options.Debug {
		fmt.Printf("DEBUG: Found %d %s elements at root level\n", count, tag)
	}
	
	// If no direct children match, nothing to do
	if count == 0 {
		return
	}
	
	// Remove matching elements directly - more efficient than reconstructing the document
	elementsToRemove.Each(func(i int, element *goquery.Selection) {
		if r.options.Debug {
			eleHTML, _ := goquery.OuterHtml(element)
			fmt.Printf("DEBUG: Removing %s: %s\n", tag, eleHTML)
		}
		element.Remove()
	})
}


// cleanMatchedNodes removes nodes that match a specific pattern
func (r *Readability) cleanMatchedNodes(e *goquery.Selection, filter func(*goquery.Selection, string) bool) {
	endOfSearchMarker := getNextNode(e, true)
	node := getNextNode(e, false)

	// Ensure we have both valid nodes to compare
	for node != nil && node.Length() > 0 && 
		endOfSearchMarker != nil && endOfSearchMarker.Length() > 0 &&
		node.Get(0) != nil && endOfSearchMarker.Get(0) != nil &&
		!isSameNode(node.Get(0), endOfSearchMarker.Get(0)) {
		
		// Build match string from class and ID attributes
		matchString := ""
		if className, exists := node.Attr("class"); exists {
			matchString += className + " "
		}
		if id, exists := node.Attr("id"); exists {
			matchString += id
		}

		// Apply filter function to determine if node should be removed
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
		// Skip special cases
		if r.shouldSkipConditionalCleaning(node, tag) {
			return
		}

		// Evaluate if node should be removed
		if r.shouldRemoveNode(node, tag) {
			node.Remove()
		}
	})
}

// shouldSkipConditionalCleaning determines if a node should be exempt from conditional cleaning
func (r *Readability) shouldSkipConditionalCleaning(node *goquery.Selection, tag string) bool {
	// Skip data tables completely
	if tag == "table" && node.AttrOr("data-readability-table-type", "") == "data" {
		return true
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
		return true
	}
	
	// For presentation tables, skip cleaning only for content-rich ones
	// but allow navigation tables to be cleaned
	if tag == "table" && node.AttrOr("data-readability-table-type", "") == "presentation" {
		// If it's explicitly marked as navigation, don't skip cleaning
		if node.AttrOr("data-readability-table-nav", "") == "true" {
			return false
		}
		
		// For other presentation tables, check if they have meaningful content
		textLength := len(getInnerText(node, true))
		if textLength > LayoutTableTextContentThreshold {
			// Check if it's not link-heavy
			linkText := 0
			node.Find("a").Each(func(i int, a *goquery.Selection) {
				linkText += len(getInnerText(a, true))
			})
			
			if textLength > 0 && float64(linkText)/float64(textLength) < 0.5 {
				// Skip cleaning for content-rich tables
				return true
			}
		}
	}

	// Skip elements inside code blocks
	if hasAncestorTag(node, "code", -1, nil) {
		return true
	}
	
	return false
}

// shouldPreserveStructure determines if a node should be preserved for structural reasons
func (r *Readability) shouldPreserveStructure(node *goquery.Selection, tag string) bool {
	// Always preserve certain heading levels
	if node.Is("h1, h2, h3") {
		return true
	}
	
	// Preserve lists with content
	if (tag == "ul" || tag == "ol") && node.Find("li").Length() > 0 {
		// If list has more than 2 items or substantial text, preserve it
		if node.Find("li").Length() >= 3 || len(getInnerText(node, true)) > MinParagraphLength {
			return true
		}
	}
	
	// Preserve content-rich elements
	if len(getInnerText(node, true)) > MinParagraphLength*2 {
		return true
	}
	
	return false
}

// shouldRemoveNode evaluates if a node should be removed during conditional cleaning
func (r *Readability) shouldRemoveNode(node *goquery.Selection, tag string) bool {
	// Check for structure preservation first
	if r.shouldPreserveStructure(node, tag) {
		return false // Keep important structural elements
	}
	
	// Calculate weight
	weight := getClassWeight(node)
	
	// Check if it has enough commas
	if getCharCount(node, ",") >= MinCommaCount {
		return false // Keep nodes with many commas
	}
	
	// Check for important link that should be preserved
	if r.hasImportantLinks(node) {
		return false
	}
	
	// Get node metrics
	metrics := r.calculateNodeMetrics(node)
	
	// Decision logic for removing the node
	shouldRemove := r.evaluateRemovalCriteria(node, tag, weight, metrics)
	
	// Special case for image galleries in lists
	if shouldRemove && (tag == "ul" || tag == "ol") && !metrics.hasListContent {
		// Check for image gallery (one image per list item)
		if metrics.imgCount == metrics.liCount {
			return false // Keep image galleries
		}
	}
	
	return shouldRemove
}

// NodeMetrics holds metrics used to evaluate if a node should be kept or removed
type NodeMetrics struct {
	paragraphCount   int
	imgCount         int
	liCount          int
	inputCount       int
	headingDensity   float64
	linkDensity      float64
	embedCount       int
	contentLength    int
	hasListContent   bool
}

// calculateNodeMetrics computes various metrics used to evaluate node content quality
func (r *Readability) calculateNodeMetrics(node *goquery.Selection) NodeMetrics {
	metrics := NodeMetrics{}
	
	// Count various element types
	metrics.paragraphCount = node.Find("p").Length()
	metrics.imgCount = node.Find("img").Length()
	metrics.liCount = node.Find("li").Length() - 100 // Subtract 100 from list item count exactly as Mozilla does
	metrics.inputCount = node.Find("input").Length()
	
	// Count headings and their text ratio
	headingText := 0
	node.Find("h1, h2, h3, h4, h5, h6").Each(func(i int, h *goquery.Selection) {
		headingText += len(getInnerText(h, true))
	})
	totalText := len(getInnerText(node, true))
	if totalText > 0 {
		metrics.headingDensity = float64(headingText) / float64(totalText)
	}
	
	// Count embeds (excluding allowed videos)
	node.Find("object, embed, iframe").Each(func(i int, embed *goquery.Selection) {
		// Skip allowed videos
		for _, attr := range embed.Get(0).Attr {
			if r.options.AllowedVideoRegex.MatchString(attr.Val) {
				return
			}
		}
		metrics.embedCount++
	})
	
	// Calculate link density and content length
	metrics.linkDensity = getLinkDensity(node)
	metrics.contentLength = len(getInnerText(node, true))
	
	// For lists, check if it's a list of links or has actual content
	if node.Is("ul") || node.Is("ol") {
		metrics.hasListContent = r.hasNonLinkListContent(node)
	}
	
	return metrics
}

// hasNonLinkListContent determines if a list contains substantive content besides links
func (r *Readability) hasNonLinkListContent(node *goquery.Selection) bool {
	totalText := 0
	totalLinks := 0
	
	node.Find("li").Each(func(i int, li *goquery.Selection) {
		text := getInnerText(li, true)
		totalText += len(text)
		
		// Count link text
		linkText := 0
		li.Find("a").Each(func(i int, a *goquery.Selection) {
			// Skip indexterm and noteref links which are just metadata and not real links
			// This matches Mozilla's behavior which doesn't count these in link density
			if dataType, exists := a.Attr("data-type"); exists && (dataType == "indexterm" || dataType == "noteref") {
				return
			}
			
			linkText += len(getInnerText(a, true))
		})
		totalLinks += linkText
	})
	
	// If list items are mostly links, it's probably not content
	if totalText > 0 {
		linkDensity := float64(totalLinks)/float64(totalText)
		
		// Accept lists with reasonable link density
		if linkDensity < ListLinkDensityThreshold {
			return true
		}
		
		// Also accept lists with substantial content even if link-heavy
		if totalText > MinParagraphLength {
			return true
		}
	}
	
	return false
}

// evaluateRemovalCriteria determines if a node should be removed based on its metrics
func (r *Readability) evaluateRemovalCriteria(node *goquery.Selection, tag string, weight int, metrics NodeMetrics) bool {
	isList := tag == "ul" || tag == "ol"
	
	// Image-heavy content without enough paragraphs (not in a figure)
	if metrics.imgCount > 1 && float64(metrics.paragraphCount)/float64(metrics.imgCount) < 0.5 && 
	   !hasAncestorTag(node, "figure", 3, nil) {
		return true
	}
	
	// Non-list with too many list items - but be more forgiving
	if !isList && metrics.liCount > metrics.paragraphCount*2 {
		// Only remove if this isn't part of a larger content structure
		if metrics.contentLength < MinContentTextLength*2 {
			return true
		}
	}
	
	// Too many input fields
	if float64(metrics.inputCount) > math.Floor(float64(metrics.paragraphCount)/3) {
		return true
	}
	
	// Non-list with low heading density, short content, and too few/many images (not in a figure)
	if !isList && metrics.headingDensity < HeadingDensityThreshold && 
	   metrics.contentLength < MinContentTextLength && 
	   (metrics.imgCount == 0 || metrics.imgCount > 2) && 
	   !hasAncestorTag(node, "figure", 3, nil) {
		return true
	}
	
	// Low weight with high link density - but exempt lists from this check
	if !isList && weight < ConditionalWeightThresholdLow && 
	   metrics.linkDensity > ConditionalLinkDensityThresholdLow {
		return true
	}
	
	// High weight with very high link density
	if weight >= ConditionalWeightThresholdLow && 
	   metrics.linkDensity > ConditionalLinkDensityThresholdHigh &&
	   // Be more forgiving with lists, especially those with many items
	   !(isList && metrics.liCount > 4) {
		return true
	}
	
	// Embeds with little surrounding content
	if (metrics.embedCount == 1 && metrics.contentLength < MinEmbedContentLength) || 
	   metrics.embedCount > 1 {
		return true
	}
	
	return false
}

// cleanHeaders removes headers that don't look like content
// and also removes duplicate headers that match the article title
func (r *Readability) cleanHeaders(e *goquery.Selection) {
	// Track already seen headings by text
	seenHeadings := make(map[string]bool)
	
	// First pass - Find headers matching the article title
	titleMatches := r.findTitleHeaders(e)
	
	// Process title matches
	if len(titleMatches) > 0 {
		r.processTitleHeaders(titleMatches, seenHeadings)
	}
	
	// Second pass - Process remaining headers
	r.processDuplicateHeaders(e, seenHeadings)
}

// findTitleHeaders identifies headers that match the article title
func (r *Readability) findTitleHeaders(e *goquery.Selection) []*goquery.Selection {
	titleMatches := []*goquery.Selection{}
	
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
	
	return titleMatches
}

// processTitleHeaders handles headers that match the article title
func (r *Readability) processTitleHeaders(titleMatches []*goquery.Selection, seenHeadings map[string]bool) {
	firstMatch := titleMatches[0]
	headerText := getInnerText(firstMatch, false)
	headingTrimmed := strings.TrimSpace(headerText)
	seenHeadings[headingTrimmed] = true
	
	// Remove all other matches
	for i := 1; i < len(titleMatches); i++ {
		titleMatches[i].Remove()
	}
}

// processDuplicateHeaders processes remaining headers looking for duplicates
func (r *Readability) processDuplicateHeaders(e *goquery.Selection, seenHeadings map[string]bool) {
	e.Find("h1, h2, h3").Each(func(i int, header *goquery.Selection) {
		// Add special handling to preserve important headings
		if len(getInnerText(header, true)) > 0 {
			// Keep important headings unless they have negative class weight
			if getClassWeight(header) >= 0 {
				// Still track seen headings to avoid duplicates
				headerText := getInnerText(header, false)
				headingTrimmed := strings.TrimSpace(headerText)
				
				// If we've seen this header text before, remove it
				if seenHeadings[headingTrimmed] {
					header.Remove()
				} else {
					seenHeadings[headingTrimmed] = true
					return // Skip removing this heading
				}
			}
		}
		
		// Original logic for other headings
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
		return similarity > TitleSimilarityThreshold
	}

	return false
}

// finalCleanupFooters handles the final cleanup of footer elements from the article content
// This is needed because in some cases, the clean function in prepArticle might not 
// have removed footer elements, especially if grabArticle returned the body element
func (r *Readability) finalCleanupFooters(article *goquery.Selection) {
	if article.Get(0) == nil {
		return
	}
	
	// Find all footers in the article
	footers := article.Find("footer, .footer")
	if r.options.Debug {
		fmt.Printf("DEBUG: Found %d footer elements in final article content\n", footers.Length())
	}
	
	// Handle footers based on options and presence
	if footers.Length() > 0 {
		if r.options.PreserveImportantLinks {
			// First, thoroughly check for important links in each footer
			hasImportantLinks := false
			allImportantLinks := r.createElement("div")
			allImportantLinks.SetAttr("class", "readability-preserved-links-container")
			
			footers.Each(func(i int, footer *goquery.Selection) {
				// Check if this footer has any important links
				importantLinksFound := false
				
				footer.Find("a").Each(func(j int, link *goquery.Selection) {
					if r.isImportantLink(link) {
						// Clone the link and create paragraph element
						linkCopy := link.Clone()
						p := r.createElement("p")
						p.AppendSelection(linkCopy)
						allImportantLinks.AppendSelection(p)
						
						importantLinksFound = true
						hasImportantLinks = true
					}
				})
				
				// Debug logging
				if r.options.Debug && importantLinksFound {
					fmt.Printf("DEBUG: Found important links in footer element #%d\n", i)
				}
			})
			
			// If any important links were found, add them to the article
			if hasImportantLinks && allImportantLinks.Children().Length() > 0 {
				// Add a clear container for the important links
				linkContainer := r.createElement("div")
				linkContainer.SetAttr("class", "readability-preserved-links-section")
				
				// Add a heading to indicate these are important links
				heading := r.createElement("h3")
				heading.SetText("Additional Links")
				linkContainer.AppendSelection(heading)
				
				// Add the important links
				linkContainer.AppendSelection(allImportantLinks)
				article.AppendSelection(linkContainer)
				
				if r.options.Debug {
					fmt.Printf("DEBUG: Added important links section to article\n")
				}
			}
			
			// Now remove all footer elements regardless of whether they had important links
			// since we've already extracted and saved the important links
			r.removeAllFooters(article, footers)
		} else {
			// If not preserving links, just remove all footers
			r.removeAllFooters(article, footers)
		}
	}
}

// This function has been replaced with a more robust implementation in finalCleanupFooters
// Keeping as a stub for backward compatibility
func (r *Readability) cleanupFootersWithLinksPreservation(article *goquery.Selection, footers *goquery.Selection) {
	// This function is now a no-op - all functionality is in finalCleanupFooters
	if r.options.Debug {
		fmt.Printf("DEBUG: cleanupFootersWithLinksPreservation is deprecated, using finalCleanupFooters\n")
	}
}

// removeAllFooters removes all footer elements from the article
func (r *Readability) removeAllFooters(article *goquery.Selection, footers *goquery.Selection) {
	// Not in preservation mode, remove all footers
	footers.Each(func(i int, footer *goquery.Selection) {
		if r.options.Debug {
			fmt.Printf("DEBUG: Removing footer in final cleanup (preservation disabled): %s\n", getOuterHTML(footer))
		}
		footer.Remove()
	})
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

// preserveImportantLinksAnywhere finds and preserves important links anywhere in the article content
func (r *Readability) preserveImportantLinksAnywhere(article *goquery.Selection) {
	if !r.options.PreserveImportantLinks {
		return
	}
	
	// Create a container for important links we find
	allImportantLinks := r.createElement("div")
	allImportantLinks.SetAttr("class", "readability-preserved-links-from-anywhere")
	
	// Find any links in the article that match our important link patterns
	foundLinks := false
	
	// First check for elements with related-links class or similar patterns
	relatedLinkContainers := article.Find("div.related-links, div.related-articles, div.more-links, ul.related-links, .more-reading")
	relatedLinkContainers.Each(func(i int, container *goquery.Selection) {
		container.Find("a").Each(func(j int, link *goquery.Selection) {
			// Clone this link and add it to our collection
			linkCopy := link.Clone()
			p := r.createElement("p")
			p.AppendSelection(linkCopy)
			allImportantLinks.AppendSelection(p)
			foundLinks = true
		})
	})
	
	// Then check for important links by text pattern
	article.Find("a").Each(func(i int, link *goquery.Selection) {
		if r.isImportantLink(link) {
			// Don't add duplicates
			linkHref, hasHref := link.Attr("href")
			if !hasHref {
				return
			}
			
			// Check if we already have this link by examining hrefs
			isDuplicate := false
			allImportantLinks.Find("a").Each(func(j int, existingLink *goquery.Selection) {
				existingHref, _ := existingLink.Attr("href")
				if existingHref == linkHref {
					isDuplicate = true
					return
				}
			})
			
			if !isDuplicate {
				// Clone this link and add it to our collection
				linkCopy := link.Clone()
				p := r.createElement("p")
				p.AppendSelection(linkCopy)
				allImportantLinks.AppendSelection(p)
				foundLinks = true
			}
		}
	})
	
	// If we found important links, add them to the article
	if foundLinks {
		// Create a clear container for these links
		linkContainer := r.createElement("div")
		linkContainer.SetAttr("class", "readability-important-links-section")
		
		// Add a heading
		heading := r.createElement("h3")
		heading.SetText("Important Links")
		linkContainer.AppendSelection(heading)
		
		// Add the important links
		linkContainer.AppendSelection(allImportantLinks)
		
		// Append to the article
		article.AppendSelection(linkContainer)
		
		if r.options.Debug {
			fmt.Printf("DEBUG: Added important links section from article content\n")
		}
	}
}

// isImportantLink checks if a link has text matching patterns we consider important
func (r *Readability) isImportantLink(link *goquery.Selection) bool {
	linkText := getInnerText(link, true)
	linkTextLower := strings.ToLower(linkText)
	
	// List of important link patterns
	importantPatterns := []string{
		"more information",
		"more info",
		"read more",
		"continue reading",
		"learn more", 
		"see more",
		"view more",
		"read full",
		"full article",
		"full story",
		"continue",
		"click for more",
		"view article",
		"continue reading",
		"see also",
		"related article",
		"more on this",
	}
	
	// Check if this is an important link by text pattern
	for _, pattern := range importantPatterns {
		if strings.Contains(linkTextLower, pattern) {
			return true
		}
	}
	
	// Also check for ellipsis pattern "..." which often indicates more content
	if strings.Contains(linkTextLower, "...") && len(linkTextLower) < 30 {
		return true
	}
	
	// Check for "more" as a standalone word or at the end of text
	if linkTextLower == "more" || strings.HasSuffix(linkTextLower, " more") {
		return true
	}
	
	return false
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

// markDataTables adds a flag to tables that appear to contain data
// and handles nested table structures
func (r *Readability) markDataTables(root *goquery.Selection) {
	// First, find the deepest nested tables and work outward
	tablesByNestingLevel := r.groupTablesByNestingLevel(root)
	
	// Process tables from deepest nesting level first
	for level := len(tablesByNestingLevel) - 1; level >= 0; level-- {
		tablesAtLevel := tablesByNestingLevel[level]
		for _, table := range tablesAtLevel {
			r.processAndClassifyTable(table, level)
		}
	}
	
	// After classification, flatten layout tables with excessive nesting
	r.flattenNestedLayoutTables(root)
}

// groupTablesByNestingLevel organizes tables by their nesting depth
func (r *Readability) groupTablesByNestingLevel(root *goquery.Selection) [][]*goquery.Selection {
	// Map to track tables by nesting level
	tablesByLevel := make(map[int][]*goquery.Selection)
	maxLevel := 0
	
	// Helper function to measure nesting level
	var calculateNestingLevel func(*goquery.Selection) int
	calculateNestingLevel = func(node *goquery.Selection) int {
		// Count how many ancestor tables this table has
		level := 0
		parent := node.Parent()
		for parent.Length() > 0 {
			if getNodeName(parent) == "TABLE" {
				level++
			}
			parent = parent.Parent()
		}
		return level
	}
	
	// Process all tables
	root.Find("table").Each(func(i int, table *goquery.Selection) {
		level := calculateNestingLevel(table)
		if level > maxLevel {
			maxLevel = level
		}
		
		if tablesByLevel[level] == nil {
			tablesByLevel[level] = make([]*goquery.Selection, 0)
		}
		tablesByLevel[level] = append(tablesByLevel[level], table)
	})
	
	// Convert map to slice of slices
	result := make([][]*goquery.Selection, maxLevel+1)
	for level := 0; level <= maxLevel; level++ {
		result[level] = tablesByLevel[level]
	}
	
	return result
}

// processAndClassifyTable analyzes a table and classifies it as data or layout
func (r *Readability) processAndClassifyTable(table *goquery.Selection, nestingLevel int) {
	// First check explicit indicators
	// Check for presentation indications
	if r.isTablePresentational(table) {
		table.SetAttr("data-readability-table-type", "presentation")
		
		// Flag navigation tables if they appear to be navigation
		if r.isNavigationTable(table) {
			table.SetAttr("data-readability-table-nav", "true")
		}
		return
	}

	// Check for data table indications
	if r.isTableData(table) {
		table.SetAttr("data-readability-table-type", "data")
		return
	}
	
	// Enhanced analysis for tables without clear indicators
	
	// Get table metrics
	metrics := r.calculateTableMetrics(table)
	
	// Get link density for navigation detection
	linkDensity := r.calculateTableLinkDensity(table)
	
	// Check if this table appears to be navigation
	if linkDensity > NavigationLinkDensityThreshold {
		table.SetAttr("data-readability-table-type", "presentation")
		table.SetAttr("data-readability-table-nav", "true")
		return
	}
	
	// Check if this is likely a layout table due to excessive nesting
	if nestingLevel > LayoutTableNestingThreshold {
		table.SetAttr("data-readability-table-type", "presentation")
		return
	}
	
	// Check for size indicators of data tables
	if metrics.rows >= DataTableMinRows || 
	   metrics.columns > DataTableMinColumns || 
	   metrics.cells > DataTableMinCells {
		table.SetAttr("data-readability-table-type", "data")
	} else {
		// Check for meaningful content that might indicate it's not just layout
		textLength := len(getInnerText(table, true))
		linkTextLength := 0
		table.Find("a").Each(func(i int, a *goquery.Selection) {
			linkTextLength += len(getInnerText(a, true))
		})
		nonLinkTextLength := textLength - linkTextLength
		
		if nonLinkTextLength > LayoutTableTextContentThreshold && linkDensity < 0.3 {
			// Table with significant non-link text is likely content
			table.SetAttr("data-readability-table-type", "data")
		} else {
			// Default to presentation
			table.SetAttr("data-readability-table-type", "presentation")
		}
	}
}

// flattenNestedLayoutTables simplifies nested presentation tables
func (r *Readability) flattenNestedLayoutTables(root *goquery.Selection) {
	// Find deeply nested presentation tables
	root.Find("table[data-readability-table-type='presentation'] table[data-readability-table-type='presentation']").Each(func(i int, nestedTable *goquery.Selection) {
		// Skip if this table has already been processed or removed
		if nestedTable.Nodes == nil || len(nestedTable.Nodes) == 0 {
			return
		}
		
		// Skip data tables completely
		parentTable := nestedTable.ParentsFiltered("table").First()
		if parentTable.AttrOr("data-readability-table-type", "") == "data" {
			return
		}
		
		// Create a div to replace the nested table
		replacement := r.createElement("div")
		replacement.SetAttr("class", "readability-flattened-table")
		
		// For navigation tables, we'll be more aggressive in simplification
		if nestedTable.AttrOr("data-readability-table-nav", "") == "true" {
			// For navigation tables, only preserve the links
			nestedTable.Find("a").Each(func(j int, link *goquery.Selection) {
				// Only keep links with text
				if len(strings.TrimSpace(link.Text())) > 0 {
					linkCopy := link.Clone()
					div := r.createElement("div")
					div.AppendSelection(linkCopy)
					replacement.AppendSelection(div)
				}
			})
		} else {
			// For regular layout tables, preserve more structure
			nestedTable.Find("tr").Each(func(j int, row *goquery.Selection) {
				// Create a div for each row
				rowDiv := r.createElement("div")
				rowDiv.SetAttr("class", "readability-table-row")
				
				// Process cells in the row
				row.Find("td").Each(func(k int, cell *goquery.Selection) {
					// Create a div for each cell
					cellDiv := r.createElement("div")
					cellDiv.SetAttr("class", "readability-table-cell")
					
					// Copy content from the cell to the div
					cellHtml, err := cell.Html()
					if err == nil {
						cellDiv.SetHtml(cellHtml)
					}
					
					rowDiv.AppendSelection(cellDiv)
				})
				
				// Only add non-empty rows
				if rowDiv.Children().Length() > 0 {
					replacement.AppendSelection(rowDiv)
				}
			})
		}
		
		// Only replace if we have content in our replacement
		if replacement.Children().Length() > 0 {
			nestedTable.ReplaceWithSelection(replacement)
		}
	})
	
	// Handle single-cell tables and table cells with single content elements
	root.Find("table[data-readability-table-type='presentation']").Each(func(i int, table *goquery.Selection) {
		// Skip if already processed or removed
		if table.Nodes == nil || len(table.Nodes) == 0 {
			return
		}
		
		// Check for single-row, single-column table structure
		rows := table.Find("tr")
		if rows.Length() == 1 {
			cells := rows.First().Find("td")
			if cells.Length() == 1 {
				cell := cells.First()
				
				// Replace with a div or paragraph based on content
				if everyNode(cell.Contents(), func(i int, s *goquery.Selection) bool {
					return s.Get(0) != nil && isPhrasingContent(s.Get(0))
				}) {
					// If the cell contains only phrasing content, replace with a paragraph
					replacement := setNodeTag(cell.Clone(), "p")
					table.ReplaceWithSelection(replacement)
				} else {
					// Otherwise, replace with a div
					replacement := setNodeTag(cell.Clone(), "div")
					table.ReplaceWithSelection(replacement)
				}
			}
		}
	})
}

// isNavigationTable checks if a table appears to be used for navigation
func (r *Readability) isNavigationTable(table *goquery.Selection) bool {
	// Navigation tables typically have high link density
	linkDensity := r.calculateTableLinkDensity(table)
	if linkDensity > NavigationLinkDensityThreshold {
		return true
	}
	
	// Navigation tables often have nav-related classes
	class, _ := table.Attr("class")
	id, _ := table.Attr("id")
	combined := strings.ToLower(class + " " + id)
	
	// Check for navigation indicators in class or id
	navigationPatterns := []string{"nav", "menu", "header", "sidebar", "topbar"}
	for _, pattern := range navigationPatterns {
		if strings.Contains(combined, pattern) {
			return true
		}
	}
	
	// Check for list-like structure with mostly links
	liCount := table.Find("li").Length()
	if liCount > 3 && 
	   table.Find("a").Length() >= int(float64(liCount) * 0.8) {
		return true
	}
	
	return false
}

// calculateTableLinkDensity measures the proportion of link text in a table
func (r *Readability) calculateTableLinkDensity(table *goquery.Selection) float64 {
	totalText := len(getInnerText(table, true))
	if totalText == 0 {
		return 0
	}
	
	linkText := 0
	table.Find("a").Each(func(i int, a *goquery.Selection) {
		linkText += len(getInnerText(a, true))
	})
	
	return float64(linkText) / float64(totalText)
}

// isTablePresentational checks if a table is for layout rather than data
func (r *Readability) isTablePresentational(table *goquery.Selection) bool {
	// Check for presentation role
	if role, exists := table.Attr("role"); exists && role == "presentation" {
		return true
	}

	// Check for datatable attribute
	if datatable, exists := table.Attr("datatable"); exists && datatable == "0" {
		return true
	}
	
	// Layout tables often have specific attributes
	width, hasWidth := table.Attr("width")
	if hasWidth && width == "100%" {
		// Check for no borders, which is common in layout tables
		border, hasBorder := table.Attr("border")
		if !hasBorder || border == "0" {
			cellspacing, hasCellspacing := table.Attr("cellspacing")
			if !hasCellspacing || cellspacing == "0" {
				return true
			}
		}
	}
	
	// Layout tables often have specific class/id names
	class, _ := table.Attr("class")
	id, _ := table.Attr("id")
	combined := strings.ToLower(class + " " + id)
	
	layoutPatterns := []string{"layout", "grid", "wrapper", "container", "outer", "inner"}
	for _, pattern := range layoutPatterns {
		if strings.Contains(combined, pattern) {
			return true
		}
	}
	
	return false
}

// isTableData checks if a table likely contains data
func (r *Readability) isTableData(table *goquery.Selection) bool {
	// Check for summary attribute
	if summary, exists := table.Attr("summary"); exists && summary != "" {
		return true
	}

	// Check for caption
	if table.Find("caption").Length() > 0 && table.Find("caption").Text() != "" {
		return true
	}

	// Check for data table descendants
	dataTableDescendants := []string{"col", "colgroup", "tfoot", "thead", "th"}
	for _, tag := range dataTableDescendants {
		if table.Find(tag).Length() > 0 {
			return true
		}
	}
	
	// Data tables often have specific class/id names
	class, _ := table.Attr("class")
	id, _ := table.Attr("id")
	combined := strings.ToLower(class + " " + id)
	
	dataPatterns := []string{"data", "stats", "statistics", "results", "info"}
	for _, pattern := range dataPatterns {
		if strings.Contains(combined, pattern) {
			return true
		}
	}
	
	// Check for consistent cell structure which is common in data tables
	if r.hasConsistentCellStructure(table) {
		return true
	}
	
	return false
}

// hasConsistentCellStructure checks if a table has uniform cells typical of data tables
func (r *Readability) hasConsistentCellStructure(table *goquery.Selection) bool {
	rows := table.Find("tr")
	if rows.Length() < 2 {
		return false
	}
	
	// Check if all rows have the same number of cells
	cellCounts := []int{}
	rows.Each(func(i int, row *goquery.Selection) {
		cellCounts = append(cellCounts, row.Find("td, th").Length())
	})
	
	// If all rows have the same cell count and it's > 1, likely a data table
	if len(cellCounts) > 0 {
		firstCount := cellCounts[0]
		if firstCount > 1 {
			allSame := true
			for _, count := range cellCounts {
				if count != firstCount {
					allSame = false
					break
				}
			}
			if allSame {
				return true
			}
		}
	}
	
	return false
}

// TableMetrics holds metrics for determining if a table is a data table
type TableMetrics struct {
	rows    int
	columns int
	cells   int
}

// calculateTableMetrics counts rows, columns, and cells in a table
func (r *Readability) calculateTableMetrics(table *goquery.Selection) TableMetrics {
	metrics := TableMetrics{
		rows: table.Find("tr").Length(),
	}
	
	// Count columns by finding the row with the most cells
	table.Find("tr").Each(func(i int, tr *goquery.Selection) {
		rowCols := 0
		tr.Find("td, th").Each(func(j int, td *goquery.Selection) {
			colspan, _ := strconv.Atoi(td.AttrOr("colspan", "1"))
			rowCols += colspan
		})
		if rowCols > metrics.columns {
			metrics.columns = rowCols
		}
	})
	
	// Get actual cell count for greater accuracy
	metrics.cells = table.Find("td, th").Length()
	
	return metrics
}