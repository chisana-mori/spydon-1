#!/usr/bin/env bash
set -euo pipefail

if command -v uv >/dev/null 2>&1; then
  uv tool install pre-commit || true
else
  if command -v pipx >/dev/null 2>&1; then
    pipx install pre-commit || true
  else
    pip install pre-commit || true
  fi
fi

git config core.hooksPath .githooks
pre-commit install --install-hooks --hook-dir .githooks
pre-commit install --hook-type commit-msg --hook-dir .githooks
echo "pre-commit installed and hooks enabled (hook-dir: .githooks)"
