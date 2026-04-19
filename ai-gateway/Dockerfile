FROM alpine:3.19

RUN apk add --no-cache go git musl-dev gcc libc-dev bash nodejs npm

WORKDIR /app

COPY go.mod go.sum ./
COPY . ./

RUN rm -rf .git

# Build Go backend
RUN GOTOOLCHAIN=auto go get modernc.org/sqlite@v1.28.0 && \
    GOTOOLCHAIN=auto go get github.com/spf13/viper@v1.18.2 && \
    GOTOOLCHAIN=auto go get github.com/golang-jwt/jwt/v5@v5.2.0 && \
    GOTOOLCHAIN=auto go get github.com/google/uuid@v1.5.0 && \
    GOTOOLCHAIN=auto go get github.com/gin-gonic/gin@v1.9.1 && \
    GOTOOLCHAIN=auto go get github.com/pkoukk/tiktoken-go@v0.1.4

RUN GOTOOLCHAIN=auto go mod download

# Build frontend (creates web/dist/)
RUN cd web && npm ci && npm run build && cd ..

RUN CGO_ENABLED=1 GOOS=linux GOTOOLCHAIN=auto go build -ldflags "-s -w" -o server ./cmd/server

RUN mkdir -p /app/logs /app/data /app/cache

ENV PORT=3000
ENV DATABASE_PATH=/app/data/gateway.db
ENV LOG_PATH=/app/logs
ENV TIKTOKEN_CACHE_DIR=/app/cache
ENV GIN_MODE=release

EXPOSE 3000

CMD ["./server"]
