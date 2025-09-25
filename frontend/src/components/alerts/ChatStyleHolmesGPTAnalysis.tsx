'use client'

import React, { useState, useRef, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { 
  Brain, 
  Play, 
  Square, 
  RotateCcw, 
  Download,
  Loader2,
  AlertCircle,
  CheckCircle2
} from 'lucide-react'
import { Alert } from '@/types/api'
import { ChatMessage } from './ChatMessage'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'

interface ChatStyleHolmesGPTAnalysisProps {
  alert: Alert
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
  }>
}

interface AnalysisState {
  status: 'idle' | 'analyzing' | 'completed' | 'error'
  messages: AnalysisMessage[]
  error?: string
}

export const ChatStyleHolmesGPTAnalysis: React.FC<ChatStyleHolmesGPTAnalysisProps> = ({ alert }) => {
  const [analysisState, setAnalysisState] = useState<AnalysisState>({
    status: 'idle',
    messages: []
  })
  const [abortController, setAbortController] = useState<AbortController | null>(null)
  const scrollAreaRef = useRef<HTMLDivElement>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  // 自动滚动到底部
  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(() => {
    scrollToBottom()
  }, [analysisState.messages])

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
      source_instance_id: 'WebUI-Chinese',
      include_tool_calls: true,
      include_tool_call_results: true,
      prompt_template: 'builtin://generic_investigation.jinja2'
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
      messages: [userMessage]
    })

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

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const data = JSON.parse(line.slice(6))
              
              if (data.type === 'analysis' || data.type === 'analysis_chunk') {
                // 处理分析内容
                const content = data.data || data.content || ''
                currentAssistantMessage = {
                  ...currentAssistantMessage,
                  content: currentAssistantMessage.content + content,
                  isStreaming: true
                }

                setAnalysisState(prev => ({
                  ...prev,
                  messages: prev.messages.map(msg =>
                    msg.id === currentAssistantMessage.id ? currentAssistantMessage : msg
                  )
                }))
              } else if (data.type === 'tool_call') {
                const toolCall = {
                  name: data.tool_name,
                  input: data.input,
                  output: data.output,
                  status: data.status as 'pending' | 'success' | 'error'
                }
                
                currentAssistantMessage = {
                  ...currentAssistantMessage,
                  toolCalls: [...(currentAssistantMessage.toolCalls || []), toolCall]
                }
                
                setAnalysisState(prev => ({
                  ...prev,
                  messages: prev.messages.map(msg => 
                    msg.id === currentAssistantMessage.id ? currentAssistantMessage : msg
                  )
                }))
              } else if (data.type === 'complete' || data.type === 'analysis_complete') {
                currentAssistantMessage = {
                  ...currentAssistantMessage,
                  isStreaming: false
                }

                setAnalysisState(prev => ({
                  status: 'completed',
                  messages: prev.messages.map(msg =>
                    msg.id === currentAssistantMessage.id ? currentAssistantMessage : msg
                  )
                }))
                break
              } else if (data.type === 'error') {
                // 处理错误
                setAnalysisState(prev => ({
                  status: 'error',
                  messages: prev.messages,
                  error: data.data?.message || '分析过程中发生错误'
                }))
                break
              }
            } catch (error) {
              console.error('解析 SSE 数据失败:', error)
            }
          }
        }
      }
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
      messages: []
    })
  }

  // 下载分析结果
  const downloadAnalysis = () => {
    const analysisText = analysisState.messages
      .map(msg => `[${msg.timestamp}] ${msg.role === 'user' ? '用户' : 'HolmesGPT'}: ${msg.content}`)
      .join('\n\n')
    
    const blob = new Blob([analysisText], { type: 'text/plain;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `holmesgpt-analysis-${alert.id}-${Date.now()}.txt`
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

  return (
    <Card className="w-full h-full flex flex-col">
      <CardHeader className="pb-3 flex-shrink-0">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            {getStatusIcon()}
            <div>
              <CardTitle className="text-lg">{getStatusText()}</CardTitle>
              <p className="text-sm text-muted-foreground mt-1">
                基于 Kubernetes 专业知识的智能故障分析
              </p>
            </div>
          </div>
          <div className="flex items-center space-x-2">
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
      </CardHeader>

      <Separator className="flex-shrink-0" />

      <CardContent className="p-0 flex-1 flex flex-col min-h-0">
        {analysisState.messages.length === 0 ? (
          <div className="flex flex-col items-center justify-center flex-1 text-center p-8">
            <Brain className="h-12 w-12 text-gray-400 mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">准备开始智能分析</h3>
            <p className="text-sm text-gray-500 max-w-md">
              点击"触发RCA分析"按钮，HolmesGPT 将为您分析告警的根本原因，并提供详细的解决方案和预防措施。
            </p>
          </div>
        ) : (
          <ScrollArea className="flex-1" ref={scrollAreaRef}>
            <div className="space-y-0 min-h-full">
              {analysisState.messages.map((message) => (
                <ChatMessage
                  key={message.id}
                  role={message.role}
                  content={message.content}
                  timestamp={message.timestamp}
                  isStreaming={message.isStreaming}
                  toolCalls={message.toolCalls}
                />
              ))}
              <div ref={messagesEndRef} />
            </div>
          </ScrollArea>
        )}
        
        {analysisState.status === 'error' && analysisState.error && (
          <div className="p-4 border-t">
            <div className="flex items-center space-x-2 text-red-600">
              <AlertCircle className="h-4 w-4" />
              <span className="text-sm font-medium">分析失败</span>
            </div>
            <p className="text-sm text-red-600 mt-1">{analysisState.error}</p>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
