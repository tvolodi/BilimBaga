# ── Stage 1: build ────────────────────────────────────────────────────────────
FROM node:20-alpine AS build

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

# ── Stage 2: The dist output is exported via named volume ─────────────────────
# nginx reads from the frontend_dist volume; nothing extra needed here.
FROM alpine:3.20

COPY --from=build /app/dist /app/dist
