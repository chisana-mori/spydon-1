# Enrichment 解析与渲染设计文档（Next.js 实现）

概述
- 目的：在 Next.js 中实现稳健的 Enrichment 解析与美化渲染模块，使其既能在服务端解析、生成可供前端渲染的结构化数据，也能在客户端以富交互组件进行展示，兼容各种 block 类型（markdown, header, json, k8s_diff, list, table, links, text_file 等）。
- 运行环境假设：Next.js 14+，TypeScript，React 18+，可部署到 Vercel 或其他支持 Edge/Serverless 的平台。

总体架构（高层）
- 接收层（Webhook / upstream）将 JSON payload 发送到我们的 Next.js 后端 API（/api/parse-finding 或 /api/receive-finding）。
- 后端解析器（lib/parser）：
  - 做 JSON Schema 校验（可选），按 block.type 调用 Block Parser 工厂，输出标准化的内部 DTO（TypeScript interface）。
  - 可做敏感信息掩码、截断、annotations 解析、version tagging。
- 存储 / 缓存（可选）：解析后的对象可以写入数据库（Postgres / Mongo）或缓存（Redis）以便历史记录与分页。
- 前端渲染层（pages 或 app route + components）：使用 EnrichmentRenderer 组件树渲染每个 enrichment 与其 blocks。
- 渲染适配器：
  - TextAdapter（用于生成短文本/邮件/简单 webhook 输出）
  - JSONAdapter（用于转发/存档）
  - UI Adapter（组件树，交互丰富）
- 扩展点：Block Parser/Renderer 注册表（plugin-like），便于新增类型。

数据模型（TypeScript 接口建议）
- 放置于: /lib/types.ts

```ts
export type EnrichmentType =
  | "graph"
  | "ai_analysis"
  | "node_info"
  | "container_info"
  | "k8s_events"
  | "alert_labels"
  | "diff"
  | "text_file"
  | "crash_info"
  | "image_pull_backoff_info"
  | "pending_pod_info"
  | string; // allow extension

export interface Link {
  url: string;
  name?: string;
  type?: string;
}

export interface FindingSubject {
  name?: string;
  subject_type?: string;
  namespace?: string;
  node?: string;
  container?: string;
  labels?: Record<string,string>;
  annotations?: Record<string,string>;
}

export interface BaseBlock {
  id?: string; // UUID optional, useful for keying React lists
  type: string;
  // rawPayload preserves original data for unknown blocks or download
  raw?: any;
}

export interface MarkdownBlock extends BaseBlock {
  type: "markdown";
  text: string;
}

export interface HeaderBlock extends BaseBlock {
  type: "header";
  text: string;
  level?: number;
}

export interface JsonBlock extends BaseBlock {
  type: "json";
  data: any;
  json_str?: string;
}

export interface K8sDiffItem {
  path: string[]; // e.g. ["spec","containers","0","image"]
  other_value?: any;
  value?: any;
  op?: "add"|"remove"|"replace";
}

export interface K8sDiffBlock extends BaseBlock {
  type: "k8s_diff" | "diff";
  diffs: K8sDiffItem[];
}

export interface ListBlock extends BaseBlock {
  type: "list";
  items: (string|any)[];
  ordered?: boolean;
}

export interface TableBlock extends BaseBlock {
  type: "table";
  headers: string[];
  rows: (string|number|any)[][];
  table_name?: string;
}

export interface LinksBlock extends BaseBlock {
  type: "links";
  links: Link[];
}

export interface TextFileBlock extends BaseBlock {
  type: "text_file" | "file";
  filename?: string;
  contents?: string; // or content_ref url
}

export interface GenericBlock extends BaseBlock {
  type: "generic";
  raw: any;
}

export type Block =
  | MarkdownBlock
  | HeaderBlock
  | JsonBlock
  | K8sDiffBlock
  | ListBlock
  | TableBlock
  | LinksBlock
  | TextFileBlock
  | GenericBlock;

export interface Enrichment {
  title?: string;
  enrichment_type?: EnrichmentType;
  annotations?: Record<string,string>;
  blocks: Block[];
}

export interface Finding {
  id?: string;
  title?: string;
  description?: string;
  severity?: string;
  aggregation_key?: string;
  subject?: FindingSubject;
  links?: Link[];
  enrichments?: Enrichment[];
  fingerprint?: string;
  creation_date?: string;
  starts_at?: string;
  ends_at?: string;
  raw?: any;
}
```

解析层（lib/parser.ts）设计
- 目标：把任意 webhook payload 转为上面 DTO，做校验与容错。
- 核心模块：
  - BlockParserRegistry: register(typeName, parserFn)
  - DefaultParsers: 实现内置类型解析（markdown, header, json, k8s_diff, list, table, links, text_file）
  - parseBlock(raw): 调用 registry 或启发式识别并返回 Block
  - parseEnrichment(raw): 遍历 blocks，解析 annotations/title/enrichment_type，并返回 Enrichment
  - parseFinding(raw): 顶层解析，返回 Finding
- 放置位置： /lib/parser.ts

示例（伪码）：

```ts
// /lib/parser.ts
import { Block, Enrichment, Finding } from "./types";

type BlockParser = (raw: any) => Block;

const registry = new Map<string, BlockParser>();

export function registerBlockParser(typeName: string, parser: BlockParser) {
  registry.set(typeName, parser);
}

export function parseBlock(raw: any): Block {
  const t = raw?.type;
  if (t && registry.has(t)) {
    return registry.get(t)!(raw);
  }
  // fallback heuristics
  if (raw?.diffs || raw?.changes) {
    return parseK8sDiff(raw);
  }
  if (raw?.data || raw?.json || raw?.json_str) {
    return parseJsonBlock(raw);
  }
  if (raw?.items) {
    return parseListBlock(raw);
  }
  if (raw?.headers && raw?.rows) {
    return parseTableBlock(raw);
  }
  if (raw?.text && !raw?.headers) {
    // ambiguous: treat as markdown
    return parseMarkdown(raw);
  }
  // unknown
  return { type: "generic", raw };
}
```

注册默认解析器（在初始化时调用）：
- registerBlockParser("markdown", parseMarkdown)
- registerBlockParser("k8s_diff", parseK8sDiff)
- etc.

后端 API 设计（Next.js API Routes）
- POST /api/parse-finding
  - 功能：接收原始 payload，调用 parseFinding -> 返回 parsed DTO（可选写入 DB）
  - 用途：用于 webhook 接收或手动上载 sample
- GET /api/finding/[id]
  - 返回已存储的 parsed Finding
- GET /api/render/finding/[id]?mode=text|json
  - 返回 text 或 JSON 渲染版本（TextAdapter / JSONAdapter）

实现注意：
- 若需要在 Edge 上运行（Vercel Edge Functions），确保依赖兼容。
- 若 payload 可能很大，使用 streaming 或 save-to-storage 后在后台解析。

前端组件设计（React + TypeScript）
- 组件目录： /components/enrichment/
- 主要组件：
  - EnrichmentRenderer.tsx
  - BlockRenderer.tsx (dispatcher)
  - MarkdownBlock.tsx
  - HeaderBlock.tsx
  - JsonBlock.tsx (带展开/折叠，高亮)
  - K8sDiffBlock.tsx (可折叠、支持 old/new 高亮)
  - ListBlock.tsx
  - TableBlock.tsx
  - LinksBlock.tsx
  - TextFileBlock.tsx
  - GenericBlock.tsx

组件接口示例：

```tsx
// /components/enrichment/EnrichmentRenderer.tsx
import React from "react";
import { Enrichment } from "@/lib/types";
import BlockRenderer from "./BlockRenderer";

export default function EnrichmentRenderer({ enrichment }: { enrichment: Enrichment }) {
  return (
    <section className="enrichment">
      <h3>{enrichment.title ?? enrichment.enrichment_type ?? "Enrichment"}</h3>
      {enrichment.annotations && (
        <div className="annotations">
          {Object.entries(enrichment.annotations).map(([k,v])=> <span key={k} className="tag">{k}: {v}</span>)}
        </div>
      )}
      <div className="blocks">
        {enrichment.blocks.map((b) => (
          <BlockRenderer key={b.id ?? JSON.stringify(b)} block={b} />
        ))}
      </div>
    </section>
  );
}
```

BlockDispatcher（简化）：

```tsx
// /components/enrichment/BlockRenderer.tsx
import React from "react";
import { Block } from "@/lib/types";
import MarkdownBlock from "./MarkdownBlock";
import JsonBlockRenderer from "./JsonBlock";
import K8sDiffBlock from "./K8sDiffBlock";
import GenericBlock from "./GenericBlock";
// ... other imports

export default function BlockRenderer({ block }: { block: Block }) {
  switch (block.type) {
    case "markdown":
      return <MarkdownBlock block={block} />;
    case "header":
      return <h4>{(block as any).text}</h4>;
    case "json":
      return <JsonBlockRenderer block={block} />;
    case "k8s_diff":
    case "diff":
      return <K8sDiffBlock block={block} />;
    case "list":
      return <ListBlock block={block} />;
    case "table":
      return <TableBlock block={block} />;
    case "links":
      return <LinksBlock block={block} />;
    case "text_file":
    case "file":
      return <TextFileBlock block={block} />;
    default:
      return <GenericBlock block={block} />;
  }
}
```

交互与 UX 建议
- JsonBlock: 默认折叠，提供“展开/复制/下载”按钮，支持语法高亮（react-syntax-highlighter）。
- K8sDiffBlock: 支持按 diff 路径折叠、并行对比 old/new（颜色区分）。提供“复制 patch”按钮。
- TableBlock: 支持列排序/过滤（若行数多）。
- LinksBlock: 每个 link 可有图标（image/video/prometheus）。
- TextFileBlock: 默认显示前 N 行，提供“下载完整文件”或“查看全部”。
- Accessibility: 所有可交互控件遵守 ARIA，键盘可达。

服务端渲染 (SSR) 与客户端渲染 (CSR) 策略
- SSR (getServerSideProps / app route server components): 适合 SEO 或首屏需在服务端渲染时显示的场景；如果 payload 需要在请求时解析并立即显示，可使用服务器端渲染。
- CSR (客户端 fetch): 若渲染逻辑高度交互且不影响 SEO，可在客户端 fetch /api/finding/[id] 后渲染。JsonBlock 展开状态等可由客户端控制。
- 建议：对大 JSON 或敏感数据使用 server-side parsing，并在前端仅请求已解析且做了必要脱敏的数据。

安全、隐私与大小控制
- size_limit：在解析器处理时应用（WebookSink.size_limit 对应），在 API 返回时也应保证不会泄露超大内容（对 text_file contents 做 snippet）。
- 敏感信息掩码：实现 ruleset（regex 或 key whitelist/blacklist），在 parse 层或 output adapter 层掩码 secrets。
- XSS 防护：在渲染 Markdown/HTML 时使用安全渲染器（如 remark + rehype + sanitize）并禁止内联脚本或危险 URL。
- CORS & Auth：保护 API 路由（webhook endpoint）用签名/认证 token，避免滥用和数据泄露。

降级策略（text-only sinks）
- TextAdapter（server 或 API route）负责把 Enrichment 渲染成短文本：
  - 按 priority 渲染：title -> key blocks (diff/json/markdown) -> links
  - 全文超过 size_limit 时仅保留前 N 字节并加 "...(truncated)"
  - 无法识别的 block 输出 "Unknown block: raw JSON snippet"

部署与扩展
- 将解析器与渲染器作为独立模块（/lib 和 /components），便于在其他项目复用。
- 新增 block 类型只需实现 parser + renderer，并在初始化时注册。
- 可为前端提供 component registry，以便通过配置按 enrichment_type 切换渲染策略（例如 ai_analysis 使用专用 summary 卡片）。

测试策略
- 单元测试（Jest/React Testing Library）：
  - Parser: 各种 raw block -> 对应 Block DTO
  - Renderers: 各个组件 snapshot 测试、交互测试（展开/复制/下载）
- 集成测试：
  - API route: POST /api/parse-finding -> 返回 expected DTO
  - End-to-end：使用 Playwright 测试 UI 渲染（展开 JSON、diff 高亮）
- Round-trip tests：
  - 构造 Finding -> serialize -> parse -> render -> 再序列化，验证关键语义（diff entries, labels, links）一致。
