#!/usr/bin/env bash
set -eo pipefail

UPSTREAM_REMOTE="upstream"
UPSTREAM_URL="https://github.com/tulir/whatsmeow.git"
CLEAN_BRANCH="main"

echo "[sync-upstream] 检查当前是否在 git 仓库中..."
if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "[sync-upstream] 错误：当前目录不是 git 仓库。请在仓库根目录运行此脚本。"
  exit 1
fi

echo "[sync-upstream] 检查工作区是否干净..."
if [[ -n "$(git status --porcelain)" ]]; then
  echo "[sync-upstream] 错误：工作区存在未提交或未跟踪的文件。"
  echo "[sync-upstream] 请先提交、清理或使用 git stash 暂存改动后再运行本脚本。"
  exit 1
fi

echo "[sync-upstream] 确认远端 origin 是否存在..."
if ! git remote get-url origin >/dev/null 2>&1; then
  echo "[sync-upstream] 错误：未配置 origin 远端，请先添加后再运行本脚本。"
  exit 1
fi

echo "[sync-upstream] 配置/校验 upstream 远端..."
if ! git remote get-url "$UPSTREAM_REMOTE" >/dev/null 2>&1; then
  echo "[sync-upstream] 未检测到 upstream，正在添加：$UPSTREAM_URL"
  git remote add "$UPSTREAM_REMOTE" "$UPSTREAM_URL"
else
  CURRENT_URL="$(git remote get-url "$UPSTREAM_REMOTE")"
  if [[ "$CURRENT_URL" != "$UPSTREAM_URL" ]]; then
    echo "[sync-upstream] 警告：upstream 当前 URL 为：$CURRENT_URL"
    echo "[sync-upstream] 将被更新为：$UPSTREAM_URL"
    git remote set-url "$UPSTREAM_REMOTE" "$UPSTREAM_URL"
  fi
fi

echo "[sync-upstream] 从 upstream 拉取最新代码..."
git fetch "$UPSTREAM_REMOTE"

echo "[sync-upstream] 检查本地分支 $CLEAN_BRANCH 是否存在..."
if ! git show-ref --verify --quiet "refs/heads/$CLEAN_BRANCH"; then
  echo "[sync-upstream] 错误：本地分支 $CLEAN_BRANCH 不存在。"
  echo "[sync-upstream] 如需创建，可执行："
  echo "  git fetch origin"
  echo "  git checkout -b $CLEAN_BRANCH origin/$CLEAN_BRANCH"
  exit 1
fi

CURRENT_BRANCH="$(git rev-parse --abbrev-ref HEAD || echo '')"
if [[ -z "$CURRENT_BRANCH" ]]; then
  echo "[sync-upstream] 错误：无法获取当前分支名称，请检查当前仓库状态。"
  exit 1
fi

if [[ "$CURRENT_BRANCH" != "$CLEAN_BRANCH" ]]; then
  echo "[sync-upstream] 当前分支为：$CURRENT_BRANCH，将切换到：$CLEAN_BRANCH"
  git checkout "$CLEAN_BRANCH"
fi

echo "[sync-upstream] 将本地 $CLEAN_BRANCH 重置为 $UPSTREAM_REMOTE/main ..."
git reset --hard "$UPSTREAM_REMOTE/main"

echo "[sync-upstream] 即将使用 --force-with-lease 推送到 origin/$CLEAN_BRANCH"
echo "[sync-upstream] 该操作会使 origin/$CLEAN_BRANCH 完全对齐 upstream/main，请确保未在 $CLEAN_BRANCH 上做自定义开发。"

git push origin "$CLEAN_BRANCH" --force-with-lease

echo "[sync-upstream] 同步完成。$CLEAN_BRANCH 已与 upstream/main 对齐并推送到 origin。"
cat <<'EOF'

后续建议操作（需手动执行）：
  # 将上游更新合入自定义主线
  git checkout custom
  git fetch origin
  # 二选一：
  git merge origin/main     # 生成 merge commit，历史保留完整
  # 或
  # git rebase origin/main  # 重写 custom 历史，保持线性（需熟悉 rebase）

请在解决冲突并完成测试后，再执行：
  git push origin custom

EOF


