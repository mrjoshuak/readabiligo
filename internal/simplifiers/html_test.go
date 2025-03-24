package simplifiers

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestSimplifyHTML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		opts    ContentOptions
		want    string
		wantErr bool
	}{
		{
			name:  "basic HTML cleanup",
			input: `<body>  <p> Hello  World </p>  </body>`,
			opts:  ContentOptions{},
			want:  `<html><head></head><body><p>Hello World</p></body></html>`,
		},
		{
			name:  "content digests",
			input: `<body><p>Hello</p><p>World</p></body>`,
			opts: ContentOptions{
				AddContentDigests: true,
			},
			want: `<html><head></head><body><p data-content-digest="185f8db32271fe25f561a6fc938b2e264306ec304eda518007d1764826381969">Hello</p><p data-content-digest="78ae647dc5544d227130a0682a51e30bc7777fbb6d8a8f17007463a3ecd1d524">World</p></body></html>`,
		},
		{
			name:  "node indexes",
			input: `<body><p>Hello</p><div><p>World</p></div></body>`,
			opts: ContentOptions{
				AddNodeIndexes: true,
			},
			want: `<html><head></head><body data-node-index="0"><p data-node-index="0.1">Hello</p><div data-node-index="0.2"><p data-node-index="0.2.1">World</p></div></body></html>`,
		},
		{
			name:  "whitespace normalization",
			input: `<body><p>Hello   World</p><p>Multiple    Spaces</p></body>`,
			opts:  ContentOptions{},
			want:  `<html><head></head><body><p>Hello World</p><p>Multiple Spaces</p></body></html>`,
		},
		{
			name:    "invalid HTML",
			input:   `<body><p>Unclosed`,
			opts:    ContentOptions{},
			wantErr: true,
		},
		{
			name:  "remove blacklisted elements",
			input: `<body><p>Text</p><script>alert('hello');</script><button>Click me</button></body>`,
			opts: ContentOptions{
				RemoveBlacklist: true,
			},
			want: `<html><head></head><body><p>Text</p></body></html>`,
		},
		{
			name:  "unwrap elements",
			input: `<body><p><span>Hello</span> <b>World</b></p></body>`,
			opts: ContentOptions{
				UnwrapElements: true,
			},
			want: `<html><head></head><body><p>Hello World</p></body></html>`,
		},
		{
			name:  "process special elements",
			input: `<body><p><q>Quote</q> and <sub>subscript</sub> and <sup>superscript</sup></p></body>`,
			opts: ContentOptions{
				ProcessSpecial: true,
			},
			want: `<html><head></head><body><p>"Quote" and _subscript and ^superscript</p></body></html>`,
		},
		{
			name:  "remove empty elements",
			input: `<html><head></head><body><p>Text</p><p></p><div>  </div></body></html>`,
			opts: ContentOptions{
				RemoveEmpty: true,
			},
			want: `<html><head></head><body><p>Text</p></body></html>`,
		},
		{
			name:  "unnest paragraphs",
			input: `<html><head></head><body><p>Before <div>Inside</div> After</p></body></html>`,
			opts: ContentOptions{
				UnnestParagraphs: true,
			},
			want: `<html><head></head><body><p>Before </p><div>Inside</div><p> After</p></body></html>`,
		},
		{
			name:  "insert paragraph breaks",
			input: `<body><p>First<br><br>Second</p></body>`,
			opts: ContentOptions{
				InsertBreaks:    true,
				ConsolidateText: true,
			},
			want: `<html><head></head><body><p>First</p><p>Second</p></body></html>`,
		},
		{
			name:  "wrap bare text",
			input: `<body>Bare text <div>Inside div</div></body>`,
			opts: ContentOptions{
				WrapBareText: true,
				RemoveEmpty:  true,
			},
			want: `<html><head></head><body><p>Bare text</p><div>Inside div</div></body></html>`,
		},
		{
			name: "full processing",
			input: `<body>
				<div class="content">
					<p style="color:red;"><span>Hello</span> <b>World</b></p>
					<script>alert('hello');</script>
					<p>First<br><br>Second</p>
					Bare text
					<p><q>Quote</q> and <sub>subscript</sub></p>
					<p></p>
				</div>
			</body>`,
			opts: ContentOptions{
				RemoveBlacklist:   true,
				UnwrapElements:    true,
				ProcessSpecial:    true,
				ConsolidateText:   true,
				RemoveEmpty:       true,
				UnnestParagraphs:  true,
				InsertBreaks:      true,
				WrapBareText:      true,
				AddContentDigests: true,
				AddNodeIndexes:    true,
			},
			want: `<html><head></head><body data-node-index="0"><div data-node-index="0.1"><p data-node-index="0.1.1" data-content-digest="78ae647dc5544d227130a0682a51e30bc7777fbb6d8a8f17007463a3ecd1d524">Hello World</p><p data-node-index="0.1.2" data-content-digest="185f8db32271fe25f561a6fc938b2e264306ec304eda518007d1764826381969">First</p><p data-node-index="0.1.3" data-content-digest="78ae647dc5544d227130a0682a51e30bc7777fbb6d8a8f17007463a3ecd1d524">Second</p><p data-node-index="0.1.4" data-content-digest="5feceb66ffc86f38d952786c6d696c79c2dbc239dd4e91b46729d73a27fb57e9">Bare text</p><p data-node-index="0.1.5" data-content-digest="b3a8e0e1f9ab1bfe3a36f231f676f78bb30a519d2b21e6c530c0eee8ebb4a5d0">"Quote" and _subscript</p></div></body></html>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SimplifyHTML(tt.input, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("SimplifyHTML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && got != tt.want {
				t.Errorf("SimplifyHTML() =\n%v\nwant\n%v", got, tt.want)
			}
		})
	}
}

func TestIsLeafNode(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`
<body>
<p>paragraph</p>
<div>division</div>
<li>list item</li>
<span>span</span>
</body>
`))
	if err != nil {
		t.Fatalf("Failed to parse test HTML: %v", err)
	}

	tests := []struct {
		name     string
		selector string
		want     bool
	}{
		{"paragraph is leaf", "p", true},
		{"list item is leaf", "li", true},
		{"div is not leaf", "div", false},
		{"span is not leaf", "span", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			el := NewPlainElement(doc.Find(tt.selector))
			if got := isLeafNode(el); got != tt.want {
				t.Errorf("isLeafNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateContentDigest(t *testing.T) {
	tests := []struct {
		name  string
		html  string
		query string
		want  string
	}{
		{
			name:  "single text node",
			html:  "<p>Hello</p>",
			query: "p",
			want:  "185f8db32271fe25f561a6fc938b2e264306ec304eda518007d1764826381969",
		},
		{
			name:  "nested elements",
			html:  "<div><p>Hello</p><p>World</p></div>",
			query: "div",
			want:  "22c4c75765836e26a3342c66abc42a4007f0fbc676e37e886a7f26c02d78e420",
		},
		{
			name:  "empty element",
			html:  "<p></p>",
			query: "p",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.html))
			if err != nil {
				t.Fatalf("Failed to parse test HTML: %v", err)
			}

			el := NewPlainElement(doc.Find(tt.query))
			if got := calculateContentDigest(el); got != tt.want {
				t.Errorf("calculateContentDigest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoveBlacklist(t *testing.T) {
	html := `<body><p>Text</p><script>alert('hello');</script><button>Click me</button></body>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse test HTML: %v", err)
	}

	removeBlacklist(doc)

	// Check that blacklisted elements are removed
	if doc.Find("script").Length() > 0 || doc.Find("button").Length() > 0 {
		t.Errorf("removeBlacklist() failed to remove blacklisted elements")
	}

	// Check that non-blacklisted elements are preserved
	if doc.Find("p").Length() == 0 {
		t.Errorf("removeBlacklist() removed non-blacklisted elements")
	}
}

func TestUnwrapElements(t *testing.T) {
	html := `<body><p><span>Hello</span> <b>World</b></p></body>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse test HTML: %v", err)
	}

	unwrapElements(doc)

	// Check that unwrapped elements are removed
	if doc.Find("span").Length() > 0 || doc.Find("b").Length() > 0 {
		t.Errorf("unwrapElements() failed to unwrap elements")
	}

	// Check that content is preserved
	if text := doc.Find("p").Text(); text != "Hello World" {
		t.Errorf("unwrapElements() did not preserve content, got %q", text)
	}
}

func TestProcessSpecialElements(t *testing.T) {
	html := `<body><p><q>Quote</q> and <sub>subscript</sub> and <sup>superscript</sup></p></body>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse test HTML: %v", err)
	}

	processSpecialElements(doc)

	// Check that special elements are unwrapped
	if doc.Find("q").Length() > 0 || doc.Find("sub").Length() > 0 || doc.Find("sup").Length() > 0 {
		t.Errorf("processSpecialElements() failed to unwrap special elements")
	}

	// Check that content is transformed correctly
	if text := doc.Find("p").Text(); !strings.Contains(text, "\"Quote\"") ||
		!strings.Contains(text, "_subscript") || !strings.Contains(text, "^superscript") {
		t.Errorf("processSpecialElements() did not transform content correctly, got %q", text)
	}
}

func TestUnnestParagraphs(t *testing.T) {
	// Create a direct test for the unnestParagraphs function
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<html><head></head><body><p>Before <div>Inside</div> After</p></body></html>`))
	if err != nil {
		t.Fatalf("Failed to parse test HTML: %v", err)
	}

	// Call the function directly
	unnestParagraphs(doc)

	// Check that the div is no longer inside the p
	if doc.Find("p div").Length() > 0 {
		t.Errorf("unnestParagraphs() failed to unnest div from p")
	}

	// Check that we now have multiple paragraphs
	if doc.Find("p").Length() < 2 {
		t.Errorf("unnestParagraphs() did not create separate paragraphs")
	}
}

func TestInsertParagraphBreaks(t *testing.T) {
	html := `<html><head></head><body><p>First<br><br>Second</p></body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse test HTML: %v", err)
	}

	insertParagraphBreaks(doc)

	// Check that br elements are removed
	if doc.Find("br").Length() > 0 {
		t.Errorf("insertParagraphBreaks() failed to remove br elements")
	}

	// Check that we now have multiple paragraphs
	if doc.Find("p").Length() < 2 {
		t.Errorf("insertParagraphBreaks() did not create separate paragraphs")
	}

	// Check the expected structure
	expectedHTML := `<html><head></head><body><p>First</p><p>Second</p></body></html>`
	actualHTML, err := doc.Html()
	if err != nil {
		t.Fatalf("Failed to get HTML: %v", err)
	}
	actualHTML = StripHTMLWhitespace(actualHTML)
	if actualHTML != expectedHTML {
		t.Errorf("insertParagraphBreaks() produced incorrect HTML:\nGot: %s\nWant: %s", actualHTML, expectedHTML)
	}
}

func TestWrapBareText(t *testing.T) {
	html := `<body>Bare text <div>Inside div</div></body>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse test HTML: %v", err)
	}

	wrapBareText(doc)

	// Check that bare text is wrapped in a paragraph
	if doc.Find("body > p").Length() == 0 {
		t.Errorf("wrapBareText() failed to wrap bare text in paragraphs")
	}

	// Check that the div content is not wrapped
	if doc.Find("div > p").Length() > 0 {
		t.Errorf("wrapBareText() incorrectly wrapped div content")
	}
}
