# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.25-alpine AS build
WORKDIR /src

# Cache dependencies first.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# The app is pure Go (pgx + glebarez/sqlite), so CGO is not required.
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# ---- runtime stage ----
FROM alpine:3.24
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 app \
    && adduser -S -D -u 10001 -G app app
WORKDIR /app
COPY --from=build /out/server /app/server
COPY docker/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

# The container runs as an unprivileged user; /app is owned by it.
USER app
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/api/v1/health/ready || exit 1

# Apply migrations before serving (idempotent, see docker/entrypoint.sh).
ENTRYPOINT ["/app/entrypoint.sh"]
