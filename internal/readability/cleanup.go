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

// removeElementsFromHTML removes elements from the article HTML by creating a temp document
func (r *Readability) removeElementsFromHTML(e *goquery.Selection, tag string) {
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
	// Skip data tables
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

	// Skip elements inside code blocks
	if hasAncestorTag(node, "code", -1, nil) {
		return true
	}
	
	return false
}

// shouldRemoveNode evaluates if a node should be removed during conditional cleaning
func (r *Readability) shouldRemoveNode(node *goquery.Selection, tag string) bool {
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
	metrics.liCount = node.Find("li").Length() - 100 // Discount list items
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
			linkText += len(getInnerText(a, true))
		})
		totalLinks += linkText
	})
	
	// If list items are mostly links, it's probably not content
	if totalText > 0 && float64(totalLinks)/float64(totalText) < ListLinkDensityThreshold {
		return true
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
	
	// Non-list with too many list items
	if !isList && metrics.liCount > metrics.paragraphCount {
		return true
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
	
	// Low weight with high link density
	if !isList && weight < ConditionalWeightThresholdLow && 
	   metrics.linkDensity > ConditionalLinkDensityThresholdLow {
		return true
	}
	
	// High weight with very high link density
	if weight >= ConditionalWeightThresholdLow && 
	   metrics.linkDensity > ConditionalLinkDensityThresholdHigh {
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
	footers := article.Find("footer")
	if r.options.Debug {
		fmt.Printf("DEBUG: Found %d footer elements in final article content\n", footers.Length())
	}
	
	// Handle footers based on options and presence
	if footers.Length() > 0 {
		if r.options.PreserveImportantLinks {
			r.cleanupFootersWithLinksPreservation(article, footers)
		} else {
			r.removeAllFooters(article, footers)
		}
	}
}

// cleanupFootersWithLinksPreservation handles footer cleanup when important links should be preserved
func (r *Readability) cleanupFootersWithLinksPreservation(article *goquery.Selection, footers *goquery.Selection) {
	// For preservation mode: Keep footer content only if it has important links
	footers.Each(func(i int, footer *goquery.Selection) {
		// Extract important links
		importantLinks := r.findAndExtractImportantLinks(footer)
		
		if importantLinks != nil && importantLinks.Children().Length() > 0 {
			// If we found important links, add them to a separate container
			if r.options.Debug {
				fmt.Printf("DEBUG: Found important links in footer, preserving\n")
			}
			
			// DON'T remove the footer, just add the links separately for redundancy
			article.AppendSelection(importantLinks)
		} else if !r.options.PreserveImportantLinks {
			// No important links and not in preservation mode, remove the footer
			if r.options.Debug {
				fmt.Printf("DEBUG: Removing footer in final cleanup: %s\n", getOuterHTML(footer))
			}
			footer.Remove()
		}
	})
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
func (r *Readability) markDataTables(root *goquery.Selection) {
	root.Find("table").Each(func(i int, table *goquery.Selection) {
		// Check for presentation indications
		if r.isTablePresentational(table) {
			table.SetAttr("data-readability-table-type", "presentation")
			return
		}

		// Check for data table indications
		if r.isTableData(table) {
			table.SetAttr("data-readability-table-type", "data")
			return
		}

		// Check table metrics
		metrics := r.calculateTableMetrics(table)
		
		// Classify by size
		if metrics.rows >= DataTableMinRows || 
		   metrics.columns > DataTableMinColumns || 
		   metrics.cells > DataTableMinCells {
			table.SetAttr("data-readability-table-type", "data")
		} else {
			table.SetAttr("data-readability-table-type", "presentation")
		}
	})
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
	
	// Check for nested tables (indicates layout)
	if table.Find("table").Length() > 0 {
		return true
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
		tr.Find("td").Each(func(j int, td *goquery.Selection) {
			colspan, _ := strconv.Atoi(td.AttrOr("colspan", "1"))
			rowCols += colspan
		})
		if rowCols > metrics.columns {
			metrics.columns = rowCols
		}
	})
	
	metrics.cells = metrics.rows * metrics.columns
	return metrics
}