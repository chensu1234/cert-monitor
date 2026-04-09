# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-04-09

### Added
- Initial release
- Multi-domain SSL/TLS certificate monitoring
- Configurable warning (default 30 days) and critical (default 7 days) thresholds
- Slack Webhook notification support
- Alert cooldown mechanism to prevent notification storms
- Detailed logging to `./log/cert-monitor.log`
- Continuous monitoring mode with configurable interval
- Support for non-standard ports
- Comprehensive README with examples
- MIT License

### Features
- Pure Bash implementation (minimal dependencies)
- OpenSSL-based certificate expiry detection
- Color-coded terminal output
- Configurable connection timeout
- Environment variable configuration support
- Systemd service example for Linux deployment
- Cron job example for automated checks

### Coming Soon
- Email notification support
- Enterprise WeChat / DingTalk integration
- Prometheus metrics export
- Extended certificate details (issuer, SAN, etc.)
- Config hot-reload
- Auto-renewal commands
