# syntax=docker/dockerfile:1.7
#
# The ShiftMaster web tier: the built React app, served by Caddy, which also
# reverse-proxies /api to the API container.
#
# Serving the SPA and proxying the API from one origin is not a convenience —
# the frontend calls a relative /api by default (VITE_API_URL is unset), and
# same-origin means no CORS preflight and no third-party cookie question.

# ── Build ────────────────────────────────────────────────────────────────────
FROM node:24-bookworm-slim AS build
WORKDIR /app

# npm ci installs exactly the committed lockfile and fails if it disagrees with
# package.json, which is what makes a container build reproducible. The peer
# flag matches CI; the React 19 ecosystem still ships packages whose peer ranges
# lag the release.
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci --legacy-peer-deps

COPY . .

# VITE_ values are inlined into the bundle at build time, so this is a build
# argument rather than a runtime environment variable. Left empty the app calls
# a relative /api, which is what the Caddy stage below serves.
ARG VITE_API_URL=""
ENV VITE_API_URL=${VITE_API_URL}

# `npm run build` is `tsc -b && vite build`: a type error fails the image rather
# than shipping a bundle nobody typechecked.
RUN npm run build

# ── Serve ────────────────────────────────────────────────────────────────────
FROM caddy:2-alpine AS web

COPY Caddyfile /etc/caddy/Caddyfile
COPY --from=build /app/dist /srv

# Three things, in order:
#
#   - an unprivileged account for the server to run as;
#   - the file capability removed from the binary. The upstream image sets
#     cap_net_bind_service so Caddy can bind port 80, which this image never
#     does — it listens on 8080. Left in place it is worse than useless: execve
#     of a file carrying permitted-set capabilities fails outright once the
#     container drops all of them, so `cap_drop: ALL` produced
#     "exec /usr/bin/caddy: operation not permitted" and a crash loop;
#   - the config validated, so a malformed Caddyfile fails the build rather than
#     the container.
RUN adduser -D -u 10001 -h /srv caddyuser \
 && apk add --no-cache libcap \
 && setcap -r /usr/bin/caddy \
 && apk del libcap \
 && caddy validate --config /etc/caddy/Caddyfile

# Caddy keeps an autosaved config and a certificate store under these paths,
# which default to /config and /data — both on the read-only root filesystem,
# where every start logged errors about being unable to write them. Nothing here
# needs to survive a restart: TLS is terminated upstream and the config is baked
# into the image, so pointing them at the container's tmpfs is honest rather
# than merely quiet.
ENV XDG_CONFIG_HOME=/tmp/caddy \
    XDG_DATA_HOME=/tmp/caddy

USER 10001:10001
EXPOSE 8080

# Caddy's own reply, not the API's: this asks whether the edge is serving, and
# the compose healthcheck on the api service asks the other question.
HEALTHCHECK --interval=15s --timeout=5s --start-period=5s --retries=3 \
    CMD ["sh", "-c", "wget -q --spider http://127.0.0.1:8080/ || exit 1"]

CMD ["caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]
