#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")"
npx --no-install tsc -p frontend/tsconfig.json
mkdir -p bin/linux-x64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -tags purego -ldflags '-s -w' -o bin/linux-x64/dl-manager .
