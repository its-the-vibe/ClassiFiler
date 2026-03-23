package classifier

// DefaultClassifier is a fallback classifier that matches every file.
// Place it last in the classifier chain.
type DefaultClassifier struct {
	name      string
	targetDir string
}

// NewDefaultClassifier returns a DefaultClassifier with the given name and
// target directory.
func NewDefaultClassifier(name, targetDir string) *DefaultClassifier {
	return &DefaultClassifier{name: name, targetDir: targetDir}
}

// Name returns the classifier name.
func (c *DefaultClassifier) Name() string { return c.name }

// TargetDir returns the target directory for matched files.
func (c *DefaultClassifier) TargetDir() string { return c.targetDir }

// Classify always returns true, making this a catch-all fallback.
func (c *DefaultClassifier) Classify(_ string) bool { return true }
