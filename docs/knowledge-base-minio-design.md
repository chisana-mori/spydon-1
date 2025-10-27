---

# 告警知识库（MinIO 存储）设计说明

版本：v1.0  日期：2025-10-23  作者：工程团队

## 1. 背景与目标

- 背景：平台已有基础 AI 分析能力，需要沉淀“按告警规则名”的处理经验，形成结构化知识库，并在告警详情页自动关联展示。
- 目标：知识库正文与资源统一存入 MinIO（对象存储），数据库仅保存一个对象 `key`（即清单/manifest 的对象键），通过该 key 关联全文与附件；前端使用 TipTap 提供富文本与 Markdown 编辑能力；告警详情页按规则名自动匹配展示。

## 2. 总体架构

- 存储分层：
  - 数据库（Postgres）：仅存知识条目元数据与 MinIO 主对象 key（manifest）。
  - MinIO：存放知识条目的清单（manifest.json）与所有富文本附件（图片/文件）。
- 访问方式：
  - 读取：后端通过 `object_key` 拉取 manifest 并返回（或直传前端），附件使用后端代理或预签名 URL 获取。
  - 写入：编辑时通过后端获取预签名上传 URL（直传 MinIO），最终由后端写入/更新 manifest，并将 manifest 的对象 key 写入数据库。

## 3. MinIO 存储设计

- Bucket：`knowledge-base`
- 对象组织（仅示例，不依赖真实目录）：
  - `knowledge/{articleId}/manifest.json`（主对象，DB 仅保存此 key）
  - `knowledge/{articleId}/assets/{uuid}.{ext}`（附件/图片）
  - `knowledge/{articleId}/versions/{version}/manifest.json`（历史版本，可选）

### 3.1 manifest.json 结构（单一主对象）

```json
{
  "schema": "kb-manifest@v1",
  "articleId": "9b2e6b2a-e8e1-4a7f-9dbe-8d2d6c5a51f1",
  "alertRuleName": "KubePodCrashLooping",
  "title": "Pod 反复重启的排查与修复",
  "severity": "high",
  "tags": ["kubernetes", "pod", "crashloopbackoff"],
  "status": "published",
  "version": 3,
  "excerpt": "定位 CrashLoopBackOff 的通用方法与常见根因...",
  "content": {
    "tiptap": { "type": "doc", "content": [/* ProseMirror JSON */] },
    "markdown": "# Pod CrashLoopBackOff 排查..."
  },
  "assets": [
    {
      "name": "diagram.png",
      "key": "knowledge/9b2e6b2a/assets/f0b1b2c3.png",
      "mime": "image/png",
      "size": 123456
    }
  ],
  "scopes": { "cluster": null, "env": null },
  "audit": {
    "createdBy": "alice",
    "createdAt": "2025-10-21T12:00:00Z",
    "updatedBy": "bob",
    "updatedAt": "2025-10-23T08:00:00Z",
    "changeSummary": "补充镜像拉取失败场景"
  }
}
```

说明：
- `manifest.json` 同时保存 TipTap JSON 与 Markdown 文本，保证渲染与可迁移性；
- `assets` 仅保存对象 key 与元数据，数据库无需存附属 key；
- 历史版本可复制 `manifest.json` 至 `versions/{version}`，实现对象级版本化。

## 4. 数据模型（数据库）

表 `knowledge_articles`
- `id` bigint PK
- `alert_rule_name` varchar(255)
- `alert_rule_name_normalized` varchar(255)（匹配索引，统一大小写/空白/分隔符）
- `title` varchar(255)
- `severity` varchar(16) null（low/medium/high/critical）
- `tags` jsonb null（字符串数组）
- `status` varchar(16) 默认 `draft`（draft/published/archived）
- `object_key` varchar(512) not null（MinIO 主对象 key：`knowledge/{articleId}/manifest.json`）
- `version` int not null default 1
- `created_by` varchar(128), `updated_by` varchar(128)
- `created_at` timestamptz, `updated_at` timestamptz

表 `knowledge_article_versions`（可选，但推荐）
- `id` bigint PK
- `article_id` bigint FK
- `version` int
- `object_key` varchar(512)（对应 `versions/{version}/manifest.json`）
- `change_summary` text
- `created_by` varchar(128), `created_at` timestamptz

索引：
- `idx_kb_rule_norm_status(alert_rule_name_normalized, status)`
- `gin` 索引在 `tags`

迁移样例：见第 10 节。

## 5. 后端接口设计（Go）

命名空间：`/api/knowledge`

- `GET /api/knowledge?alert_rule_name=...&cluster=...&env=...`：按规则名检索（仅返回 `published`，按相关性/更新时间排序，默认 1 条 + 更多计数）
- `GET /api/knowledge/:id`：获取条目（含 manifest 解析后的渲染数据或直返 manifest）
- `POST /api/knowledge`：创建草稿（生成 `articleId`，返回 `object_key` 与 `uploadPrefix`）
- `PUT /api/knowledge/:id`：更新（服务端写 manifest 并更新 `object_key` 版本）
- `POST /api/knowledge/:id/publish`：发布（`status=published`，拷贝 manifest 至 `versions/{version}`）
- `POST /api/knowledge/:id/archive`：归档
- `GET /api/knowledge/search?q=...&tags=...&severity=...`：检索分页
- `POST /api/knowledge/upload/init`：初始化上传会话（返回预签名前缀/策略）
- `POST /api/knowledge/upload/presign`：为指定 `filename/mime` 生成预签名 URL（对象 key 按规则生成）

响应建议：
- 创建：`{ id, object_key, article_id, upload_prefix }`
- 检索：`{ items: [{ id, title, object_key, updated_at, ... }], total }`

服务层：
- `KnowledgeService`：规范化匹配、别名解析、对象读写、版本化、预签名 URL 生成、权限与审计。

MinIO 交互：
- 读：`GetObject(object_key)` 读取 manifest；
- 写：`PutObject(object_key, manifestBytes)`；
- 预签名：`PresignPutObject(bucket, key, ttl)` 返回上传 URL；
- 删除（可选）：归档后清理旧版本策略。

## 6. 匹配策略

- 规范化：统一大小写、去除空白、统一分隔符（`-`/`_`/空格 → `-`），中文保留，生成 `alert_rule_name_normalized`。
- 别名：可扩展 `alert_rule_aliases`（`name_normalized` ↔ `alias_normalized`），在检索时优先 exact 命中，否则 alias 命中，最后模糊（`LIKE`）。
- 多条结果：默认展示最新 `published` 一条；“更多经验 (N)” 折叠展示。

## 7. 前端设计（Next.js + TipTap）

- 编辑器组件：`apps/frontend/src/components/knowledge/KnowledgeEditor.tsx`
  - 使用 TipTap（StarterKit、Markdown、Link、Image、Table、CodeBlock 等），`ssr: false` 动态加载。
  - 图片上传：调用后端 `presign`，前端直传 MinIO，成功后将对象 `key` 写入编辑状态；保存时由后端写入 manifest。
- 展示组件：`apps/frontend/src/components/knowledge/KnowledgeViewer.tsx`
  - 读取 manifest（只读 TipTap 渲染，或回退 Markdown 渲染）。
- 告警详情集成：`apps/frontend/src/components/alerts/AlertKnowledgePanel.tsx`
  - 按 `alert.rule`（或回退 `alert.title`）请求 `/api/knowledge`，展示命中条目与“更多经验”。
  - 空态：按钮“基于当前告警创建知识条目”，预填规则名与标签。
- 配置开关：`apps/frontend/src/config/index.ts` 增加 `knowledgeBaseEnabled`。

## 8. 安全与权限

- 权限：
  - Viewer：只读查询
  - Maintainer：创建/编辑/发布/上传
  - Admin：管理全部与归档
- 审计：记录 `create/update/publish/archive/upload`（存摘要与 key，不存大文本）。
- 预签名安全：
  - TTL（例如 10 分钟）；
  - 限制 `content-type` 与 `key` 前缀（仅允许 `knowledge/{articleId}/assets/`）。
- XSS：
  - TipTap 渲染开启安全白名单，Markdown 渲染开启 sanitize；
  - 禁止内联脚本，过滤危险 HTML。

## 9. 配置与部署

- 环境变量（后端）：
  - `MINIO_ENDPOINT`、`MINIO_REGION`（可选）
  - `MINIO_ACCESS_KEY`、`MINIO_SECRET_KEY`
  - `MINIO_BUCKET=knowledge-base`
  - `MINIO_USE_SSL=true|false`
  - `MINIO_PRESIGN_TTL=600`（秒）
- Bucket 策略：
  - 私有读写；所有访问通过后端或预签名 URL；
  - 生命周期（可选）：旧版本定期归档/清理。

## 10. 数据迁移（SQL 示例）

```sql
-- knowledge_articles
CREATE TABLE IF NOT EXISTS knowledge_articles (
  id BIGSERIAL PRIMARY KEY,
  alert_rule_name VARCHAR(255) NOT NULL,
  alert_rule_name_normalized VARCHAR(255) NOT NULL,
  title VARCHAR(255) NOT NULL,
  severity VARCHAR(16),
  tags JSONB,
  status VARCHAR(16) NOT NULL DEFAULT 'draft',
  object_key VARCHAR(512) NOT NULL,
  version INT NOT NULL DEFAULT 1,
  created_by VARCHAR(128),
  updated_by VARCHAR(128),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_kb_rule_norm_status
  ON knowledge_articles (alert_rule_name_normalized, status);

CREATE INDEX IF NOT EXISTS idx_kb_tags_gin ON knowledge_articles USING GIN (tags);

-- knowledge_article_versions（可选）
CREATE TABLE IF NOT EXISTS knowledge_article_versions (
  id BIGSERIAL PRIMARY KEY,
  article_id BIGINT NOT NULL REFERENCES knowledge_articles(id) ON DELETE CASCADE,
  version INT NOT NULL,
  object_key VARCHAR(512) NOT NULL,
  change_summary TEXT,
  created_by VARCHAR(128),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_kb_versions_unique
  ON knowledge_article_versions (article_id, version);
```

## 11. 版本与发布流程

- 状态机：`draft -> published -> archived`；从 `published` 更新产生 `version+1`。
- 版本对象：发布时将当前 `manifest.json` 复制至 `versions/{version}/manifest.json`，并写一条版本记录。
- 缓存：
  - 后端 LRU/TTL（60s）缓存按 `(rule_norm, cluster, env, status)`；
  - 前端 React Query `staleTime: 30s`。

## 12. 流程时序（创建/编辑/发布）

1) 创建：
   - 前端 `POST /api/knowledge` → 后端生成 `articleId` 与 `object_key=knowledge/{articleId}/manifest.json`，并返回。
   - 返回上传前缀 `knowledge/{articleId}/assets/` 供后续附件上传。

2) 上传附件：
   - 前端 `POST /api/knowledge/upload/presign`，携带 `articleId/filename/mime/size`；
   - 后端返回预签名 `PUT` URL 与最终 `key`；前端直传 MinIO；
   - 前端将 `key` 写入编辑状态（资产列表）。

3) 保存/更新：
   - 前端 `PUT /api/knowledge/:id` 携带 `content.tiptap`、`content.markdown`、`assets`、`title`、`tags` 等；
   - 后端写入 `manifest.json`（同 `object_key`），更新 `version` 与 `updated_by`。

4) 发布：
   - 后端将当前 `manifest.json` 复制到 `versions/{version}/manifest.json`，更新 DB `status=published`。

## 13. 安全与合规

- 上传限制：白名单（png/jpg/gif/pdf/svg），大小上限 5MB，后续可接杀毒/内容扫描；
- 链接安全：预签名 URL 只允许匹配到当前 `articleId` 的 `assets/` 前缀；
- 隐私治理：敏感信息脱敏（如内网地址/凭据片段）；
- 审计留痕：完整记录但不落正文全文到审计表以控体积。

## 14. 测试计划

- 单测（后端）：规范化与匹配、CRUD、版本化、预签名 URL 权限、MinIO 失败回退；
- 组件测（前端）：Editor 工具栏、图片上传、Viewer 渲染、详情页命中逻辑；
- 集成测：创建→上传→保存→发布→详情页自动命中；
- 性能基线：详情页引入知识库后渲染不超过原先 +100ms。

## 15. 里程碑

- M1：DB/MinIO 基础骨架（manifest 读写、presign 上传）
- M2：只读 Viewer 与详情页集成（自动匹配）
- M3：Editor 与发布流（版本化）
- M4：管理页与检索（搜索、标签、别名）
- M5：优化与安全（全文检索、生命周期、扫描）

## 16. 验收标准

- 数据库仅保留一个主对象 `object_key` 即可完整读取正文与附件；
- 告警详情页依据规则名自动展示知识库命中内容；
- 编辑器支持 Markdown/富文本与图片上传，发布后版本可回溯；
- 安全策略（权限/预签名/审计）到位，测试通过。

---

附：关键代码位置建议
- 后端：
  - 模型 `apps/backend/internal/models/knowledge.go`
  - 仓储 `apps/backend/internal/repo/knowledge_repo.go`
  - 服务 `apps/backend/internal/services/knowledge_service.go`
  - 接口 `apps/backend/internal/api/knowledge_handler.go`
  - 路由 `apps/backend/internal/api/routes.go`
- 前端：
  - 编辑器 `apps/frontend/src/components/knowledge/KnowledgeEditor.tsx`
  - 展示 `apps/frontend/src/components/knowledge/KnowledgeViewer.tsx`
  - 详情集成 `apps/frontend/src/components/alerts/AlertKnowledgePanel.tsx`

