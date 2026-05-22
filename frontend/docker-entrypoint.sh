#!/bin/sh
set -e

echo "[frontend] Refreshing dist volume from image build..."

# Remove any stale files from previous builds (old chunk hashes, removed assets, etc.)
find /app/dist -mindepth 1 -delete 2>/dev/null || true

# Copy the freshly built dist from the baked-in image layer to the shared volume
cp -r /app/dist-source/. /app/dist/

echo "[frontend] Dist volume ready ($(find /app/dist -type f | wc -l) files)."

# Stay alive so healthcheck passes and nginx depends_on: service_healthy works
exec tail -f /dev/null
