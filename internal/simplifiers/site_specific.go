package simplifiers

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// SiteSpecificHandler applies site-specific rules based on URL
func SiteSpecificHandler(doc *goquery.Document, urlStr string) *goquery.Document {
	// Parse the URL to get the domain
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return doc
	}

	domain := parsedURL.Hostname()

	// Apply site-specific rules
	switch {
	case strings.Contains(domain, "github.com"):
		return applyGitHubRules(doc)
	case strings.Contains(domain, "medium.com"):
		return applyMediumRules(doc)
	case strings.Contains(domain, "wikipedia.org"):
		return applyWikipediaRules(doc)
	case strings.Contains(domain, "nytimes.com"):
		return applyNYTimesRules(doc)
	case strings.Contains(domain, "bbc.com") || strings.Contains(domain, "bbc.co.uk"):
		return applyBBCRules(doc)
	default:
		return applyGenericRules(doc)
	}
}

// applyGitHubRules applies GitHub-specific rules
func applyGitHubRules(doc *goquery.Document) *goquery.Document {
	// Remove header
	doc.Find("header").Remove()

	// Remove footer
	doc.Find("footer").Remove()

	// Remove sidebar
	doc.Find(".sidebar").Remove()

	// Focus on README content
	doc.Find("#readme").AddClass("main-content")

	// Remove non-content elements
	doc.Find(".js-header-wrapper, .js-site-header, .site-header, .js-site-footer, .site-footer").Remove()

	return doc
}

// applyMediumRules applies Medium-specific rules
func applyMediumRules(doc *goquery.Document) *goquery.Document {
	// Remove navigation
	doc.Find("nav").Remove()

	// Remove header
	doc.Find("header").Remove()

	// Remove footer
	doc.Find("footer").Remove()

	// Remove sidebar
	doc.Find(".sidebar, [data-test-id='post-sidebar']").Remove()

	// Focus on article content
	doc.Find("article").AddClass("main-content")

	// Remove non-content elements
	doc.Find("[data-test-id='post-sidebar'], [data-test-id='post-footer'], [data-test-id='post-header']").Remove()

	return doc
}

// applyWikipediaRules applies Wikipedia-specific rules
func applyWikipediaRules(doc *goquery.Document) *goquery.Document {
	// Remove navigation
	doc.Find("#mw-navigation").Remove()

	// Remove sidebar
	doc.Find("#mw-panel").Remove()

	// Remove edit links
	doc.Find(".mw-editsection").Remove()

	// Remove footer
	doc.Find("#footer").Remove()

	// Focus on content
	doc.Find("#content").AddClass("main-content")

	// Remove non-content elements
	doc.Find("#siteSub, #contentSub, #jump-to-nav, .printfooter, #catlinks").Remove()

	return doc
}

// applyNYTimesRules applies New York Times-specific rules
func applyNYTimesRules(doc *goquery.Document) *goquery.Document {
	// Remove header
	doc.Find("header").Remove()

	// Remove footer
	doc.Find("footer").Remove()

	// Remove navigation
	doc.Find("nav").Remove()

	// Remove ads
	doc.Find(".ad").Remove()

	// Remove comments
	doc.Find("#commentsContainer").Remove()

	// Focus on article content
	doc.Find("article, .article, .story, .story-body").AddClass("main-content")

	// Remove non-content elements
	doc.Find(".NYT_BELOW_MAIN_CONTENT, .NYT_ABOVE_MAIN_CONTENT, .newsletter-signup, .comments-button").Remove()

	return doc
}

// applyBBCRules applies BBC-specific rules
func applyBBCRules(doc *goquery.Document) *goquery.Document {
	// Remove header
	doc.Find("header").Remove()

	// Remove footer
	doc.Find("footer").Remove()

	// Remove navigation
	doc.Find("nav").Remove()

	// Remove ads
	doc.Find(".bbccom_slot").Remove()

	// Remove related content
	doc.Find(".related-content").Remove()

	// Focus on article content
	doc.Find("article, .story-body, .story-body__inner").AddClass("main-content")

	// Remove non-content elements
	doc.Find(".share, .share-tools, .comments_module, .correspondent-image").Remove()

	return doc
}

// applyGenericRules applies generic rules for common non-content elements
func applyGenericRules(doc *goquery.Document) *goquery.Document {
	// Remove common non-content elements
	doc.Find("header, footer, nav, aside, .sidebar, .navigation, .menu, .ad, .advertisement").Remove()

	// Remove elements with common non-content class/ID patterns
	doc.Find("[class*='nav'], [class*='menu'], [class*='sidebar'], [class*='footer'], [class*='header'], [id*='nav'], [id*='menu'], [id*='sidebar'], [id*='footer'], [id*='header']").Remove()

	// Remove social media widgets
	doc.Find(".social, .share, .sharing, [class*='social'], [class*='share']").Remove()

	// Remove comment sections
	doc.Find(".comments, #comments, [class*='comment'], [id*='comment']").Remove()

	// Remove related content
	doc.Find(".related, .recommended, [class*='related'], [class*='recommended']").Remove()

	// Focus on content
	doc.Find("article, .article, .content, .post, .entry, main, #content, #main, .main").AddClass("main-content")

	return doc
}
