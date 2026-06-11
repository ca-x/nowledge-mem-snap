FROM --platform=$BUILDPLATFORM node:26-alpine AS web-builder

WORKDIR /src/web
COPY web/package.json web/package-lock.json* ./
RUN npm ci --fetch-retries=5 --fetch-retry-mintimeout=20000 --fetch-retry-maxtimeout=120000
COPY web/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT
ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN apk add --no-cache ca-certificates git tzdata
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go generate ./internal/persist/ent
COPY --from=web-builder /src/web/dist ./web/dist
RUN test -f web/dist/index.html
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -trimpath \
    -ldflags="-s -w -extldflags '-static' \
      -X 'github.com/ca-x/nowledge-mem-snap/version.Version=${VERSION}' \
      -X 'github.com/ca-x/nowledge-mem-snap/version.BuildTime=${BUILD_TIME}' \
      -X 'github.com/ca-x/nowledge-mem-snap/version.GitCommit=${GIT_COMMIT}'" \
    -o nowledge-mem-snap .

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app
COPY --from=builder /src/nowledge-mem-snap /app/nowledge-mem-snap
COPY assets/logo.png /app/assets/logo.png
RUN mkdir -p /app/data && chown -R appuser:appuser /app

USER appuser
ENV DATA_DIR=/app/data
ENV PORT=14335
EXPOSE 14335

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT}/healthz || exit 1

CMD ["/app/nowledge-mem-snap"]
