package simplifiers

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestIsRelevantImage(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		selector string
		expected bool
	}{
		{
			name:     "Relevant image with alt text",
			html:     `<img src="image.jpg" alt="A relevant image">`,
			selector: "img",
			expected: true,
		},
		{
			name:     "Small decorative image",
			html:     `<img src="icon.png" width="16" height="16">`,
			selector: "img",
			expected: false,
		},
		{
			name:     "Image with decorative pattern in src",
			html:     `<img src="logo.png" alt="">`,
			selector: "img",
			expected: false,
		},
		{
			name:     "Image in figure with caption",
			html:     `<figure><img src="image.jpg"><figcaption>Image caption</figcaption></figure>`,
			selector: "img",
			expected: true,
		},
		{
			name:     "Background image",
			html:     `<img src="bg.jpg" style="background-image: url('pattern.jpg');">`,
			selector: "img",
			expected: false,
		},
		{
			name:     "Image in content area",
			html:     `<div class="content"><img src="content-image.jpg"></div>`,
			selector: ".content img",
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(test.html))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}

			img := doc.Find(test.selector)
			result := IsRelevantImage(img)
			if result != test.expected {
				t.Errorf("Expected IsRelevantImage to return %v, got %v", test.expected, result)
			}
		})
	}
}

func TestProcessImages(t *testing.T) {
	html := `
		<div>
			<img src="relevant.jpg" alt="Relevant image">
			<img src="icon.png" width="16" height="16">
			<img src="logo.svg" alt="">
			<figure>
				<img src="figure-image.jpg">
				<figcaption>Image caption</figcaption>
			</figure>
		</div>
	`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	// Process images
	ProcessImages(doc)

	// Check that relevant images have the data-relevant-image attribute
	doc.Find("img[data-relevant-image='true']").Each(func(i int, s *goquery.Selection) {
		src, _ := s.Attr("src")
		if src != "relevant.jpg" && src != "figure-image.jpg" {
			t.Errorf("Unexpected image marked as relevant: %s", src)
		}
	})

	// Check that decorative images are removed
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		src, _ := s.Attr("src")
		if src == "icon.png" || src == "logo.svg" {
			t.Errorf("Decorative image not removed: %s", src)
		}
	})

	// Check that figure with relevant image has the data-has-relevant-image attribute
	figure := doc.Find("figure[data-has-relevant-image='true']")
	if figure.Length() == 0 {
		t.Errorf("Figure with relevant image not marked")
	}
}

func TestExtractImageCaption(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		selector string
		expected string
	}{
		{
			name:     "Image with figcaption",
			html:     `<figure><img src="image.jpg"><figcaption>Image caption</figcaption></figure>`,
			selector: "img",
			expected: "Image caption",
		},
		{
			name:     "Image with alt text",
			html:     `<img src="image.jpg" alt="Alt text">`,
			selector: "img",
			expected: "Alt text",
		},
		{
			name:     "Image with title",
			html:     `<img src="image.jpg" title="Image title">`,
			selector: "img",
			expected: "Image title",
		},
		{
			name:     "Image with no caption",
			html:     `<img src="image.jpg">`,
			selector: "img",
			expected: "",
		},
		{
			name:     "Image with alt and figcaption (figcaption preferred)",
			html:     `<figure><img src="image.jpg" alt="Alt text"><figcaption>Figure caption</figcaption></figure>`,
			selector: "img",
			expected: "Figure caption",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(test.html))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}

			img := doc.Find(test.selector)
			result := ExtractImageCaption(img)
			if result != test.expected {
				t.Errorf("Expected ExtractImageCaption to return %q, got %q", test.expected, result)
			}
		})
	}
}

func TestExtractImageMetadata(t *testing.T) {
	html := `
		<img src="image.jpg" alt="Alt text" title="Image title" width="800" height="600">
	`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	img := doc.Find("img")
	metadata := ExtractImageMetadata(img)

	// Check that all metadata is extracted
	expectedKeys := []string{"src", "alt", "title", "width", "height"}
	for _, key := range expectedKeys {
		if _, ok := metadata[key]; !ok {
			t.Errorf("Expected metadata to contain key %q", key)
		}
	}

	// Check specific values
	if metadata["src"] != "image.jpg" {
		t.Errorf("Expected src to be %q, got %q", "image.jpg", metadata["src"])
	}
	if metadata["alt"] != "Alt text" {
		t.Errorf("Expected alt to be %q, got %q", "Alt text", metadata["alt"])
	}
	if metadata["title"] != "Image title" {
		t.Errorf("Expected title to be %q, got %q", "Image title", metadata["title"])
	}
	if metadata["width"] != "800" {
		t.Errorf("Expected width to be %q, got %q", "800", metadata["width"])
	}
	if metadata["height"] != "600" {
		t.Errorf("Expected height to be %q, got %q", "600", metadata["height"])
	}
}

func TestEnhanceImages(t *testing.T) {
	html := `
		<div>
			<img src="image1.jpg" alt="Image 1">
			<img src="image2.jpg" alt="Image 2">
			<img src="icon.png" width="16" height="16">
		</div>
	`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	// Enhance images
	EnhanceImages(doc)

	// Check that relevant images have lazy loading
	doc.Find("img[data-relevant-image='true']").Each(func(i int, s *goquery.Selection) {
		loading, exists := s.Attr("loading")
		if !exists || loading != "lazy" {
			t.Errorf("Expected relevant image to have loading='lazy'")
		}
	})

	// Check that relevant images have responsive class
	doc.Find("img[data-relevant-image='true']").Each(func(i int, s *goquery.Selection) {
		class, exists := s.Attr("class")
		if !exists || class != "img-fluid" {
			t.Errorf("Expected relevant image to have class='img-fluid'")
		}
	})

	// Check that decorative images are removed
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		src, _ := s.Attr("src")
		if src == "icon.png" {
			t.Errorf("Decorative image not removed: %s", src)
		}
	})
}
