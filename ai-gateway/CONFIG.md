# Configuration Guide

## Environment Variables

ScoreRoute uses environment variables for security. Copy `.env.example` to `.env` and configure:

### Required Variables

| Variable | Description | How to Set |
|----------|-------------|------------|
| `ADMIN_PASSWORD` | Admin login password | Set any strong password |
| `JWT_SECRET` | JWT signing secret | Generate with `openssl rand -base64 32` |

### Optional Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 3000 | Server port |
| `DATABASE_PATH` | ./data/gateway.db | SQLite database path |
| `LOG_PATH` | ./logs | Log files directory |

## Quick Setup

```bash
# 1. Copy example config
cp .env.example .env

# 2. Generate JWT secret
openssl rand -base64 32
# Output: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# 3. Edit .env and set:
# ADMIN_PASSWORD=your_secure_password
# JWT_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# 4. Start services
docker-compose up -d
```

## Security Notes

- **Never** commit `.env` file to version control
- Use **strong passwords** for `ADMIN_PASSWORD` (min 12 characters)
- Generate a **unique** `JWT_SECRET` for each deployment
- Rotate secrets periodically
