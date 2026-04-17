# ScoreRoute - Agent Context File

## CRITICAL: About This Environment

- **THIS IS A LOCAL DEVELOPMENT ENVIRONMENT** - I have full access to this system
- **SUDO ACCESS**: Use `sudo docker ...` for Docker operations (e.g., `sudo docker build`, `sudo docker-compose up -d`)
- **WORKING DIRECTORY**: `/home/ubuntu/OpenCode/ai-gateway/`
- **SERVER**: Running on `localhost:3000` (Docker container named `ai-gateway`)
- **GIT**: Initialized with first commit - use `git log` to see history

## Important Reminders

1. I developed this entire application - all code, deployment, and configuration
2. I have sudo access and can rebuild Docker containers
3. The app is on the SAME server I'm running on - NOT remote
4. If I need to restart Docker: `sudo docker rm -f ai-gateway && sudo docker-compose up -d`
5. If I need to rebuild: `cd /home/ubuntu/OpenCode/ai-gateway && sudo docker build -t ai-gateway-app .`

## Tech Stack

- **Backend**: Go 1.21 + Gin + SQLite (file: `data/gateway.db`)
- **Frontend**: Vue3 + Element Plus + Vite
- **Container**: Docker with `docker-compose.yml`
- **Deployment**: Port 3000

## Build Commands

```bash
# Frontend rebuild
cd /home/ubuntu/OpenCode/ai-gateway/web && npm run build

# Docker rebuild
cd /home/ubuntu/OpenCode/ai-gateway && sudo docker build -t ai-gateway-app .

# Docker restart
cd /home/ubuntu/OpenCode/ai-gateway && sudo docker rm -f ai-gateway && sudo docker-compose up -d
```

## Project Structure

```
ai-gateway/
├── cmd/server/main.go          # Entry point
├── internal/
│   ├── handler/               # HTTP handlers
│   ├── service/               # Business logic
│   ├── repository/            # Database operations
│   ├── model/                # Data models
│   └── router/               # Route definitions
├── web/                       # Vue frontend
│   ├── src/views/             # Vue pages
│   ├── src/api/               # API client
│   └── dist/                  # Built frontend (served by Go server)
├── docs/开发计划.md            # Full development documentation
├── docker-compose.yml         # Docker configuration
├── Dockerfile                # Container build
└── data/                      # SQLite database
```

## API Endpoints

- **Health**: `GET /api/health` or `GET /health`
- **Auth**: `POST /api/auth/login`
- **Proxy**: `POST /v1/chat/completions` (use API Key from tokens)

## Default Credentials

- **Admin Login**: admin / dbm52100
- **API Key**: Any created token key (e.g., `sk-2af11d51-57db-4e2a-a9ab-9ab14b4ab6e2`)

## Key Features

1. **Sample Analysis**: Saves request/response samples (>1000 tokens) for 7 days, one per model
2. **User Rating**: 1-100 scale rating per model
3. **Model Rating**: Auto-calculated based on success rate, latency, stability, user rating
4. **Polling**: AUTO/POLL_ALL routes requests across all models

## If Something Goes Wrong

1. Check Docker: `sudo docker ps`
2. View logs: `sudo docker logs ai-gateway`
3. Restart container: `sudo docker restart ai-gateway`
4. Rebuild if needed: `cd /home/ubuntu/OpenCode/ai-gateway && sudo docker build -t ai-gateway-app . && sudo docker rm -f ai-gateway && sudo docker-compose up -d`
