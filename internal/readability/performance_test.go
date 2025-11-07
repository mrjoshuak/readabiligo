package readability

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

// BenchmarkGetInnerText measures the performance of text extraction
// This is a critical operation called many times during extraction
func BenchmarkGetInnerText(b *testing.B) {
	testCases := []struct {
		name string
		html string
	}{
		{
			name: "SimpleText",
			html: `<p>This is a simple paragraph with some text.</p>`,
		},
		{
			name: "NestedText",
			html: `<div><p>Paragraph 1</p><p>Paragraph 2</p><p>Paragraph 3</p></div>`,
		},
		{
			name: "DeeplyNested",
			html: `<div><div><div><div><div><p>Deeply nested text</p></div></div></div></div></div>`,
		},
		{
			name: "LargeText",
			html: `<div>` + strings.Repeat("<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit.</p>", 100) + `</div>`,
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tc.html))
			if err != nil {
				b.Fatalf("Failed to parse HTML: %v", err)
			}
			sel := doc.Selection

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = getInnerText(sel, true)
			}
		})
	}
}

// BenchmarkGetClassWeight measures the performance of class weight calculation
// This is called during scoring and candidate evaluation
func BenchmarkGetClassWeight(b *testing.B) {
	testCases := []struct {
		name string
		html string
	}{
		{
			name: "NoClasses",
			html: `<div>Content</div>`,
		},
		{
			name: "PositiveClass",
			html: `<div class="article content main">Content</div>`,
		},
		{
			name: "NegativeClass",
			html: `<div class="sidebar ad banner">Content</div>`,
		},
		{
			name: "MixedClasses",
			html: `<div class="article sidebar content">Content</div>`,
		},
		{
			name: "WithID",
			html: `<div id="article-main" class="content">Content</div>`,
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tc.html))
			if err != nil {
				b.Fatalf("Failed to parse HTML: %v", err)
			}
			sel := doc.Find("div").First()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = getClassWeight(sel)
			}
		})
	}
}

// BenchmarkGetLinkDensity measures the performance of link density calculation
// This is used in content scoring and cleanup decisions
func BenchmarkGetLinkDensity(b *testing.B) {
	testCases := []struct {
		name string
		html string
	}{
		{
			name: "NoLinks",
			html: `<p>This is text without any links.</p>`,
		},
		{
			name: "FewLinks",
			html: `<p>Text with <a href="/link1">one</a> and <a href="/link2">two</a> links.</p>`,
		},
		{
			name: "ManyLinks",
			html: `<div>` + strings.Repeat(`<a href="/link">link</a> `, 50) + `</div>`,
		},
		{
			name: "HashLinks",
			html: `<p>Text with <a href="#section1">hash</a> and <a href="#section2">links</a>.</p>`,
		},
		{
			name: "MixedLinks",
			html: `<p>Text <a href="/external">external</a> and <a href="#internal">internal</a> links.</p>`,
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tc.html))
			if err != nil {
				b.Fatalf("Failed to parse HTML: %v", err)
			}
			sel := doc.Selection

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = getLinkDensity(sel)
			}
		})
	}
}

// BenchmarkWordCount measures the performance of word counting
// Used in metadata extraction and content analysis
func BenchmarkWordCount(b *testing.B) {
	testCases := []struct {
		name string
		text string
	}{
		{
			name: "Short",
			text: "Just a few words here",
		},
		{
			name: "Medium",
			text: strings.Repeat("Lorem ipsum dolor sit amet consectetur adipiscing elit ", 10),
		},
		{
			name: "Long",
			text: strings.Repeat("Lorem ipsum dolor sit amet consectetur adipiscing elit ", 100),
		},
		{
			name: "WithPunctuation",
			text: "Words, with! lots? of. punctuation; and: symbols.",
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = wordCount(tc.text)
			}
		})
	}
}

// BenchmarkHasAncestorTag measures the performance of ancestor checking
// This is called frequently during DOM traversal
func BenchmarkHasAncestorTag(b *testing.B) {
	testCases := []struct {
		name     string
		html     string
		tagName  string
		maxDepth int
	}{
		{
			name:     "NoAncestor",
			html:     `<div><span><em>Text</em></span></div>`,
			tagName:  "table",
			maxDepth: -1,
		},
		{
			name:     "ShallowAncestor",
			html:     `<div><p><span>Text</span></p></div>`,
			tagName:  "div",
			maxDepth: -1,
		},
		{
			name:     "DeepAncestor",
			html:     `<table><tr><td><div><p><span>Text</span></p></div></td></tr></table>`,
			tagName:  "table",
			maxDepth: -1,
		},
		{
			name:     "WithMaxDepth",
			html:     `<div><div><div><div><div><span>Text</span></div></div></div></div></div>`,
			tagName:  "div",
			maxDepth: 3,
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tc.html))
			if err != nil {
				b.Fatalf("Failed to parse HTML: %v", err)
			}
			sel := doc.Find("span").First()
			if sel.Length() == 0 {
				sel = doc.Find("em").First()
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = hasAncestorTag(sel, tc.tagName, tc.maxDepth, nil)
			}
		})
	}
}

// BenchmarkUnescapeHtmlEntities measures the performance of HTML entity unescaping
// Used in metadata extraction and text cleaning
func BenchmarkUnescapeHtmlEntities(b *testing.B) {
	testCases := []struct {
		name string
		text string
	}{
		{
			name: "NoEntities",
			text: "Plain text without entities",
		},
		{
			name: "BasicEntities",
			text: "Text with &lt;tags&gt; and &amp; symbols",
		},
		{
			name: "NumericEntities",
			text: "Text with &#169; and &#x2022; symbols",
		},
		{
			name: "MixedEntities",
			text: "&lt;div&gt;Text with &#169; copyright &amp; &#x2022; bullets&lt;/div&gt;",
		},
		{
			name: "ManyEntities",
			text: strings.Repeat("&lt;p&gt;Text&lt;/p&gt; ", 20),
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = unescapeHtmlEntities(tc.text)
			}
		})
	}
}

// BenchmarkGetOuterHTML measures the performance of getting outer HTML
// Used in debug logging and cleanup operations
func BenchmarkGetOuterHTML(b *testing.B) {
	testCases := []struct {
		name string
		html string
	}{
		{
			name: "SimpleElement",
			html: `<p>Simple paragraph</p>`,
		},
		{
			name: "ElementWithAttributes",
			html: `<div class="content" id="main" data-type="article">Content</div>`,
		},
		{
			name: "NestedElements",
			html: `<div><p>Para 1</p><p>Para 2</p><p>Para 3</p></div>`,
		},
		{
			name: "LargeTree",
			html: `<div>` + strings.Repeat("<p>Content paragraph.</p>", 50) + `</div>`,
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tc.html))
			if err != nil {
				b.Fatalf("Failed to parse HTML: %v", err)
			}
			sel := doc.Selection.Children().First()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = getOuterHTML(sel)
			}
		})
	}
}

// BenchmarkTextSimilarity measures the performance of text similarity calculation
// Used in duplicate detection and content comparison
func BenchmarkTextSimilarity(b *testing.B) {
	testCases := []struct {
		name  string
		textA string
		textB string
	}{
		{
			name:  "Identical",
			textA: "This is identical text",
			textB: "This is identical text",
		},
		{
			name:  "SimilarShort",
			textA: "This is some text",
			textB: "This is similar text",
		},
		{
			name:  "DifferentShort",
			textA: "Completely different",
			textB: "Totally unrelated",
		},
		{
			name:  "SimilarLong",
			textA: strings.Repeat("Lorem ipsum dolor sit amet consectetur ", 10),
			textB: strings.Repeat("Lorem ipsum dolor sit amet adipiscing ", 10),
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = textSimilarity(tc.textA, tc.textB)
			}
		})
	}
}

// BenchmarkIsPhrasingContent checks the performance of phrasing content detection
// Called frequently during DOM traversal and text extraction
func BenchmarkIsPhrasingContent(b *testing.B) {
	testCases := []struct {
		name string
		html string
	}{
		{
			name: "SpanElement",
			html: `<span>Inline text</span>`,
		},
		{
			name: "DivElement",
			html: `<div>Block text</div>`,
		},
		{
			name: "NestedInline",
			html: `<span><em><strong>Nested inline</strong></em></span>`,
		},
		{
			name: "LinkElement",
			html: `<a href="/link">Link text</a>`,
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tc.html))
			if err != nil {
				b.Fatalf("Failed to parse HTML: %v", err)
			}
			node := doc.Selection.Get(0).FirstChild

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = isPhrasingContent(node)
			}
		})
	}
}
