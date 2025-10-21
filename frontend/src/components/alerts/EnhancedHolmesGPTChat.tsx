'use client'

import React, { useState, useRef, useEffect, useCallback } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { 
  Brain, 
  Play, 
  Square, 
  RotateCcw, 
  Download,
  Loader2,
  AlertCircle,
  CheckCircle2,
  MessageSquare,
  Settings,
  Volume2,
  VolumeX
} from 'lucide-react'
import { Alert } from '@/types/api'
import { ChatMessage, HolmesStructuredData, HolmesTaskItem, HolmesTaskSection } from './ChatMessage'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'

interface EnhancedHolmesGPTChatProps {
  alert: Alert
  showCard?: boolean  // 控制是否显示外层Card
}

interface AnalysisMessage {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  timestamp: string
  isStreaming?: boolean
  toolCalls?: Array<{
    name: string
    input: any
    output?: any
    status: 'pending' | 'success' | 'error'
    command?: string
  }>
  structuredData?: HolmesStructuredData
}

interface AnalysisState {
  status: 'idle' | 'analyzing' | 'completed' | 'error'
  messages: AnalysisMessage[]
  error?: string
  totalSteps?: number
  completedSteps?: number
}

interface ChatSettings {
  autoScroll: boolean
  soundEnabled: boolean
  showToolCalls: boolean
  language: 'zh-CN' | 'en-US'
}

export const EnhancedHolmesGPTChat: React.FC<EnhancedHolmesGPTChatProps> = ({ 
  alert, 
  showCard = true 
}) => {
  const [analysisState, setAnalysisState] = useState<AnalysisState>({
    status: 'idle',
    messages: []
  })
  const [abortController, setAbortController] = useState<AbortController | null>(null)
  const [settings, setSettings] = useState<ChatSettings>({
    autoScroll: false,
    soundEnabled: false,
    showToolCalls: true,
    language: 'zh-CN'
  })
  const [showSettings, setShowSettings] = useState(false)
  const allTasksCompletedRef = useRef(false);
  // 固定在底部的任务与结论卡片
  const [pinnedTasksData, setPinnedTasksData] = useState<HolmesStructuredData | undefined>(undefined)
  const [pinnedSummaryData, setPinnedSummaryData] = useState<HolmesStructuredData | undefined>(undefined)
  const tasksMessageIdRef = useRef<string>('tasks-pinned')
  const summaryMessageIdRef = useRef<string>('summary-pinned')
  
  
  const scrollAreaRef = useRef<HTMLDivElement>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const [isUserNearBottom, setIsUserNearBottom] = useState(true)
  const [showScrollToLatest, setShowScrollToLatest] = useState(false)
  const lastContentSignatureRef = useRef<string>('')

  const scrollToBottom = useCallback((behavior: ScrollBehavior = 'smooth') => {
    requestAnimationFrame(() => {
      messagesEndRef.current?.scrollIntoView({ behavior })
    })
  }, [])

  const handleScroll = useCallback(() => {
    const container = scrollAreaRef.current
    if (!container) return
    const threshold = 120
    const distanceToBottom = container.scrollHeight - container.scrollTop - container.clientHeight
    const nearBottom = distanceToBottom <= threshold
    setIsUserNearBottom(nearBottom)
  }, [])

  const handleScrollToLatest = useCallback(() => {
    scrollToBottom()
    setShowScrollToLatest(false)
  }, [scrollToBottom])

  useEffect(() => {
    if (isUserNearBottom) {
      setShowScrollToLatest(false)
    }
  }, [isUserNearBottom])

  useEffect(() => {
    const signatureParts: string[] = []
    const lastMessage = analysisState.messages[analysisState.messages.length - 1]
    const lastMessageSignature = lastMessage
      ? `${analysisState.messages.length}:${lastMessage.id}:${lastMessage.content?.length ?? 0}:${lastMessage.toolCalls?.length ?? 0}:${lastMessage.isStreaming ? 1 : 0}`
      : `${analysisState.messages.length}:none`
    signatureParts.push(lastMessageSignature)

    if (pinnedTasksData) {
      const sectionSignature = pinnedTasksData.taskSections
        ?.map(section => `${section.id}:${section.tasks?.length ?? 0}:${section.statusText ?? ''}`)
        .join('|') ?? ''
      const tasksSignature = `${pinnedTasksData.tasks?.length ?? 0}:${sectionSignature}:${(pinnedTasksData as any).progressText ?? ''}`
      signatureParts.push(tasksSignature)
    } else {
      signatureParts.push('no-tasks')
    }

    if (pinnedSummaryData?.summary) {
      signatureParts.push(`${pinnedSummaryData.summary.length}`)
    } else {
      signatureParts.push('no-summary')
    }

    const signature = signatureParts.join('||')
    if (lastContentSignatureRef.current === signature) {
      return
    }
    lastContentSignatureRef.current = signature

    if (settings.autoScroll && isUserNearBottom) {
      scrollToBottom()
      return
    }

    if (!isUserNearBottom) {
      setShowScrollToLatest(true)
    }
  }, [analysisState.messages, pinnedTasksData, pinnedSummaryData, settings.autoScroll, isUserNearBottom, scrollToBottom])

  useEffect(() => {
    const frame = requestAnimationFrame(() => {
      handleScroll()
    })
    return () => cancelAnimationFrame(frame)
  }, [analysisState.messages, pinnedTasksData, pinnedSummaryData, handleScroll])

  // 构建 HolmesGPT 调查请求
  const buildInvestigateRequest = () => {
    return {
      source: 'robusta',
      title: `${alert.title}`,
      description: `${alert.description || ''}

🚨 重要指令 - CRITICAL INSTRUCTION:
你必须使用中文回答所有问题！You MUST respond in Chinese for all questions!
请用中文提供详细的根因分析、解决方案和预防措施。
Please provide detailed root cause analysis, solutions, and preventive measures in Chinese.

分析要求：
1. 使用中文描述问题现象
2. 使用中文分析根本原因
3. 使用中文提供解决步骤
4. 使用中文给出预防建议
5. 所有技术术语请用中文解释

Analysis Requirements:
1. Describe the problem in Chinese
2. Analyze root causes in Chinese  
3. Provide solution steps in Chinese
4. Give prevention suggestions in Chinese
5. Explain all technical terms in Chinese`,
      subject: {
        alert_id: alert.id,
        fingerprint: alert.fingerprint,
        cluster_id: alert.cluster_id,
        severity: alert.severity,
        status: alert.status,
        labels: alert.labels || {},
        annotations: alert.annotations || {},
        created_at: alert.created_at,
        starts_at: alert.starts_at,
        ends_at: alert.ends_at
      },
      context: {
        cluster_name: alert.cluster_id,
        alert_fingerprint: alert.fingerprint,
        alert_severity: alert.severity,
        alert_status: alert.status,
        alert_labels: alert.labels || {},
        alert_annotations: alert.annotations || {},
        language_requirement: '必须使用中文 - MUST USE CHINESE',
        response_language: 'zh-CN',
        instruction: '请用中文回答所有问题，包括技术分析、解决方案和建议。Please respond in Chinese for all technical analysis, solutions and recommendations.'
      },
      source_instance_id: 'WebUI-Chinese-Enhanced',
      include_tool_calls: settings.showToolCalls,
      include_tool_call_results: settings.showToolCalls,
      prompt_template: 'builtin://generic_investigation.jinja2'
    }
  }

  // 解析任务进度
  const parseProgress = (structuredData?: HolmesStructuredData) => {
    if (!structuredData) return { total: 0, completed: 0 }
    
    const allTasks = [
      ...(structuredData.tasks || []),
      ...(structuredData.taskSections?.flatMap(section => section.tasks) || [])
    ]
    
    const completed = allTasks.filter(task => task.status === 'completed').length
    const total = allTasks.length
    
    return { total, completed }
  }

  // 音效播放
  const playNotificationSound = () => {
    if (settings.soundEnabled && typeof window !== 'undefined') {
      try {
        const audio = new Audio('/notification.mp3')
        audio.volume = 0.3
        audio.play().catch(() => {
          // 忽略播放失败
        })
      } catch (error) {
        // 忽略音频错误
      }
    }
  }

  // JSON解析逻辑（复用原有逻辑）
  const normalizeTaskStatus = (status: any): HolmesTaskItem['status'] => {
    const value = String(status ?? '').toLowerCase()
    if (value.includes('progress') || value.includes('running')) return 'in_progress'
    if (value.includes('complete') || value.includes('done') || value.includes('success')) return 'completed'
    return 'pending'
  }

  const parseJsonObjectSequence = (raw: string): any[] => {
    const trimmed = raw.trim()
    if (!trimmed) return []

    const result: any[] = []
    let buffer = ''
    let depth = 0
    let inString = false
    let escape = false

    const flushBuffer = () => {
      const candidate = buffer.trim()
      if (!candidate) {
        buffer = ''
        return
      }
      try {
        result.push(JSON.parse(candidate))
      } catch {
        // 忽略无法解析的片段
      }
      buffer = ''
    }

    for (let i = 0; i < trimmed.length; i += 1) {
      const char = trimmed[i]
      buffer += char

      if (escape) {
        escape = false
        continue
      }

      if (char === '\\') {
        escape = true
        continue
      }

      if (char === '"') {
        inString = !inString
        continue
      }

      if (!inString) {
        if (char === '{' || char === '[') depth += 1
        if (char === '}' || char === ']') depth -= 1
      }

      if (depth === 0 && !inString) {
        flushBuffer()
      }
    }

    flushBuffer()
    return result
  }

  const extractTodos = (payload: any): HolmesTaskItem[] | undefined => {
    console.log('Extracting todos from payload:', payload)
    
    // 支持多种数据结构，包括最新的JSON格式
    const candidates = [
      payload?.todos,
      payload?.params?.todos,
      payload?.data?.todos,
      payload?.result?.params?.todos,
      // 新增：直接从result.data解析任务状态文本
      payload?.result?.data ? parseTasksFromStatusText(payload.result.data) : null
    ].filter(item => item && (Array.isArray(item) || typeof item === 'string'))

    console.log('Todo candidates found:', candidates)

    if (!candidates.length) return undefined

    const seen = new Map<string, HolmesTaskItem>()

    candidates.forEach(candidate => {
      if (Array.isArray(candidate)) {
        candidate.forEach((item: any, index: number) => {
          if (!item) return
          const content = typeof item.content === 'string' ? item.content.trim() : undefined
          if (!content) return
          const id = String(item.id ?? index)
          const note = typeof item.note === 'string' ? item.note : undefined
          seen.set(id || content, {
            id,
            content,
            status: normalizeTaskStatus(item.status),
            note
          })
        })
      } else if (typeof candidate === 'string') {
        // 从状态文本中解析任务
        const tasks = parseTasksFromStatusText(candidate)
        if (tasks) {
          tasks.forEach(task => {
            seen.set(task.id, task)
          })
        }
      }
    })

    return Array.from(seen.values())
  }

  // 新增：从状态文本解析任务列表
  const parseTasksFromStatusText = (statusText: string): HolmesTaskItem[] | null => {
    if (typeof statusText !== 'string') return null
    
    const tasks: HolmesTaskItem[] = []
    const lines = statusText.split('\n')
    
    console.log('Parsing status text lines:', lines)
    
    for (const line of lines) {
      // 匹配格式：[✓] [1] 任务内容 或 [ ] [2] 任务内容 或 [~] [3] 任务内容
      const taskMatch = line.match(/\[(.*?)\]\s*\[(\d+)\]\s*(.*)/)
      if (taskMatch) {
        const [, statusSymbol, id, content] = taskMatch
        let status: HolmesTaskItem['status'] = 'pending'
        
        if (statusSymbol.includes('✓')) {
          status = 'completed'
        } else if (statusSymbol.includes('~') || statusSymbol.includes('▶')) {
          status = 'in_progress'
        } else {
          status = 'pending'
        }
        
        const task = {
          id,
          content: content.trim(),
          status
        }
        
        console.log('Parsed task:', task)
        tasks.push(task)
      } else {
        console.log('Line did not match task pattern:', line)
      }
    }
    
    console.log('Final parsed tasks:', tasks)
    return tasks.length > 0 ? tasks : null
  }

  const deriveStatusText = (payload: any): string | undefined => {
    const direct = [payload?.status_text, payload?.status, payload?.message]
      .find((value) => typeof value === 'string' && value.trim())
    if (direct) return (direct as string).trim()

    const resultText = typeof payload?.result?.data === 'string' ? payload.result.data : undefined
    if (!resultText) return undefined
    const taskLine = resultText.split('\n').find((line: string) => line.includes('Task Status'))
    return (taskLine || resultText.split('\n')[0]).trim()
  }

  const extractTaskStatus = (text?: string): string | undefined => {
    if (typeof text !== 'string') return undefined
    const match = text.match(/Task Status\*\*:?\s*(\d+)\s*completed,\s*(\d+)\s*in progress,\s*(\d+)\s*pending/i)
    if (!match) return undefined
    const [, completed, inProgress, pending] = match
    return `完成 ${completed} · 进行中 ${inProgress} · 待处理 ${pending}`
  }

  const extractHolmesStructuredDataFromObject = (payload: any): HolmesStructuredData | undefined => {
    if (!payload || typeof payload !== 'object') return undefined

    console.log('Extracting structured data from object:', payload)

    const planTextCandidates = [payload.content, payload.result?.message]
      .filter((value) => typeof value === 'string' && value.trim()) as string[]
    let planText: string | undefined = planTextCandidates[0]?.trim()

    const toolName = typeof payload.tool_name === 'string'
      ? payload.tool_name
      : typeof payload.name === 'string'
        ? payload.name
        : undefined

    console.log('Tool name:', toolName)

    const tasks = extractTodos(payload)
    console.log('Extracted tasks:', tasks)
    const statusText = deriveStatusText(payload)
    let summary: string | undefined
    let progressText: string | undefined

    // 检测是否为最终分析报告
    const isFinalReport = payload.content && typeof payload.content === 'string' && 
      (payload.content.includes('# 问题分析报告') || 
       payload.content.includes('## 问题描述') ||
       payload.content.includes('## 根本原因分析') ||
       payload.content.includes('## 解决方案'))

    if (isFinalReport) {
      summary = payload.content
      planText = undefined
    } else {
      if ((!planText || /^write\s*\[/i.test(planText)) && tasks && tasks.length) {
        planText = 'HolmesGPT 已生成调查任务清单，以下为建议的调查步骤。'
      }
      if (!planText && typeof payload.content === 'string') {
        planText = payload.content.trim()
      }

      if ((!tasks || tasks.length === 0) && planText && !isFinalReport) {
        // 非最终报告且无任务信息的纯文本，按“排查进度”展示，而不是“分析结论”
        progressText = planText
        planText = undefined
      }
    }

    const commands = collectCommands(payload)

    const hasInfo = Boolean(planText || progressText || (tasks && tasks.length) || commands.length || toolName || statusText || summary)

    if (!hasInfo) {
      return undefined
    }

    // 为同一个工具调用生成稳定的ID，避免重复创建卡片
    const canonicalToolNameRaw = typeof payload.tool_name === 'string'
      ? payload.tool_name
      : typeof payload.name === 'string'
        ? payload.name
        : undefined
    const canonicalToolName = canonicalToolNameRaw?.trim()

    const isTodoWrite = canonicalToolName ? canonicalToolName.toLowerCase().includes('todo') : false

    const sectionId = (() => {
      if (isTodoWrite) {
        return 'todo-write-main'
      }

      if (payload.tool_call_id) return String(payload.tool_call_id)
      if (payload.id) return String(payload.id)
      if (payload.call_id) return String(payload.call_id)

      if (canonicalToolName) {
        return `section-${canonicalToolName.toLowerCase().replace(/[^a-z0-9]/g, '-')}`
      }

      return 'section-default'
    })()

    const statusSummary = extractTaskStatus(payload.result?.data)
    const sectionTitle = (() => {
      if (isTodoWrite) {
        return statusSummary ? `任务更新 · ${statusSummary}` : '任务更新'
      }
      if (canonicalToolName) {
        return `${canonicalToolName} 调用`
      }
      return 'HolmesGPT 调用'
    })()

    const result: HolmesStructuredData = {
      planText,
      tasks,
      toolName,
      statusText: statusSummary || statusText || undefined,
      progressText,
      summary,
      taskSections: ((tasks && tasks.length) || commands.length) ? [{
        id: sectionId,
        title: sectionTitle,
        toolName: toolName || 'HolmesGPT',
        tasks: tasks || [],
        statusText: statusSummary || statusText || undefined,
        commands: commands.length ? commands : undefined,
      }] : undefined,
      raw: payload
    }

    console.log('Extracted structured data result:', result)
    return result
  }

  const extractHolmesStructuredData = (payload: any): HolmesStructuredData | undefined => {
    if (!payload) return undefined

    if (typeof payload === 'string') {
      const objects = parseJsonObjectSequence(payload)
      if (!objects.length) {
        return {
          summary: payload,
        }
      }
      return objects.reduce<HolmesStructuredData | undefined>((acc, item) => {
        const structured = extractHolmesStructuredData(item)
        return mergeHolmesStructuredData(acc, structured)
      }, undefined)
    }

    if (Array.isArray(payload)) {
      return payload.reduce<HolmesStructuredData | undefined>((acc, item) => {
        const structured = extractHolmesStructuredData(item)
        return mergeHolmesStructuredData(acc, structured)
      }, undefined)
    }

    return extractHolmesStructuredDataFromObject(payload)
  }

  const mergeHolmesStructuredData = (
    previous?: HolmesStructuredData,
    next?: HolmesStructuredData
  ): HolmesStructuredData | undefined => {
    if (!previous && !next) return undefined
    if (!previous) return next
    if (!next) return previous

    // If 'next' is just a summary, merge it simply to avoid complex logic overwriting tasks.
    if (next.summary && !next.planText && !next.tasks && !next.taskSections && !next.toolName) {
        const newSummary = appendTextChunk(previous.summary || '', next.summary);
        return {
            ...previous,
            summary: newSummary,
        };
    }
    // If 'next' is just a progressText, merge it similarly
    if (next.progressText && !next.planText && !next.tasks && !next.taskSections && !next.toolName && !next.summary) {
        const newProgress = appendTextChunk((previous as any).progressText || '', next.progressText);
        return {
            ...previous,
            progressText: newProgress,
        } as HolmesStructuredData;
    }

    const merged: HolmesStructuredData = {
      planText: next.planText || previous.planText,
      toolName: next.toolName || previous.toolName,
      statusText: next.statusText || previous.statusText,
      raw: undefined,
      tasks: []
    }

    const collectRaw = (value?: any) => {
      if (value === undefined) return []
      return Array.isArray(value) ? value : [value]
    }

    const rawCombined = [...collectRaw(previous.raw), ...collectRaw(next.raw)]
    if (rawCombined.length) {
      merged.raw = rawCombined
    }

    const order: string[] = []
    const taskMap = new Map<string, HolmesTaskItem>()

    const registerTask = (task?: HolmesTaskItem) => {
      if (!task) return
      const key = task.id || task.content
      if (!taskMap.has(key)) {
        order.push(key)
      }
      taskMap.set(key, task)
    }

    previous.tasks?.forEach(registerTask)
    next.tasks?.forEach(registerTask)

    merged.tasks = order.map(key => taskMap.get(key)!).filter(Boolean)

    let summaryText = previous.summary || ''
    if (next.summary) {
      summaryText = appendTextChunk(summaryText, next.summary)
    }
    if (summaryText.trim()) {
      merged.summary = summaryText
    }

    // merge progressText
    const prevProgress = (previous as any).progressText || ''
    const incomingProgress = next.progressText || ''
    const combinedProgress = incomingProgress ? appendTextChunk(prevProgress, incomingProgress) : prevProgress
    if (combinedProgress && combinedProgress.trim()) {
      (merged as any).progressText = combinedProgress
    }

    const sectionMap = new Map<string, HolmesTaskSection>()
    const sectionOrder: string[] = []

    const registerSection = (section?: HolmesTaskSection) => {
      if (!section) return
       
      if (!sectionMap.has(section.id)) {
        sectionOrder.push(section.id)
        sectionMap.set(section.id, section)
        } else {
          // 合并相同ID的section，主要是合并任务列表
          const existing = sectionMap.get(section.id)!
          const mergedSection: HolmesTaskSection = {
            id: section.id,
            // 保留现有的基本信息，只更新可变的状态信息
            title: section.title || existing.title, // 优先使用新标题
            statusText: section.statusText || existing.statusText, // 更新状态文本
            toolName: section.toolName || existing.toolName,
            tasks: [],
            commands: appendCommands(existing.commands, section.commands),
          }
        
        // 合并任务：以ID或content为键，新状态覆盖旧状态
        const taskMap = new Map()
        const taskOrder: string[] = []
        
        const addTask = (task: any) => {
          if (!task) return
          const key = task.id || task.content
          if (!taskMap.has(key)) {
            taskOrder.push(key)
          }
          // 新的任务状态覆盖旧的（例如：pending -> completed）
          taskMap.set(key, task)
        }
        
        // 先添加现有任务，再添加新任务（新任务会覆盖相同ID的旧任务）
        existing.tasks?.forEach(addTask)
        section.tasks?.forEach(addTask)
        
        mergedSection.tasks = taskOrder.map(key => taskMap.get(key)!).filter(Boolean)
        sectionMap.set(section.id, mergedSection)
      }
    }

    previous.taskSections?.forEach(registerSection)
    next.taskSections?.forEach(section => {
      if (!section) return
      if (!section.tasks?.length) return

      // 如果是 TodoWrite 且新增任务，与现有段 ID 合并
      if (section.id === 'todo-write-main' && sectionMap.has('todo-write-main')) {
        registerSection(section)
        return
      }

      registerSection(section)
    })

    const sections = sectionOrder.map(id => sectionMap.get(id)!).filter(Boolean)
    if (sections.length > 0) {
      merged.taskSections = sections
    }

    if (!merged.tasks?.length) {
      delete merged.tasks
    }

    return merged
  }

  function appendTextChunk(base: string, chunk: string): string {
    if (!chunk) return base
    const trimmedChunk = chunk.trim()
    if (!trimmedChunk) return base
    const normalizedChunk = trimmedChunk.replace(/\s+/g, ' ')
    const normalizedBase = base.replace(/\s+/g, ' ')
    if (normalizedBase.includes(normalizedChunk)) return base
    return base ? `${base}\n\n${trimmedChunk}` : trimmedChunk
  }

  function appendCommands(existing?: string[], incoming?: string[]): string[] | undefined {
    const merged = new Set<string>()
    existing?.forEach(cmd => {
      if (cmd && cmd.trim()) merged.add(cmd.trim())
    })
    incoming?.forEach(cmd => {
      if (cmd && cmd.trim()) merged.add(cmd.trim())
    })
    return merged.size ? Array.from(merged) : undefined
  }

  function collectCommands(payload: any): string[] {
    const commands: string[] = []

    const tryAdd = (value?: any) => {
      if (typeof value === 'string') {
        const trimmed = value.trim()
        if (trimmed && !commands.includes(trimmed)) {
          commands.push(trimmed)
        }
      }
    }

    tryAdd(payload?.invocation)
    tryAdd(payload?.description)
    tryAdd(payload?.command)
    tryAdd(payload?.result?.invocation)
    tryAdd(payload?.result?.description)

    return commands
  }

  const formatTaskListMarkdown = (tasks: HolmesTaskItem[], indent = ''): string => {
    if (!tasks?.length) return ''
    return tasks
      .map(task => {
        const checkbox = task.status === 'completed' ? '[x]' : task.status === 'in_progress' ? '[-]' : '[ ]'
        const lines = [`${indent}- ${checkbox} ${task.content}`]
        if (task.note) {
          lines.push(`${indent}  > 备注：${task.note}`)
        }
        return lines.join('\n')
      })
      .join('\n')
  }

  const structuredDataToMarkdown = (
    data?: HolmesStructuredData,
    options?: { headingLevel?: number; summaryTitle?: string }
  ): string | undefined => {
    if (!data) return undefined
    const { headingLevel = 3, summaryTitle = '分析结论' } = options || {}
    const sections: string[] = []
    const heading = (title: string, levelOffset = 0) => {
      const level = Math.min(6, headingLevel + levelOffset)
      return `${'#'.repeat(level)} ${title}`
    }

    if (data.planText) {
      sections.push(`${heading('分析计划')}\n${data.planText}`)
    }

    if (data.progressText) {
      sections.push(`${heading('排查进度')}\n${data.progressText}`)
    }

    if (data.tasks?.length) {
      const tasksMarkdown = formatTaskListMarkdown(data.tasks)
      if (tasksMarkdown) {
        sections.push(`${heading('任务列表')}\n${tasksMarkdown}`)
      }
    }

    if (data.taskSections?.length) {
      const sectionLines: string[] = []
      data.taskSections.forEach(section => {
        if (!section || !section.tasks?.length) return
        sectionLines.push(`${heading(section.title || section.toolName || '任务', 1)}`)
        if (section.statusText) {
          sectionLines.push(`> 状态：${section.statusText}`)
        }
        const tasksMarkdown = formatTaskListMarkdown(section.tasks, '  ')
        if (tasksMarkdown) {
          sectionLines.push(tasksMarkdown)
        }
        if (section.commands?.length) {
          sectionLines.push('  - 执行命令：')
          section.commands.forEach(cmd => {
            sectionLines.push(`    - \`${cmd}\``)
          })
        }
      })
      if (sectionLines.length) {
        sections.push(sectionLines.join('\n'))
      }
    }

    if (data.summary) {
      sections.push(`${heading(summaryTitle)}\n${data.summary}`)
    }

    return sections.length ? sections.join('\n\n') : undefined
  }

  // 开始分析
  const startAnalysis = async () => {
    const controller = new AbortController()
    setAbortController(controller)

    // 添加用户消息
    const userMessage: AnalysisMessage = {
      id: `user-${Date.now()}`,
      role: 'user',
      content: `请分析告警：${alert.title}\n\n描述：${alert.description || '无描述'}`,
      timestamp: format(new Date(), 'HH:mm:ss', { locale: zhCN })
    }

    setAnalysisState({
      status: 'analyzing',
      messages: [userMessage],
      totalSteps: 0,
      completedSteps: 0
    })
    allTasksCompletedRef.current = false;
    setPinnedTasksData(undefined)
    setPinnedSummaryData(undefined)

    try {
      const response = await fetch('/api/holmesgpt/stream/investigate', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(buildInvestigateRequest()),
        signal: controller.signal
      })

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      const reader = response.body?.getReader()
      if (!reader) {
        throw new Error('无法读取响应流')
      }

      let currentAssistantMessage: AnalysisMessage = {
        id: `assistant-${Date.now()}`,
        role: 'assistant',
        content: '',
        timestamp: format(new Date(), 'HH:mm:ss', { locale: zhCN }),
        isStreaming: true,
        toolCalls: []
      }

      setAnalysisState(prev => ({
        ...prev,
        messages: [...prev.messages, currentAssistantMessage]
      }))
      

      const decoder = new TextDecoder()
      let buffer = ''

      // 处理单个数据项的函数
      const processDataItem = (item: any) => {
        console.log('Processing data item:', item)
        
        // 0. 优先检查是否为工具调用结果 - 创建独立的工具调用消息
        if (item.role === 'tool' && item.tool_call_id) {
          // 对于 TodoWrite，不创建独立消息，直接更新任务卡片
          if (item.name === 'TodoWrite') {
            console.log('TodoWrite tool result received (early branch) - update structured data only')
            const structured = extractHolmesStructuredData(item)
            const mergedStructured = mergeHolmesStructuredData(pinnedTasksData, structured)

            // 当所有任务完成后，标记进入结论阶段
            if (mergedStructured) {
              const allTasks = mergedStructured.tasks || mergedStructured.taskSections?.flatMap(s => s.tasks) || []
              if (allTasks.length > 0 && allTasks.every(t => t.status === 'completed')) {
                console.log('All tasks completed (early branch), expecting final summary.')
                allTasksCompletedRef.current = true
              }
              const progress = parseProgress(mergedStructured)
              setAnalysisState(prev => ({
                ...prev,
                totalSteps: progress.total,
                completedSteps: progress.completed
              }))
            }
            setPinnedTasksData(mergedStructured)
            return
          }
          const toolCall = {
            name: item.name,
            input: item.description,
            output: item.result,
            status: (item.result?.status === 'success' ? 'success' : item.result?.status === 'error' ? 'error' : 'pending') as 'pending' | 'success' | 'error',
            command: item.result?.invocation || item.description || ''
          }
          
          console.log('=== CREATING TOOL CALL MESSAGE ===')
          console.log('Tool call data:', toolCall)
          
          // 创建独立的工具调用消息，插入时间线
          const toolCallMessage: AnalysisMessage = {
            id: `tool-${item.tool_call_id || Date.now()}`,
            role: 'assistant',
            content: '', // 工具调用消息没有文本内容
            timestamp: format(new Date(), 'HH:mm:ss', { locale: zhCN }),
            isStreaming: false,
            toolCalls: [toolCall]
          }
          
          // 先完成当前的助手消息（如果有内容的话）
          if (currentAssistantMessage.content.trim()) {
            currentAssistantMessage = {
              ...currentAssistantMessage,
              isStreaming: false
            }
            
            setAnalysisState(prev => ({
              ...prev,
              messages: [
                ...prev.messages.map(msg => 
                  msg.id === currentAssistantMessage.id ? currentAssistantMessage : msg
                ),
                toolCallMessage // 添加工具调用消息
              ]
            }))
            
            // 创建新的助手消息用于后续内容
            currentAssistantMessage = {
              id: `assistant-${Date.now()}-${Math.random().toString(36).slice(2)}`,
              role: 'assistant',
              content: '',
              timestamp: format(new Date(), 'HH:mm:ss', { locale: zhCN }),
              isStreaming: true,
              toolCalls: []
            }
            // 追加新的助手消息，确保随后的增量更新可定位
            setAnalysisState(prev => ({
              ...prev,
              messages: [...prev.messages, currentAssistantMessage]
            }))
          } else {
            // 如果当前助手消息没有内容，直接添加工具调用消息
            setAnalysisState(prev => ({
              ...prev,
              messages: [...prev.messages, toolCallMessage]
            }))
          }
          
          console.log('Tool call message created:', toolCallMessage)
          console.log('=== END TOOL CALL MESSAGE CREATION ===')
        }
        
        // 1. 处理文本内容
        else if (item.content && typeof item.content === 'string') {
          // 若任务全部完成，将文本归入固定“分析结论”卡片，避免插入中部
          if (allTasksCompletedRef.current) {
            setPinnedSummaryData(prev => mergeHolmesStructuredData(prev, { summary: item.content }))
            // 不再把结论文本追加到中间消息，保证结论出现在底部
            return
          }
          // 否则按过程文本/计划解析
          const structured = extractHolmesStructuredData(item)
          // 若解析出了summary（例如isFinalReport），也固定到底部卡片，并从结构中移除
          if (structured?.summary) {
            setPinnedSummaryData(prev => mergeHolmesStructuredData(prev, { summary: structured.summary }))
            structured.summary = undefined
          }
          const mergedStructured = mergeHolmesStructuredData(currentAssistantMessage.structuredData, structured)
          currentAssistantMessage = {
            ...currentAssistantMessage,
            structuredData: mergedStructured,
            content: appendTextChunk(currentAssistantMessage.content, item.content),
            isStreaming: true
          }
          updateMessageState()
        }
        
        // 2. 处理工具调用开始（tool_name + id）
        else if (item.tool_name && item.id && !item.role) {
          console.log('Tool call started:', item.tool_name, item.id)
          // 可以在这里显示工具调用开始的状态
        }
        
        // 3. 处理结构化数据或任务更新
        else if (item.type === 'analysis' || item.type === 'analysis_chunk') {
          const payload = item.data ?? item.content

          if (payload && typeof payload === 'object') {
            const structured = extractHolmesStructuredData(payload)
            const hasTodo = structured?.taskSections?.some(s => s.id === 'todo-write-main')
            if (hasTodo) {
              const mergedPinned = mergeHolmesStructuredData(pinnedTasksData, structured)
              setPinnedTasksData(mergedPinned)
              const progress = parseProgress(mergedPinned)
              setAnalysisState(prev => ({
                ...prev,
                totalSteps: progress.total,
                completedSteps: progress.completed
              }))
              return
            }
            const mergedStructured = mergeHolmesStructuredData(currentAssistantMessage.structuredData, structured)
            currentAssistantMessage = {
              ...currentAssistantMessage,
              structuredData: mergedStructured,
              isStreaming: true
            }
            updateMessageState()
          }
        }
      }
      
      // 更新消息状态的辅助函数
      const updateMessageState = () => {
        const progress = parseProgress(currentAssistantMessage.structuredData)
        
        setAnalysisState(prev => {
          const exists = prev.messages.some(msg => msg.id === currentAssistantMessage.id)
          return {
            ...prev,
            messages: exists
              ? prev.messages.map(msg =>
                  msg.id === currentAssistantMessage.id ? currentAssistantMessage : msg
                )
              : [...prev.messages, currentAssistantMessage],
            totalSteps: progress.total,
            completedSteps: progress.completed
          }
        })
      }

      // 处理SSE格式的分析数据
      const processAnalysisData = (analysisData: any) => {
        console.log('Processing analysis data:', analysisData)
        
        // 1. 处理AI消息内容
        if (analysisData.content && typeof analysisData.content === 'string') {
          console.log('Processing AI message content:', analysisData.content);
          if (allTasksCompletedRef.current) {
            // 结论阶段：固定在底部
            setPinnedSummaryData(prev => mergeHolmesStructuredData(prev, { summary: analysisData.content }))
            return
          }
          // 过程阶段：解析为进度/计划
          const structured = extractHolmesStructuredData(analysisData)
          if (structured?.summary) {
            // 若模型直接给了带标题的总结，仍固定到底部
            setPinnedSummaryData(prev => mergeHolmesStructuredData(prev, { summary: structured.summary }))
            structured.summary = undefined
          }

          const mergedStructured = mergeHolmesStructuredData(currentAssistantMessage.structuredData, structured)
          currentAssistantMessage = {
            ...currentAssistantMessage,
            structuredData: mergedStructured,
            content: appendTextChunk(currentAssistantMessage.content, analysisData.content),
            isStreaming: true
          }
          updateMessageState()
        }
        
        // 2. 处理工具调用开始
        else if (analysisData.tool_name && analysisData.id) {
          console.log('Tool call started:', analysisData.tool_name, analysisData.id)
          // 可以在这里显示工具调用开始的状态
        }
        
        // 3. 处理工具调用结果
        else if (analysisData.role === 'tool' && analysisData.tool_call_id) {
          const toolCall = {
            name: analysisData.name,
            input: analysisData.description,
            output: analysisData.result,
            status: (analysisData.result?.status === 'success' ? 'success' : analysisData.result?.status === 'error' ? 'error' : 'pending') as 'pending' | 'success' | 'error',
            command: analysisData.result?.invocation || analysisData.description || ''
          }
          
          console.log('=== CREATING TOOL CALL MESSAGE ===')
          console.log('Tool call data:', toolCall)
          console.log('Tool name:', analysisData.name)
          
          // 对于TodoWrite工具，只更新固定任务卡片，不创建独立的工具调用消息
          if (analysisData.name === 'TodoWrite') {
            console.log('Processing TodoWrite result - updating structured data only')
            const structured = extractHolmesStructuredData(analysisData)
            const mergedStructured = mergeHolmesStructuredData(pinnedTasksData, structured)

            if (mergedStructured) {
              const allTasks = mergedStructured.tasks || mergedStructured.taskSections?.flatMap(s => s.tasks) || [];
              if (allTasks.length > 0 && allTasks.every(t => t.status === 'completed')) {
                  console.log('All tasks completed, expecting final summary.');
                  allTasksCompletedRef.current = true;
              }
              const progress = parseProgress(mergedStructured)
              setAnalysisState(prev => ({
                ...prev,
                totalSteps: progress.total,
                completedSteps: progress.completed
              }))
            }
            setPinnedTasksData(mergedStructured)
            return // 直接返回，不创建工具调用消息
          }
          
          // 对于其他工具，创建独立的工具调用消息，插入时间线
          const toolCallMessage: AnalysisMessage = {
            id: `tool-${analysisData.tool_call_id || Date.now()}`,
            role: 'assistant',
            content: '', // 工具调用消息没有文本内容
            timestamp: format(new Date(), 'HH:mm:ss', { locale: zhCN }),
            isStreaming: false,
            toolCalls: [toolCall]
          }
          
          // 先完成当前的助手消息（如果有内容的话）
          if (currentAssistantMessage.content.trim()) {
            currentAssistantMessage = {
              ...currentAssistantMessage,
              isStreaming: false
            }
            
            setAnalysisState(prev => ({
              ...prev,
              messages: [
                ...prev.messages.map(msg => 
                  msg.id === currentAssistantMessage.id ? currentAssistantMessage : msg
                ),
                toolCallMessage // 添加工具调用消息
              ]
            }))
            
            // 创建新的助手消息用于后续内容
            currentAssistantMessage = {
              id: `assistant-${Date.now()}-${Math.random().toString(36).slice(2)}`,
              role: 'assistant',
              content: '',
              timestamp: format(new Date(), 'HH:mm:ss', { locale: zhCN }),
              isStreaming: true,
              toolCalls: []
            }
            // 追加新的助手消息，确保随后的增量更新可定位
            setAnalysisState(prev => ({
              ...prev,
              messages: [...prev.messages, currentAssistantMessage]
            }))
          } else {
            // 如果当前助手消息没有内容，直接添加工具调用消息
            setAnalysisState(prev => ({
              ...prev,
              messages: [...prev.messages, toolCallMessage]
            }))
          }
          
          console.log('Tool call message created:', toolCallMessage)
          console.log('=== END TOOL CALL MESSAGE CREATION ===')
        }
        
        // 4. 处理最终分析结论（ai_answer_end的data是字符串）
        else if (typeof analysisData === 'string') {
          console.log('Final analysis conclusion received');
          if (allTasksCompletedRef.current) {
            // 最终文本固定到底部结论卡片
            setPinnedSummaryData(prev => mergeHolmesStructuredData(prev, { summary: analysisData }))
          } else {
            const mergedStructured = mergeHolmesStructuredData(currentAssistantMessage.structuredData, { progressText: analysisData })
            currentAssistantMessage = {
              ...currentAssistantMessage,
              structuredData: mergedStructured,
              content: appendTextChunk(currentAssistantMessage.content, analysisData),
              isStreaming: true,
            }
            updateMessageState()
          }
        }
      }

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          if (line.startsWith('event: ')) {
            // 处理 SSE 事件类型（忽略，下一行是数据）
            continue
          } else if (line.startsWith('data: ')) {
            try {
              const rawData = line.slice(6)
              
              // 处理特殊的结束标记
              if (rawData === '{"type":"complete","data":{}}') {
                console.log('Analysis complete signal received')
                // 完成分析
                currentAssistantMessage = {
                  ...currentAssistantMessage,
                  isStreaming: false
                }
                setAnalysisState(prev => {
                  const exists = prev.messages.some(msg => msg.id === currentAssistantMessage.id)
                  const updatedMessages = exists
                    ? prev.messages.map(msg => msg.id === currentAssistantMessage.id ? currentAssistantMessage : msg)
                    : [...prev.messages, currentAssistantMessage]
                  return {
                    status: 'completed',
                    messages: updatedMessages,
                    totalSteps: prev.totalSteps,
                    completedSteps: prev.completedSteps
                  }
                })
                break
              }
              
              const data = JSON.parse(rawData)
              
              // 处理SSE格式的数据 {"type":"analysis","data":{...}}
              if (data.type === 'analysis' && data.data) {
                processAnalysisData(data.data)
              }
              // 处理数组格式的数据流（向后兼容）
              else if (Array.isArray(data)) {
                data.forEach(item => processDataItem(item))
              } 
              // 处理直接的数据项
              else {
                processDataItem(data)
              }
              
            } catch (error) {
              console.error('JSON parse error:', error, 'Line:', line)
            }
          }
        }
      }
      
      // 正常结束时，确保最后的消息被标记为完成
      if (currentAssistantMessage.content.trim()) {
        currentAssistantMessage = {
          ...currentAssistantMessage,
          isStreaming: false
        }
        
        setAnalysisState(prev => ({
          status: 'completed',
          messages: prev.messages.map(msg => 
            msg.id === currentAssistantMessage.id ? currentAssistantMessage : msg
          ),
          totalSteps: prev.totalSteps,
          completedSteps: prev.completedSteps
        }))
      }
      
      playNotificationSound()
      
    } catch (error: any) {
      if (error.name === 'AbortError') {
        setAnalysisState(prev => ({
          status: 'idle',
          messages: prev.messages.slice(0, -1) // 移除未完成的助手消息
        }))
      } else {
        console.error('分析失败:', error)
        setAnalysisState(prev => ({
          status: 'error',
          messages: prev.messages,
          error: error.message || '分析过程中发生错误'
        }))
      }
    } finally {
      setAbortController(null)
    }
  }

  // 停止分析
  const stopAnalysis = () => {
    if (abortController) {
      abortController.abort()
    }
  }

  // 重新分析
  const restartAnalysis = () => {
    setAnalysisState({
      status: 'idle',
      messages: [],
      totalSteps: 0,
      completedSteps: 0
    })
    allTasksCompletedRef.current = false;
    setPinnedTasksData(undefined)
    setPinnedSummaryData(undefined)
  }

  // 下载分析结果
  const downloadAnalysis = () => {
    const markdown = buildAnalysisMarkdown()
    const blob = new Blob([markdown], { type: 'text/markdown;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `holmesgpt-analysis-${alert.id}-${Date.now()}.md`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
  }

  const getStatusIcon = () => {
    switch (analysisState.status) {
      case 'analyzing':
        return <Loader2 className="h-5 w-5 animate-spin text-blue-600" />
      case 'completed':
        return <CheckCircle2 className="h-5 w-5 text-green-600" />
      case 'error':
        return <AlertCircle className="h-5 w-5 text-red-600" />
      default:
        return <Brain className="h-5 w-5 text-gray-600" />
    }
  }

  const getStatusText = () => {
    switch (analysisState.status) {
      case 'analyzing':
        return '正在分析中...'
      case 'completed':
        return '分析完成'
      case 'error':
        return '分析失败'
      default:
        return 'HolmesGPT 智能分析'
    }
  }

  const getProgressText = () => {
    const prog = parseProgress(pinnedTasksData)
    if (prog.total > 0) {
      return `${prog.completed}/${prog.total} 步骤`
    }
    if (analysisState.totalSteps && analysisState.completedSteps !== undefined) {
      return `${analysisState.completedSteps}/${analysisState.totalSteps} 步骤`
    }
    return null
  }

  const getRoleLabel = (role: AnalysisMessage['role']) => {
    switch (role) {
      case 'assistant':
        return 'HolmesGPT'
      case 'user':
        return '用户'
      case 'system':
        return '系统'
      default:
        return role
    }
  }

  const buildAnalysisMarkdown = () => {
    const lines: string[] = []
    lines.push('# HolmesGPT 分析报告')
    lines.push(`- 告警标题：${alert.title}`)
    lines.push(`- 告警指纹：${alert.fingerprint || '未知'}`)
    lines.push(`- 集群：${alert.cluster_id || '未知'}`)
    lines.push(`- 严重程度：${alert.severity || '未知'}`)
    lines.push(`- 导出时间：${format(new Date(), 'yyyy-MM-dd HH:mm:ss', { locale: zhCN })}`)

    analysisState.messages.forEach((message, index) => {
      const roleLabel = getRoleLabel(message.role)
      const timestamp = message.timestamp ? `（${message.timestamp}）` : ''
      lines.push(`\n## ${index + 1}. ${roleLabel}${timestamp}`)
      if (message.content?.trim()) {
        lines.push(message.content.trim())
      }
      const structuredMarkdown = structuredDataToMarkdown(message.structuredData)
      if (structuredMarkdown) {
        lines.push(structuredMarkdown)
      }
      if (message.toolCalls?.length) {
        lines.push('### 工具调用记录')
        message.toolCalls.forEach(call => {
          const statusLabel = call.status === 'success' ? '✅ 成功' : call.status === 'error' ? '⚠️ 失败' : '⏳ 进行中'
          lines.push(`- ${call.name}：${statusLabel}`)
          if (call.command) {
            lines.push(`  - 命令：\`${call.command}\``)
          }
          if (typeof call.output === 'string' && call.output.trim()) {
            lines.push(`  - 输出：${call.output.trim()}`)
          } else if (call.output) {
            lines.push('  - 输出：')
            lines.push('    ```json')
            lines.push(JSON.stringify(call.output, null, 2))
            lines.push('    ```')
          }
        })
      }
    })

    if (pinnedTasksData) {
      const taskMarkdown = structuredDataToMarkdown(pinnedTasksData, { headingLevel: 2, summaryTitle: '任务总结' })
      if (taskMarkdown) {
        lines.push('\n## 调查任务进度')
        lines.push(taskMarkdown)
      }
    }

    if (pinnedSummaryData) {
      const summaryMarkdown = structuredDataToMarkdown(pinnedSummaryData, { headingLevel: 2, summaryTitle: '最终结论' })
      if (summaryMarkdown) {
        lines.push(summaryMarkdown)
      }
    }

    return lines.filter(Boolean).join('\n\n')
  }

  // 内容区域组件
  const ContentArea = () => (
    <div className="flex-1 flex flex-col min-h-0">
      <div
        ref={scrollAreaRef}
        onScroll={handleScroll}
        className="flex-1 overflow-y-auto px-6 py-4 flex flex-col"
      >
        {analysisState.messages.length === 0 ? (
          <div className="flex flex-col items-center justify-center flex-1 text-center p-8">
            <div className="relative mb-6">
              <div className="w-16 h-16 bg-blue-100 rounded-full flex items-center justify-center">
                <Brain className="h-8 w-8 text-blue-600" />
              </div>
              <div className="absolute -top-1 -right-1 w-6 h-6 bg-green-500 rounded-full flex items-center justify-center">
                <MessageSquare className="h-3 w-3 text-white" />
              </div>
            </div>
            <h3 className="text-lg font-medium text-gray-900 mb-2">准备开始智能分析</h3>
            <p className="text-sm text-gray-500 max-w-md leading-relaxed">
              点击触发RCA分析按钮，HolmesGPT 将为您分析告警的根本原因，并提供详细的解决方案和预防措施。
              分析过程包括任务规划、并行调查和结论总结。
            </p>
            <div className="mt-6 grid grid-cols-3 gap-4 text-center">
              <div className="p-3 bg-blue-50 rounded-lg">
                <div className="text-blue-600 font-semibold text-lg">📋</div>
                <div className="text-xs text-blue-700 mt-1">任务规划</div>
              </div>
              <div className="p-3 bg-green-50 rounded-lg">
                <div className="text-green-600 font-semibold text-lg">🔍</div>
                <div className="text-xs text-green-700 mt-1">并行调查</div>
              </div>
              <div className="p-3 bg-purple-50 rounded-lg">
                <div className="text-purple-600 font-semibold text-lg">📝</div>
                <div className="text-xs text-purple-700 mt-1">结论总结</div>
              </div>
            </div>
          </div>
        ) : (
          <div className="space-y-0">
            {analysisState.messages.map((message) => {
              const sanitized = sanitizeStructuredData(message.structuredData)
              return (
                <ChatMessage
                  key={message.id}
                  role={message.role}
                  content={message.content}
                  timestamp={message.timestamp}
                  isStreaming={message.isStreaming}
                  toolCalls={message.toolCalls || []}
                  structuredData={sanitized}
                />
              )
            })}
            {pinnedTasksData && (() => {
              const prog = parseProgress(pinnedTasksData)
              const pinnedIsStreaming = analysisState.status === 'analyzing' && (!pinnedSummaryData) && (prog.total === 0 || prog.completed < prog.total)
              return (
                <ChatMessage
                  key={tasksMessageIdRef.current}
                  role="assistant"
                  content={''}
                  timestamp={format(new Date(), 'HH:mm:ss', { locale: zhCN })}
                  isStreaming={pinnedIsStreaming}
                  toolCalls={[]}
                  structuredData={pinnedTasksData}
                />
              )
            })()}
            {pinnedSummaryData && (
              <ChatMessage
                key={summaryMessageIdRef.current}
                role="assistant"
                content={''}
                timestamp={format(new Date(), 'HH:mm:ss', { locale: zhCN })}
                isStreaming={false}
                toolCalls={[]}
                structuredData={pinnedSummaryData}
              />
            )}
            {showScrollToLatest && (
              <div className="sticky bottom-4 flex justify-end">
                <Button
                  size="sm"
                  variant="secondary"
                  className="shadow"
                  onClick={handleScrollToLatest}
                >
                  回到最新
                </Button>
              </div>
            )}
            <div ref={messagesEndRef} />
          </div>
        )}
      </div>
      
      {analysisState.status === 'error' && analysisState.error && (
        <div className="p-4 border-t bg-red-50">
          <div className="flex items-center space-x-2 text-red-600">
            <AlertCircle className="h-4 w-4" />
            <span className="text-sm font-medium">分析失败</span>
          </div>
          <p className="text-sm text-red-600 mt-1">{analysisState.error}</p>
          <Button 
            onClick={restartAnalysis} 
            variant="outline" 
            size="sm" 
            className="mt-2 border-red-200 text-red-600 hover:bg-red-50"
          >
            重试分析
          </Button>
        </div>
      )}
    </div>
  )

  // 过滤掉 TodoWrite 任务段，避免在普通消息中重复展示
  function sanitizeStructuredData(data?: HolmesStructuredData): HolmesStructuredData | undefined {
    if (!data) return data
    let changed = false
    const next: HolmesStructuredData = { ...data }
    const hadSections = Array.isArray(next.taskSections) && next.taskSections.length > 0
    const filtered = hadSections ? next.taskSections!.filter(s => s.id !== 'todo-write-main') : []
    const removedTodo = hadSections && filtered.length !== next.taskSections!.length
    if (removedTodo) {
      next.taskSections = filtered
      changed = true
    }
    const isTodoLike = removedTodo || (typeof next.toolName === 'string' && next.toolName.toLowerCase().includes('todo'))
    if (isTodoLike) {
      if (next.planText) { next.planText = undefined as any; changed = true }
      if (next.statusText) { next.statusText = undefined as any; changed = true }
      if (next.toolName) { next.toolName = undefined as any; changed = true }
      // 避免顶层 tasks（若存在）残留导致重复
      if (next.tasks && next.tasks.length) { delete (next as any).tasks; changed = true }
    }
    // 不在普通消息中展示结论，结论固定在底部
    if ((next as any).summary) { delete (next as any).summary; changed = true }
    // 如果去除后不再包含任何可渲染的结构信息，则返回 undefined 以隐藏该卡片
    const hasRenderable = Boolean(
      next.planText ||
      (next as any).progressText ||
      (next.taskSections && next.taskSections.length) ||
      next.summary
    )
    if (!hasRenderable) return undefined
    return changed ? next : data
  }

  // 控制栏组件
  const ControlBar = () => (
    <div className="flex-shrink-0 border-b bg-white px-6 py-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          {getStatusIcon()}
          <div>
            <div className="text-lg font-semibold flex items-center space-x-2">
              <span>{getStatusText()}</span>
              {getProgressText() && (
                <Badge variant="outline" className="text-xs">
                  {getProgressText()}
                </Badge>
              )}
            </div>
            <p className="text-sm text-muted-foreground mt-1">
              基于 Kubernetes 专业知识的智能故障分析
            </p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setShowSettings(!showSettings)}
            className="h-8 w-8 p-0"
          >
            <Settings className="h-4 w-4" />
          </Button>
          
          {analysisState.status === 'idle' && (
            <Button onClick={startAnalysis} size="sm">
              <Play className="h-4 w-4 mr-2" />
              触发RCA分析
            </Button>
          )}
          {analysisState.status === 'analyzing' && (
            <Button onClick={stopAnalysis} variant="outline" size="sm">
              <Square className="h-4 w-4 mr-2" />
              停止分析
            </Button>
          )}
          {(analysisState.status === 'completed' || analysisState.status === 'error') && (
            <>
              <Button onClick={restartAnalysis} variant="outline" size="sm">
                <RotateCcw className="h-4 w-4 mr-2" />
                重新分析
              </Button>
              {analysisState.messages.length > 0 && (
                <Button onClick={downloadAnalysis} variant="outline" size="sm">
                  <Download className="h-4 w-4 mr-2" />
                  下载结果
                </Button>
              )}
            </>
          )}
        </div>
      </div>

      {/* 设置面板 */}
      {showSettings && (
        <div className="mt-4 p-4 bg-gray-50 rounded-lg border">
          <h4 className="text-sm font-medium mb-3">聊天设置</h4>
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-sm">自动滚动</span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setSettings(prev => ({ ...prev, autoScroll: !prev.autoScroll }))}
                className={settings.autoScroll ? 'bg-blue-50 border-blue-200' : ''}
              >
                {settings.autoScroll ? '已启用' : '已禁用'}
              </Button>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm">完成提示音</span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setSettings(prev => ({ ...prev, soundEnabled: !prev.soundEnabled }))}
                className={settings.soundEnabled ? 'bg-blue-50 border-blue-200' : ''}
              >
                {settings.soundEnabled ? <Volume2 className="h-4 w-4" /> : <VolumeX className="h-4 w-4" />}
              </Button>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm">显示工具调用</span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setSettings(prev => ({ ...prev, showToolCalls: !prev.showToolCalls }))}
                className={settings.showToolCalls ? 'bg-blue-50 border-blue-200' : ''}
              >
                {settings.showToolCalls ? '已启用' : '已禁用'}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )

  // 根据showCard决定返回结构
  if (!showCard) {
    return (
      <div className="w-full h-full flex flex-col bg-white border rounded-lg overflow-hidden">
        <ControlBar />
        <ContentArea />
      </div>
    )
  }

  return (
    <Card className="w-full h-full flex flex-col">
      <CardHeader className="pb-3 flex-shrink-0">
        <ControlBar />
      </CardHeader>

      <Separator className="flex-shrink-0" />

      <CardContent className="p-0 flex-1 flex flex-col min-h-0">
        <ContentArea />
      </CardContent>
    </Card>
  )
}
