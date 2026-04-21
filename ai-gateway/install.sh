#!/bin/bash
# ============================================
# ScoreRoute 一键安装脚本 v2.0.0
# 完全自动化，无需交互
# ============================================

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

DEFAULT_PORT=3000
PROJECT_NAME="ScoreRoute"
REPO_URL="https://github.com/DING-BAOMING/ScoreRoute.git"
INSTALL_DIR="${HOME}/scoreroute"

PORT=${PORT:-$DEFAULT_PORT}
NON_INTERACTIVE=${NON_INTERACTIVE:-false}

while [[ $# -gt 0 ]]; do
    case $1 in
        --port)
            PORT="$2"
            shift 2
            ;;
        --password)
            ADMIN_PASSWORD="$2"
            shift 2
            ;;
        --dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        --non-interactive)
            NON_INTERACTIVE=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [--port PORT] [--password PASS] [--dir DIR] [--non-interactive]"
            exit 0
            ;;
        *)
            echo "Unknown: $1"
            exit 1
            ;;
    esac
done

PORT=${PORT:-$DEFAULT_PORT}

show_banner() {
    echo -e "${GREEN}"
    echo "╔════════════════════════════════════════════╗"
    echo "║     ScoreRoute 一键安装脚本 v2.0.0         ║"
    echo "║     AI Gateway - 智能路由网关               ║"
    echo "╚════════════════════════════════════════════╝"
    echo -e "${NC}"
}

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_ok() { echo -e "${GREEN}[OK]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

check_docker() {
    log_info "检查 Docker..."
    command -v docker >/dev/null || { log_error "Docker未安装"; exit 1; }
    docker info >/dev/null 2>&1 || { log_error "Docker未运行"; exit 1; }
    if docker compose version >/dev/null 2>&1; then
        DC="docker compose"
    elif command -v docker-compose >/dev/null; then
        DC="docker-compose"
    else
        log_error "Docker Compose未安装"
        exit 1
    fi
    log_ok "Docker就绪"
}

setup_directory() {
    log_info "创建目录: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
    cd "$INSTALL_DIR"
    if [ -d ".git" ]; then
        git fetch origin && git reset --hard origin/main
    else
        git clone "$REPO_URL" .
    fi
    log_ok "目录就绪"
}

generate_config() {
    log_info "生成配置..."
    ADMIN_PASSWORD=${ADMIN_PASSWORD:-$(openssl rand -base64 16 | tr -dc 'a-zA-Z0-9' | head -c 16)}
    JWT_SECRET=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)
    cat > .env << EOF
PORT=${PORT}
DATABASE_PATH=./data/gateway.db
LOG_PATH=./logs
ADMIN_PASSWORD=${ADMIN_PASSWORD}
JWT_SECRET=${JWT_SECRET}
EOF
    mkdir -p data logs cache
    log_ok "密码: ${ADMIN_PASSWORD}"
}

update_docker_compose() {
    log_info "配置Docker..."
    sed -i "s/\${PORT:-3000}/${PORT}/" docker-compose.yml
    log_ok "Docker配置完成"
}

build_and_start() {
    log_info "构建镜像..."
    $DC down 2>/dev/null || true
    $DC up -d --build
    log_info "等待服务启动(60s)..."
    for i in $(seq 1 60); do
        curl -s http://localhost:${PORT}/health >/dev/null 2>&1 && { log_ok "服务启动成功"; break; }
        sleep 1
    done
}

show_complete() {
    echo ""
    echo "════════════════════════════════════"
    echo "  ScoreRoute 安装完成!"
    echo "════════════════════════════════════"
    echo ""
    echo "访问: http://localhost:${PORT}"
    echo "账号: admin"
    echo "密码: ${ADMIN_PASSWORD}"
    echo ""
    echo "命令: $DC ps / logs / restart / down"
}

main() {
    show_banner
    check_docker
    setup_directory
    generate_config
    update_docker_compose
    build_and_start
    show_complete
}

main
