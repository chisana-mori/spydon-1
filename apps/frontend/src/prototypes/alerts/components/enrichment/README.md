# Enrichment 渲染系统

基于 `docs/rawplay.md` 设计文档实现的 Enrichment 解析与美化渲染模块。

## 功能特性

- ✅ 支持多种 Block 类型的解析和渲染
- ✅ 美化的交互式 UI 展示
- ✅ 可折叠/展开的内容区域
- ✅ 复制、下载等实用功能
- ✅ 支持原始 JSON/YAML 视图切换
- ✅ 完全类型安全（TypeScript）

## 支持的 Block 类型

### 1. Markdown Block
显示格式化的文本内容。

### 2. Header Block
显示标题（支持不同级别）。

### 3. JSON Block
- 可折叠/展开
- 语法高亮
- 一键复制功能

### 4. K8s Diff Block
- 显示 Kubernetes 资源的变更
- 支持 add/remove/replace 操作
- 新旧值对比展示
- 颜色区分不同操作类型

### 5. Table Block
- 使用 shadcn/ui Table 组件
- 支持表头和多行数据

### 6. List Block
- 支持有序/无序列表
- 自动格式化

### 7. Links Block
- 显示相关链接
- 外部链接图标
- 新标签页打开

### 8. Text File Block
- 显示文件内容
- 支持预览和完整展示
- 下载功能

### 9. Generic Block
- 处理未知类型
- 显示原始 JSON 数据

## 使用方法

### 基本用法

```tsx
import { RawPayloadViewer } from '@prototypes/alerts/components/alerts/RawPayloadViewer';

<RawPayloadViewer
  rawPayloadKey="path/to/payload"
  alertId="alert-123"
  alertTitle="Alert Title"
/>
```

### 单独使用 Enrichment 渲染器

```tsx
import { EnrichmentRenderer } from '@prototypes/alerts/components/enrichment';
import { Enrichment } from '@prototypes/alerts/types/enrichment';

const enrichment: Enrichment = {
  title: "AI Analysis",
  enrichment_type: "ai_analysis",
  blocks: [
    {
      type: "markdown",
      text: "This is a markdown block"
    },
    {
      type: "json",
      data: { key: "value" }
    }
  ]
};

<EnrichmentRenderer enrichment={enrichment} />
```

### 解析原始数据

```tsx
import { parseFinding } from '@prototypes/alerts/lib/enrichment-parser';

const rawData = { /* ... */ };
const finding = parseFinding(rawData);

// 访问解析后的数据
finding.enrichments?.forEach(enrichment => {
  console.log(enrichment.title);
  enrichment.blocks.forEach(block => {
    console.log(block.type);
  });
});
```

## 扩展新的 Block 类型

### 1. 定义类型

在 `types/enrichment.ts` 中添加新的 Block 接口：

```typescript
export interface CustomBlock extends BaseBlock {
  type: "custom";
  customField: string;
}

export type Block = 
  | MarkdownBlock
  | CustomBlock  // 添加到联合类型
  | ...;
```

### 2. 创建解析器

在 `lib/enrichment-parser.ts` 中注册解析器：

```typescript
function parseCustomBlock(raw: any): CustomBlock {
  return {
    id: raw.id || crypto.randomUUID(),
    type: 'custom',
    customField: raw.customField,
    raw,
  };
}

// 注册
registry.register('custom', parseCustomBlock);
```

### 3. 创建渲染组件

创建 `components/enrichment/CustomBlock.tsx`：

```tsx
'use client';

import React from 'react';
import { CustomBlock as CustomBlockType } from '../../types/enrichment';

interface CustomBlockProps {
  block: CustomBlockType;
}

export function CustomBlock({ block }: CustomBlockProps) {
  return (
    <div>
      {block.customField}
    </div>
  );
}
```

### 4. 更新 BlockRenderer

在 `BlockRenderer.tsx` 中添加新的 case：

```tsx
case 'custom':
  return <CustomBlock block={block as any} />;
```

## 架构说明

```
types/enrichment.ts          # 类型定义
lib/enrichment-parser.ts     # 解析逻辑
components/enrichment/
  ├── EnrichmentRenderer.tsx # 主渲染器
  ├── BlockRenderer.tsx      # Block 分发器
  ├── MarkdownBlock.tsx      # 各种 Block 组件
  ├── JsonBlock.tsx
  ├── K8sDiffBlock.tsx
  └── ...
```

## 视图模式

RawPayloadViewer 支持 4 种视图模式：

1. **美化视图** (formatted) - 默认，使用 Enrichment 渲染器
2. **JSON 视图** - 格式化的 JSON
3. **YAML 视图** - 简化的 YAML 格式
4. **原始视图** - 未格式化的原始数据

## 注意事项

- 所有 Block 组件都是客户端组件（'use client'）
- 使用 shadcn/ui 组件库保持 UI 一致性
- 支持深色模式
- 所有交互操作都有适当的错误处理
