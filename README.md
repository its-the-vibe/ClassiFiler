# ClassiFiler

ClassiFiler is a Go service that consumes file-result messages from [SlackFiler](https://github.com/its-the-vibe/SlackFiler), classifies each file using a configurable chain of classifiers, moves it to the appropriate target folder, and publishes the classification result to a Redis pub/sub channel for downstream consumers.

---

## Table of Contents

- [Requirements](#requirements)
- [How it Works](#how-it-works)
- [Configuration](#configuration)
- [Sensitive Credentials (.env)](#sensitive-credentials-env)
- [Classifiers](#classifiers)
- [Redis Message Formats](#redis-message-formats)
- [Running Locally](#running-locally)
- [Running with Docker Compose](#running-with-docker-compose)

---

## Requirements

- Go 1.24+
- An external Redis instance (not bundled with ClassiFiler)
- [SlackFiler](https://github.com/its-the-vibe/SlackFiler) running and writing to its output queue

---

## How it Works

1. The service polls a Redis list (`input_queue`) with `LPOP` every second.
2. For each message it receives from SlackFiler's output queue:
   - The filename is extracted from `target_file_path`.
   - The classifier chain is evaluated in order; the first match wins.
   - The file is moved to the matching classifier's `target_dir`.\
     Cross-device moves (different Docker volumes) are handled automatically.
   - A classification result is published to the Redis pub/sub `output_channel`.
3. If no classifier matches (and no `default` classifier is configured) the message is skipped with a warning.

---

## Configuration

Copy `config.example.yaml` to `config.yaml` and edit it:

```yaml
redis:
  host: "localhost"
  port: 6379
  input_queue: "slack_file_results"   # must match SlackFiler's output_list
  output_channel: "classifiler_results"

classifiers:
  - name: "images"
    type: "filename_regex"
    pattern: "(?i)\\.(jpg|jpeg|png|gif|bmp|svg|webp)$"
    target_dir: "/classified/images"

  - name: "documents"
    type: "filename_regex"
    pattern: "(?i)\\.(pdf|doc|docx|txt|md|csv)$"
    target_dir: "/classified/documents"

  - name: "default"
    type: "default"
    target_dir: "/classified/other"
```

`config.yaml` is gitignored; only `config.example.yaml` is committed.

The path to the config file defaults to `config.yaml` in the working directory.
Override it with the `CONFIG_PATH` environment variable.

---

## Sensitive Credentials (.env)

Copy `.env.example` to `.env` and fill in the values:

```
REDIS_PASSWORD=your-redis-password-here
```

`.env` is gitignored and must never be committed. In Docker Compose the file is
passed via `env_file`.

---

## Classifiers

Classifiers are evaluated in the order they appear in `config.yaml`. The first
one that matches the filename is used.

| Type             | Description                                              |
|------------------|----------------------------------------------------------|
| `filename_regex` | Matches the filename against a Go regular expression.    |
| `default`        | Catch-all fallback — always matches. Place it last.      |

Additional classifier types can be added by implementing the `classifier.Classifier`
interface in `internal/classifier/`.

---

## Redis Message Formats

### Input (LPOP from `input_queue`)

Produced by SlackFiler's output queue:

```json
{
  "file_info": {
    "id": "F0123456789",
    "name": "report.pdf",
    "title": "Q4 Report",
    "mimetype": "application/pdf",
    "size": 204800
  },
  "target_file_path": "/downloads/general/report.pdf"
}
```

### Output (Publish to `output_channel`)

Published to the configured Redis pub/sub channel after a successful classification:

```json
{
  "file_info": {
    "id": "F0123456789",
    "name": "report.pdf",
    "title": "Q4 Report",
    "mimetype": "application/pdf",
    "size": 204800
  },
  "original_path": "/downloads/general/report.pdf",
  "new_path": "/classified/documents/report.pdf",
  "classifier_name": "documents",
  "classified_at": "2025-01-15T12:34:56Z"
}
```

---

## Running Locally

```bash
# 1. Copy and edit the config
cp config.example.yaml config.yaml

# 2. Copy and edit the .env
cp .env.example .env

# 3. Build and run
go build -o classifiler .
./classifiler
```

Override the config path if needed:

```bash
CONFIG_PATH=/etc/classifiler/config.yaml ./classifiler
```

---

## Running with Docker Compose

```bash
# 1. Copy and edit config and .env (see above)

# 2. Ensure the external Redis network exists
docker network create redis_net

# 3. Edit docker-compose.yml to set the correct volume paths for your
#    SlackFiler download directories and classification target directories

# 4. Build and start
docker compose up --build -d
```

The service container runs **read-only** (`read_only: true`). The config file is
mounted read-only at `/etc/classifiler/config.yaml`. The download and
classification directories are mounted as separate read-write volumes.

> **Note:** the `source` and `destination` directories do not need to be on the
> same filesystem — ClassiFiler automatically handles cross-device moves.
