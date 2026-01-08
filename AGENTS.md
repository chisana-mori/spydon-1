# Spydon / robusta-web：AGENTS 工作指南（精简版）

> **目标**：压制技术债，优先可维护性、可读性与可验证性。

## 0. 使用方式（给 Agent）


1. **按需再读**：`docs/` 下相关文档（渐进式披露），不要把大量细节重复写进本文件。
2. **以代码与自动化检查为准**：文档可能滞后。
3. **谨慎操作**：任何会安装依赖、下载大文件、或改动运行环境的操作，先征求确认。

## 1. 项目地图（WHAT）

核心目录（以 `robusta-web/` 为根）：

```text
apps/
  backend/    # Go API（Gin），入口 cmd/server
  frontend/   # Next.js（App Router）+ TS + Tailwind
infrastructure/  # docker-compose / k8s / nginx / scripts
docs/            # 架构、设计、部署、CI 等文档
var/             # 工具/缓存（如 Go modules）
```

## 2. 我们在优化什么（WHY）

- **可删除/可重构**：按业务域切片，避免“全局大包/大组件”。
- **可排障**：错误信息要带上下文，日志要可追踪。
- **可验证**：改动必须能通过现有测试、lint、type-check（至少在相关模块可跑）。

## 3. 代码组织与改动策略（HOW）

### 3.1 通用原则

- **KISS + YAGNI**：优先最小改动、最少抽象。
- **别制造债**：避免“通用 utils/manager 大杂烩”；文件/函数过大时优先拆分。
- **遵循现有结构**：新代码尽量放到已有的 feature/domain 目录中。

### 3.2 后端（Go）

- **分层建议**：路由/handler（入参校验）→ service（业务逻辑）→ db/repo（数据访问）。
- **Feature-First 优先**：能放进 `apps/backend/internal/features/<feature>/...` 的新逻辑，优先放这里；不要继续扩大“全局包”。
- **错误处理**：不要忽略错误；返回时用 `%w` 包装上下文（便于定位）。
- **并发**：需要并发时优先用 `errgroup`/`WaitGroup`；避免不必要的 channel 与全局状态。

### 3.3 前端（Next.js / TypeScript）

- **类型安全**：禁止 `any`；优先定义清晰的 `type/interface`。
- **异步风格**：统一 `async/await`，避免链式 `.then()` 堆叠。
- **目录约定**：业务功能放 `apps/frontend/src/features`；通用 UI 放 `src/components` / `src/components/ui`。
- **组件体量**：超大组件请拆分（UI/数据/状态/副作用分离），避免单文件承担多职责。

## 4. UI / 设计规则（精简版）

1. **页面容器宽度**：高层 page wrapper 不要随意加 `container` / `mx-auto` / `max-w-*`，避免与侧边栏对齐漂移。
2. **页面主标题（h1）**：统一 `text-2xl font-bold`。
3. **标题图标**：主标题前放语义化 icon，并使用一致的容器样式（示例见旧实现/现有页面）。
4. **颜色与主题**：不要硬编码 hex；遵循语义化颜色与暗黑模式规则。

> 颜色与语义的**单一事实来源**：`docs/notes/frontend-design.mdc`（不要在本文件复制整套表格）。

## 5. 验证方式（建议最小闭环）

按改动范围选择最小集合运行：

### 5.1 代码质量（仓库级）

- `make pre-commit`（通过 `uvx` 运行，无需额外安装）

### 5.2 后端（Go）

- `cd apps/backend && go test ./...`
- 更全面（会尝试拉起依赖）：`cd apps/backend && bash test/tests.sh`

### 5.3 前端（Next.js）

- `cd apps/frontend && npm run lint`
- `cd apps/frontend && npm run type-check`
- `cd apps/frontend && npm test`
- 端到端：`cd apps/frontend && npm run test:e2e`（如需安装浏览器，先确认再执行）

## 6. 常见反模式（请避免）

- ❌ “顺手塞进 utils/”导致不可维护
- ❌ 忽略错误 / 吞异常 / 缺少上下文
- ❌ 为了“未来可能需要”过度抽象
- ❌ UI 规则各写一套（颜色/间距/标题样式不一致）

## 7. 进一步阅读（渐进式披露）

- 前端配色与语义：`docs/notes/frontend-design.mdc`
