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

// Scoring constants for content extraction
const (
	// BaseContentScore is the starting score for content elements
	BaseContentScore = 1.0

	// CommaBonus is the score bonus per comma found in content
	CommaBonus = 1.0

	// MaxLengthBonus is the maximum bonus for text length
	MaxLengthBonus = 3.0

	// TextLengthDivisor is the value by which text length is divided to calculate the bonus
	TextLengthDivisor = 100.0

	// MinContentTextLength is minimum text length required for a node to be considered content
	MinContentTextLength = 25

	// SiblingScoreMultiplier is multiplier for calculating sibling score threshold
	SiblingScoreMultiplier = 0.2

	// MinimumSiblingScoreThreshold is the minimum threshold for sibling scores when topCandidate has no score
	MinimumSiblingScoreThreshold = 10.0

	// SameClassSiblingBonus is the bonus for siblings with the same class as the top candidate
	SameClassSiblingBonus = 0.2

	// MinParagraphLength is the minimum length for a paragraph to be included
	MinParagraphLength = 80

	// MaxShortParagraphLength is the maximum length for a short paragraph that can be included if it has specific qualities
	MaxShortParagraphLength = 80

	// ParagraphLinkDensityThreshold is the maximum allowed link density for paragraphs to be included
	ParagraphLinkDensityThreshold = 0.25
)

// Cleanup and conditional removal constants
const (
	// MinCommaCount is the minimum number of commas needed for a node to pass the comma test in conditional cleaning
	MinCommaCount = 10

	// MinEmbedContentLength is the minimum content length for a node with a single embed to be kept
	MinEmbedContentLength = 75

	// TitleSimilarityThreshold is the threshold for determining if a heading is similar to the article title
	TitleSimilarityThreshold = 0.75

	// ListLinkDensityThreshold is the maximum ratio of link text to total text for lists to be considered content
	ListLinkDensityThreshold = 0.7 // Less aggressive to preserve more lists

	// ConditionalLinkDensityThresholdLow is link density threshold for nodes with low weight
	ConditionalLinkDensityThresholdLow = 0.3 // Less aggressive

	// ConditionalLinkDensityThresholdHigh is link density threshold for nodes with high weight
	ConditionalLinkDensityThresholdHigh = 0.6 // Less aggressive

	// ConditionalWeightThresholdLow is the low weight threshold for conditional cleaning
	ConditionalWeightThresholdLow = 25

	// HeadingDensityThreshold is the maximum ratio of heading text to total text
	HeadingDensityThreshold = 0.9

	// DataTableMinRows is the minimum number of rows for a table to be considered a data table
	DataTableMinRows = 3

	// DataTableMinColumns is the minimum number of columns for a table to be considered a data table
	DataTableMinColumns = 2

	// DataTableMinCells is the minimum number of cells for a table to be considered a data table
	DataTableMinCells = 10
	
	// LayoutTableNestingThreshold is the maximum allowed nesting level for layout tables
	LayoutTableNestingThreshold = 2
	
	// NavigationLinkDensityThreshold is the link density threshold for considering a table to be navigation
	NavigationLinkDensityThreshold = 0.8
	
	// LayoutTableTextContentThreshold is the minimum text content (non-link) for a table to be kept
	LayoutTableTextContentThreshold = 50
)

// Node scoring constants
const (
	// Initial scores based on tag name
	DivInitialScore          = 5.0
	BlockquoteInitialScore   = 3.0
	NegativeListInitialScore = -3.0
	HeadingInitialScore      = -5.0

	// Class weight adjustments
	ClassWeightNegative = -25
	ClassWeightPositive = 25
)

// Ancestor scoring constants
const (
	// AncestorLevelDepth is the max depth for getting node ancestors
	AncestorLevelDepth = 5

	// AncestorScoreDividerL0 is the score divider for level 0 ancestors
	AncestorScoreDividerL0 = 1.0

	// AncestorScoreDividerL1 is the score divider for level 1 ancestors
	AncestorScoreDividerL1 = 2.0

	// AncestorScoreDividerMultiplier is the multiplier for levels > 1
	AncestorScoreDividerMultiplier = 3.0
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
	RegexpUnlikelyCandidates = regexp.MustCompile(`-ad-|ai2html|banner|breadcrumbs|combx|comment|community|cover-wrap|disqus|extra|footer|gdpr|header|legends|menu|nav|navigation|related|remark|replies|rss|shoutbox|sidebar|skyscraper|social|sponsor|supplemental|ad-break|agegate|pagination|pager|popup|yom-remote`)

	// Candidates that might be content despite matching the unlikelyCandidates pattern
	RegexpMaybeCandidate = regexp.MustCompile(`and|article|body|column|content|main|shadow`)

	// Positive indicators of content
	RegexpPositive = regexp.MustCompile(`article|body|content|entry|hentry|h-entry|main|page|pagination|post|text|blog|story`)

	// Negative indicators of content - adjusted to be consistent with Readability.js
	RegexpNegative = regexp.MustCompile(`-ad-|hidden|^hid$| hid$| hid |^hid |banner|combx|comment|com-|contact|footer|gdpr|masthead|media|meta|outbrain|promo|related|scroll|share|shoutbox|sidebar|skyscraper|sponsor|shopping|tags|widget`)

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