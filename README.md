# cert-monitor 🖥️🔒

> SSL/TLS 证书监控与过期告警工具

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Shell](https://img.shields.io/badge/Shell-Bash-green.svg)](https://www.gnu.org/software/bash/)
[![OpenSSL](https://img.shields.io/badge/OpenSSL-Required-blue.svg)](https://www.openssl.org/)

一个简单实用的 SSL/TLS 证书监控工具，检测网站证书有效期，异常时发送告警通知。

## ✨ 特性

- 🔒 **Bash 4.0+ 实现** - 零依赖，仅需 OpenSSL + curl
- ⚡ **支持多域名批量检查** - 单次检查任意数量域名
- 🔔 **多渠道告警** - Slack Webhook 通知支持
- ⏰ **智能阈值告警** - 可配置警告/严重双阈值
- 🛡️ **防重复告警** - 内置冷却机制避免告警风暴
- 📝 **详细日志记录** - 每次检查完整记录
- 🔄 **持续监控模式** - 定时自动检查
- 🌐 **支持任意端口** - 非标准端口也能检查

## 🏃 快速开始

### 环境要求

- Bash 4.0+ (macOS 需安装新版 Bash: `brew install bash`)
- OpenSSL
- curl (用于 Slack 通知)
- 网络连接

### 安装

```bash
# 克隆项目
git clone https://github.com/chensu1234/cert-monitor.git
cd cert-monitor

# 添加执行权限
chmod +x bin/cert-monitor.sh
```

### 快速使用

```bash
# 使用默认配置检查
./bin/cert-monitor.sh

# 指定配置文件
./bin/cert-monitor.sh -c /path/to/domains.conf

# 设置警告阈值(14天) 和 严重阈值(3天)
./bin/cert-monitor.sh --warn 14 --critical 3

# 启用 Slack 通知
./bin/cert-monitor.sh --webhook "https://hooks.slack.com/services/xxx"

# 持续监控模式 (每小时检查一次)
./bin/cert-monitor.sh --monitor --interval 3600
```

## ⚙️ 配置说明

### 配置文件

编辑 `config/domains.conf` 文件：

```bash
# 格式: domain:port (端口默认 443 可省略)
# 支持 # 注释

# 基础用法
example.com
example.com:443

# 非标准端口
api.example.com:8443
mail.example.com:993

# 注释
# 下面是需要监控的域名
github.com
google.com
cloudflare.com
```

### 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `CONFIG_FILE` | 配置文件路径 | `./config/domains.conf` |
| `LOG_FILE` | 日志文件路径 | `./log/cert-monitor.log` |
| `INTERVAL` | 检查间隔(秒) | `86400` |
| `WARN_DAYS` | 警告阈值(天) | `30` |
| `CRIT_DAYS` | 严重阈值(天) | `7` |
| `TIMEOUT` | 连接超时(秒) | `10` |
| `NOTIFY_WEBHOOK` | Slack Webhook URL | - |

### Slack 集成

1. 在 Slack 创建 Incoming Webhook: https://api.slack.com/messaging/webhooks
2. 获取 Webhook URL
3. 使用 `--webhook` 参数或设置 `NOTIFY_WEBHOOK` 环境变量

```bash
# 示例
./bin/cert-monitor.sh --webhook "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXX"
```

## 📋 命令行选项

| 选项 | 说明 | 默认值 |
|------|--------|--------|
| `-c, --config FILE` | 配置文件路径 | `./config/domains.conf` |
| `-i, --interval SEC` | 检查间隔秒数(监控模式) | `86400` |
| `-w, --warn DAYS` | 警告阈值天数 | `30` |
| `-k, --critical DAYS` | 严重阈值天数 | `7` |
| `-t, --timeout SEC` | 连接超时秒数 | `10` |
| `--webhook URL` | Slack Webhook URL | - |
| `-m, --monitor` | 启用持续监控模式 | `false` |
| `-h, --help` | 显示帮助信息 | - |
| `-v, --version` | 显示版本信息 | - |

## 📁 项目结构

```
cert-monitor/
├── bin/
│   └── cert-monitor.sh       # 主脚本
├── config/
│   └── domains.conf           # 域名配置
├── log/                       # 日志目录
│   └── .gitkeep
├── README.md
├── LICENSE
└── CHANGELOG.md
```

## 📝 日志说明

日志默认保存在 `./log/cert-monitor.log`，包含：

- 启动和配置信息
- 每个域名的检查结果
- 证书状态变化
- 告警发送记录
- 错误信息

```
[2026-04-09 12:00:00] [INFO] ========== 开始证书检查 ==========
[2026-04-09 12:00:01] [INFO] 检查证书: github.com:443
[2026-04-09 12:00:02] [OK] 证书正常: github.com:443 (到期: 2026-08-15 14:22:00)
```

## 🔔 告警规则

| 剩余天数 | 级别 | 颜色 | 说明 |
|---------|------|------|------|
| > 30 天 | ✅ 正常 | 绿色 | 无需操作 |
| 8-30 天 | ⚠️ 警告 | 黄色 | 建议续期 |
| 3-7 天 | 🔴 严重 | 红色 | 尽快续期 |
| < 0 天 | 🚨 过期 | 红色 | 已过期! |

## 🚀 使用场景

### 定时任务 (crontab)

```bash
# 每天早上 9 点检查一次
0 9 * * * /path/to/cert-monitor/bin/cert-monitor.sh -c /path/to/cert-monitor/config/domains.conf --webhook "https://hooks.slack.com/xxx" >> /path/to/cert-monitor/log/cert-monitor.log 2>&1
```

### Docker 部署

```bash
docker run -d \
  --name cert-monitor \
  -v $(pwd)/config:/app/config \
  -v $(pwd)/log:/app/log \
  -e NOTIFY_WEBHOOK="https://hooks.slack.com/xxx" \
  -e WARN_DAYS=14 \
  -e CRIT_DAYS=3 \
  your-docker-repo/cert-monitor
```

### Systemd 服务 (Linux)

```bash
# 创建 systemd 服务
sudo tee /etc/systemd/system/cert-monitor.service > /dev/null << EOF
[Unit]
Description=SSL Certificate Monitor
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$USER
WorkingDirectory=/path/to/cert-monitor
ExecStart=/path/to/cert-monitor/bin/cert-monitor.sh --monitor --interval 3600
Restart=on-failure
RestartSec=60

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable cert-monitor
sudo systemctl start cert-monitor
```

## 🔧 扩展计划

- [ ] 邮件通知支持
- [ ] 企业微信/钉钉通知
- [ ] Prometheus 指标导出
- [ ] 证书详情报告 (颁发者、SAN 等)
- [ ] 配置文件热重载
- [ ] 主动续期命令

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 👤 作者

Chen Su

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 🙏 致谢

- [OpenSSL](https://www.openssl.org/) - SSL/TLS 工具
- [Slack API](https://api.slack.com/) - 告警通知
