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
		
	case ContentTypePaywall:
		// For paywall content, extract all available content and ignore the paywall
		r.flags = r.flags & ^FlagCleanConditionally // Disable conditional cleaning for max content
		r.flags = r.flags | FlagStripUnlikelys | FlagWeightClasses // Still strip unlikely elements
		r.options.PreserveImportantLinks = true // Keep important links that might bypass the paywall
		r.options.CharThreshold = r.options.CharThreshold / 4 // Very low threshold to extract everything
		
		// Preserve premium content markers
		r.options.ClassesToPreserve = append(
			r.options.ClassesToPreserve,
			"premium-content", "paid-content", "subscriber-content",
		)
		
		if r.options.Debug {
			fmt.Printf("DEBUG: Using paywall content extraction settings (maximum content extraction)\n")
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
		
	case ContentTypePaywall:
		// For paywall content, apply special handling to extract everything
		if r.options.Debug {
			fmt.Printf("DEBUG: Applying paywall content cleanup rules (premium content extraction)\n")
		}
		
		// Apply paywall-specific cleanup
		cleanupPaywallContent(article)

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
	// Enhanced cleanup for minimal content pages
	if article == nil || article.Length() == 0 {
		return
	}
	
	// First, try to identify the main login/form container
	formContainer := findMainFormContainer(article)
	
	if formContainer != nil && formContainer.Length() > 0 {
		// We found the main form container, focus only on it and a limited context
		handleFormContainer(article, formContainer)
	} else {
		// No form container found, clean up based on general rules for minimal pages
		performGeneralMinimalPageCleanup(article)
	}
	
	// Final cleanup specific to minimal content:
	// Remove all external links and non-essential elements
	removeNonEssentialElements(article)
}

// findMainFormContainer identifies the primary login/form container in a minimal page
func findMainFormContainer(article *goquery.Selection) *goquery.Selection {
	// Look for authentication containers by common class names and IDs
	authContainerSelectors := []string{
		".login-container", ".login-form", ".auth-container", "#login", "#login-form", 
		".signup-container", ".signup-form", "#signup", "#signup-form",
		".register-container", ".registration-form", "#register", "#registration",
		".auth-form", ".authentication", "#auth", "#authentication",
		"form.login", "form.signup", "form.register", ".form-container",
	}
	
	// Try to find the main container using common selectors
	for _, selector := range authContainerSelectors {
		container := article.Find(selector)
		if container.Length() > 0 {
			return container
		}
	}
	
	// Look for forms with password fields (likely login/signup forms)
	formWithPassword := article.Find("form:has(input[type='password'])")
	if formWithPassword.Length() > 0 {
		return formWithPassword
	}
	
	// Look for restricted access messages with nearby forms
	restrictedMsgSelectors := []string{
		":contains('must be logged in')", ":contains('login required')", 
		":contains('please log in')", ":contains('sign in to')",
		":contains('members only')", ":contains('restricted access')",
		":contains('please sign in')", ":contains('login to continue')",
	}
	
	for _, msgSelector := range restrictedMsgSelectors {
		msgElement := article.Find(msgSelector)
		if msgElement.Length() > 0 {
			// Look for nearby forms
			// Check parent elements up to 3 levels up
			parent := msgElement.Parent()
			for i := 0; i < 3 && parent.Length() > 0; i++ {
				if parent.Find("form").Length() > 0 {
					return parent
				}
				parent = parent.Parent()
			}
			
			// Check siblings for form elements
			siblings := msgElement.Siblings()
			formSibling := siblings.Filter("form")
			// Also look for containers
			containerSiblings := siblings.Filter(".form-container, .login-container")
			
			if formSibling.Length() > 0 {
				return formSibling
			} else if containerSiblings.Length() > 0 {
				return containerSiblings
			}
		}
	}
	
	// Fall back to any form with standard login/signup input fields
	authForm := article.Find("form:has(input#username), form:has(input#email), form:has(input[name='username']), form:has(input[name='email'])")
	if authForm.Length() > 0 {
		return authForm
	}
	
	// No specific form container found
	return nil
}

// handleFormContainer focuses extraction on a form container and its context
func handleFormContainer(article *goquery.Selection, formContainer *goquery.Selection) {
	// Identify the contextual container - could be the form's parent or a nearby container
	contextContainer := formContainer
	
	// Try to include the parent container if it seems to provide context
	parent := formContainer.Parent()
	if parent.Length() > 0 && parent.Find("h1, h2, h3, .heading, .title").Length() > 0 {
		contextContainer = parent
	}
	
	// Look for a form title/heading to include
	formTitle := article.Find("h1:contains('Login'), h1:contains('Sign In'), h2:contains('Login'), h2:contains('Sign In'), h1:contains('Register'), h2:contains('Register')")
	
	// If title is outside our container but nearby, try to include its container
	if formTitle.Length() > 0 {
		// Check if the title is not already inside our context container
		titleInside := false
		formTitle.Each(func(i int, title *goquery.Selection) {
			titleParents := title.Parents()
			titleParents.Each(func(j int, parent *goquery.Selection) {
				if isSameNode(parent.Get(0), contextContainer.Get(0)) {
					titleInside = true
					return
				}
			})
		})
		
		if !titleInside {
			// Check if it's a sibling or nearby element
			titleParent := formTitle.Parent()
			if titleParent.Length() > 0 {
				// Create a container for both elements
				newContainer := getEmptyDiv()
				newContainer.AppendSelection(titleParent.Clone())
				newContainer.AppendSelection(contextContainer.Clone())
				contextContainer = newContainer
			}
		}
	}
	
	// Include messages that might be outside the form but relevant (e.g., error messages)
	messageSelectors := []string{
		".message", ".alert", ".error-message", ".success-message", 
		".notification", ".info-message", ".validation-message",
	}
	
	for _, selector := range messageSelectors {
		messages := article.Find(selector)
		messages.Each(func(i int, message *goquery.Selection) {
			// Check if the message is already inside our container
			messageInside := false
			messageParents := message.Parents()
			messageParents.Each(func(j int, parent *goquery.Selection) {
				if parent.Length() > 0 && contextContainer.Length() > 0 {
					if parent.Get(0) != nil && contextContainer.Get(0) != nil {
						if isSameNode(parent.Get(0), contextContainer.Get(0)) {
							messageInside = true
							return
						}
					}
				}
			})
			
			if !messageInside {
				// Add relevant messages only if they're near the form
				messageTxt := strings.ToLower(message.Text())
				if strings.Contains(messageTxt, "password") || 
				   strings.Contains(messageTxt, "login") || 
				   strings.Contains(messageTxt, "sign in") || 
				   strings.Contains(messageTxt, "username") ||
				   strings.Contains(messageTxt, "account") {
					// Clone the message and add it to our container
					contextContainer.AppendSelection(message.Clone())
				}
			}
		})
	}
	
	// Replace article content with just the context container
	articleHtml, _ := contextContainer.Html()
	article.SetHtml(articleHtml)
}

// performGeneralMinimalPageCleanup cleans minimal pages when no specific form is found
func performGeneralMinimalPageCleanup(article *goquery.Selection) {
	// Remove all obviously non-content elements
	article.Find("aside, nav, .sidebar, .nav, .navigation, .menu, .header, .footer").Remove()
	article.Find(".share, .sharing, .social, .ad, .advertisement, .related, .recommended").Remove()
	
	// Look for the main content container
	mainSelectors := []string{
		"main", "#main", ".main-content", ".content", "#content", 
		"article", ".article", ".page-content", "#page-content",
		".auth-content", ".auth-container", ".minimal-content",
	}
	
	// Try to find the most relevant content container
	var bestContainer *goquery.Selection
	maxScore := 0
	
	for _, selector := range mainSelectors {
		containers := article.Find(selector)
		containers.Each(func(i int, container *goquery.Selection) {
			// Score based on content elements and forms
			score := container.Find("p").Length()*2 + 
					 container.Find("h1, h2, h3").Length()*3 +
					 container.Find("form, input[type='text'], input[type='password']").Length()*5
			
			// Check for key content indicators
			text := strings.ToLower(container.Text())
			if strings.Contains(text, "login") || 
			   strings.Contains(text, "sign in") || 
			   strings.Contains(text, "register") ||
			   strings.Contains(text, "password") {
				score += 10
			}
			
			if score > maxScore {
				maxScore = score
				bestContainer = container
			}
		})
	}
	
	// If we found a good container, focus on it
	if bestContainer != nil && bestContainer.Length() > 0 && maxScore > 5 {
		// Replace article content with just the best container
		articleHtml, _ := bestContainer.Html()
		article.SetHtml(articleHtml)
	} else {
		// Default cleanup - Remove navigation elements
		article.Find("ul li a[href='/'], ul li a[href='#']").Parent().Parent().Remove()
		article.Find("ul.menu, ul.nav, nav ul").Remove()
		
		// Keep only the most relevant blocks
		article.Children().Each(func(i int, child *goquery.Selection) {
			// Skip if it's an important element
			if child.Is("h1, h2, form, .content, .main") {
				return
			}
			
			// Check if this element has any forms or important content
			hasForm := child.Find("form, input[type='text'], input[type='password'], button[type='submit']").Length() > 0
			hasHeading := child.Find("h1, h2, h3").Length() > 0
			
			// Calculate a relevance score
			score := 0
			if hasForm {
				score += 10
			}
			if hasHeading {
				score += 5
			}
			score += child.Find("p").Length() * 2
			
			// Remove if it doesn't seem relevant
			if score < 3 {
				child.Remove()
			}
		})
	}
}

// cleanupPaywallContent performs specialized cleanup for paywall content
func cleanupPaywallContent(article *goquery.Selection) {
	// First, identify all paywall elements to be removed or bypassed
	paywallSelectors := []string{
		".paywall", "#paywall", ".paywall-container", "#paywall-container",
		".subscription-required", ".subscription-wall", ".meter-paywall",
		".metered-content", ".paid-content-container", ".paid-overlay",
		".subscriber-only", ".subscriber-overlay", ".reg-gate", ".registration-gate",
		".article-gate", ".content-gate", ".gated-content", ".article-paywall",
		".dynamic-paywall", ".reader-paywall", ".paid-content-gate",
	}
	
	// Remove the paywall container entirely to reveal the content beneath
	for _, selector := range paywallSelectors {
		article.Find(selector).Remove()
	}
	
	// Remove any blur effects or opacity filters that might be hiding content
	article.Find("[style*='blur'], [style*='opacity'], .blurred, .blur-content, .dimmed, .fade-out").Each(func(_ int, s *goquery.Selection) {
		// Replace the style with normal visibility
		s.RemoveAttr("style")
		s.RemoveClass("blurred blur-content dimmed fade-out")
		s.AddClass("readability-preserve")
	})
	
	// Preserve all paragraphs that might be part of the premium content
	article.Find(".premium-content, .paid-content, .premium, .subscriber-content, .subscribers-only").Each(func(_ int, s *goquery.Selection) {
		// Make these elements visible and preserve them
		s.RemoveAttr("style") // Remove any hiding styles
		s.AddClass("readability-preserve")
		
		// Find all nested paragraphs and make sure they're preserved
		s.Find("p, h1, h2, h3, h4, h5, h6, blockquote, ul, ol, li").Each(func(_ int, content *goquery.Selection) {
			content.AddClass("readability-preserve")
		})
	})
	
	// Look for any hidden premium content and make it visible
	article.Find("p[class*='premium'], p[class*='paid'], div[class*='premium'], div[class*='paid']").Each(func(_ int, s *goquery.Selection) {
		s.RemoveAttr("style")
		s.AddClass("readability-preserve")
	})
	
	// Remove subscription CTA elements that get in the way of content
	subscriptionCTASelectors := []string{
		".subscribe-button", ".subscription-button", ".subscribe-now",
		".subscription-prompt", ".subscription-message", ".subscribe-overlay",
		".subscription-offer", ".paywall-prompt", ".login-prompt",
		".subscribe-cta", ".subscriber-cta", ".subscription-callout",
	}
	
	for _, selector := range subscriptionCTASelectors {
		article.Find(selector).Remove()
	}
	
	// Remove any content blockers or overlays
	article.Find(".content-blocker, .content-overlay, .article-overlay, .article-blur, .content-fade").Remove()
	
	// Preserve any headings that might be part of the premium content
	article.Find("h1, h2, h3, h4, h5, h6").AddClass("readability-preserve")
	
	// Preserve all paragraphs in the article body
	article.Find("article p, .article-body p, .article-content p, .story p, .story-body p").AddClass("readability-preserve")
	
	// Remove any modal or dialog elements that could be used for paywall prompts
	article.Find(".modal, .dialog, .popup, .notification-modal").Remove()
	
	// Remove any elements with inline styles that might hide content
	article.Find("[style*='display: none'], [style*='visibility: hidden']").Each(func(_ int, s *goquery.Selection) {
		// Only if it appears to be content, not navigation
		if s.Find("p, h1, h2, h3, blockquote").Length() > 0 {
			s.RemoveAttr("style")
			s.AddClass("readability-preserve")
		}
	})
	
	// Remove any "login to read more" messages
	article.Find(".login-message, .login-prompt, .continue-reading-prompt").Remove()
	
	// Look for classes that might indicate premium article content and preserve them
	article.Find("[class*='article-'], [class*='content-'], [class*='story-']").Each(func(_ int, s *goquery.Selection) {
		if s.Find("p, h1, h2, h3, blockquote, ul, ol").Length() > 0 {
			s.AddClass("readability-preserve")
		}
	})
	
	// Clean up standard elements unrelated to the main content
	article.Find(".share, .sharing, .social, .comments, .comment-section").Remove()
	article.Find(".related-articles, .read-more, .more-articles, .trending").Remove()
	article.Find(".ad, .advertisement, .ad-unit, .banner").Remove()
}

// removeNonEssentialElements performs final cleanup for minimal pages
func removeNonEssentialElements(article *goquery.Selection) {
	// Remove links that don't look necessary for the login functionality
	article.Find("a").Each(func(i int, link *goquery.Selection) {
		linkText := strings.ToLower(link.Text())
		href, _ := link.Attr("href")
		
		// Keep only login-related links
		isAuthLink := strings.Contains(linkText, "login") || 
					  strings.Contains(linkText, "sign in") || 
					  strings.Contains(linkText, "register") ||
					  strings.Contains(linkText, "sign up") || 
					  strings.Contains(linkText, "forgot password") ||
					  strings.Contains(linkText, "reset password") ||
					  strings.Contains(linkText, "create account")
		
		// Also keep links that have auth-related hrefs
		isAuthHref := strings.Contains(href, "login") || 
					  strings.Contains(href, "signin") || 
					  strings.Contains(href, "register") ||
					  strings.Contains(href, "signup") || 
					  strings.Contains(href, "password") ||
					  strings.Contains(href, "account")
		
		// If it's not an auth link and not a special case, remove it
		if !isAuthLink && !isAuthHref && href != "#" && href != "/" {
			// Check if it's inside a form (might be important for form function)
			inForm := link.ParentsFiltered("form").Length() > 0
			
			if !inForm {
				link.Remove()
			}
		}
	})
	
	// Remove any remaining social media elements
	article.Find(".social-links, .social-media, .follow-us, .share-buttons").Remove()
	
	// Remove any leftover banners or promotions
	article.Find(".banner, .promo, .promotion, .advertisement, .ad-unit").Remove()
	
	// Remove visually hidden elements that might contain SEO content
	article.Find("[aria-hidden='true'], .hidden, .sr-only, .visually-hidden").Remove()
}

// Note: We're using the isSameNode function from dom_helpers.go

// Helper function to create an empty div for new containers
func getEmptyDiv() *goquery.Selection {
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader("<div></div>"))
	return doc.Find("div")
}