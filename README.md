# cert-monitor

> 🔔 TLS/SSL certificate expiration monitoring tool — check, alert, and stay ahead of expired certificates.

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Build](https://img.shields.io/badge/Build-Passing-brightgreen.svg)]()

## ✨ Features

- **Fast concurrent checks** — checks multiple hosts simultaneously with configurable connection timeouts
- **Rich certificate analysis** — extracts issuer, CN, SANs, wildcard status, key algorithm, serial number
- **Multiple notification channels** — webhook (with Go templates), email (SMTP/TLS), Slack blocks
- **Daemon mode** — long-running HTTP server with a sleek web dashboard and scheduled checks
- **Issue severity levels** — `ok`, `info`, `warning`, `critical`, `expired`, `error`
- **Exit codes** — exit 0 (all OK), exit 1 (warnings), exit 2 (critical/expired)
- **YAML configuration** — full-featured config file with sensible defaults
- **JSON output** — machine-readable output for integration with monitoring pipelines
- **SHA-1 deprecation warning** — automatically flags certificates using deprecated SHA-1 signatures
- **Wildcard detection** — identifies wildcard certificates
- **No external dependencies** — pure Go standard library + minimal deps

## 🏃 Quick Start

### Installation

```bash
# From source
git clone https://github.com/YOUR_HANDLE/cert-monitor.git
cd cert-monitor
make build

# Download a binary from Releases
curl -L https://github.com/YOUR_HANDLE/cert-monitor/releases/latest/download/cert-monitor-darwin-arm64 -o cert-monitor
chmod +x cert-monitor
```

### Usage

```bash
# Check a single host
cert-monitor check google.com:443

# Check multiple hosts from CLI
cert-monitor check google.com:443 github.com:443 cloudflare.com:443

# Check using a config file
cert-monitor check --config config.yaml

# Check and output JSON
cert-monitor check google.com:443 github.com:443 --json

# Check and send notifications
cert-monitor check --config config.yaml --notify

# Generate a sample config
cert-monitor gen-config --output config.yaml

# Run as a daemon with web dashboard
cert-monitor serve --config config.yaml --addr :8080
```

### First-time Setup

```bash
# 1. Generate a configuration file
cert-monitor gen-config --output config.yaml

# 2. Edit config.yaml — add your targets and notification settings
vim config.yaml

# 3. Run a one-shot check
cert-monitor check --config config.yaml --notify
```

## ⚙️ Configuration

All settings are optional — only `targets` is required for checks.

```yaml
# config.yaml
targets:
  - "google.com:443"
  - "github.com:443"
  - "your-site.com:8443"

check:
  warn_days: 30       # Warn if cert expires within N days (default: 30)
  critical_days: 7    # Critical alert if within N days (default: 7)
  timeout: 10         # Connection timeout in seconds (default: 10)
  interval: 3600      # Check interval in seconds — daemon mode only (default: 3600)

notify:
  enabled: true

  # Generic HTTP webhook with Go template body
  webhook:
    enabled: false
    url: "https://your-webhook-endpoint.com/notify"
    method: "POST"
    headers:
      Authorization: "Bearer YOUR_TOKEN"
    body: |
      {
        "host": "{{.Host}}",
        "days_remaining": {{.DaysRemaining}},
        "issue": "{{.Issue}}"
      }

  # Email via SMTP
  email:
    enabled: false
    host: "smtp.example.com"
    port: 587
    username: "user"
    password: "pass"
    from: "cert-monitor@example.com"
    to: "admin@example.com"

  # Slack incoming webhook
  slack:
    enabled: false
    webhook: "https://hooks.slack.com/services/XXX/YYY/ZZZ"
    channel: "#alerts"

server:
  addr: ":8080"
  port: 8080

logging:
  level: "info"   # debug, info, warn, error
  file: ""         # leave empty for stdout
  format: "text"   # text or json
```

## 📋 Command Line Options

### Global Flags

| Flag | Alias | Default | Description |
|------|-------|---------|-------------|
| `--config` | `-c` | `config.yaml` | Path to configuration file |
| `--log-level` | `-l` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `--log-file` | `-o` | stdout | Log output file path |

### `check` Command

| Flag | Alias | Default | Description |
|------|-------|---------|-------------|
| `--warn-days` | `-w` | `30` | Warn if cert expires within N days |
| `--critical-days` | `-k` | `7` | Critical if cert expires within N days |
| `--json` | `-j` | `false` | Output results as JSON |
| `--notify` | `-n` | `false` | Send notifications for issues found |

### `serve` Command

| Flag | Alias | Default | Description |
|------|-------|---------|-------------|
| `--addr` | `-a` | `:8080` | HTTP server bind address |
| `--interval` | `-i` | `3600` | Check interval in seconds |

### `gen-config` Command

| Flag | Alias | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `config.yaml` | Output file path |

## 📁 Project Structure

```
cert-monitor/
├── README.md
├── CHANGELOG.md
├── LICENSE
├── Makefile
├── .gitignore
├── go.mod
├── go.sum
├── cmd/
│   └── cert-monitor/
│       ├── main.go          # CLI entry point, commands
│       └── server.go        # HTTP daemon + web dashboard
├── internal/
│   ├── checker/
│   │   ├── checker.go       # Core TLS certificate checking logic
│   │   └── output.go        # Table & JSON output formatters
│   ├── config/
│   │   └── config.go        # YAML config loading + defaults
│   └── notifier/
│       └── notifier.go      # Webhook, email, Slack dispatch
└── config/
    └── config.yaml.example  # Example configuration
```

## 🌐 HTTP API (Daemon Mode)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | Web dashboard |
| `/api/status` | GET | Summary counts by severity |
| `/api/results` | GET | Full check results as JSON |
| `/api/check` | POST | Trigger an immediate check |
| `/api/health` | GET | Health check (`{"status":"ok"}`) |

## 🚨 Exit Codes

| Code | Meaning |
|------|---------|
| `0` | All certificates OK (no issues) |
| `1` | One or more warnings or info-level issues |
| `2` | One or more critical, expired, or error states |
| `>2` | Fatal error (bad config, etc.) |

## 📝 CHANGELOG

All notable changes are documented in [CHANGELOG.md](CHANGELOG.md).

## 📄 License

MIT License — see [LICENSE](LICENSE) for details.

## 🔧 Development

```bash
# Build
make build

# Run tests
make test

# Clean build artifacts
make clean

# Format code
make fmt
```
