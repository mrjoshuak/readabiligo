// Package readability provides a pure Go implementation of Mozilla's Readability.js
// for extracting the main content from web pages.
package readability

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// adjustForContentType modifies the extraction parameters based on the content type
func (r *Readability) adjustForContentType() {
	switch r.contentType {
	case ContentTypeReference:
		// For reference content like Wikipedia, preserve more structure
		r.flags = r.flags & ^FlagCleanConditionally // Disable conditional cleaning
		r.flags = r.flags & ^FlagStripUnlikelys     // Disable stripping of unlikely candidates
		r.options.PreserveImportantLinks = true     // Preserve important links
		r.options.CharThreshold = r.options.CharThreshold / 2 // Lower threshold to keep more content
		
		// Modify score weights to favor structured content
		// We'll implement customizations in applyContentTypeCleanup
		
		if r.options.Debug {
			fmt.Printf("DEBUG: Using reference content extraction settings (maximum structure preservation)\n")
		}

	case ContentTypeTechnical:
		// For technical content, preserve code blocks and structure
		r.options.PreserveImportantLinks = true
		r.flags = r.flags | FlagCleanConditionally  // Enable conditional cleaning
		r.flags = r.flags & ^FlagStripUnlikelys     // Disable stripping for code blocks
		
		// Add code-related classes to preserve
		r.options.ClassesToPreserve = append(
			r.options.ClassesToPreserve, 
			"code", "highlight", "syntax", "pre", "codeblock", "language-*",
		)
		
		if r.options.Debug {
			fmt.Printf("DEBUG: Using technical content extraction settings (code block preservation)\n")
		}

	case ContentTypeError:
		// For error pages, be more aggressive in cleaning
		r.flags = r.flags | FlagStripUnlikelys | FlagWeightClasses | FlagCleanConditionally
		r.options.PreserveImportantLinks = false
		r.options.CharThreshold = r.options.CharThreshold * 2 // Higher threshold to exclude more content
		
		if r.options.Debug {
			fmt.Printf("DEBUG: Using error page extraction settings (aggressive cleaning)\n")
		}

	case ContentTypeMinimal:
		// For minimal content pages, focus on core content only
		r.flags = r.flags | FlagStripUnlikelys | FlagWeightClasses | FlagCleanConditionally
		r.options.PreserveImportantLinks = false
		r.options.CharThreshold = r.options.CharThreshold * 3/2 // Higher threshold to exclude more content
		
		if r.options.Debug {
			fmt.Printf("DEBUG: Using minimal content extraction settings (focus on core content)\n")
		}

	case ContentTypeArticle:
		// Standard article extraction (default) - balanced approach
		r.flags = FlagStripUnlikelys | FlagWeightClasses | FlagCleanConditionally
		r.options.PreserveImportantLinks = false // Standard article usually doesn't need footer links
		
		if r.options.Debug {
			fmt.Printf("DEBUG: Using standard article extraction settings (balanced cleaning)\n")
		}

	default:
		// Unknown content type - use standard settings
		r.flags = FlagStripUnlikelys | FlagWeightClasses | FlagCleanConditionally
		if r.options.Debug {
			fmt.Printf("DEBUG: Using default extraction settings for unknown content type\n")
		}
	}
}

// applyContentTypeCleanup performs additional content-type specific cleanup
func (r *Readability) applyContentTypeCleanup(article *goquery.Selection) {
	switch r.contentType {
	case ContentTypeReference:
		// For reference content like Wikipedia, preserve more structure
		// Keep tables, infoboxes, and citations by default
		if r.options.Debug {
			fmt.Printf("DEBUG: Applying reference content cleanup rules (maximum structure preservation)\n")
		}
		
		// Specific cleanup for Wikipedia-like content:
		// Remove only edit links but keep all other structure
		article.Find("span.mw-editsection, a.mw-editsection").Remove()
		
		// Add class to preserve all headings
		article.Find("h1, h2, h3, h4, h5, h6").AddClass("readability-preserve")
		
		// Add class to preserve all lists
		article.Find("ul, ol").AddClass("readability-preserve")
		
		// Add class to preserve tables that look like they contain data
		article.Find("table").Each(func(_ int, s *goquery.Selection) {
			// Keep tables with structure that looks like data/infoboxes
			if s.Find("th, td").Length() > 4 || s.HasClass("infobox") || s.HasClass("wikitable") {
				s.AddClass("readability-preserve")
			}
		})
		
		// Make sure footnotes and citations are preserved
		article.Find(".references, .citation, .footnote, .reference, .cite").AddClass("readability-preserve")
		
		// Keep figures and captions
		article.Find("figure, figcaption").AddClass("readability-preserve")

	case ContentTypeTechnical:
		// For technical content, ensure code blocks are preserved
		if r.options.Debug {
			fmt.Printf("DEBUG: Applying technical content cleanup rules (code preservation)\n")
		}
		
		// Preserve code blocks and technical content
		preserveCodeElements(article)
		
		// Also preserve headings, which are important in technical content
		article.Find("h1, h2, h3, h4, h5, h6").AddClass("readability-preserve")
		
		// Add class to preserve syntax highlighting elements
		article.Find(".syntax, .language-*, [class*='language-']").AddClass("readability-preserve")
		
		// Preserve console output and command examples
		article.Find(".console, .terminal, .shell, .command").AddClass("readability-preserve")
		
		// Make sure tables that contain code or examples are preserved
		article.Find("table").Each(func(_ int, s *goquery.Selection) {
			if s.Find("code, pre, .syntax").Length() > 0 {
				s.AddClass("readability-preserve")
			}
		})

	case ContentTypeError:
		// For error pages, be more aggressive in cleaning
		if r.options.Debug {
			fmt.Printf("DEBUG: Applying error page cleanup rules (aggressive cleaning)\n")
		}
		
		// Apply very aggressive cleaning to error pages
		cleanupErrorPage(article)
		
		// Further reduce content to the core error message
		// Remove everything except the main error message and a few key elements
		article.Children().Each(func(_ int, s *goquery.Selection) {
			// Keep only if it looks like the main error container
			text := s.Text()
			if !strings.Contains(strings.ToLower(text), "error") && 
			   !strings.Contains(strings.ToLower(text), "not found") &&
			   !strings.Contains(strings.ToLower(text), "404") &&
			   !strings.Contains(strings.ToLower(text), "sorry") {
				if !s.Is("h1, h2, .error, .error-container, .not-found") {
					s.Remove()
				}
			}
		})
		
		// Only keep the most important links on error pages
		article.Find("a").Each(func(_ int, s *goquery.Selection) {
			text := strings.ToLower(s.Text())
			href, _ := s.Attr("href")
			// Only keep homepage/contact/help links
			if !strings.Contains(text, "home") && 
			   !strings.Contains(text, "contact") && 
			   !strings.Contains(text, "help") &&
			   href != "/" {
				s.Remove()
			}
		})

	case ContentTypeMinimal:
		// For minimal content pages, focus on core content only
		if r.options.Debug {
			fmt.Printf("DEBUG: Applying minimal content cleanup rules (core content focus)\n")
		}
		
		// Apply strict cleaning to minimal pages
		cleanupMinimalPage(article)
		
		// Remove anything that isn't clearly main content
		article.Find("aside, nav, footer, header, .sidebar, .related, .recommendations").Remove()
		
		// Keep only the main content area if it can be identified
		mainContainers := article.Find("main, #main, .main, article, .article, .content, #content")
		if mainContainers.Length() > 1 {
			var bestContainer *goquery.Selection
			maxScore := 0
			
			mainContainers.Each(func(_ int, s *goquery.Selection) {
				// Score the container based on content
				score := s.Find("p").Length()*3 + s.Find("h1, h2, h3").Length()*2
				
				// If this container has a better score, select it
				if score > maxScore {
					maxScore = score
					bestContainer = s
				}
			})
			
			// If we found a best container, keep only that one
			if bestContainer != nil && maxScore > 0 {
				// Clone the best container
				bestClone := bestContainer.Clone()
				
				// Remove all content
				article.Children().Remove()
				
				// Add the best container back
				article.AppendSelection(bestClone)
			}
		}

	case ContentTypeArticle:
		// Standard article cleanup - balanced approach
		if r.options.Debug {
			fmt.Printf("DEBUG: Applying standard article cleanup rules (balanced cleaning)\n")
		}
		
		// Standard articles should have moderate cleaning
		// Remove clearly non-content elements but keep article structure
		
		// Remove social sharing and comment sections
		article.Find(".share, .sharing, .social, .comments, .comment-section").Remove()
		
		// Remove author bios that appear at the end if they're in separate elements
		article.Find(".author-bio, .bio, .about-author").Remove()
		
		// Keep the main heading
		article.Find("h1").First().AddClass("readability-preserve")
		
		// Make sure bylines are preserved
		article.Find(".byline, .author, .meta").AddClass("readability-preserve")
	}
}

// preserveCodeElements ensures code blocks are preserved in technical content
func preserveCodeElements(article *goquery.Selection) {
	// Add the 'readability-preserve' class to code elements so they're not removed
	article.Find("pre, code, .syntax, .highlight, [class*='language-'], .codeblock").AddClass("readability-preserve")
	
	// Also preserve parent elements of code blocks to maintain context
	article.Find("pre, code").Each(func(_ int, s *goquery.Selection) {
		// Find parent elements of code blocks (up to 2 levels) and preserve them
		s.Parent().AddClass("readability-preserve")
		s.Parent().Parent().AddClass("readability-preserve")
	})
	
	// Look for elements that might contain code but don't have specific classes
	article.Find("div, span").Each(func(_ int, s *goquery.Selection) {
		// Check if the element's text looks like code
		text := s.Text()
		if strings.Contains(text, "function ") || 
		   strings.Contains(text, "class ") || 
		   strings.Contains(text, "var ") || 
		   strings.Contains(text, "const ") ||
		   strings.Contains(text, "import ") ||
		   strings.Contains(text, "package ") ||
		   strings.Contains(text, "def ") {
			s.AddClass("readability-preserve")
		}
	})
}

// cleanupErrorPage performs aggressive cleanup on error pages
func cleanupErrorPage(article *goquery.Selection) {
	// Remove all navigation elements found in error pages
	article.Find("nav, .nav, .navigation, .menu, .header, .footer, .sidebar").Remove()
	
	// Remove all link lists that are likely navigation
	article.Find("ul, ol").Each(func(_ int, s *goquery.Selection) {
		linkCount := s.Find("a").Length()
		if linkCount > 2 && linkCount == s.Find("li").Length() {
			s.Remove() // Remove lists that are just collections of links
		}
	})
	
	// Remove elements that often contain navigation on error pages
	article.Find("ul li a[href='/'], ul li a[href='#']").Parent().Parent().Remove()
	
	// Remove all social sharing, search, and non-essential elements
	article.Find(".share, .social, .search, .subscribe, .newsletter, .related").Remove()
	
	// Keep mainly text content and error messages
	article.Find("div").Each(func(_ int, s *goquery.Selection) {
		// If the div doesn't contain paragraphs or error messages, remove it
		if s.Find("p, h1, h2, h3, strong, em, .error, .message").Length() == 0 {
			s.Remove()
		}
	})
	
	// Remove any elements that don't contain text about errors/not found
	article.Find("*").Each(func(_ int, s *goquery.Selection) {
		// Skip essential elements
		if s.Is("body, html, head, h1, h2, p") {
			return
		}
		
		// Check if the element or its children mention errors
		text := strings.ToLower(s.Text())
		if !strings.Contains(text, "error") && 
		   !strings.Contains(text, "not found") && 
		   !strings.Contains(text, "404") &&
		   !strings.Contains(text, "sorry") {
			// Also check if it's the main error container by class
			if !s.HasClass("error") && !s.HasClass("not-found") {
				// Only remove if it doesn't have important error-related children
				if s.Find(".error, .not-found, .error-message").Length() == 0 {
					s.Remove()
				}
			}
		}
	})
}

// cleanupMinimalPage removes non-essential elements from minimal pages
func cleanupMinimalPage(article *goquery.Selection) {
	// Remove sidebars, navigation, and other non-essential elements
	article.Find("aside, nav, .sidebar, .nav, .navigation, .menu, .header, .footer").Remove()
	
	// Remove all social sharing, ads, and non-essential elements
	article.Find(".share, .sharing, .social, .ad, .advertisement, .related, .recommended").Remove()
	
	// Remove all but the most important links
	article.Find("a").Each(func(_ int, s *goquery.Selection) {
		// Keep only links that are part of the main content
		inMainContent := s.ParentsFiltered("main, article, .content, .article, #content, .main").Length() > 0
		if !inMainContent {
			s.Remove()
		}
	})
	
	// Focus on the main content area
	article.Find("form, input, button").Each(func(_ int, s *goquery.Selection) {
		// Keep only if it's in a main content area
		parents := s.ParentsFiltered("main, article, .content, .main").Length()
		if parents == 0 {
			s.Remove()
		}
	})
	
	// If there are multiple main content areas, try to identify the most important one
	mainContainers := article.Find("main, #main, .main, article, .article, .content, #content")
	if mainContainers.Length() > 1 {
		var bestContainer *goquery.Selection
		maxScore := 0
		
		mainContainers.Each(func(_ int, s *goquery.Selection) {
			// Score the container based on content
			score := s.Find("p").Length()*3 + s.Find("h1, h2, h3").Length()*2
			
			// If this container has a better score, select it
			if score > maxScore {
				maxScore = score
				bestContainer = s
			}
		})
		
		// If we found a best container, keep only that one
		if bestContainer != nil && maxScore > 0 {
			// Clone the best container
			bestClone := bestContainer.Clone()
			
			// Remove all content
			article.Children().Remove()
			
			// Add the best container back
			article.AppendSelection(bestClone)
		}
	}
}