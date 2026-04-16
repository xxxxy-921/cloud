# Multi-stage Dockerfile for Metis editions.
# Build args control which edition is produced:
#   EDITION      — Go build tag (e.g., edition_license). Empty = full.
#   APPS         — Frontend module filter (e.g., system,license). Empty = all.
#   BINARY_NAME  — Output binary name (e.g., license). Default: server.
#   CMD_PATH     — Go main package path. Default: ./cmd/server.
#
# Example:
#   docker build \
#     --build-arg EDITION=edition_license \
#     --build-arg APPS=system,license \
#     --build-arg BINARY_NAME=license \
#     -t license:latest .

# ── Stage 1: Frontend ────────────────────────────────────────────
FROM oven/bun:1 AS frontend

WORKDIR /app

COPY web/package.json web/bun.lock* web/
COPY scripts/ scripts/

ARG APPS=""

RUN cd web && bun install --frozen-lockfile

COPY web/ web/

RUN if [ -n "$APPS" ]; then APPS=$APPS ./scripts/gen-registry.sh; fi && \
    cd web && bun run build

# ── Stage 2: Backend ─────────────────────────────────────────────
FROM golang:1.26-bookworm AS backend

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /app/web/dist/ web/dist/

ARG EDITION=""
ARG BINARY_NAME="server"
ARG CMD_PATH="./cmd/server"
ARG VERSION="dev"
ARG GIT_COMMIT=""
ARG BUILD_TIME=""

RUN CGO_ENABLED=0 go build \
    ${EDITION:+-tags $EDITION} \
    -ldflags "-X metis/internal/version.Version=${VERSION} -X metis/internal/version.GitCommit=${GIT_COMMIT} -X metis/internal/version.BuildTime=${BUILD_TIME}" \
    -o /out/${BINARY_NAME} ${CMD_PATH}

# ── Stage 3: Runtime ─────────────────────────────────────────────
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates tzdata && \
    rm -rf /var/lib/apt/lists/* && \
    groupadd -r metis && useradd -r -g metis -m metis

ARG BINARY_NAME="server"

COPY --from=backend /out/${BINARY_NAME} /usr/local/bin/metis

RUN mkdir -p /data && chown metis:metis /data

USER metis
WORKDIR /data

EXPOSE 8080

ENTRYPOINT ["metis"]
