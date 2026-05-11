# Manual Deployment Guide

This guide explains how to deploy ScoreRoute without Docker.

## Prerequisites

- Go 1.21 or higher
- Node.js 18+ and npm
- SQLite (usually included with Go)

## Backend Setup

1. Clone the repository:
```bash
git clone https://github.com/DING-BAOMING/ScoreRoute.git
cd ScoreRoute
```

2. Build the backend binary:
```bash
go build -o server ./cmd/server
```

3. Create required directories:
```bash
mkdir -p data logs cache web/dist
```

4. Build the frontend:
```bash
cd web
npm install
npm run build
cd ..
cp -r web/dist/* web/dist/.* data/ 2>/dev/null || true
```

5. Create configuration file (.env):
```bash
cat > .env << EOF
PORT=3000
DATABASE_PATH=./data/gateway.db
LOG_PATH=./logs
ADMIN_PASSWORD=your-secure-password
JWT_SECRET=your-secret-key-minimum-32-chars
EOF
```

6. Start the server:
```bash
./server
```

## Troubleshooting

### Port Already in Use
If port 3000 is in use, specify a different port:
```bash
PORT=8080 ./server
```

### Frontend Assets Not Found
Ensure web/dist contents are copied to the correct location:
```bash
mkdir -p web/dist
cp -r web/dist/* ./web/dist/
```

### Database Errors
Ensure the data directory exists and is writable:
```bash
chmod 755 data
```
