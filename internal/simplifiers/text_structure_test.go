package simplifiers

import (
	"testing"
)

// TestExtractParagraphs tests the ExtractParagraphs function
// This test is commented out because it requires the goquery package
// and will be enabled when we have proper imports working
/*
func TestExtractParagraphs(t *testing.T) {
	// Test implementation will go here
}
*/

func TestCountSentences(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "Empty text",
			text:     "",
			expected: 0,
		},
		{
			name:     "Single sentence with period",
			text:     "This is a sentence.",
			expected: 1,
		},
		{
			name:     "Single sentence with exclamation mark",
			text:     "This is a sentence!",
			expected: 1,
		},
		{
			name:     "Single sentence with question mark",
			text:     "Is this a sentence?",
			expected: 1,
		},
		{
			name:     "Multiple sentences with different terminators",
			text:     "This is the first sentence. This is the second sentence! Is this the third sentence?",
			expected: 3,
		},
		{
			name:     "Sentence without terminator",
			text:     "This is a sentence without a terminator",
			expected: 1,
		},
		{
			name:     "Mixed sentences with and without terminators",
			text:     "This is the first sentence. This is the second sentence without a terminator",
			expected: 2,
		},
		{
			name:     "Sentences with numbers",
			text:     "This is sentence 1. This is sentence 2.",
			expected: 2,
		},
		{
			name:     "Sentences with abbreviations",
			text:     "Dr. Smith went to the store. He bought some apples.",
			expected: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := CountSentences(test.text)
			if result != test.expected {
				t.Errorf("Expected %d sentences, got %d", test.expected, result)
			}
		})
	}
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "Empty text",
			text:     "",
			expected: 0,
		},
		{
			name:     "Single word",
			text:     "Word",
			expected: 1,
		},
		{
			name:     "Multiple words",
			text:     "These are multiple words",
			expected: 4,
		},
		{
			name:     "Words with punctuation",
			text:     "This, is a sentence. With punctuation!",
			expected: 6,
		},
		{
			name:     "Words with extra whitespace",
			text:     "  Words  with  extra  whitespace  ",
			expected: 4,
		},
		{
			name:     "Words with numbers",
			text:     "There are 3 words here",
			expected: 5,
		},
		{
			name:     "Words with mixed case",
			text:     "These Words Have Mixed CASE",
			expected: 5,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := CountWords(test.text)
			if result != test.expected {
				t.Errorf("Expected %d words, got %d", test.expected, result)
			}
		})
	}
}

func TestCalculateReadingLevel(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected float64
		delta    float64 // Allowed difference due to floating point precision
	}{
		{
			name:     "Empty text",
			text:     "",
			expected: 0,
			delta:    0.01,
		},
		{
			name:     "Simple text",
			text:     "This is a simple text. It has short sentences.",
			expected: 2.5,
			delta:    2.0, // Allow some variance due to syllable counting differences
		},
		{
			name:     "Complex text",
			text:     "The complexity of this particular text is significantly higher than the previous example. It contains longer sentences with more sophisticated vocabulary and intricate grammatical structures.",
			expected: 15.0,
			delta:    5.0, // Allow some variance due to syllable counting differences
		},
		{
			name:     "Text with no sentences",
			text:     "no sentences",
			expected: 0,
			delta:    0.01,
		},
		{
			name:     "Text with no words",
			text:     ".",
			expected: 0,
			delta:    0.01,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := CalculateReadingLevel(test.text)
			diff := result - test.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > test.delta {
				t.Errorf("Expected reading level around %f (±%f), got %f", test.expected, test.delta, result)
			}
		})
	}
}

func TestCountSyllables(t *testing.T) {
	tests := []struct {
		name     string
		word     string
		expected int
	}{
		{
			name:     "One syllable word",
			word:     "cat",
			expected: 1,
		},
		{
			name:     "Two syllable word",
			word:     "hello",
			expected: 2,
		},
		{
			name:     "Three syllable word",
			word:     "beautiful",
			expected: 3,
		},
		{
			name:     "Word with silent e",
			word:     "make",
			expected: 1,
		},
		{
			name:     "Empty word",
			word:     "",
			expected: 1, // Minimum is 1
		},
		{
			name:     "Word with no vowels",
			word:     "rhythm",
			expected: 2, // 'y' counts as a vowel here
		},
		{
			name:     "Word with consecutive vowels",
			word:     "read",
			expected: 1,
		},
		{
			name:     "Word with 'y' as vowel",
			word:     "happy",
			expected: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := countSyllables(test.word)
			if result != test.expected {
				t.Errorf("Expected %d syllables for %q, got %d", test.expected, test.word, result)
			}
		})
	}
}

func TestIsVowelRune(t *testing.T) {
	vowels := []rune{'a', 'e', 'i', 'o', 'u', 'y'}
	consonants := []rune{'b', 'c', 'd', 'f', 'g', 'h', 'j', 'k', 'l', 'm', 'n', 'p', 'q', 'r', 's', 't', 'v', 'w', 'x', 'z'}

	for _, vowel := range vowels {
		if !isVowelRune(vowel) {
			t.Errorf("Expected %c to be a vowel", vowel)
		}
	}

	for _, consonant := range consonants {
		if isVowelRune(consonant) {
			t.Errorf("Expected %c to be a consonant", consonant)
		}
	}
}
