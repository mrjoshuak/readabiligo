package simplifiers

import (
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Common patterns for decorative images
var decorativeImagePatterns = []string{
	"spacer", "pixel", "transparent", "blank", "dot", "separator",
	"icon", "logo", "avatar", "button", "banner", "ad", "advertisement",
	"tracking", "analytics", "pixel.gif", "1x1", "1px", "badge",
}

// IsRelevantImage determines if an image is relevant to the content
func IsRelevantImage(s *goquery.Selection) bool {
	// Check size attributes
	width, _ := strconv.Atoi(s.AttrOr("width", "0"))
	height, _ := strconv.Atoi(s.AttrOr("height", "0"))

	// Small images are likely decorative
	if width > 0 && height > 0 && width < 100 && height < 100 {
		return false
	}

	// Check for common decorative image patterns
	src := s.AttrOr("src", "")
	for _, pattern := range decorativeImagePatterns {
		if strings.Contains(strings.ToLower(src), pattern) {
			return false
		}
	}

	// Check if the image has alt text
	alt := s.AttrOr("alt", "")
	if alt != "" && !isGenericAltText(alt) {
		return true
	}

	// Check if the image is in a figure with a caption
	if s.ParentsFiltered("figure").Length() > 0 && s.ParentsFiltered("figure").Find("figcaption").Length() > 0 {
		return true
	}

	// Check if the image is in a content area
	if s.ParentsFiltered("[id*='content'], [class*='content'], article, main, .post, .entry").Length() > 0 {
		return true
	}

	// Check if the image is a background image
	if s.AttrOr("role", "") == "presentation" || s.HasClass("bg") || s.HasClass("background") ||
		strings.Contains(s.AttrOr("style", ""), "background-image") {
		return false
	}

	// Default to true for images that don't match any of the above criteria
	return true
}

// isGenericAltText checks if the alt text is generic and not descriptive
func isGenericAltText(alt string) bool {
	alt = strings.ToLower(alt)
	genericPatterns := []string{
		"image", "picture", "photo", "img", "graphic", "icon", "logo",
		"button", "banner", "thumbnail", "placeholder", "spacer",
	}

	for _, pattern := range genericPatterns {
		if alt == pattern {
			return true
		}
	}

	return false
}

// ProcessImages processes images in the document
func ProcessImages(doc *goquery.Document) *goquery.Document {
	// Find all images
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		// Check if the image is relevant
		if !IsRelevantImage(s) {
			// Remove non-relevant images
			s.Remove()
		} else {
			// Enhance relevant images
			EnhanceImage(s)

			// Mark as relevant
			s.SetAttr("data-relevant-image", "true")

			// Mark parent figure if it exists
			if s.ParentsFiltered("figure").Length() > 0 {
				s.ParentsFiltered("figure").SetAttr("data-has-relevant-image", "true")
			}
		}
	})

	return doc
}

// EnhanceImages enhances images in the document (for backward compatibility with tests)
func EnhanceImages(doc *goquery.Document) *goquery.Document {
	// Find all images
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		// Check if the image is relevant
		if !IsRelevantImage(s) {
			// Remove non-relevant images
			s.Remove()
		} else {
			// Mark as relevant
			s.SetAttr("data-relevant-image", "true")

			// Add lazy loading
			s.SetAttr("loading", "lazy")

			// Add responsive class
			s.SetAttr("class", "img-fluid")

			// Mark parent figure if it exists
			if s.ParentsFiltered("figure").Length() > 0 {
				s.ParentsFiltered("figure").SetAttr("data-has-relevant-image", "true")
			}
		}
	})

	return doc
}

// EnhanceImage enhances an image with additional attributes
func EnhanceImage(s *goquery.Selection) {
	// Add loading="lazy" attribute for better performance
	s.SetAttr("loading", "lazy")

	// Add alt text if missing
	if s.AttrOr("alt", "") == "" {
		// Try to extract alt text from surrounding context
		altText := ExtractImageCaption(s)
		if altText != "" {
			s.SetAttr("alt", altText)
		}
	}

	// Add title attribute if missing
	if s.AttrOr("title", "") == "" {
		// Use alt text as title if available
		altText := s.AttrOr("alt", "")
		if altText != "" {
			s.SetAttr("title", altText)
		}
	}

	// Extract and add metadata
	metadata := ExtractImageMetadata(s)
	for key, value := range metadata {
		s.SetAttr("data-"+key, value)
	}
}

// ExtractImageCaption extracts a caption for an image
func ExtractImageCaption(s *goquery.Selection) string {
	// Check for figcaption
	if s.ParentsFiltered("figure").Length() > 0 {
		figcaption := s.ParentsFiltered("figure").Find("figcaption")
		if figcaption.Length() > 0 {
			return figcaption.Text()
		}
	}

	// Check for alt text
	alt := s.AttrOr("alt", "")
	if alt != "" && !isGenericAltText(alt) {
		return alt
	}

	// Check for title
	title := s.AttrOr("title", "")
	if title != "" {
		return title
	}

	// Check for aria-label
	ariaLabel := s.AttrOr("aria-label", "")
	if ariaLabel != "" {
		return ariaLabel
	}

	// No caption found
	return ""
}

// ExtractImageMetadata extracts metadata from an image
func ExtractImageMetadata(s *goquery.Selection) map[string]string {
	metadata := make(map[string]string)

	// Extract src
	src := s.AttrOr("src", "")
	if src != "" {
		metadata["src"] = src
	}

	// Extract dimensions
	width := s.AttrOr("width", "")
	height := s.AttrOr("height", "")
	if width != "" {
		metadata["width"] = width
	}
	if height != "" {
		metadata["height"] = height
	}

	// Extract alt text
	alt := s.AttrOr("alt", "")
	if alt != "" {
		metadata["alt"] = alt
	}

	// Extract title
	title := s.AttrOr("title", "")
	if title != "" {
		metadata["title"] = title
	}

	// Extract caption
	caption := ExtractImageCaption(s)
	if caption != "" {
		metadata["caption"] = caption
	}

	return metadata
}
