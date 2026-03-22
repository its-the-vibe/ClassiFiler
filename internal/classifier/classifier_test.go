package classifier_test

import (
	"testing"

	"github.com/its-the-vibe/classifiler/internal/classifier"
)

func TestFilenameClassifier_Match(t *testing.T) {
	c, err := classifier.NewFilenameClassifier("images", `(?i)\.(jpg|jpeg|png|gif)$`, "/target/images")
	if err != nil {
		t.Fatalf("unexpected error creating classifier: %v", err)
	}

	tests := []struct {
		filename string
		want     bool
	}{
		{"photo.jpg", true},
		{"photo.JPG", true},
		{"photo.jpeg", true},
		{"image.PNG", true},
		{"animation.gif", true},
		{"document.pdf", false},
		{"archive.tar.gz", false},
		{"photo.jpg.exe", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			if got := c.Classify(tt.filename); got != tt.want {
				t.Errorf("Classify(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestFilenameClassifier_InvalidPattern(t *testing.T) {
	_, err := classifier.NewFilenameClassifier("bad", `[invalid`, "/target")
	if err == nil {
		t.Error("expected error for invalid regex pattern, got nil")
	}
}

func TestFilenameClassifier_Metadata(t *testing.T) {
	c, _ := classifier.NewFilenameClassifier("docs", `\.pdf$`, "/docs")
	if c.Name() != "docs" {
		t.Errorf("Name() = %q, want %q", c.Name(), "docs")
	}
	if c.TargetDir() != "/docs" {
		t.Errorf("TargetDir() = %q, want %q", c.TargetDir(), "/docs")
	}
}

func TestDefaultClassifier_AlwaysMatches(t *testing.T) {
	c := classifier.NewDefaultClassifier("default", "/other")

	filenames := []string{"anything.txt", "whatever.xyz", "no-extension", ""}
	for _, f := range filenames {
		if !c.Classify(f) {
			t.Errorf("DefaultClassifier.Classify(%q) = false, want true", f)
		}
	}
}

func TestDefaultClassifier_Metadata(t *testing.T) {
	c := classifier.NewDefaultClassifier("fallback", "/fallback")
	if c.Name() != "fallback" {
		t.Errorf("Name() = %q, want %q", c.Name(), "fallback")
	}
	if c.TargetDir() != "/fallback" {
		t.Errorf("TargetDir() = %q, want %q", c.TargetDir(), "/fallback")
	}
}

func TestChain_FirstMatch(t *testing.T) {
	img, _ := classifier.NewFilenameClassifier("images", `(?i)\.(jpg|png)$`, "/images")
	doc, _ := classifier.NewFilenameClassifier("documents", `(?i)\.pdf$`, "/docs")
	def := classifier.NewDefaultClassifier("default", "/other")

	chain := []classifier.Classifier{img, doc, def}

	if got := classifier.Chain(chain, "photo.jpg"); got == nil || got.Name() != "images" {
		t.Errorf("expected 'images' classifier for photo.jpg, got %v", got)
	}
	if got := classifier.Chain(chain, "report.pdf"); got == nil || got.Name() != "documents" {
		t.Errorf("expected 'documents' classifier for report.pdf, got %v", got)
	}
	if got := classifier.Chain(chain, "unknown.xyz"); got == nil || got.Name() != "default" {
		t.Errorf("expected 'default' classifier for unknown.xyz, got %v", got)
	}
}

func TestChain_EmptyChain(t *testing.T) {
	if got := classifier.Chain(nil, "file.txt"); got != nil {
		t.Errorf("Chain(nil, ...) = %v, want nil", got)
	}
	if got := classifier.Chain([]classifier.Classifier{}, "file.txt"); got != nil {
		t.Errorf("Chain(empty, ...) = %v, want nil", got)
	}
}

func TestChain_NoMatch(t *testing.T) {
	img, _ := classifier.NewFilenameClassifier("images", `(?i)\.(jpg|png)$`, "/images")
	chain := []classifier.Classifier{img}

	if got := classifier.Chain(chain, "document.pdf"); got != nil {
		t.Errorf("expected nil, got %q", got.Name())
	}
}
