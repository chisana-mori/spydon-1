'use client'

import React, { useState, useRef, useEffect, useCallback } from 'react'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { Brain, Loader2, AlertCircle, CheckCircle2 } from 'lucide-react'
import { Alert } from '@/types/api'
import { HolmesStructuredData, HolmesTaskItem, HolmesTaskSection, formatSummaryText } from './ChatMessage'
import {
  parseProgress,
  normalizePlainText,
  appendTextChunk,
  deduplicateSummaryBlocks,
  buildSummarySignature,
  extractHolmesStructuredData,
  mergeHolmesStructuredData,
  looksLikeStructuredSummary,
  structuredDataToMarkdown,
  sanitizeStructuredData
} from './holmesUtils'
import { buildCachedSSEChunks, createCachedChunkIterable, createResponseChunkIterable, consumeSSEChunks } from './sseHelpers'
import { ControlBar } from './ControlBar'
import { ContentArea } from './ContentArea'
import { EnhancedHolmesGPTChatProps, AnalysisState, ChatSettings, StreamProcessor } from './holmesTypes'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { appConfig } from '@/config'

// 生成唯一ID的计数器
let messageIdCounter = 0
const generateUniqueId = (prefix: string = 'msg') => {
  messageIdCounter++
  return `${prefix}-${Date.now()}-${messageIdCounter}-${Math.random().toString(36).slice(2)}`
}



export const EnhancedHolmesGPTChat: React.FC<EnhancedHolmesGPTChatProps> = ({
  alert,
  showCard = true,
  cachedResult,
  loadingCache = false
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
  const [isReplayingCache, setIsReplayingCache] = useState(false)
  const lastSummarySignatureRef = useRef<string | null>(null)

  const restartAnalysis = useCallback(() => {
    setAnalysisState({
      status: 'idle',
      messages: [],
      totalSteps: 0,
      completedSteps: 0
    })
    allTasksCompletedRef.current = false
    setPinnedTasksData(undefined)
    setPinnedSummaryData(undefined)
    lastSummarySignatureRef.current = null
  }, [])

  const scrollAreaRef = useRef<HTMLDivElement>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const [isUserNearBottom, setIsUserNearBottom] = useState(true)
  const [showScrollToLatest, setShowScrollToLatest] = useState(false)
  const lastContentSignatureRef = useRef<string>('')
  const lastReplayedCacheKeyRef = useRef<string | null>(null)

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

  // 加载缓存结果并回放
  useEffect(() => {
    if (!cachedResult) {
      lastReplayedCacheKeyRef.current = null
    }
  }, [cachedResult])

  useEffect(() => {
    if (!cachedResult || loadingCache || isReplayingCache) {
      return
    }

    const runId = cachedResult.run_id || cachedResult.runId || cachedResult.RunID
    const cacheKey = runId || (cachedResult.cached_at ? `${cachedResult.cached_at}` : JSON.stringify(cachedResult))

    if (!cacheKey) {
      console.warn('缓存结果缺少可识别的运行ID，将跳过自动回放')
      return
    }

    if (lastReplayedCacheKeyRef.current === cacheKey) {
      return
    }

    if (analysisState.status !== 'idle' || analysisState.messages.length > 0) {
      restartAnalysis()
    }

    lastReplayedCacheKeyRef.current = cacheKey
    console.log('✅ 找到RCA缓存，将自动回放:', cachedResult)
    replayCachedResult(cachedResult)
  }, [cachedResult, loadingCache, isReplayingCache, analysisState.status, analysisState.messages.length, restartAnalysis])









  // 回放缓存的RCA结果
  const replayCachedResult = async (cached: any) => {
    setIsReplayingCache(true)
    console.log('🎬 开始回放缓存的RCA结果')

    try {
      const chunks = buildCachedSSEChunks(cached)

      if (!chunks.length) {
        console.warn('⚠️ 缓存中没有可用的内容')
        setAnalysisState({
          status: 'error',
          messages: [],
          error: '缓存中没有可回放的HolmesGPT数据'
        })
        return
      }

      setAnalysisState({
        status: 'analyzing',
        messages: [],
        totalSteps: 0,
        completedSteps: 0
      })
      allTasksCompletedRef.current = false
      setPinnedTasksData(undefined)
      setPinnedSummaryData(undefined)
      lastSummarySignatureRef.current = null

      const baseDate = cached?.cached_at ? new Date(cached.cached_at) : new Date()
      const timestampFactory = () => format(baseDate, 'HH:mm:ss', { locale: zhCN })
      const processor = createStreamProcessor({
        initialTimestamp: timestampFactory(),
        timestampFactory
      })

      await consumeSSEChunks(createCachedChunkIterable(chunks), processor)

      processor.finalize('completed')
      playNotificationSound()
    } catch (error) {
      console.error('❌ 回放缓存结果失败:', error)
      setAnalysisState({
        status: 'error',
        messages: [],
        error: '加载缓存结果失败'
      })
    } finally {
      setIsReplayingCache(false)
    }
  }

  // 构建 HolmesGPT 调查请求（仅包含必要参数，其余由后端组装）
  const buildInvestigateRequest = () => {
    return {
      alert_id: alert.id,
      depth: 'standard',
      language: settings.language || 'zh-CN',
      include_tool_calls: settings.showToolCalls,
    }
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













  const createStreamProcessor = (options: {
    initialTimestamp: string
    timestampFactory: () => string
  }): StreamProcessor => {
    let buffer = ''
    let completed = false
    let lastAiAnswerSignature: string | null = null

    let currentAssistantMessage: AnalysisMessage = {
      id: generateUniqueId('assistant'),
      role: 'assistant',
      content: '',
      timestamp: options.initialTimestamp,
      isStreaming: true,
      toolCalls: []
    }

    setAnalysisState(prev => ({
      ...prev,
      messages: [...prev.messages, currentAssistantMessage]
    }))

    const syncCurrentMessage = () => {
      const progress = parseProgress(currentAssistantMessage.structuredData)

      setAnalysisState(prev => {
        const exists = prev.messages.some(msg => msg.id === currentAssistantMessage.id)
        const messages = exists
          ? prev.messages.map(msg => (msg.id === currentAssistantMessage.id ? currentAssistantMessage : msg))
          : [...prev.messages, currentAssistantMessage]

        return {
          ...prev,
          messages,
          totalSteps: progress.total,
          completedSteps: progress.completed
        }
      })
    }

    const mergePinnedTasks = (structured?: HolmesStructuredData) => {
      if (!structured) return undefined

      let merged: HolmesStructuredData | undefined
      setPinnedTasksData(prev => {
        merged = mergeHolmesStructuredData(prev, structured)
        return merged
      })

      if (merged) {
        const progress = parseProgress(merged)
        setAnalysisState(prev => ({
          ...prev,
          totalSteps: progress.total,
          completedSteps: progress.completed
        }))

        const allTasks =
          merged.tasks ||
          merged.taskSections?.flatMap(section => section.tasks) ||
          []

        if (allTasks.length > 0 && allTasks.every(task => task.status === 'completed')) {
          allTasksCompletedRef.current = true
        }
      }

      return merged
    }

    const appendSummary = (
      summary: string,
      extraStructured?: HolmesStructuredData
    ) => {
      if (!summary) return
      const normalizedSummary = normalizePlainText(summary)
      if (!normalizedSummary) return
      const dedupedSummary = deduplicateSummaryBlocks(normalizedSummary)
      if (!dedupedSummary.trim()) return

      // 生成签名用于去重
      const signature = buildSummarySignature(dedupedSummary)
      if (signature && signature === lastSummarySignatureRef.current) {
        console.log('检测到重复的summary，跳过添加')
        return
      }

      if (signature) {
        lastSummarySignatureRef.current = signature
      }

      setPinnedSummaryData(prev => {
        const merged = mergeHolmesStructuredData(
          prev,
          extraStructured
            ? { ...extraStructured, summary: dedupedSummary }
            : { summary: dedupedSummary }
        )
        if (merged?.summary) {
          merged.summary = deduplicateSummaryBlocks(merged.summary)
        }
        return merged
      })
    }

    const removeSummaryFromStructuredData = (
      data?: HolmesStructuredData
    ): HolmesStructuredData | undefined => {
      if (!data || data.summary === undefined) {
        return data
      }
      const cloned: HolmesStructuredData = { ...data }
      delete (cloned as any).summary
      return cloned
    }

    const startNewAssistantMessage = () => {
      currentAssistantMessage = {
        id: generateUniqueId('assistant'),
        role: 'assistant',
        content: '',
        timestamp: options.timestampFactory(),
        isStreaming: true,
        toolCalls: []
      }

      setAnalysisState(prev => ({
        ...prev,
        messages: [...prev.messages, currentAssistantMessage]
      }))
    }

    const processDataItem = (item: any, eventType?: string) => {
      if (!item) return
      console.log('Processing data item:', item, 'eventType:', eventType)

      // 处理 ai_message 事件 - 这些是中间过程消息，不是最终结论
      if (eventType === 'ai_message') {
        const messageContent = item.content || item.data?.content
        if (messageContent && typeof messageContent === 'string') {
          const normalized = normalizePlainText(messageContent)
          if (normalized) {
            // 作为进度文本添加到当前消息，不作为summary
            const mergedStructured = mergeHolmesStructuredData(currentAssistantMessage.structuredData, { progressText: normalized })
            currentAssistantMessage = {
              ...currentAssistantMessage,
              structuredData: mergedStructured,
              content: appendTextChunk(currentAssistantMessage.content, normalized),
              isStreaming: true
            }
            syncCurrentMessage()
          }
        }
        return
      }

      if (eventType === 'ai_answer_end') {
        const payload = item && typeof item === 'object' && 'data' in item ? item.data : item
        let combinedStructured: HolmesStructuredData | undefined

        const structuredFromPayload = extractHolmesStructuredData(payload)
        combinedStructured = mergeHolmesStructuredData(combinedStructured, structuredFromPayload)

        let summaryText = ''
        if (structuredFromPayload?.summary) {
          summaryText = appendTextChunk(summaryText, structuredFromPayload.summary)
        }

        const additionalSummaries: string[] = []
        if (typeof payload === 'string') {
          additionalSummaries.push(payload)
        } else if (payload && typeof payload === 'object') {
          if (typeof payload.analysis === 'string' && payload.analysis.trim()) {
            additionalSummaries.push(payload.analysis.trim())
          }
          if (typeof payload.content === 'string' && payload.content.trim()) {
            additionalSummaries.push(payload.content.trim())
          }
        }

        if (typeof item === 'string') {
          additionalSummaries.push(item)
        }

        additionalSummaries.forEach(candidate => {
          summaryText = appendTextChunk(summaryText, candidate)
        })

        const dedupedSummaryText = deduplicateSummaryBlocks(summaryText)
        const signature = dedupedSummaryText ? buildSummarySignature(dedupedSummaryText) : null

        if (signature && signature === lastAiAnswerSignature) {
          console.log('检测到重复 ai_answer_end 事件，跳过处理')
          return
        }

        if (signature) {
          lastAiAnswerSignature = signature
        }

        if (dedupedSummaryText.trim()) {
          // 只添加到pinnedSummary，不添加到当前消息
          appendSummary(dedupedSummaryText)
        }

        // 不要将summary添加到当前消息的structuredData中
        if (combinedStructured) {
          const structuredWithoutSummary = removeSummaryFromStructuredData(combinedStructured)
          const mergedForMessage = removeSummaryFromStructuredData(
            mergeHolmesStructuredData(currentAssistantMessage.structuredData, structuredWithoutSummary)
          )
          currentAssistantMessage = {
            ...currentAssistantMessage,
            structuredData: mergedForMessage,
            isStreaming: true
          }
          syncCurrentMessage()
        }

        return
      }

      if (item.role === 'tool' && item.tool_call_id) {
        if (item.name === 'TodoWrite') {
          console.log('TodoWrite tool result received (unified branch)')
          const structured = extractHolmesStructuredData(item)
          mergePinnedTasks(structured)
          return
        }

        const toolCall = {
          name: item.name,
          input: item.description,
          output: item.result,
          status: (item.result?.status === 'success'
            ? 'success'
            : item.result?.status === 'error'
            ? 'error'
            : 'pending') as 'pending' | 'success' | 'error',
          command: item.result?.invocation || item.description || ''
        }

        console.log('=== CREATING TOOL CALL MESSAGE ===')
        console.log('Tool call data:', toolCall)

        const toolCallMessage: AnalysisMessage = {
          id: `tool-${item.tool_call_id || Date.now()}`,
          role: 'assistant',
          content: '',
          timestamp: options.timestampFactory(),
          isStreaming: false,
          toolCalls: [toolCall]
        }

        if (currentAssistantMessage.content.trim()) {
          currentAssistantMessage = {
            ...currentAssistantMessage,
            isStreaming: false
          }

          setAnalysisState(prev => {
            const exists = prev.messages.some(msg => msg.id === currentAssistantMessage.id)
            const updated = exists
              ? prev.messages.map(msg =>
                  msg.id === currentAssistantMessage.id ? currentAssistantMessage : msg
                )
              : [...prev.messages, currentAssistantMessage]

            return {
              ...prev,
              messages: [...updated, toolCallMessage]
            }
          })

          startNewAssistantMessage()
        } else {
          setAnalysisState(prev => ({
            ...prev,
            messages: [...prev.messages, toolCallMessage]
          }))
        }

        console.log('Tool call message created:', toolCallMessage)
        console.log('=== END TOOL CALL MESSAGE CREATION ===')
        return
      }

      if (item.content && typeof item.content === 'string') {
        const textContent = normalizePlainText(item.content)
        if (!textContent) {
          return
        }

        if (allTasksCompletedRef.current) {
          // 任务完成后，所有内容都作为summary处理
          appendSummary(textContent)
          return
        }

        const structured = extractHolmesStructuredData(item)
        const structuredForMessage = removeSummaryFromStructuredData(structured)

        // 如果提取到了summary，只添加到pinnedSummary，不添加到消息内容
        if (structured?.summary) {
          appendSummary(structured.summary)
          // 不要将summary添加到content中
        }

        const mergedStructured = mergeHolmesStructuredData(currentAssistantMessage.structuredData, structuredForMessage)

        // 只有在没有summary的情况下才添加到content
        const shouldAddToContent = !structured?.summary

        currentAssistantMessage = {
          ...currentAssistantMessage,
          structuredData: mergedStructured,
          content: shouldAddToContent ? appendTextChunk(currentAssistantMessage.content, textContent) : currentAssistantMessage.content,
          isStreaming: true
        }
        syncCurrentMessage()
        return
      }

      if (item.tool_name && item.id && !item.role) {
        console.log('Tool call started:', item.tool_name, item.id)
        return
      }

      if (item.type === 'analysis' || item.type === 'analysis_chunk') {
        const payload = item.data ?? item.content

        if (payload && typeof payload === 'object') {
          const structured = extractHolmesStructuredData(payload)
          const hasTodo = structured?.taskSections?.some(section => section.id === 'todo-write-main')
          if (hasTodo) {
            mergePinnedTasks(structured)
            return
          }

          const structuredForMessage = removeSummaryFromStructuredData(structured)

          // 如果有summary，只添加到pinnedSummary
          if (structured?.summary) {
            appendSummary(structured.summary)
          }

          const mergedStructured = mergeHolmesStructuredData(currentAssistantMessage.structuredData, structuredForMessage)
          currentAssistantMessage = {
            ...currentAssistantMessage,
            structuredData: mergedStructured,
            isStreaming: true
          }
          syncCurrentMessage()
          return
        }

        if (typeof payload === 'string') {
          const normalizedPayload = normalizePlainText(payload)
          if (!normalizedPayload) {
            return
          }

          // 检查是否看起来像结构化的summary
          if (looksLikeStructuredSummary(normalizedPayload)) {
            appendSummary(normalizedPayload)
            return
          }

          // 如果所有任务已完成，将内容作为summary处理
          if (allTasksCompletedRef.current) {
            appendSummary(normalizedPayload)
          } else {
            // 否则作为进度文本处理
            const mergedStructured = mergeHolmesStructuredData(currentAssistantMessage.structuredData, { progressText: normalizedPayload })
            currentAssistantMessage = {
              ...currentAssistantMessage,
              structuredData: mergedStructured,
              content: appendTextChunk(currentAssistantMessage.content, normalizedPayload),
              isStreaming: true
            }
            syncCurrentMessage()
          }
        }
      }
    }

    const processAnalysisData = (analysisData: any, eventType?: string) => {
      console.log('Processing analysis data:', analysisData, 'eventType:', eventType)

      if (typeof analysisData === 'string') {
        const trimmed = analysisData.trim()

        // 只有 ai_answer_end 事件才作为summary处理
        // ai_message 事件已经在 processDataItem 中单独处理
        if (eventType === 'ai_answer_end') {
          if (trimmed) {
            const normalized = normalizePlainText(trimmed)
            allTasksCompletedRef.current = true
            appendSummary(normalized)
            // 清理当前消息中的summary，确保不重复
            currentAssistantMessage = {
              ...currentAssistantMessage,
              structuredData: removeSummaryFromStructuredData(currentAssistantMessage.structuredData),
              content: '', // 清空content，避免重复显示
              isStreaming: true
            }
            syncCurrentMessage()
          }
          return
        }

        // 尝试解析JSON
        if (trimmed.startsWith('{') && trimmed.endsWith('}')) {
          try {
            const parsed = JSON.parse(trimmed)
            if (parsed && typeof parsed === 'object') {
              processDataItem(parsed, 'ai_answer_end')
              return
            }
          } catch (error) {
            console.warn('Failed to parse analysis string as JSON:', error)
          }
        }

        // 检查是否看起来像结构化的summary（只在非ai_message事件时）
        if (eventType !== 'ai_message' && looksLikeStructuredSummary(trimmed)) {
          appendSummary(trimmed)
          return
        }

        // 如果所有任务已完成且不是ai_message事件，作为summary处理
        if (allTasksCompletedRef.current && eventType !== 'ai_message') {
          appendSummary(trimmed)
        } else {
          // 否则作为进度文本处理
          const mergedStructured = mergeHolmesStructuredData(currentAssistantMessage.structuredData, { progressText: trimmed })
          currentAssistantMessage = {
            ...currentAssistantMessage,
            structuredData: mergedStructured,
            content: appendTextChunk(currentAssistantMessage.content, trimmed),
            isStreaming: true
          }
          syncCurrentMessage()
        }
        return
      }

      if (Array.isArray(analysisData)) {
        analysisData.forEach(item => processDataItem(item, eventType))
        return
      }

      if (analysisData && typeof analysisData === 'object') {
        processDataItem(analysisData, eventType)
      }
    }

    const markCompleted = () => {
      if (completed) {
        return
      }

      currentAssistantMessage = {
        ...currentAssistantMessage,
        isStreaming: false
      }
      syncCurrentMessage()
      setAnalysisState(prev => ({
        ...prev,
        status: 'completed'
      }))
      completed = true
    }

    let currentEventType: string | null = null

    const appendChunk = (chunk: string): boolean => {
      buffer += chunk
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      let stop = false

      for (const line of lines) {
        if (line === '') {
          // 空行表示一个事件结束，重置事件类型
          currentEventType = null
          continue
        }

        if (line.startsWith('event: ')) {
          currentEventType = line.slice(7).trim()
          console.log('SSE Event Type:', currentEventType)
          continue
        }
        if (!line.startsWith('data: ')) {
          continue
        }

        const rawData = line.slice(6)

        if (!rawData || rawData === '[DONE]') {
          continue
        }

        if (rawData === '{"type":"complete","data":{}}') {
          console.log('Analysis complete signal received (unified)')
          markCompleted()
          stop = true
          continue
        }

        try {
          const data = JSON.parse(rawData)

          // 使用当前的事件类型来处理数据
          const effectiveEventType = currentEventType || data.type
          console.log('Processing with event type:', effectiveEventType, 'data:', data)

          if (data.type === 'analysis' && data.data !== undefined) {
            processAnalysisData(data.data, effectiveEventType)
          } else if (Array.isArray(data)) {
            data.forEach(item => processDataItem(item, effectiveEventType))
          } else {
            processDataItem(data, effectiveEventType)
          }
        } catch (error) {
          console.error('JSON parse error:', error, 'Line:', line)
        }
      }

      return stop
    }

    const finalize = (status: AnalysisState['status'] = 'completed') => {
      if (status === 'completed') {
        markCompleted()
      } else if (!completed) {
        currentAssistantMessage = {
          ...currentAssistantMessage,
          isStreaming: false
        }
        syncCurrentMessage()
      }

      setAnalysisState(prev => ({
        ...prev,
        status: status === 'completed' && prev.status === 'error' ? prev.status : status
      }))

      if (status === 'completed') {
        completed = true
      }
    }

    return {
      appendChunk,
      finalize,
      processAnalysisData,
      processDataItem
    }
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
    lastSummarySignatureRef.current = null

    try {
      // 构建 API 路径，包含 basePath（如果配置了）
      const basePath = appConfig.basePath || ''
      const apiPath = `${basePath}/api/holmesgpt/stream/investigate`

      const response = await fetch(apiPath, {
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

      const timestampFactory = () => format(new Date(), 'HH:mm:ss', { locale: zhCN })
      const processor = createStreamProcessor({
        initialTimestamp: timestampFactory(),
        timestampFactory
      })
      const { iterable, stop } = createResponseChunkIterable(response)
      await consumeSSEChunks(iterable, processor, { onStop: stop })

      processor.finalize('completed')

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

  const handleReanalyze = () => {
    void startAnalysis()
  }

  // 停止分析
  const stopAnalysis = () => {
    if (abortController) {
      abortController.abort()
    }
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

  // 内容区域组件（已拆分，保留旧实现以便审阅）
  const LegacyContentArea = () => null



  // 控制栏组件
  // 控制栏组件（已拆分，保留旧实现以便审阅）
  const LegacyControlBar = () => null

  // 根据showCard决定返回结构
  if (!showCard) {
    return (
      <div className="w-full h-full flex flex-col overflow-hidden">
        <ControlBar
          statusIcon={getStatusIcon()}
          statusText={getStatusText()}
          progressText={getProgressText()}
          showSettings={showSettings}
          toggleSettings={() => setShowSettings(!showSettings)}
          settings={settings}
          setSettings={setSettings}
          analysisStatus={analysisState.status}
          loadingCache={loadingCache}
          cachedResult={cachedResult}
          isReplayingCache={isReplayingCache}
          onStart={startAnalysis}
          onStop={stopAnalysis}
          onReanalyze={handleReanalyze}
          onDownload={downloadAnalysis}
        />
        <ContentArea
          analysisState={analysisState}
          scrollAreaRef={scrollAreaRef}
          messagesEndRef={messagesEndRef}
          handleScroll={handleScroll}
          showScrollToLatest={showScrollToLatest}
          handleScrollToLatest={handleScrollToLatest}
          pinnedTasksData={pinnedTasksData}
          pinnedSummaryData={pinnedSummaryData}
          tasksMessageId={tasksMessageIdRef.current}
          summaryMessageId={summaryMessageIdRef.current}
          onReanalyze={handleReanalyze}
        />
      </div>
    )
  }

  return (
    <Card className="w-full h-full flex flex-col">
      <CardHeader className="pb-3 flex-shrink-0">
        <ControlBar
          statusIcon={getStatusIcon()}
          statusText={getStatusText()}
          progressText={getProgressText()}
          showSettings={showSettings}
          toggleSettings={() => setShowSettings(!showSettings)}
          settings={settings}
          setSettings={setSettings}
          analysisStatus={analysisState.status}
          loadingCache={loadingCache}
          cachedResult={cachedResult}
          isReplayingCache={isReplayingCache}
          onStart={startAnalysis}
          onStop={stopAnalysis}
          onReanalyze={handleReanalyze}
          onDownload={downloadAnalysis}
        />
      </CardHeader>

      <Separator className="flex-shrink-0" />

      <CardContent className="p-0 flex-1 flex flex-col min-h-0">
        <ContentArea
          analysisState={analysisState}
          scrollAreaRef={scrollAreaRef}
          messagesEndRef={messagesEndRef}
          handleScroll={handleScroll}
          showScrollToLatest={showScrollToLatest}
          handleScrollToLatest={handleScrollToLatest}
          pinnedTasksData={pinnedTasksData}
          pinnedSummaryData={pinnedSummaryData}
          tasksMessageId={tasksMessageIdRef.current}
          summaryMessageId={summaryMessageIdRef.current}
          onReanalyze={handleReanalyze}
        />
      </CardContent>
    </Card>
  )
}
