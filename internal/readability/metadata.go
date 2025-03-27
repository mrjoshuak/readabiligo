package readability

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

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