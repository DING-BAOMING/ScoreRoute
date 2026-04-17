# CLAUDE.md - ScoreRoute

This project uses the ScoreRoute application. For essential context, see @AGENTS.md

## Quick Start

```bash
# Make API request
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer sk-2af11d51-57db-4e2a-a9ab-9ab14b4ab6e2" \
  -H "Content-Type: application/json" \
  -d '{"model": "MiniMax-M2.5", "messages": [{"role": "user", "content": "Hi"}]}'
```

## Important Reminders for Claude Code

1. I developed this entire project - I know it fully
2. I have sudo access - use `sudo docker ...` for Docker commands
3. The server is local (localhost:3000), not remote
4. Build commands work as documented in AGENTS.md
