# ── Stage 1: build ────────────────────────────────────────────────────────────
FROM node:20-alpine AS build

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

# ── Stage 2: refresh dist volume on every container start ─────────────────────
# Bake the freshly built dist into the image (outside the volume mount path).
# The entrypoint clears the named volume and copies fresh files on each start,
# so nginx always serves the current build — never a stale cached volume.
FROM alpine:3.20

COPY --from=build /app/dist /app/dist-source
RUN mkdir -p /app/dist
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

ENTRYPOINT ["/docker-entrypoint.sh"]
