package extractors

import (
	"sort"
	"strings"

	"github.com/antchfx/htmlquery"
	"github.com/mrjoshuak/readabiligo/internal/simplifiers"
	"golang.org/x/net/html"
)

// XPathScore represents an XPath query with a confidence score
type XPathScore struct {
	XPath string
	Score int
}

// ExtractedElement represents an extracted element with its score and the XPaths used to find it
type ExtractedElement struct {
	Score  int
	XPaths []string
}

// ProcessDictFunc is a function that processes the extracted elements dictionary
type ProcessDictFunc func(map[string]*ExtractedElement) map[string]*ExtractedElement

// ExtractElement extracts elements from HTML using a list of XPath queries with confidence scores
func ExtractElement(htmlContent string, xpaths []XPathScore, processDictFn ProcessDictFunc) map[string]*ExtractedElement {
	// Parse the HTML
	doc, err := htmlquery.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil
	}

	// Extract elements using XPath queries
	extractedStrings := make(map[string]*ExtractedElement)
	for _, xpathScore := range xpaths {
		nodes, err := htmlquery.QueryAll(doc, xpathScore.XPath)
		if err != nil {
			continue
		}

		for _, node := range nodes {
			var element string
			// Check if this is an attribute node by looking at the XPath
			if strings.Contains(xpathScore.XPath, "@") && node.Type == html.ElementNode {
				// For attribute nodes, we need to extract the attribute value
				// The attribute name is in the XPath after the @
				attrName := xpathScore.XPath[strings.LastIndex(xpathScore.XPath, "@")+1:]
				if strings.Contains(attrName, "]") {
					attrName = attrName[:strings.Index(attrName, "]")]
				}
				for _, attr := range node.Attr {
					if attr.Key == attrName {
						element = attr.Val
						break
					}
				}
			} else {
				element = htmlquery.InnerText(node)
			}

			// Normalize whitespace
			element = simplifiers.NormalizeWhitespace(element)
			if element == "" {
				continue
			}

			// Add or update the element in the map
			if _, exists := extractedStrings[element]; !exists {
				extractedStrings[element] = &ExtractedElement{
					Score:  xpathScore.Score,
					XPaths: []string{xpathScore.XPath},
				}
			} else {
				extractedStrings[element].Score += xpathScore.Score
				extractedStrings[element].XPaths = append(extractedStrings[element].XPaths, xpathScore.XPath)
				sort.Strings(extractedStrings[element].XPaths)
			}
		}
	}

	// Process the dictionary if a processing function is provided
	if processDictFn != nil {
		extractedStrings = processDictFn(extractedStrings)
	}

	return extractedStrings
}
