#!/usr/bin/env bash
set -euo pipefail

TARGET_BRANCH="custom"

echo "[tag-custom] 检查当前是否在 git 仓库中..."
if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "[tag-custom] 错误：当前目录不是 git 仓库。请在仓库根目录运行此脚本。"
  exit 1
fi

echo "[tag-custom] 检查工作区是否干净..."
if [[ -n "$(git status --porcelain)" ]]; then
  echo "[tag-custom] 错误：工作区存在未提交或未跟踪的文件。"
  echo "[tag-custom] 请先提交、清理或使用 git stash 暂存改动后再运行本脚本。"
  exit 1
fi

CURRENT_BRANCH="$(git rev-parse --abbrev-ref HEAD)"
if [[ "$CURRENT_BRANCH" != "$TARGET_BRANCH" ]]; then
  echo "[tag-custom] 错误：当前分支为 $CURRENT_BRANCH，仅允许在 $TARGET_BRANCH 分支上打 tag。"
  echo "[tag-custom] 请先执行：git checkout $TARGET_BRANCH"
  exit 1
fi

echo "[tag-custom] 确认远端 origin 是否存在..."
if ! git remote get-url origin >/dev/null 2>&1; then
  echo "[tag-custom] 错误：未配置 origin 远端，请先添加后再运行本脚本。"
  exit 1
fi

echo "[tag-custom] 从 origin 拉取最新 $TARGET_BRANCH ..."
git fetch origin "$TARGET_BRANCH"

LOCAL_HASH="$(git rev-parse HEAD)"
REMOTE_HASH="$(git rev-parse origin/$TARGET_BRANCH)"

if [[ "$LOCAL_HASH" != "$REMOTE_HASH" ]]; then
  echo "[tag-custom] 错误：本地 $TARGET_BRANCH 与 origin/$TARGET_BRANCH 不一致。"
  echo "[tag-custom] 本地 HEAD:   $LOCAL_HASH"
  echo "[tag-custom] 远程 HEAD:   $REMOTE_HASH"
  echo "[tag-custom] 请先执行 git pull --ff-only origin $TARGET_BRANCH 或手动同步后再打 tag。"
  exit 1
fi

if [[ $# -ge 1 ]]; then
  TAG_NAME="$1"
else
  SHORT_SHA="$(git rev-parse --short HEAD)"
  TAG_NAME="custom-$(date +%Y%m%d-%H%M%S)-${SHORT_SHA}"
fi

echo "[tag-custom] 目标 tag 名称：$TAG_NAME"

if git rev-parse -q --verify "refs/tags/$TAG_NAME" >/dev/null 2>&1; then
  echo "[tag-custom] 错误：tag $TAG_NAME 已存在，请换一个名称。"
  exit 1
fi

echo "[tag-custom] 当前分支：$CURRENT_BRANCH"
echo "[tag-custom] 当前提交：$LOCAL_HASH"

git tag -a "$TAG_NAME" -m "tag on custom: $TAG_NAME"
git push origin "$TAG_NAME"

cat <<EOF
[tag-custom] 已在分支 $TARGET_BRANCH 打 tag：$TAG_NAME，并推送到 origin。

下游项目可在 go.mod 中使用类似配置：

  require go.mau.fi/whatsmeow $TAG_NAME
  replace go.mau.fi/whatsmeow => github.com/2025Restart/whatsmeow $TAG_NAME

EOF


