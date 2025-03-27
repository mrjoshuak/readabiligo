// Package readability provides a pure Go implementation of Mozilla's Readability.js
// for extracting the main content from web pages.
package readability

import (
	"regexp"
)

// Flags for controlling the content extraction process
const (
	FlagStripUnlikelys      = 0x1
	FlagWeightClasses       = 0x2
	FlagCleanConditionally  = 0x4
)

// Node types from the HTML package
const (
	ElementNode = 1
	TextNode    = 3
	CommentNode = 8
	DoctypeNode = 10
)

// Default settings
const (
	// DefaultMaxElemsToParse is the maximum number of elements to parse (0 = no limit)
	DefaultMaxElemsToParse = 0

	// DefaultNTopCandidates is the number of top candidates to consider
	DefaultNTopCandidates = 5

	// DefaultCharThreshold is the minimum number of characters required for content
	DefaultCharThreshold = 500
)

// DefaultTagsToScore defines the element tags that should be scored
var DefaultTagsToScore = []string{"SECTION", "H2", "H3", "H4", "H5", "H6", "P", "TD", "PRE"}

// ClassesToPreserve defines CSS classes that should be preserved in the output
var ClassesToPreserve = []string{"page"}

// UnlikelyRoles defines ARIA roles that suggest a node is not content
var UnlikelyRoles = []string{"menu", "menubar", "complementary", "navigation", "alert", "alertdialog", "dialog"}

// DivToPElems defines elements that can appear inside a <div> but should be promoted to paragraphs
var DivToPElems = []string{"BLOCKQUOTE", "DL", "DIV", "IMG", "OL", "P", "PRE", "TABLE", "UL"}

// AlterToDivExceptions defines elements that should not be converted to <div>
var AlterToDivExceptions = []string{"DIV", "ARTICLE", "SECTION", "P"}

// PresentationalAttributes defines presentational attributes to remove
var PresentationalAttributes = []string{"align", "background", "bgcolor", "border", "cellpadding", "cellspacing", "frame", "hspace", "rules", "style", "valign", "vspace"}

// DeprecatedSizeAttributeElems defines elements with deprecated size attributes
var DeprecatedSizeAttributeElems = []string{"TABLE", "TH", "TD", "HR", "PRE"}

// PhrasingElems defines elements that qualify as phrasing content
var PhrasingElems = []string{
	"ABBR", "AUDIO", "B", "BDO", "BR", "BUTTON", "CITE", "CODE", "DATA",
	"DATALIST", "DFN", "EM", "EMBED", "I", "IMG", "INPUT", "KBD", "LABEL",
	"MARK", "MATH", "METER", "NOSCRIPT", "OBJECT", "OUTPUT", "PROGRESS", "Q",
	"RUBY", "SAMP", "SCRIPT", "SELECT", "SMALL", "SPAN", "STRONG", "SUB",
	"SUP", "TEXTAREA", "TIME", "VAR", "WBR",
}

// HTMLEscapeMap defines HTML entities that need to be escaped
var HTMLEscapeMap = map[string]string{
	"lt":   "<",
	"gt":   ">",
	"amp":  "&",
	"quot": "\"",
	"apos": "'",
}

// Regular expressions used in the Readability algorithm
var (
	// Unlikely candidates for content
	RegexpUnlikelyCandidates = regexp.MustCompile(`-ad-|ai2html|banner|breadcrumbs|combx|comment|community|cover-wrap|disqus|extra|footer|gdpr|header|legends|menu|related|remark|replies|rss|shoutbox|sidebar|skyscraper|social|sponsor|supplemental|ad-break|agegate|pagination|pager|popup|yom-remote`)

	// Candidates that might be content despite matching the unlikelyCandidates pattern
	RegexpMaybeCandidate = regexp.MustCompile(`and|article|body|column|content|main|shadow`)

	// Positive indicators of content
	RegexpPositive = regexp.MustCompile(`article|body|content|entry|hentry|h-entry|main|page|pagination|post|text|blog|story`)

	// Negative indicators of content
	RegexpNegative = regexp.MustCompile(`-ad-|hidden|^hid$| hid$| hid |^hid |banner|combx|comment|com-|contact|foot|footer|footnote|gdpr|masthead|media|meta|outbrain|promo|related|scroll|share|shoutbox|sidebar|skyscraper|sponsor|shopping|tags|tool|widget`)

	// Extraneous content areas
	RegexpExtraneous = regexp.MustCompile(`print|archive|comment|discuss|e[\-]?mail|share|reply|all|login|sign|single|utility`)

	// Byline indicators
	RegexpByline = regexp.MustCompile(`byline|author|dateline|writtenby|p-author`)

	// Font elements to replace
	RegexpReplaceFonts = regexp.MustCompile(`<(/?)font[^>]*>`)

	// Normalize whitespace
	RegexpNormalize = regexp.MustCompile(`\s{2,}`)

	// Video services to preserve
	RegexpVideos = regexp.MustCompile(`//(www\.)?((dailymotion|youtube|youtube-nocookie|player\.vimeo|v\.qq)\.com|(archive|upload\.wikimedia)\.org|player\.twitch\.tv)`)

	// Share elements
	RegexpShareElements = regexp.MustCompile(`(\b|_)(share|sharedaddy)(\b|_)`)

	// Next page links
	RegexpNextLink = regexp.MustCompile(`(next|weiter|continue|>([^\|]|$)|»([^\|]|$))`)

	// Previous page links
	RegexpPrevLink = regexp.MustCompile(`(prev|earl|old|new|<|«)`)

	// Tokenize text
	RegexpTokenize = regexp.MustCompile(`\W+`)

	// Whitespace
	RegexpWhitespace = regexp.MustCompile(`^\s*$`)

	// Has content
	RegexpHasContent = regexp.MustCompile(`\S$`)

	// Hash URL
	RegexpHashUrl = regexp.MustCompile(`^#.+`)

	// Srcset URL
	RegexpSrcsetUrl = regexp.MustCompile(`(\S+)(\s+[\d.]+[xw])?(\s*(?:,|$))`)

	// Base64 data URL
	RegexpB64DataUrl = regexp.MustCompile(`^data:\s*([^\s;,]+)\s*;\s*base64\s*,`)

	// JSON-LD article types
	RegexpJsonLdArticleTypes = regexp.MustCompile(`^Article|AdvertiserContentArticle|NewsArticle|AnalysisNewsArticle|AskPublicNewsArticle|BackgroundNewsArticle|OpinionNewsArticle|ReportageNewsArticle|ReviewNewsArticle|Report|SatiricalArticle|ScholarlyArticle|MedicalScholarlyArticle|SocialMediaPosting|BlogPosting|LiveBlogPosting|DiscussionForumPosting|TechArticle|APIReference$`)
)