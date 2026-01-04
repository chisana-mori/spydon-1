# HolmesGPT 分析组件使用指南

## 概述

本项目默认提供 **EnhancedHolmesGPTChat** 作为 HolmesGPT 分析界面的唯一实现，它整合了实时任务进度、增强的聊天体验以及结论汇总等能力。

## 快速开始

### 使用 EnhancedHolmesGPTChat

```tsx
import { EnhancedHolmesGPTChat } from '@/components/alerts/EnhancedHolmesGPTChat'

function AlertAnalysisPage({ alert }: { alert: Alert }) {
  return (
    <div className="container mx-auto p-6">
      <div className="max-w-4xl mx-auto">
        <EnhancedHolmesGPTChat alert={alert} />
      </div>
    </div>
  )
}
```

## JSON 数据解析

组件支持解析以下 JSON 格式：

### 1. 任务更新格式
```json
{
  "tool_name": "TodoWrite",
  "result": {
    "data": "✅ Investigation plan updated with 5 tasks...",
    "params": {
      "todos": [
        {
          "id": "1",
          "content": "检查 pod 的详细状态",
          "status": "completed"
        }
      ]
    }
  }
}
```

### 2. 工具调用格式
```json
{
  "tool_name": "kubectl_describe",
  "id": "call_xxx",
  "result": {
    "status": "success",
    "data": "Pod 详细信息..."
  }
}
```

### 3. 最终报告格式
```json
{
  "content": "# 问题分析报告\n\n## 问题描述\n...",
  "reasoning": null
}
```

## 功能特性

### 任务状态管理

组件会自动解析和显示任务状态：

- 🔴 **待处理 (pending)**: 尚未开始的任务
- 🔵 **进行中 (in_progress)**: 正在执行的任务，带有动画效果
- 🟢 **已完成 (completed)**: 已完成的任务，带有完成标识

### Markdown 渲染增强

- **代码块**: 支持语法高亮和一键复制/下载
- **表格**: 响应式表格布局
- **引用**: 美化的引用样式
- **列表**: 优化的列表间距和样式

### 实时进度显示

```typescript
// 自动计算任务进度
const progress = parseProgress(structuredData)
// 返回: { total: 6, completed: 4 }
```

### 设置选项 (仅增强组件)

- **自动滚动**: 控制消息自动滚动行为
- **音效提示**: 分析完成时播放提示音
- **工具调用显示**: 控制是否显示详细的工具调用信息

## 自定义样式

### 主题颜色

```css
/* 任务状态颜色 */
.task-pending { @apply text-gray-500; }
.task-in-progress { @apply text-blue-600; }
.task-completed { @apply text-green-600; }

/* 分析阶段颜色 */
.analysis-plan { @apply bg-blue-50 border-blue-200; }
.analysis-conclusion { @apply bg-green-50 border-green-200; }
```

### 响应式布局

组件已适配移动端，推荐容器宽度：

- **桌面端**: `max-w-4xl` 或 `max-w-6xl`
- **平板**: `max-w-2xl`
- **手机**: `w-full`

## API 集成

### 后端接口要求

```typescript
// POST /api/holmesgpt/stream/investigate
interface InvestigateRequest {
  source: string
  title: string
  description: string
  subject: AlertSubject
  context: AnalysisContext
  include_tool_calls: boolean
  include_tool_call_results: boolean
}

// SSE 响应格式
interface StreamResponse {
  type: 'analysis' | 'analysis_chunk' | 'tool_call' | 'complete' | 'error'
  data: any
}
```

### 错误处理

组件内置错误处理机制：

```typescript
try {
  // 分析逻辑
} catch (error) {
  if (error.name === 'AbortError') {
    // 用户中止操作
  } else {
    // 其他错误，显示错误信息
    setAnalysisState(prev => ({
      ...prev,
      status: 'error',
      error: error.message
    }))
  }
}
```

## 性能优化

### 1. 内存管理

- 使用 `AbortController` 管理网络请求
- 及时清理事件监听器
- 优化大量消息的渲染性能

### 2. 渲染优化

```typescript
// 使用 React.memo 优化子组件
const ChatMessage = React.memo(({ ... }) => {
  // 组件逻辑
})

// 避免不必要的重新渲染
const memoizedStructuredData = useMemo(() => {
  return extractHolmesStructuredData(rawData)
}, [rawData])
```

### 3. 流式数据处理

```typescript
// 增量解析 JSON 流
const parseJsonObjectSequence = (raw: string): any[] => {
  // 逐步解析 JSON 对象，避免阻塞 UI
}
```

## 故障排除

### 常见问题

1. **任务状态不更新**
   - 检查 JSON 格式是否正确
   - 确认 `tool_call_id` 匹配

2. **Markdown 渲染异常**
   - 检查 `react-markdown` 依赖版本
   - 确认 `remarkGfm` 插件加载

3. **音效不播放**
   - 检查浏览器音频权限
   - 确认音频文件路径正确

### 调试模式

```typescript
// 启用调试日志
const DEBUG = process.env.NODE_ENV === 'development'

if (DEBUG) {
  console.log('Parsed structured data:', structuredData)
}
```

## 未来改进

- [ ] 支持多语言切换
- [ ] 添加历史记录管理
- [ ] 集成语音识别输入
- [ ] 支持自定义主题
- [ ] 添加分析结果导出为 PDF

## 技术栈

- **React 18** + **TypeScript**
- **Tailwind CSS** - 样式系统
- **shadcn/ui** - UI 组件库
- **react-markdown** - Markdown 渲染
- **react-syntax-highlighter** - 代码高亮
- **date-fns** - 日期格式化
- **lucide-react** - 图标库
