PRE_COMMIT=uvx pre-commit

setup-precommit:
	$(PRE_COMMIT) install --install-hooks
	$(PRE_COMMIT) install --hook-type commit-msg
	@echo "pre-commit 已启用（uvx 运行，无需额外安装）"

pre-commit:
	$(PRE_COMMIT) run --all-files

.PHONY: setup-precommit pre-commit
