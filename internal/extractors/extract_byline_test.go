package extractors

import (
	"testing"
)

func TestExtractByline(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "Meta tag with article:author",
			html:     `<html><head><meta property="article:author" content="John Doe"></head><body></body></html>`,
			expected: "John Doe",
		},
		{
			name:     "Meta tag with og:article:author",
			html:     `<html><head><meta property="og:article:author" content="Jane Smith"></head><body></body></html>`,
			expected: "Jane Smith",
		},
		{
			name:     "Meta tag with author",
			html:     `<html><head><meta name="author" content="Bob Johnson"></head><body></body></html>`,
			expected: "Bob Johnson",
		},
		{
			name:     "Meta tag with sailthru.author",
			html:     `<html><head><meta name="sailthru.author" content="Alice Williams"></head><body></body></html>`,
			expected: "Alice Williams",
		},
		{
			name:     "Meta tag with byl",
			html:     `<html><head><meta name="byl" content="By Charlie Brown"></head><body></body></html>`,
			expected: "Charlie Brown",
		},
		{
			name:     "Meta tag with twitter:creator",
			html:     `<html><head><meta name="twitter:creator" content="@DavidMiller"></head><body></body></html>`,
			expected: "@DavidMiller",
		},
		{
			name:     "A tag with rel=author",
			html:     `<html><body><a rel="author">Emily Davis</a></body></html>`,
			expected: "Emily Davis",
		},
		{
			name:     "Span with author class",
			html:     `<html><body><span class="author">Frank Wilson</span></body></html>`,
			expected: "Frank Wilson",
		},
		{
			name:     "Div with author class",
			html:     `<html><body><div class="author">Grace Taylor</div></body></html>`,
			expected: "Grace Taylor",
		},
		{
			name:     "Span with byline class",
			html:     `<html><body><span class="byline">By Henry Martin</span></body></html>`,
			expected: "Henry Martin",
		},
		{
			name:     "Span with itemprop=author",
			html:     `<html><body><span itemprop="author">Ivy Clark</span></body></html>`,
			expected: "Ivy Clark",
		},
		{
			name:     "Schema.org Person with author role",
			html:     `<html><body><div itemtype="http://schema.org/Person"><span itemprop="name">Jack Adams</span><span itemprop="role">Author</span></div></body></html>`,
			expected: "",
		},
		{
			name:     "Schema.org Article with author",
			html:     `<html><body><div itemtype="http://schema.org/Article"><span itemprop="author">Karen White</span></div></body></html>`,
			expected: "",
		},
		{
			name:     "Paragraph with 'By' prefix",
			html:     `<html><body><p>By Laura Green</p></body></html>`,
			expected: "Laura Green",
		},
		{
			name:     "Paragraph with 'Written by' prefix",
			html:     `<html><body><p>Written by Michael Lee</p></body></html>`,
			expected: "Michael Lee",
		},
		{
			name:     "Multiple author patterns (should pick highest confidence)",
			html:     `<html><head><meta name="author" content="Nancy Hall"></head><body><span class="byline">By Oliver King</span></body></html>`,
			expected: "Nancy Hall",
		},
		{
			name:     "No author information",
			html:     `<html><body><p>This is an article with no author information.</p></body></html>`,
			expected: "",
		},
		{
			name:     "Author with suffix",
			html:     `<html><body><span class="author">Patricia Evans | Writer</span></body></html>`,
			expected: "Patricia Evans",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := ExtractByline(test.html)
			if result != test.expected {
				t.Errorf("Expected byline %q, got %q", test.expected, result)
			}
		})
	}
}

func TestCleanByline(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple byline",
			input:    "John Doe",
			expected: "John Doe",
		},
		{
			name:     "Byline with 'By' prefix",
			input:    "By Jane Smith",
			expected: "Jane Smith",
		},
		{
			name:     "Byline with 'Author:' prefix",
			input:    "Author: Bob Johnson",
			expected: "Bob Johnson",
		},
		{
			name:     "Byline with 'Written by' prefix",
			input:    "Written by Alice Williams",
			expected: "Alice Williams",
		},
		{
			name:     "Byline with 'Posted by' prefix",
			input:    "Posted by Charlie Brown",
			expected: "Charlie Brown",
		},
		{
			name:     "Byline with 'Published by' prefix",
			input:    "Published by David Miller",
			expected: "David Miller",
		},
		{
			name:     "Byline with 'Reported by' prefix",
			input:    "Reported by Emily Davis",
			expected: "Emily Davis",
		},
		{
			name:     "Byline with '| Author' suffix",
			input:    "Frank Wilson | Author",
			expected: "Frank Wilson",
		},
		{
			name:     "Byline with '| Writer' suffix",
			input:    "Grace Taylor | Writer",
			expected: "Grace Taylor",
		},
		{
			name:     "Byline with '| Reporter' suffix",
			input:    "Henry Martin | Reporter",
			expected: "Henry Martin",
		},
		{
			name:     "Byline with '| Staff' suffix",
			input:    "Ivy Clark | Staff",
			expected: "Ivy Clark",
		},
		{
			name:     "Byline with whitespace",
			input:    "  Jack Adams  ",
			expected: "Jack Adams",
		},
		{
			name:     "Byline with prefix and suffix",
			input:    "By Karen White | Writer",
			expected: "Karen White",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := cleanByline(test.input)
			if result != test.expected {
				t.Errorf("Expected cleaned byline %q, got %q", test.expected, result)
			}
		})
	}
}
