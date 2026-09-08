# syntax=docker/dockerfile:1.7
#
# The ShiftMaster API, and the migration tool that has to run before it.
#
# Two runnable targets:
#
#   --target api      the default. No language model, so the assistant reports
#                     itself unavailable and the rest of the product is
#                     unaffected. This is what most deployments want.
#   --target api-ai   adds the llama.cpp server so AI_MANAGED=true can supervise
#                     a model the host mounts in. The model file itself is never
#                     baked in: it is gigabytes, it changes independently of the
#                     code, and an image is the wrong place for it.
#
# Both stages carry cmd/migrate as well as cmd/api, so the same image can run
# the migration step — a second image would only let the two drift apart.

# ── Build ────────────────────────────────────────────────────────────────────
FROM golang:1.26.1-bookworm AS build
WORKDIR /src

# Dependencies resolve in their own layer, so editing application code does not
# re-download the module graph.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

# CGO is off: the binaries are static, which is what lets the runtime stage stay
# a plain slim base with no toolchain in it. -trimpath keeps build paths out of
# the binary.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
        -o /out/shiftmaster-api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
        -o /out/shiftmaster-migrate ./cmd/migrate

# ── Runtime ──────────────────────────────────────────────────────────────────
FROM debian:bookworm-slim AS api

# ca-certificates for outbound TLS (Microsoft Graph mail, web push), tzdata
# because every schedule in this product is a local-time question, and curl so
# the container can answer its own HEALTHCHECK.
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates tzdata curl \
 && rm -rf /var/lib/apt/lists/*

# A fixed high uid keeps file ownership legible on a bind-mounted uploads
# directory, where the host has no idea what a container-local user is.
RUN groupadd --gid 10001 shiftmaster \
 && useradd --uid 10001 --gid 10001 --home-dir /app --no-create-home --shell /usr/sbin/nologin shiftmaster

WORKDIR /app

COPY --from=build /out/shiftmaster-api /out/shiftmaster-migrate /usr/local/bin/
# The migration tool reads the series from disk at runtime, so the SQL travels
# with the binary that applies it.
COPY internal/database/migrations /app/migrations

# Uploads are the one path the service writes. Declared here so the directory
# exists with the right owner even when nothing is mounted over it.
RUN install -d -o shiftmaster -g shiftmaster -m 0750 /app/uploads
VOLUME ["/app/uploads"]

ENV SERVER_HOST=0.0.0.0 \
    SERVER_PORT=8080 \
    UPLOAD_BASE_PATH=/app/uploads \
    ENV=production \
    LOG_FORMAT=json

USER 10001:10001
EXPOSE 8080

# /health checks the database round-trip, not just process liveness, so a
# container that has lost its pool is reported unhealthy rather than up.
HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=3 \
    CMD ["sh", "-c", "curl -fsS http://127.0.0.1:8080/health || exit 1"]

ENTRYPOINT ["shiftmaster-api"]

# ── llama.cpp, for the AI target ─────────────────────────────────────────────
# Pinned to the same release deploy/provision-ai.sh installs, so a container and
# a bare-metal host run identical model server code.
#
# linux/amd64 only, and not by choice: the release publishes exactly one general
# Linux build, llama-<ver>-bin-ubuntu-x64.zip. There is no ubuntu-arm64 asset,
# so an arm64 build of this target can only 404 — which is what it did, with a
# curl error that said nothing about why. Building llama.cpp from source here
# would drag a C++ toolchain into the image for a target most deployments do not
# use; an arm64 host that wants the assistant should build llama-server itself
# and mount it in, or run the API outside a container.
FROM debian:bookworm-slim AS llama
ARG LLAMA_VERSION=b7062
ARG TARGETARCH
RUN apt-get update \
 && apt-get install -y --no-install-recommends curl unzip ca-certificates \
 && rm -rf /var/lib/apt/lists/*
RUN set -eux; \
    case "${TARGETARCH}" in \
      amd64) asset="llama-${LLAMA_VERSION}-bin-ubuntu-x64.zip" ;; \
      *) echo "llama.cpp ${LLAMA_VERSION} ships no Linux ${TARGETARCH} binary; build this target with --platform linux/amd64, or supply your own llama-server" >&2; exit 1 ;; \
    esac; \
    curl -fL --retry 3 -o /tmp/llama.zip \
      "https://github.com/ggml-org/llama.cpp/releases/download/${LLAMA_VERSION}/${asset}"; \
    unzip -q -o /tmp/llama.zip -d /tmp/x; \
    install -d /out/bin /out/lib; \
    find /tmp/x -type f -name 'llama-server' -exec install -m 0755 {} /out/bin/ \; ; \
    find /tmp/x -type f -name '*.so*' -exec install -m 0644 {} /out/lib/ \; ; \
    test -x /out/bin/llama-server

FROM api AS api-ai
USER 0:0
# The prebuilt server is dynamically linked against libgomp and the C++ runtime.
RUN apt-get update \
 && apt-get install -y --no-install-recommends libgomp1 libstdc++6 \
 && rm -rf /var/lib/apt/lists/*
COPY --from=llama /out/bin/llama-server /usr/local/bin/llama-server
COPY --from=llama /out/lib/ /usr/local/lib/llama/
ENV LD_LIBRARY_PATH=/usr/local/lib/llama
# Where the compose file mounts the GGUF. Nothing is downloaded at runtime; if
# the file is absent the assistant simply reports itself unavailable.
ENV AI_MODEL_PATH=/app/models/model.gguf \
    AI_MANAGED=true \
    AI_BASE_URL=http://127.0.0.1:8081
RUN install -d -o shiftmaster -g shiftmaster -m 0750 /app/models
USER 10001:10001
