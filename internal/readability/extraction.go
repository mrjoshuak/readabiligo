package readability

import (
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

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
	
	// Get the document body, creating one if needed
	body := r.initializeDocumentBody()
	
	// Prepare nodes for scoring
	elementsToScore := r.prepareNodesForScoring(body)
	
	// Score candidate elements
	candidates := r.scoreNodes(elementsToScore)
	
	// If no candidates found, return article with whole body
	if len(candidates) == 0 {
		return r.doc.Find("body")
	}
	
	// Find the top candidate and build article
	return r.buildArticleFromCandidates(candidates)
}

// initializeDocumentBody ensures we have a valid body element to work with
func (r *Readability) initializeDocumentBody() *goquery.Selection {
	body := r.doc.Find("body")
	if body.Length() > 0 {
		return body
	}
	
	// Create a synthetic body with the document's content
	body = r.createElement("body")
	if body == nil || body.Length() == 0 {
		// If we can't create a body element, try to return the document itself as a last resort
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
	
	return body
}

// prepareNodesForScoring traverses the DOM and prepares nodes for scoring
func (r *Readability) prepareNodesForScoring(body *goquery.Selection) []*goquery.Selection {
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
			return elementsToScore // Return empty slice
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
			} else if hasSingleTagInsideElement(node, "P") && getLinkDensity(node) < ParagraphLinkDensityThreshold {
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
	
	return elementsToScore
}

// scoreNodes calculates scores for all candidate nodes
func (r *Readability) scoreNodes(elementsToScore []*goquery.Selection) []*NodeInfo {
	candidates := []*NodeInfo{}
	
	for _, elem := range elementsToScore {
		// Skip elements with no parent
		parent := elem.Parent()
		if parent.Length() == 0 {
			continue
		}

		// Skip elements with less than minimum content length
		innerText := getInnerText(elem, true)
		if len(innerText) < MinContentTextLength {
			continue
		}

		// Get ancestors up to specified level
		ancestors := getNodeAncestors(elem, AncestorLevelDepth)
		if len(ancestors) == 0 {
			continue
		}

		// Calculate content score for this element
		contentScore := BaseContentScore                                // Base score
		contentScore += float64(getCharCount(elem, ",")) * CommaBonus   // Bonus for commas
		contentScore += math.Min(float64(len(innerText))/TextLengthDivisor, MaxLengthBonus) // Bonus for text length

		// Initialize and score ancestors
		candidates = r.scoreAncestors(ancestors, candidates, contentScore)
	}
	
	return candidates
}

// scoreAncestors assigns scores to the ancestors of content elements
func (r *Readability) scoreAncestors(ancestors []*goquery.Selection, candidates []*NodeInfo, contentScore float64) []*NodeInfo {
	for level, ancestor := range ancestors {
		// Skip nodes without tag name or parent
		if getNodeName(ancestor) == "" || ancestor.Parent().Length() == 0 {
			continue
		}

		// Calculate score divider based on level
		var scoreDivider float64
		if level == 0 {
			scoreDivider = AncestorScoreDividerL0
		} else if level == 1 {
			scoreDivider = AncestorScoreDividerL1
		} else {
			scoreDivider = float64(level) * AncestorScoreDividerMultiplier
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
				scoreInitial = DivInitialScore
			case "PRE", "TD", "BLOCKQUOTE":
				scoreInitial = BlockquoteInitialScore
			case "ADDRESS", "OL", "UL", "DL", "DD", "DT", "LI", "FORM":
				scoreInitial = NegativeListInitialScore
			case "H1", "H2", "H3", "H4", "H5", "H6", "TH":
				scoreInitial = HeadingInitialScore
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
	
	return candidates
}

// buildArticleFromCandidates creates an article element from the top candidate
func (r *Readability) buildArticleFromCandidates(candidates []*NodeInfo) *goquery.Selection {
	// Sort candidates by adjusted score (accounting for link density)
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

	// Add the top candidate to the article
	article.AppendSelection(topCandidate.node.Clone())
	
	// Find and add high-quality siblings
	r.addSiblings(article, topCandidate, candidates)
	
	return article
}

// addSiblings finds and adds high-quality siblings to the article
func (r *Readability) addSiblings(article *goquery.Selection, topCandidate *NodeInfo, candidates []*NodeInfo) {
	// Calculate sibling score threshold
	var siblingScoreThreshold float64
	if topCandidate.contentScore > 0 {
		siblingScoreThreshold = topCandidate.contentScore * SiblingScoreMultiplier
	} else {
		siblingScoreThreshold = MinimumSiblingScoreThreshold
	}

	// Get siblings of the top candidate
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
				siblingScore += topCandidate.contentScore * SameClassSiblingBonus
			}
		}

		// Add sibling if score is high enough
		if siblingScore >= siblingScoreThreshold {
			article.AppendSelection(sibling.Clone())
		} else if getNodeName(sibling) == "P" {
			// Special case for paragraphs - look for paragraphs that might be good content
			r.addParagraphIfGoodContent(article, sibling)
		}
	})
}

// addParagraphIfGoodContent adds a paragraph to the article if it appears to contain good content
func (r *Readability) addParagraphIfGoodContent(article *goquery.Selection, paragraph *goquery.Selection) {
	linkDensity := getLinkDensity(paragraph)
	nodeContent := getInnerText(paragraph, true)
	nodeLength := len(nodeContent)

	// Decide whether to include this paragraph
	if nodeLength > MinParagraphLength && linkDensity < ParagraphLinkDensityThreshold {
		// Longer paragraphs with low link density are likely good content
		article.AppendSelection(paragraph.Clone())
	} else if nodeLength < MaxShortParagraphLength && nodeLength > 0 && linkDensity == 0 &&
		strings.Contains(nodeContent, ". ") {
		// Short paragraphs with no links that contain a period might be good content
		article.AppendSelection(paragraph.Clone())
	}
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
					if RegexpImageExtension.MatchString(attr.Val) {
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

		// Process the images
		r.replaceLazyLoadedImage(prevElement, tempDoc)
	})
}

// replaceLazyLoadedImage replaces a lazy-loaded image with a real one from noscript
func (r *Readability) replaceLazyLoadedImage(prevElement *goquery.Selection, tempDoc *goquery.Document) {
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
			if attr.Key == "src" || attr.Key == "srcset" || RegexpImageExtension.MatchString(attr.Val) {
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
}

// RegexpImageExtension is a regex to match image file extensions
var RegexpImageExtension = regexp.MustCompile(`\.(jpg|jpeg|png|webp)`)