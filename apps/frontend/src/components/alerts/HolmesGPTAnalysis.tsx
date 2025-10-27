'use client'

import React, { useState, useRef, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'

import {
  Brain,
  Play,
  Square,
  Loader2,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Copy,
  Download,
  RotateCcw
} from 'lucide-react'
import { toast } from 'sonner'
import { AIAnalysisDisplay } from './AIAnalysisDisplay'

interface Alert {
  id: string
  title: string
  description: string
  severity: string
  status: string
  cluster_id: string
  labels?: Record<string, any>
  annotations?: Record<string, any>
  fingerprint: string
  created_at: string
  starts_at?: string
  ends_at?: string
}

interface HolmesGPTAnalysisProps {
  alert: Alert
  className?: string
}

interface StreamEvent {
  type: 'analysis' | 'tool_call' | 'error' | 'complete'
  data: any
}

export function HolmesGPTAnalysis({ alert, className }: HolmesGPTAnalysisProps) {
  const [isAnalyzing, setIsAnalyzing] = useState(false)
  const [analysis, setAnalysis] = useState<string>('')
  const [toolCalls, setToolCalls] = useState<any[]>([])
  const [error, setError] = useState<string | null>(null)
  const [isComplete, setIsComplete] = useState(false)
  const abortControllerRef = useRef<AbortController | null>(null)

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

  // 启动流式分析
  const startAnalysis = async () => {
    if (isAnalyzing) return

    setIsAnalyzing(true)
    setAnalysis('')
    setToolCalls([])
    setError(null)
    setIsComplete(false)

    // 创建新的 AbortController
    abortControllerRef.current = new AbortController()

    try {
      const investigateRequest = buildInvestigateRequest()
      
      const response = await fetch('/api/holmesgpt/stream/investigate', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(investigateRequest),
        signal: abortControllerRef.current.signal,
      })

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }

      if (!response.body) {
        throw new Error('响应体为空')
      }

      const reader = response.body.getReader()
      const decoder = new TextDecoder()

      while (true) {
        const { done, value } = await reader.read()
        
        if (done) {
          setIsComplete(true)
          break
        }

        const chunk = decoder.decode(value, { stream: true })
        const lines = chunk.split('\n')

        for (const line of lines) {
          if (line.trim() === '') continue
          
          try {
            // 处理 Server-Sent Events 格式
            if (line.startsWith('data: ')) {
              const eventData = line.slice(6) // 移除 'data: ' 前缀

              if (eventData === '[DONE]') {
                setIsComplete(true)
                continue
              }

              const event: StreamEvent = JSON.parse(eventData)

              switch (event.type) {
                case 'analysis':
                  // 累积分析文本
                  if (typeof event.data === 'string') {
                    setAnalysis(prev => prev + event.data)
                  } else {
                    // 如果是对象，尝试提取文本内容
                    const text = event.data.analysis || event.data.content || JSON.stringify(event.data)
                    setAnalysis(prev => prev + text)
                  }
                  break
                case 'tool_call':
                  // 添加工具调用记录
                  if (Array.isArray(event.data)) {
                    setToolCalls(prev => [...prev, ...event.data])
                  } else {
                    setToolCalls(prev => [...prev, event.data])
                  }
                  break
                case 'error':
                  setError(event.data.message || event.data || '分析过程中发生错误')
                  break
                case 'complete':
                  setIsComplete(true)
                  break
                default:
                  // Unknown event type
              }
            }
          } catch (parseError) {
            // 尝试将整行作为分析文本处理
            if (line.trim() && !line.startsWith('data: ')) {
              setAnalysis(prev => prev + line + '\n')
            }
          }
        }
      }

      toast.success('HolmesGPT 分析完成')
    } catch (err: any) {
      if (err.name === 'AbortError') {
        toast.info('分析已停止')
      } else {
        setError(err.message || '分析失败')
        toast.error(`分析失败: ${err.message}`)
      }
    } finally {
      setIsAnalyzing(false)
      abortControllerRef.current = null
    }
  }

  // 停止分析
  const stopAnalysis = () => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort()
    }
  }

  // 复制分析结果
  const copyAnalysis = async () => {
    try {
      await navigator.clipboard.writeText(analysis)
      toast.success('分析结果已复制到剪贴板')
    } catch (err) {
      toast.error('复制失败')
    }
  }

  // 下载分析结果
  const downloadAnalysis = () => {
    const content = `# HolmesGPT 分析报告

## 告警信息
- 标题: ${alert.title}
- 描述: ${alert.description}
- 严重级别: ${alert.severity}
- 状态: ${alert.status}
- 集群: ${alert.cluster_id}
- 指纹: ${alert.fingerprint}

## 分析结果
${analysis}

## 工具调用记录
${toolCalls.map((tool, index) => `
### 工具 ${index + 1}: ${tool.tool_name}
- 描述: ${tool.description}
- 结果: ${JSON.stringify(tool.result, null, 2)}
`).join('\n')}

---
生成时间: ${new Date().toLocaleString('zh-CN')}
`

    const blob = new Blob([content], { type: 'text/markdown' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `holmes-analysis-${alert.id}-${Date.now()}.md`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    
    toast.success('分析报告已下载')
  }

  // 重新分析
  const restartAnalysis = () => {
    setAnalysis('')
    setToolCalls([])
    setError(null)
    setIsComplete(false)
    startAnalysis()
  }

  return (
    <Card className={className}>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className="w-10 h-10 bg-gradient-to-br from-purple-500 to-blue-600 rounded-lg flex items-center justify-center">
              <Brain className="h-5 w-5 text-white" />
            </div>
            <div>
              <CardTitle className="text-lg">HolmesGPT 智能分析</CardTitle>
              <p className="text-sm text-muted-foreground">
                AI 驱动的根因分析和故障排查
              </p>
            </div>
          </div>
          
          <div className="flex items-center space-x-2">
            {analysis && (
              <>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={copyAnalysis}
                  className="h-8"
                >
                  <Copy className="h-4 w-4" />
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={downloadAnalysis}
                  className="h-8"
                >
                  <Download className="h-4 w-4" />
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={restartAnalysis}
                  className="h-8"
                  disabled={isAnalyzing}
                >
                  <RotateCcw className="h-4 w-4" />
                </Button>
              </>
            )}
            
            <Button
              onClick={isAnalyzing ? stopAnalysis : startAnalysis}
              disabled={false}
              size="sm"
              className={isAnalyzing ? 'bg-red-600 hover:bg-red-700' : ''}
            >
              {isAnalyzing ? (
                <>
                  <Square className="h-4 w-4 mr-2" />
                  停止分析
                </>
              ) : (
                <>
                  <Play className="h-4 w-4 mr-2" />
                  触发RCA分析
                </>
              )}
            </Button>
          </div>
        </div>
      </CardHeader>
      
      <CardContent className="p-0">
        {/* 使用新的 AI 分析展示组件 */}
        <AIAnalysisDisplay
          analysis={analysis}
          isAnalyzing={isAnalyzing}
          isComplete={isComplete}
          error={error}
          className="p-6"
        />

        {/* 工具调用记录 */}
        {toolCalls.length > 0 && (
          <div className="border-t border-gray-200 p-6">
            <div className="flex items-center space-x-2 mb-4">
              <AlertTriangle className="h-5 w-5 text-gray-600" />
              <h4 className="font-semibold text-gray-900">工具调用记录</h4>
              <Badge variant="secondary" className="ml-2">
                {toolCalls.length}
              </Badge>
            </div>
            <div className="space-y-3 max-h-64 overflow-auto">
              {toolCalls.map((tool, index) => (
                <div key={index} className="border border-gray-200 rounded-lg p-3 bg-gray-50">
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center space-x-2">
                      <Badge variant="outline" className="text-xs">
                        #{index + 1}
                      </Badge>
                      <span className="font-medium text-sm text-gray-900">{tool.tool_name}</span>
                    </div>
                    <span className="text-xs text-gray-500 bg-white px-2 py-1 rounded">
                      {tool.result?.status || 'unknown'}
                    </span>
                  </div>
                  <p className="text-sm text-gray-600 mb-2">{tool.description}</p>
                  <details className="text-xs">
                    <summary className="cursor-pointer text-gray-500 hover:text-gray-700">
                      查看详细结果
                    </summary>
                    <div className="mt-2 bg-white rounded p-2 border">
                      <pre className="text-gray-700 whitespace-pre-wrap overflow-auto max-h-32">
                        {JSON.stringify(tool.result, null, 2)}
                      </pre>
                    </div>
                  </details>
                </div>
              ))}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
