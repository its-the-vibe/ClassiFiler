package classifier

import (
	"fmt"
	"regexp"
)

// FilenameClassifier matches filenames against a configurable regular expression.
type FilenameClassifier struct {
	name      string
	pattern   *regexp.Regexp
	targetDir string
}

// NewFilenameClassifier compiles pattern and returns a FilenameClassifier.
// An error is returned if pattern is not a valid regular expression.
func NewFilenameClassifier(name, pattern, targetDir string) (*FilenameClassifier, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("compiling pattern %q: %w", pattern, err)
	}
	return &FilenameClassifier{
		name:      name,
		pattern:   re,
		targetDir: targetDir,
	}, nil
}

// Name returns the classifier name.
func (c *FilenameClassifier) Name() string { return c.name }

// TargetDir returns the target directory for matched files.
func (c *FilenameClassifier) TargetDir() string { return c.targetDir }

// Classify returns true if filename matches the configured pattern.
func (c *FilenameClassifier) Classify(filename string) bool {
	return c.pattern.MatchString(filename)
}
