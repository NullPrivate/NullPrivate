#!/usr/bin/env bash

# 说明：
# - 使用非交互方式登录 Docker Hub（--password-stdin），避免 VSCode 任务非 TTY 环境报错。
# - 通过临时 `DOCKER_CONFIG` 避免系统全局 credsStore/credential helper 触发交互登录。
# - 从阿里云镜像仓库拉取指定版本镜像，打上 Docker Hub 的版本与 latest 标签并推送。

set -Eeuo pipefail

if [[ ${1-} == "" ]]; then
  echo "用法：$0 <版本号，例如 v1.4.2>" >&2
  exit 2
fi

VERSION="$1"

# 必要环境变量检查
if [[ -z "${DOCKER_HUB_TOKEN_NULLPRIVATE:-}" ]]; then
  echo "错误：环境变量 DOCKER_HUB_TOKEN_NULLPRIVATE 未设置。" >&2
  echo "请在 VSCode 终端或系统环境中导出：export DOCKER_HUB_TOKEN_NULLPRIVATE=\"<Docker Hub Personal Access Token>\"" >&2
  exit 1
fi

# 临时 Docker 配置目录，隔离可能需要交互的 credential helper
DOCKER_CONFIG_DIR="$(mktemp -d -t docker-config-XXXXXX)"
export DOCKER_CONFIG="$DOCKER_CONFIG_DIR"
cleanup() {
  rm -rf "$DOCKER_CONFIG_DIR" || true
}
trap cleanup EXIT

echo "[1/5] 使用临时 DOCKER_CONFIG：$DOCKER_CONFIG"

echo "[2/5] 登录 Docker Hub（非交互）……"
printf "%s" "${DOCKER_HUB_TOKEN_NULLPRIVATE}" | docker login --username "nullprivate" --password-stdin docker.io

SRC="registry.cn-hangzhou.aliyuncs.com/adguardprivate/adguardprivate:${VERSION}"
DST_VER="docker.io/nullprivate/nullprivate:${VERSION}"
DST_LATEST="docker.io/nullprivate/nullprivate:latest"

echo "[3/5] 拉取镜像：$SRC"
docker pull "$SRC"

echo "[4/5] 打标签：$DST_VER, $DST_LATEST"
docker tag "$SRC" "$DST_VER"
docker tag "$SRC" "$DST_LATEST"

echo "[5/5] 推送到 Docker Hub：$DST_VER, $DST_LATEST"
docker push "$DST_VER"
docker push "$DST_LATEST"

echo "完成：$DST_VER 与 $DST_LATEST 已推送。"

