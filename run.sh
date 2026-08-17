#!/usr/bin/env bash
# 编译（如需要）并启动示例执行器
# 用法:
#   ./run.sh
#   ./run.sh --rebuild
#   XXL_JOB_ADMIN=http://127.0.0.1:8080/xxl-job-admin ./run.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

OUTPUT="${OUTPUT:-bin/xxl-job-executor}"
REBUILD=0
for arg in "$@"; do
	case "$arg" in
	-h | --help)
		cat <<'EOF'
Usage: ./run.sh [--rebuild]

Environment:
  XXL_JOB_ADMIN          调度中心地址，默认 http://127.0.0.1/xxl-job-admin
  XXL_JOB_ACCESS_TOKEN   请求令牌，默认空
  XXL_EXECUTOR_IP        执行器 IP，默认 127.0.0.1
  XXL_EXECUTOR_PORT      执行器端口，默认 9999
  XXL_REGISTRY_KEY       执行器名称，默认 golang-jobs
  XXL_PHP_BIN            PHP 可执行文件，默认 php
  XXL_LOG_DIR            GLUE 脚本/日志目录，默认空
  OUTPUT                 编译产物路径，默认 bin/xxl-job-executor
EOF
		exit 0
		;;
	--rebuild | -B)
		REBUILD=1
		;;
	*)
		echo "unknown arg: $arg" >&2
		exit 1
		;;
	esac
done

need_build=0
if [[ "$REBUILD" -eq 1 || ! -x "$OUTPUT" ]]; then
	need_build=1
fi

if [[ "$need_build" -eq 1 ]]; then
	"$ROOT/build.sh"
fi

export XXL_JOB_ADMIN="${XXL_JOB_ADMIN:-http://127.0.0.1/xxl-job-admin}"
export XXL_EXECUTOR_PORT="${XXL_EXECUTOR_PORT:-9999}"
export XXL_REGISTRY_KEY="${XXL_REGISTRY_KEY:-golang-jobs}"

echo "starting $OUTPUT"
echo "  admin:    $XXL_JOB_ADMIN"
echo "  listen:   ${XXL_EXECUTOR_IP:-127.0.0.1}:$XXL_EXECUTOR_PORT"
echo "  registry: $XXL_REGISTRY_KEY"
exec "$OUTPUT"
