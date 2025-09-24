# Robusta 前端整体设计

## 目标概述
- 提供面向 SRE/开发的自托管 UI，用于查看 Robusta Runner 产生的事件、指标及 Playbook 执行结果。
- 支持多集群与命名空间过滤，具备实时刷新能力，兼容最新 Robusta 版本（以 0.11.x 为基线）。

## 技术栈
- **前端框架**：Vite + React + TypeScript，利用模块化架构和快速开发体验。
- **样式体系**：Tailwind CSS 4 α 版本 + shadcn/ui 组件集，确保设计语言统一，可快速迭代。
- **状态管理**：React Query（@tanstack/react-query）负责数据获取与缓存；Zustand 轻量保存全局 UI 状态（如筛选条件）。
- **可视化**：ECharts（按需引入）展示指标趋势与集群健康度。
- **实时更新**：使用 SSE（Server-Sent Events）或 WebSocket 通道订阅新事件，前端通过通知中心反馈。

## 目录结构
```
robusta-web/
├─ docs/                    # 架构与接口文档
├─ frontend/
│  ├─ src/
│  │  ├─ app/               # 布局、路由、上下文
│  │  ├─ components/        # 基础通用组件（含 shadcn 包装）
│  │  ├─ features/
│  │  │  ├─ dashboard/      # 仪表盘模块
│  │  │  ├─ events/         # 事件列表与详情
│  │  │  └─ playbooks/      # 自动化执行记录
│  │  ├─ lib/               # 工具函数、API 客户端
│  │  ├─ pages/             # 路由页面
│  │  ├─ stores/            # Zustand 状态容器
│  │  └─ types/             # TS 类型定义
│  ├─ public/
│  ├─ tests/                # Vitest/RTL 单测
│  └─ playwright/           # 端到端测试配置
└─ ...
```

## 接口契约（BFF 层）
- `GET /api/clusters/summary`：集群健康度、告警统计、关键指标（CPU、内存、故障率）。
- `GET /api/events`：分页获取事件列表，支持参数 `cluster`、`namespace`、`severity`、`keyword`、`since`。
- `GET /api/events/{id}`：事件详情，包含根因分析、关联资源、Playbook 输出。
- `GET /api/playbooks/runs`：自动化剧本执行历史，含状态、耗时、触发来源。
- `GET /api/metrics/timeseries`：指标时间序列数据，参数 `metric`, `cluster`, `namespace`, `range`。
- `GET /api/events/stream`：SSE 通道，推送新增事件。

## 权限与安全
- 默认对接企业 IAM/OIDC，前端使用 JWT 存储在 HttpOnly Cookie；所有请求需携带认证。
- 前端路由基于权限守卫，防止越权访问；关键操作记录审计日志。

## 性能考虑
- 按需拆分路由与特性模块，启用 React 懒加载。
- 使用 Tailwind JIT 自定义预设，减少生成 CSS 体积。
- 数据层设置合理的缓存时间与错误重试策略，避免对 Runner 造成压力。

## 测试策略
- 单元测试覆盖核心交互逻辑、API 适配器、状态容器。
- 使用 Playwright 进行关键路径的端到端验证（事件筛选、详情查看）。
- 导出 Storybook（后续扩展）便于组件与状态组合测试。

## 迭代路线
1. MVP：仪表盘 + 事件列表 + 事件详情，支持基础筛选与实时提醒。
2. 扩展：Playbook 运行记录、指标可视化、角色权限管理。
3. 优化：多集群切换体验、移动端适配、可插拔的数据源。
