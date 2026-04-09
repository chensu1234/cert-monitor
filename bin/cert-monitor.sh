#!/bin/bash
#
# cert-monitor - SSL/TLS 证书监控与过期告警工具
# 作者: Chen Su
# 许可证: MIT
#
# 用法:
#   ./bin/cert-monitor.sh                    # 使用默认配置
#   ./bin/cert-monitor.sh -c config/domains.conf   # 指定配置文件
#   ./bin/cert-monitor.sh -i 3600            # 设置检查间隔(秒)
#   ./bin/cert-monitor.sh -w "https://hooks.slack.com/xxx"  # Slack 通知
#

set -euo pipefail

# ============================================================
# 颜色定义
# ============================================================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# ============================================================
# 默认配置 (可通过环境变量覆盖)
# ============================================================
CONFIG_FILE="${CONFIG_FILE:-./config/domains.conf}"
LOG_FILE="${LOG_FILE:-./log/cert-monitor.log}"
INTERVAL="${INTERVAL:-86400}"          # 默认检查间隔: 24小时
WARN_DAYS="${WARN_DAYS:-30}"            # 默认警告阈值: 30天
CRIT_DAYS="${CRIT_DAYS:-7}"             # 默认严重阈值: 7天
TIMEOUT="${TIMEOUT:-10}"                # 连接超时(秒)
NOTIFY_WEBHOOK="${NOTIFY_WEBHOOK:-}"    # Slack Webhook URL
NOTIFY_EMAIL="${NOTIFY_EMAIL:-}"        # 邮件通知(预留)
WATCH_MODE="${WATCH_MODE:-false}"       # 持续监控模式

# ============================================================
# 全局状态
# ============================================================
declare -A cert_expiry_cache    # 缓存证书到期时间
declare -A last_alert_time      # 防止重复告警
ALERT_COOLDOWN=3600             # 告警冷却时间(秒)

# ============================================================
# 帮助信息
# ============================================================
show_help() {
    cat << EOF
${BOLD}cert-monitor${NC} - SSL/TLS 证书监控与过期告警工具

${BOLD}用法:${NC}
    $(basename "$0") [选项]

${BOLD}选项:${NC}
    -c, --config FILE    配置文件路径 (默认: ./config/domains.conf)
    -i, --interval SEC   检查间隔秒数，仅监控模式有效 (默认: 86400)
    -w, --warn DAYS      警告阈值天数 (默认: 30)
    -k, --critical DAYS  严重阈值天数 (默认: 7)
    -t, --timeout SEC    连接超时秒数 (默认: 10)
    -w, --webhook URL    Slack Webhook URL
    -m, --monitor        持续监控模式
    -h, --help           显示帮助信息
    -v, --version        显示版本信息

${BOLD}示例:${NC}
    $(basename "$0") -c /etc/cert-monitor/domains.conf -i 3600
    $(basename "$0") --webhook https://hooks.slack.com/services/xxx
    $(basename "$0") --monitor --warn 14 --critical 3

${BOLD}配置文件格式:${NC}
    # domain:port  (默认端口 443 可省略)
    example.com
    example.com:443
    sub.example.com:8443

    # 支持 # 注释

${BOLD}退出码:${NC}
    0   所有证书正常
    1   部分证书过期或即将过期
    2   检查失败(网络/配置错误)

EOF
}

# 显示版本
show_version() {
    echo "cert-monitor v1.0.0"
}

# ============================================================
# 日志函数
# ============================================================
log() {
    local level="$1"
    shift
    local msg="[$(date '+%Y-%m-%d %H:%M:%S')] [$level] $*"
    echo -e "$msg" | tee -a "$LOG_FILE"
}

log_info()  { log "INFO" "$@"; }
log_warn()  { log "${YELLOW}WARN${NC}" "$@"; }
log_error() { log "${RED}ERROR${NC}" "$@"; }
log_ok()    { log "${GREEN}OK${NC}" "$@"; }

# ============================================================
# 解析命令行参数
# ============================================================
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -c|--config)
                CONFIG_FILE="$2"
                shift 2
                ;;
            -i|--interval)
                INTERVAL="$2"
                shift 2
                ;;
            --warn)
                WARN_DAYS="$2"
                shift 2
                ;;
            -w|--warn-days)
                WARN_DAYS="$2"
                shift 2
                ;;
            -k|--critical)
                CRIT_DAYS="$2"
                shift 2
                ;;
            -t|--timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            --webhook)
                NOTIFY_WEBHOOK="$2"
                shift 2
                ;;
            -m|--monitor)
                WATCH_MODE="true"
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            -v|--version)
                show_version
                exit 0
                ;;
            *)
                echo -e "${RED}未知选项: $1${NC}"
                show_help
                exit 1
                ;;
        esac
    done
}

# ============================================================
# 解析配置文件
# 返回格式: host:port 列表(空格分隔)
# ============================================================
parse_config() {
    if [[ ! -f "$CONFIG_FILE" ]]; then
        log_error "配置文件不存在: $CONFIG_FILE"
        exit 2
    fi

    local entries=()
    while IFS= read -r line; do
        # 跳过空行和纯注释行
        [[ -z "$line" ]] && continue
        [[ "$line" =~ ^[[:space:]]*# ]] && continue
        # 去除首尾空白
        line="$(echo "$line" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
        [[ -z "$line" ]] && continue

        entries+=("$line")
    done < "$CONFIG_FILE"

    printf '%s\n' "${entries[@]}"
}

# ============================================================
# 获取证书信息
# 返回: 证书到期时间戳
# ============================================================
get_cert_expiry() {
    local host="$1"
    local port="${2:-443}"

    # 尝试使用 openssl 获取证书到期时间
    # 使用 timeout 防止连接挂起
    local expiry_date
    expiry_date=$(echo | timeout "$TIMEOUT" openssl s_client -servername "$host" -connect "$host:$port" 2>/dev/null \
        | openssl x509 -noout -dates 2>/dev/null \
        | grep 'notAfter=' \
        | cut -d'=' -f2) || true

    if [[ -z "$expiry_date" ]]; then
        return 1
    fi

    # 转换为时间戳
    date -j -f "%b %d %H:%M:%S %Y %Z" "$expiry_date" +%s 2>/dev/null \
        || date --date="$expiry_date" +%s 2>/dev/null \
        || echo "$expiry_date"
}

# ============================================================
# 获取证书详情
# ============================================================
get_cert_details() {
    local host="$1"
    local port="${2:-443}"

    local cert_info
    cert_info=$(echo | timeout "$TIMEOUT" openssl s_client -servername "$host" -connect "$host:$port" 2>/dev/null \
        | openssl x509 -noout -subject -issuer -dates 2>/dev/null) || true

    echo "$cert_info"
}

# ============================================================
# 计算距离过期的天数
# ============================================================
days_until_expiry() {
    local expiry_ts="$1"
    local now_ts
    now_ts=$(date +%s)

    echo $(( (expiry_ts - now_ts) / 86400 ))
}

# ============================================================
# 格式化日期
# ============================================================
format_date() {
    local expiry_ts="$1"
    date -j -f %s "$expiry_ts" "+%Y-%m-%d %H:%M:%S" 2>/dev/null \
        || date -d "@$expiry_ts" "+%Y-%m-%d %H:%M:%S" 2>/dev/null \
        || echo "$expiry_ts"
}

# ============================================================
# 发送 Slack 通知
# ============================================================
send_slack_notification() {
    local color="$1"    # good / warning / danger
    local title="$2"
    local body="$3"
    local domain="$4"
    local days_left="$5"

    if [[ -z "$NOTIFY_WEBHOOK" ]]; then
        return 0
    fi

    # 根据剩余天数设置 emoji
    local emoji
    case "$days_left" in
        -[0-9]*) emoji=":rotating_light:" ;;
        0)       emoji=":warning:" ;;
        [0-6])   emoji=":warning:" ;;
        *)       emoji=":white_check_mark:" ;;
    esac

    local payload
    payload=$(cat << EOF
{
    "attachments": [
        {
            "color": "$color",
            "blocks": [
                {
                    "type": "header",
                    "text": {
                        "type": "plain_text",
                        "text": "$emoji $title",
                        "emoji": true
                    }
                },
                {
                    "type": "section",
                    "fields": [
                        {
                            "type": "mrkdwn",
                            "text": "*域名:*\n$domain"
                        },
                        {
                            "type": "mrkdwn",
                            "text": "*剩余天数:*\n${days_left}天"
                        }
                    ]
                },
                {
                    "type": "context",
                    "elements": [
                        {
                            "type": "mrkdwn",
                            "text": "$body"
                        }
                    ]
                }
            ]
        }
    ]
}
EOF
)

    curl -s -X POST "$NOTIFY_WEBHOOK" \
        -H 'Content-Type: application/json' \
        -d "$payload" > /dev/null 2>&1 || true
}

# ============================================================
# 检查是否应该发送告警(防重复)
# ============================================================
should_alert() {
    local domain="$1"
    local now
    now=$(date +%s)

    local last_alert="${last_alert_time[$domain]:-0}"

    if (( now - last_alert > ALERT_COOLDOWN )); then
        last_alert_time[$domain]="$now"
        return 0
    fi
    return 1
}

# ============================================================
# 检查单个域名证书
# ============================================================
check_cert() {
    local entry="$1"
    local host="${entry%:*}"
    local port="${entry#*:}"

    # 如果 port 等于 host，说明没有指定端口
    if [[ "$port" == "$host" ]]; then
        port="443"
    fi

    log_info "检查证书: ${host}:${port}"

    # 获取证书到期时间
    local expiry_ts
    expiry_ts=$(get_cert_expiry "$host" "$port") || {
        log_error "无法获取证书: ${host}:${port}"
        send_slack_notification "danger" "证书检查失败" "无法连接到 ${host}:${port}" "$host:$port" "N/A"
        return 1
    }

    local expiry_str
    expiry_str=$(format_date "$expiry_ts")

    local days_left
    days_left=$(days_until_expiry "$expiry_ts")

    # 计算百分比(假设证书有效期为395天)
    local total_days=395
    local used_days=$((total_days - days_left))
    local pct=$(( used_days * 100 / total_days ))
    (( pct > 100 )) && pct=100

    # 更新缓存
    cert_expiry_cache[$entry]="$days_left"

    # 根据剩余天数判断状态
    local status_color="${GREEN}"
    local status_text="${GREEN}正常${NC}"
    local severity="good"
    local exit_code=0

    if (( days_left < 0 )); then
        status_color="${RED}"
        status_text="${RED}已过期 ${days_left#-} 天${NC}"
        severity="danger"
        exit_code=1

        if should_alert "$entry"; then
            log_error "证书已过期: ${host}:${port} (过期 ${days_left#-} 天)"
            send_slack_notification "danger" "🚨 证书已过期!" "证书于 $expiry_str 过期，已过期 ${days_left#-} 天" "$host:$port" "$days_left"
        fi
    elif (( days_left <= CRIT_DAYS )); then
        status_color="${RED}"
        status_text="${RED}严重! 剩余 ${days_left} 天${NC}"
        severity="danger"
        exit_code=1

        if should_alert "$entry"; then
            log_warn "证书严重警告: ${host}:${port} (剩余 ${days_left} 天)"
            send_slack_notification "danger" "🔴 证书严重告警!" "证书将于 $expiry_str 到期，剩余 ${days_left} 天" "$host:$port" "$days_left"
        fi
    elif (( days_left <= WARN_DAYS )); then
        status_color="${YELLOW}"
        status_text="${YELLOW}警告 剩余 ${days_left} 天${NC}"
        severity="warning"
        exit_code=1

        if should_alert "$entry"; then
            log_warn "证书即将过期: ${host}:${port} (剩余 ${days_left} 天)"
            send_slack_notification "warning" "🟡 证书即将过期" "证书将于 $expiry_str 到期，剩余 ${days_left} 天" "$host:$port" "$days_left"
        fi
    else
        status_text="${GREEN}正常 剩余 ${days_left} 天${NC}"
        log_ok "证书正常: ${host}:${port} (到期: $expiry_str)"
    fi

    # 打印状态行
    printf "  %-45s %s\n" "${host}:${port}" "$status_text"

    return $exit_code
}

# ============================================================
# 打印 ASCII 进度条
# ============================================================
print_bar() {
    local pct="$1"
    local width=20
    local filled=$(( pct * width / 100 ))
    local empty=$(( width - filled ))

    printf "["
    printf "%${filled}s" | tr ' ' '█'
    printf "%${empty}s" | tr ' ' '░'
    printf "] %3d%%" "$pct"
}

# ============================================================
# 打印统计报告
# ============================================================
print_summary() {
    local total="$1"
    local ok="$2"
    local warn="$3"
    local crit="$4"
    local expired="$5"

    echo ""
    echo "========================================"
    echo -e "  ${BOLD}证书检查统计${NC}"
    echo "========================================"
    echo -e "  总计:    ${BOLD}${total}${NC}"
    echo -e "  ${GREEN}正常:${NC}    $ok"
    echo -e "  ${YELLOW}警告:${NC}    $warn"
    echo -e "  ${RED}严重:${NC}    $crit"
    echo -e "  ${RED}已过期:${NC}  $expired"
    echo "========================================"
}

# ============================================================
# 确保必要目录存在
# ============================================================
ensure_dirs() {
    mkdir -p "$(dirname "$LOG_FILE")" 2>/dev/null || true
    mkdir -p "$(dirname "$CONFIG_FILE")" 2>/dev/null || true
}

# ============================================================
# 主检查逻辑
# ============================================================
run_check() {
    ensure_dirs

    log_info "========================================="
    log_info "开始证书检查"
    log_info "配置文件: $CONFIG_FILE"
    log_info "警告阈值: ${WARN_DAYS} 天"
    log_info "严重阈值: ${CRIT_DAYS} 天"
    log_info "========================================="

    echo ""
    echo -e "${BOLD}${CYAN}cert-monitor - SSL/TLS 证书监控${NC}"
    echo -e "${CYAN}========================================${NC}"
    echo -e "检查时间: $(date '+%Y-%m-%d %H:%M:%S')"
    echo -e "警告阈值: ${WARN_DAYS} 天 | 严重阈值: ${CRIT_DAYS} 天"
    echo -e "${CYAN}========================================${NC}"
    echo ""

    # 解析配置
    mapfile -t entries < <(parse_config)

    if [[ ${#entries[@]} -eq 0 ]]; then
        log_error "配置文件中没有有效的域名条目"
        exit 2
    fi

    echo -e "${BOLD}域名                                          状态${NC}"
    echo "------------------------------------------------------------"

    local total=0 ok=0 warn=0 crit=0 expired=0
    local has_error=0

    for entry in "${entries[@]}"; do
        (( total++ ))
        check_cert "$entry" || {
            local ret=$?
            if (( ret == 1 )); then
                # 可能是过期或警告
                local days_left="${cert_expiry_cache[$entry]:-0}"
                if (( days_left < 0 )); then
                    (( expired++ ))
                elif (( days_left <= CRIT_DAYS )); then
                    (( crit++ ))
                elif (( days_left <= WARN_DAYS )); then
                    (( warn++ ))
                fi
            fi
            # 检查是否连接错误
            if (( ret == 1 )) && [[ "${cert_expiry_cache[$entry]:-}" == "" ]]; then
                (( has_error++ ))
            fi
        }

        # 检查返回值
        local days_left="${cert_expiry_cache[$entry]:-0}"
        if (( days_left > WARN_DAYS )); then
            (( ok++ ))
        elif (( days_left > CRIT_DAYS )); then
            (( warn++ ))
        elif (( days_left >= 0 )); then
            (( crit++ ))
        fi
    done

    print_summary "$total" "$ok" "$warn" "$crit" "$expired"

    log_info "检查完成"

    # 返回适当的退出码
    if (( expired > 0 || crit > 0 )); then
        return 1
    elif (( warn > 0 )); then
        return 1
    fi
    return 0
}

# ============================================================
# 持续监控模式
# ============================================================
run_monitor() {
    log_info "启动持续监控模式，间隔: ${INTERVAL} 秒"
    echo -e "${YELLOW}监控模式: 每 ${INTERVAL} 秒检查一次${NC}"
    echo -e "按 Ctrl+C 停止"
    echo ""

    while true; do
        run_check
        local result=$?

        echo ""
        echo -e "${BLUE}下次检查: $(date -v+${INTERVAL}S '+%Y-%m-%d %H:%M:%S' 2>/dev/null || date -d "+${INTERVAL} seconds" '+%Y-%m-%d %H:%M:%S') ${NC}"
        echo ""

        sleep "$INTERVAL"
    done
}

# ============================================================
# 主入口
# ============================================================
main() {
    parse_args "$@"
    ensure_dirs

    if [[ "$WATCH_MODE" == "true" ]]; then
        run_monitor
    else
        run_check
    fi
}

main "$@"
