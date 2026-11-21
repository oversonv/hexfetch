#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
msg="${1:-Update $(date '+%Y-%m-%d %H:%M')}"
git add -A
if ! git diff --cached --quiet; then
    git commit -m "$msg"
fi
git pull --rebase
git push origin HEAD
CGO_ENABLED=0 go build -o hexfetch ./src/cmd/hexfetch
echo "built: $(pwd)/hexfetch"
