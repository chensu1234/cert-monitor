# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] — 2026-04-21

### Added
- Initial release
- `check` command for one-shot TLS certificate expiration checks
- `serve` command for long-running HTTP daemon with web dashboard
- `gen-config` command for generating a sample configuration file
- Concurrent host checking with configurable timeouts
- Rich certificate analysis: issuer, CN, SANs, wildcard status, key algorithm, serial number, signature algorithm
- Issue severity levels: `ok`, `info`, `warning`, `critical`, `expired`, `error`
- Notification support: webhook (Go templates), email (SMTP/TLS), Slack (Block Kit)
- YAML configuration file with sensible defaults
- JSON output mode for machine-readable results
- Tabular ASCII output for human readability
- Exit codes for integration with monitoring systems
- SHA-1 deprecation warning (flags SHA-1 signed certificates)
- Built-in HTTP web dashboard with live refresh
- REST API: `/api/status`, `/api/results`, `/api/check`, `/api/health`
- Graceful shutdown support
- Comprehensive test coverage for core checking logic
