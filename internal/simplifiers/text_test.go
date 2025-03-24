package simplifiers

import (
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
		cat  string
		want bool
	}{
		{"Cc", "Cc", true},
		{"Cf", "Cf", true},
		{"Cn", "Cn", true},
		{"Co", "Co", true},
		{"Cs", "Cs", true},
		{"Lu", "Lu", false}, // Uppercase letter
		{"Ll", "Ll", false}, // Lowercase letter
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsControlCategory(tt.cat); got != tt.want {
				t.Errorf("IsControlCategory(%q) = %v, want %v", tt.cat, got, tt.want)
			}
		})
	}
}
