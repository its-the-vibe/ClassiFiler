// Package classifier provides the Classifier interface and a chain runner.
package classifier

// Classifier classifies a file by its name and returns whether it matches.
type Classifier interface {
	// Name returns the human-readable identifier of the classifier.
	Name() string
	// Classify returns true if the classifier matches the given filename.
	Classify(filename string) bool
	// TargetDir returns the directory files matching this classifier are moved to.
	TargetDir() string
}

// Chain iterates classifiers in order and returns the first one that matches
// filename. It returns nil if no classifier matches.
func Chain(classifiers []Classifier, filename string) Classifier {
	for _, c := range classifiers {
		if c.Classify(filename) {
			return c
		}
	}
	return nil
}
