# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

## [2.32.0] - 2026-05-11

### Fixed
- Extra Ratings API route mismatch (GET /api/extra-ratings/records)
- Vite config manualChunks function format for rolldown compatibility
- Frontend API consistency (/extra-rating -> /extra-ratings)

## [2.31.0] - 2026-05-09

### Fixed
- Issue #301: unknown model returns 404 instead of 500
- Dashboard password_less_mode toggle redirect
- Docs.vue htmlContent bug
- 流式请求不保存样本问题

### Changed
- Updated workflow.md and session_state.md documentation

## [2.30.0] - 2026-05-08

### Fixed
- Extra Ratings API supports model_name lookup
- Batch create API improvements
- Logs cleanup API improvements

### Changed
- gofmt fixes in extra_rating.go and model.go

## [2.29.0] - 2026-05-07

### Fixed
- GetNextModelSmart returns error instead of silent fallback
- Smart dispatch improvements
- Dispatcher streaming samples fix

## [2.28.0] - 2026-05-06

### Changed
- Added gin.ReleaseMode for production
- Disabled demo channel
- Updated version string

## [2.0.0] - 2026-04-17

### Added
- AI Gateway with multi-provider support (OpenAI, Anthropic, NVIDIA, MiniMax)
- Smart model routing based on scoring
- Token management system
- Admin dashboard with Vue3
- Docker deployment support
- Multi-channel API management
- Request/response sample analysis
- Model rating system
- User rating system
- Token rate limiting

### Security
- Environment-based secret management
- JWT authentication
- API key authentication for external access

## [1.0.0] - 2026-04-17

### Added
- Initial release
