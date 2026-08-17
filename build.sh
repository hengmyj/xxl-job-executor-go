#!/usr/bin/env bash
# 编译示例执行器到 bin/xxl-job-executor
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

OUTPUT="${OUTPUT:-bin/xxl-job-executor}"
mkdir -p "$(dirname "$OUTPUT")"

echo "building ./example -> $OUTPUT"
go build -o "$OUTPUT" ./example
echo "ok: $ROOT/$OUTPUT"
