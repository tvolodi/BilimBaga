# ── Stage 1: Build React frontend ─────────────────────────────────────────────
FROM node:20-alpine AS frontend-builder

WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# ── Stage 2: nginx with baked-in static files ─────────────────────────────────
# The Go API runs in a separate container ("api") on the same compose network.
# nginx proxies /api/ → http://api:8080 and serves the SPA for all other paths.
FROM nginx:alpine

COPY --from=frontend-builder /app/dist /usr/share/nginx/html
COPY deploy/nginx.conf /etc/nginx/conf.d/default.conf
