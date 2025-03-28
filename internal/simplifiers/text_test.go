package simplifiers

import (
	"strings"
	"testing"
)

func TestNormalizeUnicode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "NFKC normalization combines characters",
			input: "e\u0301", // é as two characters
			want:  "é",       // é as single character
		},
		{
			name:  "NFKC normalization handles special spaces",
			input: "hello\u2003world", // em space
			want:  "hello world",      // regular space
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeUnicode(tt.input); got != tt.want {
				t.Errorf("NormalizeUnicode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "multiple types of whitespace",
			input: "Hello \t World  with\n\rmultiple  spaces",
			want:  "Hello World with multiple spaces",
		},
		{
			name:  "tabs and newlines",
			input: "hello\t\nworld",
			want:  "hello world",
		},
		{
			name:  "leading and trailing whitespace",
			input: "  hello world  ",
			want:  "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeWhitespace(tt.input); got != tt.want {
				t.Errorf("NormalizeWhitespace() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStripControlChars(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "retain normal text",
			input: "Hello, World!",
			want:  "Hello, World!",
		},
		{
			name:  "retain specified whitespace",
			input: "Hello\t\n\r\fWorld",
			want:  "Hello\t\n\r\fWorld",
		},
		{
			name:  "strip other control chars",
			input: "Hello\u0000World\u001F",
			want:  "HelloWorld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripControlChars(tt.input); got != tt.want {
				t.Errorf("StripControlChars() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "combined operations",
			input: "Hello\t\t  World   with\n\rmultiple\f\fspaces  ",
			want:  "Hello World with multiple spaces",
		},
		{
			name:  "control characters and spaces",
			input: "\x00Hello\x01  \x02World\x03\n  ",
			want:  "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeText(tt.input); got != tt.want {
				t.Errorf("NormalizeText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStripHTMLWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "remove space around tags",
			input: "Hello < p >World< /p >",
			want:  "Hello<p>World</p>",
		},
		{
			name:  "preserve inner spaces",
			input: "< p >Hello  World< /p >",
			want:  "<p>Hello World</p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripHTMLWhitespace(tt.input); got != tt.want {
				t.Errorf("StripHTMLWhitespace() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsControlCategory(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		cat  string
		want bool
	}{
		{"Cc", 0x00, "Cc", true},
		{"Cf", 0x061C, "Cf", true},
		{"Cn", 0x10FFFF, "Cn", true},
		{"Co", 0xE000, "Co", true},
		{"Cs", 0xD800, "Cs", true},
		{"Lu", 'A', "Lu", false}, // Uppercase letter
		{"Ll", 'a', "Ll", false}, // Lowercase letter
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsControlCategory(tt.r, tt.cat); got != tt.want {
				t.Errorf("IsControlCategory(%q, %q) = %v, want %v", tt.r, tt.cat, got, tt.want)
			}
		})
	}
}

// Test data for benchmarks
var (
	// Short text for benchmarking simple operations
	shortText = "This is a short text with some special characters: " +
		"\u2013 \u2014 \u2018 \u2019 \u201c \u201d \u2026 \u00a0 \u00ad " +
		"\u2022 \u2023 \u2043 \u2212 \u00b7 \u00b0 \u00ae \u00a9 \u2122"

	// Medium text with a mix of normal text and special characters
	mediumText = strings.Repeat("This is a paragraph with special chars: "+
		"\u2013 \u2014 \u2018 \u2019 \u201c \u201d \u2026 \u00a0 \u00ad "+
		"and some control characters: \u0001 \u0002 \u0003 \u0004 \u0005 "+
		"and lots of spaces and tabs:     \t    \t    \n\r\n\r\n\r\n\r\n\r\n\r\n", 10)

	// Long text with many special characters and HTML-like content
	longText = strings.Repeat("<div> <p> This is a longer text with many " +
		"special characters: \u2013 \u2014 \u2018 \u2019 \u201c \u201d \u2026 "+
		"\u00a0 \u00ad \u2022 \u2023 \u2043 \u2212 \u00b7 \u00b0 \u00ae \u00a9 "+
		"\u2122 and     lots  of   spaces \t\t\t and \r\n\r\n\r\n newlines "+
		"as well as HTML-like content: < span > test < / span > </p> </div>", 50)

	// Text with HTML entities
	htmlText = strings.Repeat("This text has HTML entities: &lt;div&gt; " +
		"&amp; &quot;quoted text&quot; &apos;single quotes&apos; &mdash; " +
		"&ndash; &hellip; &lsquo;left single quote&rsquo; &ldquo;left double " +
		"quote&rdquo; &bull; &middot; &plusmn; &times; &divide; &not; &micro; " +
		"&para; &degree; &frac14; &frac12; &frac34; &iquest; &iexcl; &szlig; " +
		"&agrave; &aacute; &acirc; &atilde; &auml; &aring; &aelig; &ccedil; " +
		"&egrave; &eacute; &ecirc; &euml; &igrave; &iacute; &icirc; &iuml; " +
		"&ntilde; &ograve; &oacute; &ocirc; &otilde; &ouml; &oslash; &ugrave; " +
		"&uacute; &ucirc; &uuml; &yacute; &yuml; &thorn; &eth;", 5)
)

// BenchmarkNormalizeUnicode benchmarks Unicode normalization
func BenchmarkNormalizeUnicode(b *testing.B) {
	b.Run("ShortText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NormalizeUnicode(shortText)
		}
	})

	b.Run("MediumText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NormalizeUnicode(mediumText)
		}
	})

	b.Run("LongText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NormalizeUnicode(longText)
		}
	})
}

// BenchmarkNormalizeWhitespace benchmarks whitespace normalization
func BenchmarkNormalizeWhitespace(b *testing.B) {
	b.Run("ShortText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NormalizeWhitespace(shortText)
		}
	})

	b.Run("MediumText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NormalizeWhitespace(mediumText)
		}
	})

	b.Run("LongText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NormalizeWhitespace(longText)
		}
	})
}

// BenchmarkStripControlChars benchmarks control character removal
func BenchmarkStripControlChars(b *testing.B) {
	b.Run("ShortText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			StripControlChars(shortText)
		}
	})

	b.Run("MediumText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			StripControlChars(mediumText)
		}
	})

	b.Run("LongText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			StripControlChars(longText)
		}
	})
}

// BenchmarkNormalizeText benchmarks the full text normalization process
func BenchmarkNormalizeText(b *testing.B) {
	b.Run("ShortText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NormalizeText(shortText)
		}
	})

	b.Run("MediumText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NormalizeText(mediumText)
		}
	})

	b.Run("LongText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NormalizeText(longText)
		}
	})
}

// BenchmarkDecodeHtmlEntities benchmarks HTML entity decoding
func BenchmarkDecodeHtmlEntities(b *testing.B) {
	b.Run("HtmlText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			DecodeHtmlEntities(htmlText)
		}
	})
}

// BenchmarkStripHTMLWhitespace benchmarks HTML whitespace removal
func BenchmarkStripHTMLWhitespace(b *testing.B) {
	b.Run("HtmlText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			StripHTMLWhitespace(htmlText)
		}
	})
}
