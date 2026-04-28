#!/bin/bash
# ScoreRoute一键安装脚本v2.28.0 完全自动化
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

DEFAULT_PORT=3000
REPO_URL="https://github.com/DING-BAOMING/ScoreRoute.git"
GITEE_URL="https://gitee.com/BM-D/ScoreRoute.git"

CURRENT_DIR="$(cd "$(dirname "$0")" && pwd)"
INSTALL_DIR="${CURRENT_DIR}/scoreroute"

PORT="${DEFAULT_PORT}"
export NON_INTERACTIVE="false"
USE_GITEE="false"
ADMIN_PASSWORD=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --port) PORT="$2"; shift 2 ;;
        --password) ADMIN_PASSWORD="$2"; shift 2 ;;
        --dir) INSTALL_DIR="$2"; shift 2 ;;
        --non-interactive) NON_INTERACTIVE=true; shift ;;
        --help|-h) echo "Usage: $0 [--port PORT] [--password PASS] [--dir DIR] [--non-interactive] [--gitee]"; exit 0 ;;
        --gitee) USE_GITEE=true; shift ;;
        *) echo "Unknown: $1"; exit 1 ;;
    esac
done

show_banner() {
    echo -e "${GREEN}ScoreRoute一键安装v2.28.0${NC}"
}

log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_ok() { echo -e "${GREEN}[OK]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

check_docker() {
    log_info "检查Docker..."
    command -v docker >/dev/null || log_error "Docker未安装"
    docker info >/dev/null 2>&1 || log_error "Docker未运行"
    if docker compose version >/dev/null 2>&1; then
        DC="docker compose"
    elif command -v docker-compose >/dev/null; then
        DC="docker-compose"
    else
        log_error "Docker Compose未安装"
    fi
    log_ok "Docker就绪"
}

validate_port() {
    if ! [[ "$PORT" =~ ^[0-9]+$ ]] || [ "$PORT" -lt 1 ] || [ "$PORT" -gt 65535 ]; then
        log_error "端口必须是1-65535之间的数字，当前值: $PORT"
    fi
}

setup_directory() {
    log_info "创建目录: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
    cd "$INSTALL_DIR"
    
    if [ -d ".git" ]; then
        log_info "更新代码..."
        git fetch origin && git reset --hard origin/main
    else
        log_info "克隆代码..."
        if [ "$USE_GITEE" = "true" ]; then
            git clone "$GITEE_URL" .
        else
            git clone "$REPO_URL" .
        fi
    fi
    
    if [ -d "ai-gateway" ] && [ ! -f "docker-compose.yml" ]; then
        if [ -f "ai-gateway/docker-compose.yml" ]; then
            log_info "移动ai-gateway内容到当前目录..."
            mv ai-gateway/* ai-gateway/.* . 2>/dev/null || true
            rmdir ai-gateway 2>/dev/null || true
        else
            log_info "进入ai-gateway目录..."
            cd ai-gateway
        fi
    fi
    
    log_ok "目录就绪: $(pwd)"
}

generate_config() {
    log_info "生成配置..."
    ADMIN_PASSWORD="${ADMIN_PASSWORD:-$(openssl rand -base64 16 | tr -dc 'a-zA-Z0-9' | head -c 16)}"
    # Generate secure JWT secret with fallback
    if command -v python3 >/dev/null 2>&1; then
        JWT_SECRET=$(python3 -c "import secrets; print(secrets.token_urlsafe(32))")
    else
        JWT_SECRET=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)
    fi

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
    if [ -f "docker-compose.yml" ]; then
        sed -i "s/\${PORT:-3000}/${PORT}/" docker-compose.yml
        log_ok "Docker配置完成"
    else
        log_error "未找到docker-compose.yml，请检查目录结构"
    fi
}

build_and_start() {
    log_info "构建镜像(首次可能需要2-5分钟)..."
    "$DC" down 2>/dev/null || true
    "$DC" up -d --build
    log_info "等待服务启动..."
    for i in $(seq 1 120); do
        if curl -s "http://localhost:${PORT}/health" >/dev/null 2>&1; then
            log_ok "服务启动成功"
            return 0
        fi
        if [ "$i" -eq 30 ] || [ "$i" -eq 60 ] || [ "$i" -eq 90 ]; then
            log_info "仍在构建中，请耐心等待..."
        fi
        sleep 1
    done
    log_error "服务启动超时(>120秒)，请检查Docker日志: docker logs ai-gateway-app-1"
}

show_complete() {
    echo "====================================="
    echo "ScoreRoute安装完成!"
    echo "====================================="
    echo "安装目录: $(pwd)"
    echo "访问地址: http://localhost:${PORT}"
    echo "API端点: http://localhost:${PORT}/v1"
    echo "账号: admin"
    echo "密码: ${ADMIN_PASSWORD}"
}

show_banner
validate_port
check_docker
setup_directory
generate_config
update_docker_compose
build_and_start
show_complete
