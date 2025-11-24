'use client'

import React, { useState, useRef, useEffect, useCallback } from 'react'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { Brain, Loader2, AlertCircle, CheckCircle2, Clock, Bot } from 'lucide-react'
import { structuredDataToMarkdown } from './holmesUtils'
import { ControlBar } from './ControlBar'
import { ContentArea } from './ContentArea'
import { ApprovalDialog } from './ApprovalDialog'
import { EnhancedHolmesGPTChatProps, ChatSettings, AnalysisMessage } from './holmesTypes'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { useHolmesAnalysis } from './useHolmesAnalysis'

export const EnhancedHolmesGPTChat: React.FC<EnhancedHolmesGPTChatProps> = ({
  alert,
  showCard = true,
  cachedResult,
  loadingCache = false
}) => {
  const [settings, setSettings] = useState<ChatSettings>({
    autoScroll: false,
    soundEnabled: false,
    showToolCalls: true,
    language: 'zh-CN'
  })
  const [showSettings, setShowSettings] = useState(false)

  // Refs for scrolling
  const scrollAreaRef = useRef<HTMLDivElement>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const [isUserNearBottom, setIsUserNearBottom] = useState(true)
  const [showScrollToLatest, setShowScrollToLatest] = useState(false)
  const lastContentSignatureRef = useRef<string>('')
  const lastReplayedCacheKeyRef = useRef<string | null>(null)
  const tasksMessageIdRef = useRef<string>('tasks-pinned')
  const summaryMessageIdRef = useRef<string>('summary-pinned')

  // Sound player
  const playNotificationSound = useCallback(() => {
    if (settings.soundEnabled && typeof window !== 'undefined') {
      try {
        const audio = new Audio('/notification.mp3')
        audio.volume = 0.3
        audio.play().catch(() => { })
      } catch (error) { }
    }
  }, [settings.soundEnabled])

  // Use the custom hook
  const {
    analysisState,
    pinnedTasksData,
    pinnedSummaryData,
    pendingApproval,
    isSubmittingApproval,
    isReplayingCache,
    startAnalysis,
    stopAnalysis,
    replayCachedResult,
    sendApprovalDecision
  } = useHolmesAnalysis(alert.id, {
    onPlaySound: playNotificationSound
  })

  // Scrolling logic
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

  // Auto-scroll effect
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

  // Scroll listener
  useEffect(() => {
    const frame = requestAnimationFrame(() => {
      handleScroll()
    })
    return () => cancelAnimationFrame(frame)
  }, [analysisState.messages, pinnedTasksData, pinnedSummaryData, handleScroll])

  // Cache replay effect
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
      // If already analyzing something else, don't interrupt automatically?
      // Or should we restart? The original code restarted.
      // But here we can't easily "restart" inside the effect without causing loops if not careful.
      // The hook's replayCachedResult handles reset.
    }

    lastReplayedCacheKeyRef.current = cacheKey
    console.log('✅ 找到RCA缓存，将自动回放:', cachedResult)
    replayCachedResult(cachedResult)
  }, [cachedResult, loadingCache, isReplayingCache, analysisState.status, analysisState.messages.length, replayCachedResult])

  const handleStartAnalysis = () => {
    startAnalysis(alert, settings)
  }

  const handleReanalyze = () => {
    startAnalysis(alert, settings)
  }

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
    // We can use the hook's state for steps if available, or parse from pinned data
    if (analysisState.totalSteps && analysisState.completedSteps !== undefined) {
      return `${analysisState.completedSteps}/${analysisState.totalSteps} 步骤`
    }
    return null
  }

  const getRoleLabel = (role: AnalysisMessage['role']) => {
    switch (role) {
      case 'assistant': return 'HolmesGPT'
      case 'user': return '用户'
      case 'system': return '系统'
      default: return role
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
          onStart={handleStartAnalysis}
          onStop={stopAnalysis}
          onReanalyze={handleReanalyze}
          onDownload={downloadAnalysis}
        />
        {pendingApproval && !isReplayingCache && (
          <ApprovalDialog
            approval={pendingApproval}
            onDecision={sendApprovalDecision}
            disabled={isSubmittingApproval}
          />
        )}
        <ContentArea
          analysisState={analysisState}
          scrollAreaRef={scrollAreaRef as React.RefObject<HTMLDivElement>}
          messagesEndRef={messagesEndRef as React.RefObject<HTMLDivElement>}
          handleScroll={handleScroll}
          showScrollToLatest={showScrollToLatest}
          handleScrollToLatest={handleScrollToLatest}
          pinnedTasksData={pinnedTasksData}
          pinnedSummaryData={pinnedSummaryData}
          tasksMessageId={tasksMessageIdRef.current}
          summaryMessageId={summaryMessageIdRef.current}
          onReanalyze={handleReanalyze}
          isReplayingCache={isReplayingCache}
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
          onStart={handleStartAnalysis}
          onStop={stopAnalysis}
          onReanalyze={handleReanalyze}
          onDownload={downloadAnalysis}
        />
      </CardHeader>

      <Separator className="flex-shrink-0" />

      <CardContent className="p-0 flex-1 flex flex-col min-h-0">
        {pendingApproval && !isReplayingCache && (
          <ApprovalDialog
            approval={pendingApproval}
            onDecision={sendApprovalDecision}
            disabled={isSubmittingApproval}
          />
        )}
        <ContentArea
          analysisState={analysisState}
          scrollAreaRef={scrollAreaRef as React.RefObject<HTMLDivElement>}
          messagesEndRef={messagesEndRef as React.RefObject<HTMLDivElement>}
          handleScroll={handleScroll}
          showScrollToLatest={showScrollToLatest}
          handleScrollToLatest={handleScrollToLatest}
          pinnedTasksData={pinnedTasksData}
          pinnedSummaryData={pinnedSummaryData}
          tasksMessageId={tasksMessageIdRef.current}
          summaryMessageId={summaryMessageIdRef.current}
          onReanalyze={handleReanalyze}
          isReplayingCache={isReplayingCache}
        />
      </CardContent>
    </Card>
  )
}
