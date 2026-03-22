package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"github.com/its-the-vibe/classifiler/internal/classifier"
	"github.com/its-the-vibe/classifiler/internal/config"
)

// InputMessage is the message format produced by SlackFiler's output queue.
type InputMessage struct {
	FileInfo       FileInfo `json:"file_info"`
	TargetFilePath string   `json:"target_file_path"`
}

// FileInfo contains Slack file metadata forwarded by SlackFiler.
type FileInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Title    string `json:"title"`
	Mimetype string `json:"mimetype"`
	Size     int64  `json:"size"`
}

// OutputMessage is published to the Redis pub/sub channel after classification.
type OutputMessage struct {
	FileInfo       FileInfo  `json:"file_info"`
	OriginalPath   string    `json:"original_path"`
	NewPath        string    `json:"new_path"`
	ClassifierName string    `json:"classifier_name"`
	ClassifiedAt   time.Time `json:"classified_at"`
}

func main() {
	// Load .env if present; ignore error when file is absent.
	_ = godotenv.Load()

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	classifiers, err := buildClassifiers(cfg)
	if err != nil {
		slog.Error("failed to build classifiers", "error", err)
		os.Exit(1)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: os.Getenv("REDIS_PASSWORD"),
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to Redis", "host", cfg.Redis.Host, "port", cfg.Redis.Port)
	slog.Info("ClassiFiler started",
		"input_queue", cfg.Redis.InputQueue,
		"output_channel", cfg.Redis.OutputChannel,
	)

	run(ctx, rdb, cfg, classifiers)
	slog.Info("ClassiFiler stopped")
}

// run is the main consumer loop. It polls the Redis input queue and processes
// each message until ctx is cancelled.
func run(ctx context.Context, rdb *redis.Client, cfg *config.Config, classifiers []classifier.Classifier) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		result, err := rdb.LPop(ctx, cfg.Redis.InputQueue).Result()
		if errors.Is(err, redis.Nil) {
			// Queue is empty; wait before polling again.
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("failed to pop from queue", "queue", cfg.Redis.InputQueue, "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}

		if err := processMessage(ctx, rdb, cfg, classifiers, result); err != nil {
			slog.Error("failed to process message", "error", err)
		}
	}
}

// buildClassifiers constructs the classifier chain from the config.
func buildClassifiers(cfg *config.Config) ([]classifier.Classifier, error) {
	classifiers := make([]classifier.Classifier, 0, len(cfg.Classifiers))
	for _, cc := range cfg.Classifiers {
		switch cc.Type {
		case "filename_regex":
			c, err := classifier.NewFilenameClassifier(cc.Name, cc.Pattern, cc.TargetDir)
			if err != nil {
				return nil, fmt.Errorf("building classifier %q: %w", cc.Name, err)
			}
			classifiers = append(classifiers, c)
		case "default":
			classifiers = append(classifiers, classifier.NewDefaultClassifier(cc.Name, cc.TargetDir))
		default:
			return nil, fmt.Errorf("unknown classifier type %q for classifier %q", cc.Type, cc.Name)
		}
	}
	return classifiers, nil
}

// processMessage classifies a single file, moves it, and publishes the result.
func processMessage(ctx context.Context, rdb *redis.Client, cfg *config.Config, classifiers []classifier.Classifier, raw string) error {
	var msg InputMessage
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		return fmt.Errorf("parsing message: %w", err)
	}

	filename := filepath.Base(msg.TargetFilePath)
	if filename == "" || filename == "." {
		filename = msg.FileInfo.Name
	}

	matched := classifier.Chain(classifiers, filename)
	if matched == nil {
		slog.Warn("no classifier matched, skipping", "file", filename)
		return nil
	}

	if err := os.MkdirAll(matched.TargetDir(), 0o755); err != nil {
		return fmt.Errorf("creating target directory %q: %w", matched.TargetDir(), err)
	}

	newPath := filepath.Join(matched.TargetDir(), filename)
	if err := moveFile(msg.TargetFilePath, newPath); err != nil {
		return fmt.Errorf("moving file from %q to %q: %w", msg.TargetFilePath, newPath, err)
	}

	slog.Info("classified file",
		"file", filename,
		"classifier", matched.Name(),
		"from", msg.TargetFilePath,
		"to", newPath,
	)

	out := OutputMessage{
		FileInfo:       msg.FileInfo,
		OriginalPath:   msg.TargetFilePath,
		NewPath:        newPath,
		ClassifierName: matched.Name(),
		ClassifiedAt:   time.Now().UTC(),
	}
	outJSON, err := json.Marshal(out)
	if err != nil {
		return fmt.Errorf("marshaling output message: %w", err)
	}

	if err := rdb.Publish(ctx, cfg.Redis.OutputChannel, string(outJSON)).Err(); err != nil {
		return fmt.Errorf("publishing result to %q: %w", cfg.Redis.OutputChannel, err)
	}

	return nil
}

// moveFile moves src to dst. It tries an atomic os.Rename first and falls back
// to a copy-then-delete when the source and destination are on different
// filesystems (EXDEV).
func moveFile(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// Handle cross-device link error (different filesystems / Docker volumes).
	var linkErr *os.LinkError
	if !errors.As(err, &linkErr) || !errors.Is(linkErr.Err, syscall.EXDEV) {
		return err
	}

	return copyAndDelete(src, dst)
}

// copyAndDelete copies src to dst and removes src on success.
func copyAndDelete(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source: %w", err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("creating destination: %w", err)
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return fmt.Errorf("copying data: %w", err)
	}

	if err := out.Sync(); err != nil {
		out.Close()
		os.Remove(dst)
		return fmt.Errorf("syncing destination: %w", err)
	}

	if err := out.Close(); err != nil {
		os.Remove(dst)
		return fmt.Errorf("closing destination: %w", err)
	}

	if err := os.Remove(src); err != nil {
		slog.Warn("failed to remove source file after copy", "path", src, "error", err)
	}

	return nil
}
