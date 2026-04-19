#!/bin/bash
# ============================================
# ScoreRoute 一键安装脚本
# ============================================

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

DEFAULT_PORT=3000
PROJECT_NAME="ScoreRoute"
REPO_URL="https://github.com/DING-BAOMING/ScoreRoute.git"
INSTALL_DIR="${HOME}/scoreroute"

show_banner() {
    echo -e "${GREEN}"
    echo "╔════════════════════════════════════════════╗"
    echo "║     ScoreRoute 一键安装脚本 v1.0.1           ║"
    echo "║     AI Gateway - 智能路由网关               ║"
    echo "╚════════════════════════════════════════════╝"
    echo -e "${NC}"
}

check_port() {
    local port=$1
    echo -e "${YELLOW}[检查] 检测端口 ${port}...${NC}"
    
    if lsof -Pi :${port} -sTCP:LISTEN -t &> /dev/null; then
        echo -e "${RED}✗ 端口 ${port} 已被占用${NC}"
        return 1
    fi
    
    echo -e "${GREEN}✓ 端口 ${port} 可用${NC}"
    return 0
}

check_docker() {
    echo -e "${YELLOW}[1/7] 检查 Docker 环境...${NC}"
    
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}✗ Docker 未安装${NC}"
        exit 1
    fi
    
    if ! docker compose version &> /dev/null && ! command -v docker-compose &> /dev/null; then
        echo -e "${RED}✗ Docker Compose 未安装${NC}"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        echo -e "${RED}✗ Docker 服务未运行${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✓ Docker 环境检查通过${NC}"
}

select_port() {
    echo -e "${YELLOW}[2/7] 选择端口...${NC}"
    
    read -p "请输入端口号 [${DEFAULT_PORT}]: " PORT
    PORT=${PORT:-$DEFAULT_PORT}
    
    if ! [[ "$PORT" =~ ^[0-9]+$ ]] || [ "$PORT" -lt 1 ] || [ "$PORT" -gt 65535 ]; then
        echo -e "${RED}✗ 无效端口号${NC}"
        exit 1
    fi
    
    if ! check_port $PORT; then
        echo -e "${YELLOW}请选择其他端口或停止占用端口的服务${NC}"
        select_port
    fi
}

setup_directory() {
    echo -e "${YELLOW}[3/7] 准备安装目录...${NC}"
    
    if [ -d "$INSTALL_DIR" ]; then
        echo -e "${YELLOW}检测到已有安装目录${NC}"
        read -p "是否要重新安装？(y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "安装已取消"
            exit 0
        fi
        rm -rf "$INSTALL_DIR"
    fi
    
    mkdir -p "$INSTALL_DIR"
    echo -e "${GREEN}✓ 创建目录: $INSTALL_DIR${NC}"
    echo -e "${YELLOW}正在克隆代码...${NC}"
    git clone "$REPO_URL" "$INSTALL_DIR"
    cd "$INSTALL_DIR/ai-gateway"
    echo -e "${GREEN}✓ 代码准备完成: $(pwd)${NC}"
}

generate_config() {
    echo -e "${YELLOW}[4/7] 生成安全配置...${NC}"
    
    ADMIN_PASSWORD=$(openssl rand -base64 16 | tr -dc 'a-zA-Z0-9' | head -c 16)
    JWT_SECRET=$(openssl rand -base64 32)
    
    cat > .env << ENVEOF
PORT=${PORT}
DATABASE_PATH=./data/gateway.db
LOG_PATH=./logs

ADMIN_PASSWORD=${ADMIN_PASSWORD}
JWT_SECRET=${JWT_SECRET}
ENVEOF
    
    mkdir -p data logs cache
    echo -e "${GREEN}✓ 配置文件已生成${NC}"
    echo -e "${GREEN}✓ 目录已创建: data/, logs/, cache/${NC}"
}

update_docker_compose() {
    echo -e "${YELLOW}[5/7] 配置 Docker 端口...${NC}"
    
    sed -i "s/\${PORT:-3000}/${PORT}/" docker-compose.yml
    sed -i 's/\${PORT:-3000}/3000/' docker-compose.yml
    echo -e "${GREEN}✓ 端口已配置为 ${PORT}${NC}"
}

build_and_start() {
    echo -e "${YELLOW}[6/7] 构建并启动 Docker 容器...${NC}"
    
    echo -e "${YELLOW}正在构建 Docker 镜像 (这可能需要几分钟)...${NC}"
    if ! docker build -t scoreroute-app .; then
        echo -e "${RED}✗ Docker 镜像构建失败${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ Docker 镜像构建成功${NC}"
    
    docker ps -a --format '{{.Names}}' | grep -E '^ai-gateway|scoreroute' | xargs -r docker rm -f &> /dev/null || true
    
    echo -e "${YELLOW}正在启动容器...${NC}"
    if ! docker compose up -d; then
        echo -e "${RED}✗ 容器启动失败${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ 容器启动成功${NC}"
    
    sleep 5
}

show_complete() {
    echo ""
    echo -e "${GREEN}════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}         ScoreRoute 安装完成！${NC}"
    echo -e "${GREEN}════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "${YELLOW}访问地址:${NC} http://localhost:${PORT}"
    echo -e "${YELLOW}管理员账号:${NC} admin"
    echo -e "${YELLOW}管理员密码:${NC} ${ADMIN_PASSWORD}"
    echo ""
    echo -e "${YELLOW}重要:${NC} 请立即修改默认密码！"
    echo ""
    echo -e "${YELLOW}常用命令:${NC}"
    echo "  查看状态: docker ps"
    echo "  查看日志: docker logs -f scoreroute-app"
    echo "  重启服务: docker restart scoreroute-app"
    echo "  停止服务: docker compose down"
    echo ""
    echo -e "${GREEN}安装目录:${NC} $INSTALL_DIR/ai-gateway"
    echo ""
}

uninstall() {
    echo -e "${YELLOW}正在卸载 ScoreRoute...${NC}"
    docker ps -a --format '{{.Names}}' | grep -E '^ai-gateway|scoreroute' | xargs -r docker rm -f &> /dev/null || true
    docker rmi scoreroute-app &> /dev/null || true
    rm -rf "$INSTALL_DIR"
    echo -e "${GREEN}✓ 卸载完成${NC}"
}

main() {
    show_banner
    
    case "${1:-}" in
        uninstall)
            uninstall
            ;;
        help|--help|-h)
            echo "用法: $0 [选项]"
            echo "  (无参数)     执行完整安装"
            echo "  uninstall    卸载 ScoreRoute"
            ;;
        *)
            check_docker
            select_port
            setup_directory
            generate_config
            update_docker_compose
            build_and_start
            show_complete
            ;;
    esac
}

main "$@"
