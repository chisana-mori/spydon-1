'use client'

import React, { FC, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import remarkMath from 'remark-math'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { oneDark } from 'react-syntax-highlighter/dist/cjs/styles/prism'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { 
  Copy, 
  Download, 
  Check, 
  Bot, 
  User, 
  Clock,
  AlertTriangle,
  ListChecks,
  CheckCircle2,
  Circle,
  Loader2,
  ChevronDown
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { JsonViewer } from './JsonViewer'

export interface HolmesTaskItem {
  id: string
  content: string
  status: 'pending' | 'in_progress' | 'completed'
  note?: string
}

export interface HolmesTaskSection {
  id: string
  title: string
  toolName: string
  tasks: HolmesTaskItem[]
  statusText?: string
  commands?: string[]
}

export interface HolmesStructuredData {
  planText?: string
  tasks?: HolmesTaskItem[]
  toolName?: string
  statusText?: string
  taskSections?: HolmesTaskSection[]
  // 新增：用于展示中间过程的排查进度（而非最终结论）
  progressText?: string
  summary?: string
  raw?: any
}

interface ChatMessageProps {
  role: 'user' | 'assistant' | 'system'
  content: string
  timestamp?: string
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

interface CodeBlockProps {
  language: string
  value: string
}

const programmingLanguages: Record<string, string> = {
  javascript: '.js',
  python: '.py',
  java: '.java',
  c: '.c',
  cpp: '.cpp',
  'c++': '.cpp',
  'c#': '.cs',
  ruby: '.rb',
  php: '.php',
  swift: '.swift',
  'objective-c': '.m',
  kotlin: '.kt',
  typescript: '.ts',
  go: '.go',
  perl: '.pl',
  rust: '.rs',
  scala: '.scala',
  haskell: '.hs',
  lua: '.lua',
  shell: '.sh',
  bash: '.sh',
  sh: '.sh',
  zsh: '.sh',
  sql: '.sql',
  html: '.html',
  css: '.css',
  yaml: '.yaml',
  yml: '.yaml',
  json: '.json',
  xml: '.xml',
  dockerfile: 'Dockerfile',
  docker: 'Dockerfile',
  nginx: '.conf',
  apache: '.conf',
  config: '.conf',
  conf: '.conf'
}

const languageDisplayNames: Record<string, string> = {
  javascript: 'JavaScript',
  typescript: 'TypeScript',
  python: 'Python',
  java: 'Java',
  cpp: 'C++',
  c: 'C',
  csharp: 'C#',
  ruby: 'Ruby',
  php: 'PHP',
  go: 'Go',
  rust: 'Rust',
  shell: 'Shell',
  bash: 'Bash',
  sh: 'Shell',
  zsh: 'Zsh',
  sql: 'SQL',
  html: 'HTML',
  css: 'CSS',
  yaml: 'YAML',
  yml: 'YAML',
  json: 'JSON',
  xml: 'XML',
  dockerfile: 'Dockerfile',
  docker: 'Dockerfile',
  nginx: 'Nginx Config',
  apache: 'Apache Config',
  config: 'Config',
  conf: 'Config',
  kubectl: 'Kubectl',
  k8s: 'Kubernetes',
  kubernetes: 'Kubernetes'
}

const CodeBlock: FC<CodeBlockProps> = ({ language, value }) => {
  const { isCopied, copyToClipboard } = useCopyToClipboard({ timeout: 2000 })

  // 智能语言检测
  const detectLanguage = (lang: string, code: string): string => {
    const lowerLang = lang.toLowerCase()
    
    // 直接匹配
    if (lowerLang && programmingLanguages[lowerLang]) {
      return lowerLang
    }
    
    // 基于内容检测
    if (code.includes('kubectl ') || code.includes('docker ') || code.includes('helm ')) {
      return 'bash'
    }
    if (code.includes('apiVersion:') || code.includes('kind:') || code.includes('metadata:')) {
      return 'yaml'
    }
    if (code.includes('nslookup ') || code.includes('dig ') || code.includes('ping ')) {
      return 'bash'
    }
    if (code.includes('SELECT ') || code.includes('FROM ') || code.includes('WHERE ')) {
      return 'sql'
    }
    
    return lowerLang || 'text'
  }

  const detectedLanguage = detectLanguage(language, value)
  const displayName = languageDisplayNames[detectedLanguage] || detectedLanguage.toUpperCase()
  const lineCount = value.split('\n').length

  const downloadAsFile = () => {
    if (typeof window === 'undefined') return
    
    const fileExtension = programmingLanguages[detectedLanguage] || '.txt'
    const suggestedFileName = `code-${Date.now()}${fileExtension}`
    const fileName = window.prompt('输入文件名', suggestedFileName)
    
    if (!fileName) return
    
    const blob = new Blob([value], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.download = fileName
    link.href = url
    link.style.display = 'none'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
  }

  const onCopy = () => {
    if (isCopied) return
    copyToClipboard(value)
  }

  return (
    <div className="codeblock relative w-full bg-gray-950 font-mono rounded-lg overflow-hidden border border-gray-800 shadow-lg my-4">
      {/* 代码块头部 */}
      <div className="flex w-full items-center justify-between bg-gray-900 px-4 py-3 border-b border-gray-800">
        <div className="flex items-center space-x-3">
          <div className="flex space-x-1.5">
            <div className="w-3 h-3 rounded-full bg-red-500"></div>
            <div className="w-3 h-3 rounded-full bg-yellow-500"></div>
            <div className="w-3 h-3 rounded-full bg-green-500"></div>
          </div>
          <div className="flex items-center space-x-2">
            <span className="text-sm font-medium text-gray-300">{displayName}</span>
            <span className="text-xs text-gray-500">·</span>
            <span className="text-xs text-gray-500">{lineCount} 行</span>
          </div>
        </div>
        <div className="flex items-center space-x-1">
          <Button
            variant="ghost"
            size="sm"
            className="h-8 w-8 p-0 hover:bg-gray-800 text-gray-400 hover:text-white transition-colors"
            onClick={downloadAsFile}
            title="下载文件"
          >
            <Download className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="h-8 w-8 p-0 hover:bg-gray-800 text-gray-400 hover:text-white transition-colors"
            onClick={onCopy}
            title={isCopied ? '已复制' : '复制代码'}
          >
            {isCopied ? <Check className="h-4 w-4 text-green-400" /> : <Copy className="h-4 w-4" />}
          </Button>
        </div>
      </div>
      
      {/* 代码内容 */}
      <div className="relative">
        <SyntaxHighlighter
          language={detectedLanguage}
          style={oneDark}
          showLineNumbers={lineCount > 3}
          lineNumberStyle={{
            minWidth: '3em',
            paddingRight: '1em',
            color: '#6B7280',
            backgroundColor: 'transparent',
            fontSize: '12px',
            userSelect: 'none'
          }}
          customStyle={{
            margin: 0,
            padding: '1rem',
            width: '100%',
            background: 'transparent',
            fontSize: '14px',
            lineHeight: '1.5',
            fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Consolas, "Liberation Mono", Menlo, monospace'
          }}
          codeTagProps={{
            style: {
              fontFamily: 'inherit'
            }
          }}
        >
          {value}
        </SyntaxHighlighter>
        
        {/* 复制成功提示 */}
        {isCopied && (
          <div className="absolute top-4 right-4 bg-green-600 text-white px-3 py-1 rounded-md text-sm font-medium shadow-lg animate-in fade-in-0 zoom-in-95 duration-300">
            已复制到剪贴板
          </div>
        )}
      </div>
    </div>
  )
}

export const ChatMessage: FC<ChatMessageProps> = ({
  role,
  content,
  timestamp,
  isStreaming = false,
  toolCalls = [],
  structuredData
}) => {
  const { isCopied: isAnalysisCopied, copyToClipboard } = useCopyToClipboard({ timeout: 2000 })

  // 简单的 JSON 检测
  const isJsonContent = (text: string): boolean => {
    const trimmed = text.trim()
    return (trimmed.startsWith('{') && trimmed.endsWith('}')) || 
           (trimmed.startsWith('[') && trimmed.endsWith(']'))
  }

  const parseJsonSafely = (text: string) => {
    try {
      return JSON.parse(text)
    } catch {
      return null
    }
  }

  const jsonData = !structuredData && isJsonContent(content) ? parseJsonSafely(content) : null

  const getRoleIcon = () => {
    switch (role) {
      case 'assistant':
        return <Bot className="h-6 w-6 text-blue-600" />
      case 'user':
        return <User className="h-6 w-6 text-green-600" />
      case 'system':
        return <AlertTriangle className="h-6 w-6 text-orange-600" />
      default:
        return <Bot className="h-6 w-6 text-gray-600" />
    }
  }

  const getRoleName = () => {
    switch (role) {
      case 'assistant':
        return 'HolmesGPT'
      case 'user':
        return '用户'
      case 'system':
        return '系统'
      default:
        return '未知'
    }
  }

  const getRoleColor = () => {
    switch (role) {
      case 'assistant':
        return 'bg-blue-50 border-blue-200'
      case 'user':
        return 'bg-green-50 border-green-200'
      case 'system':
        return 'bg-orange-50 border-orange-200'
      default:
        return 'bg-gray-50 border-gray-200'
    }
  }

  const renderTaskStatusIcon = (status: HolmesTaskItem['status']) => {
    switch (status) {
      case 'completed':
        return <CheckCircle2 className="h-4 w-4 text-green-600 mt-0.5" />
      case 'in_progress':
        return <Loader2 className="h-4 w-4 text-blue-600 mt-0.5 animate-spin" />
      default:
        return <Circle className="h-4 w-4 text-gray-400 mt-0.5" />
    }
  }

  const getTaskStatusLabel = (status: HolmesTaskItem['status']) => {
    switch (status) {
      case 'completed':
        return '已完成'
      case 'in_progress':
        return '进行中'
      default:
        return '待处理'
    }
  }

  const getTaskBadgeVariant = (status: HolmesTaskItem['status']) => {
    switch (status) {
      case 'completed':
        return 'default' as const
      case 'in_progress':
        return 'secondary' as const
      default:
        return 'outline' as const
    }
  }

  const [openSections, setOpenSections] = useState<Record<string, boolean>>({})
  const [expandedTasks, setExpandedTasks] = useState<Record<string, boolean>>({})

  const toggleSection = (id: string) => {
    setOpenSections(prev => ({
      ...prev,
      [id]: !prev[id],
    }))
  }

  const toggleTaskExpanded = (taskId: string) => {
    setExpandedTasks(prev => ({
      ...prev,
      [taskId]: !prev[taskId]
    }))
  }

  const renderTaskList = (tasks: HolmesTaskItem[]) => (
    <ul className="divide-y divide-gray-100">
      {tasks.map((task, index) => {
        const isFirst = index === 0
        const isLast = index === tasks.length - 1
        const prevTask = index > 0 ? tasks[index - 1] : null
        const nextTask = index < tasks.length - 1 ? tasks[index + 1] : null
        
        return (
          <li key={task.id} className={cn(
            "relative px-4 py-3 flex items-start justify-between transition-colors",
            task.status === 'in_progress' ? 'bg-blue-50/50' : '',
            task.status === 'completed' ? 'bg-green-50/30' : ''
          )}>
            {/* 连接线 */}
            {!isFirst && (
              <div className="absolute left-6 top-0 w-0.5 h-3 bg-gray-200" />
            )}
            {!isLast && (
              <div className="absolute left-6 bottom-0 w-0.5 h-3 bg-gray-200" />
            )}
            
            <div className="flex items-start space-x-3 pr-3 flex-1">
              <div className="relative z-10 bg-white">
                {renderTaskStatusIcon(task.status)}
              </div>
              <div className="flex-1 min-w-0">
                <p className={cn(
                  "text-sm leading-relaxed break-words",
                  task.status === 'completed' ? 'text-gray-700 line-through' : 'text-gray-900',
                  task.status === 'in_progress' ? 'text-blue-900 font-medium' : ''
                )}>
                  {task.content}
                </p>
                {task.note && (
                  <p className="mt-1 text-xs text-gray-500 whitespace-pre-wrap break-words">
                    {task.note}
                  </p>
                )}
                {task.status === 'in_progress' && (
                  <div className="mt-2 flex items-center space-x-1 text-xs text-blue-600">
                    <div className="w-1 h-1 bg-blue-500 rounded-full animate-pulse" />
                    <span>正在执行...</span>
                  </div>
                )}
              </div>
            </div>
            <div className="flex flex-col items-end space-y-1">
              <Badge variant={getTaskBadgeVariant(task.status)} className="text-xs whitespace-nowrap">
                {getTaskStatusLabel(task.status)}
              </Badge>
              {task.status === 'completed' && (
                <span className="text-xs text-green-600 flex items-center">
                  <CheckCircle2 className="h-3 w-3 mr-1" />
                  已完成
                </span>
              )}
            </div>
          </li>
        )
      })}
    </ul>
  )

  const renderStructuredContent = (data: HolmesStructuredData) => {
    const {
      planText,
      tasks = [],
      toolName,
      statusText,
      progressText,
      summary,
      taskSections = [],
    } = data

    const copyText = buildStructuredCopyText(data)
  const hasActiveTasks = taskSections.some(section => 
    section.tasks.some(task => task.status === 'in_progress')
  ) || tasks.some(task => task.status === 'in_progress')

  const hasPendingTasks = taskSections.some(section => 
    section.tasks.some(task => task.status === 'pending')
  ) || tasks.some(task => task.status === 'pending')

  const allTasksFlattened = (
    taskSections.flatMap(section => section.tasks) 
      .concat(tasks)
  )

  const completedCount = allTasksFlattened.filter(task => task.status === 'completed').length
  const totalCount = allTasksFlattened.length
  const progressRatioText = totalCount > 0 ? `${completedCount}/${totalCount} 完成` : undefined

  return (
    <div className="space-y-4">
      {(planText || toolName) && (
        <div className="rounded-lg border border-blue-200 bg-blue-50/80 p-4 shadow-sm">
          <div className="flex items-center space-x-2 text-blue-800 mb-2">
            <ListChecks className="h-4 w-4" />
            <span className="text-sm font-semibold">分析计划</span>
            <div className="flex items-center space-x-2 ml-auto">
              {progressRatioText && (
                <Badge variant="secondary" className="text-xs">
                  {progressRatioText}
                </Badge>
              )}
              {hasActiveTasks && (
                <Badge variant="secondary" className="text-xs bg-blue-100 text-blue-700">
                  <Loader2 className="h-3 w-3 animate-spin mr-1" />
                  执行中
                </Badge>
              )}
              {hasPendingTasks && !hasActiveTasks && completedCount < totalCount && (
                <Badge variant="secondary" className="text-xs">
                  待执行 {totalCount - completedCount}
                </Badge>
              )}
            </div>
          </div>
            {planText ? (
              <div className="prose prose-sm max-w-none text-blue-900">
                <ReactMarkdown 
                  remarkPlugins={[remarkGfm, remarkMath]}
                  components={{
                    p: ({ children }) => <p className="text-sm leading-relaxed mb-2 last:mb-0">{children}</p>,
                    strong: ({ children }) => <strong className="font-semibold text-blue-800">{children}</strong>
                  }}
                >
                  {planText}
                </ReactMarkdown>
              </div>
            ) : (
              <p className="text-xs text-blue-700">HolmesGPT 正在生成分析思路...</p>
            )}
            {toolName && (
              <Badge variant="outline" className="mt-3 border-blue-200 text-xs text-blue-700">
                使用工具：{toolName}
              </Badge>
            )}
            {statusText && (
              <p className="mt-2 text-xs text-blue-700/80">当前状态：{statusText}</p>
            )}
          </div>
        )}

        {taskSections.length > 0 ? (
          <div className="space-y-3">
            {taskSections.map(section => {
              const isOpen = openSections[section.id] ?? true
              const sectionActiveTasks = section.tasks.filter(task => task.status === 'in_progress').length
              const sectionCompletedTasks = section.tasks.filter(task => task.status === 'completed').length
              const sectionTotalTasks = section.tasks.length
                // 从工具调用中提取相关命令
                const relatedCommands = toolCalls
                  ?.filter(call => 
                    call.status === 'success' && 
                    (section.title?.includes(call.name || '') || 
                     call.name?.toLowerCase().includes('kubectl') ||
                     call.name?.toLowerCase().includes('describe') ||
                     call.name?.toLowerCase().includes('get'))
                  )
                  ?.map(call => {
                    // 优先使用预处理的命令字段
                    if (call.command) {
                      return call.command
                    }
                    if (typeof call.input === 'string') {
                      return call.input
                    } else if (call.input?.command) {
                      return call.input.command
                    } else if (call.input?.args) {
                      return `${call.name} ${call.input.args.join(' ')}`
                    }
                    return `${call.name} 调用`
                  }) || []

              return (
                <div key={section.id} className="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm">
                  <button
                    className="w-full flex items-center justify-between px-4 py-2 text-left text-sm font-semibold text-gray-700 bg-gray-50 hover:bg-gray-100 transition"
                    onClick={() => toggleSection(section.id)}
                  >
                    <div className="flex flex-col sm:flex-row sm:items-center sm:space-x-2 space-y-2 sm:space-y-0">
                      <div className="flex items-center space-x-2">
                        <ChevronDown
                          className={cn(
                            'h-4 w-4 transition-transform',
                            isOpen ? 'rotate-180' : 'rotate-0'
                          )}
                        />
                        <span>{section.title || `${section.toolName} 调用`}</span>
                        {sectionActiveTasks > 0 && (
                          <Badge variant="secondary" className="text-xs ml-2">
                            <Loader2 className="h-3 w-3 animate-spin mr-1" />
                            进行中 {sectionActiveTasks}
                          </Badge>
                        )}
                      </div>
                      {relatedCommands.length > 0 && (
                        <div className="text-xs text-gray-500 font-mono break-all">
                          {relatedCommands.join('；')}
                        </div>
                      )}
                    </div>
                    <div className="flex items-center space-x-2">
                      {section.statusText && (
                        <span className="text-xs text-gray-500">{section.statusText}</span>
                      )}
                      <span className="text-xs text-gray-500">
                        {sectionCompletedTasks}/{sectionTotalTasks} 完成
                      </span>
                    </div>
                  </button>
                  {isOpen && (
                    <div className="space-y-2">
                      {relatedCommands.length > 0 && (
                        <div className="px-4 pt-3 text-xs text-gray-500 space-y-1 border-b border-gray-100">
                          {relatedCommands.map((cmd, index) => (
                            <div key={`${section.id}-cmd-${index}`} className="font-mono break-all">
                              {cmd}
                            </div>
                          ))}
                        </div>
                      )}
                      {renderTaskList(section.tasks)}
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        ) : tasks.length > 0 ? (
          <div className="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm">
            <div className="border-b bg-gray-50 px-4 py-2 flex items-center justify-between">
              <span className="text-sm font-semibold text-gray-700">调查任务</span>
              <span className="text-xs text-gray-500">
                {tasks.filter(t => t.status === 'completed').length}/{tasks.length} 完成
              </span>
            </div>
            {renderTaskList(tasks)}
          </div>
        ) : null}

        {progressText && (
          <div className="rounded-lg border border-blue-200 bg-blue-50/80 p-4 shadow-sm">
            <div className="flex items-center space-x-2 text-blue-800 mb-3">
              <ListChecks className="h-5 w-5" />
              <span className="font-semibold text-base">排查进度</span>
            </div>
            <div className="prose prose-sm max-w-none text-blue-900">
              <ReactMarkdown 
                remarkPlugins={[remarkGfm, remarkMath]}
                components={{
                  p: ({ children }) => <p className="text-sm leading-relaxed mb-2 last:mb-0">{children}</p>,
                  strong: ({ children }) => <strong className="font-semibold text-blue-800">{children}</strong>,
                  code: ({ children }) => (
                    <code className="bg-blue-100 text-blue-800 px-1.5 py-0.5 rounded text-sm font-mono">
                      {children}
                    </code>
                  ),
                }}
              >
                {progressText}
              </ReactMarkdown>
            </div>
          </div>
        )}

        {summary && (
          <div className="rounded-lg border border-green-200 bg-green-50/80 p-4 shadow-sm">
            <div className="flex items-center space-x-2 text-green-800 mb-3">
              <CheckCircle2 className="h-5 w-5" />
              <span className="font-semibold text-base">分析结论</span>
            </div>
            <div className="prose prose-sm max-w-none text-green-900">
              <ReactMarkdown 
                remarkPlugins={[remarkGfm, remarkMath]}
                components={{
                  h1: ({ children }) => <h1 className="text-lg font-bold mb-3 text-green-800">{children}</h1>,
                  h2: ({ children }) => <h2 className="text-base font-semibold mb-2 text-green-700">{children}</h2>,
                  h3: ({ children }) => <h3 className="text-sm font-medium mb-2 text-green-600">{children}</h3>,
                  p: ({ children }) => <p className="text-sm leading-relaxed mb-2 last:mb-0">{children}</p>,
                  ul: ({ children }) => <ul className="list-disc list-inside space-y-1 mb-3 text-sm">{children}</ul>,
                  ol: ({ children }) => <ol className="list-decimal list-inside space-y-1 mb-3 text-sm">{children}</ol>,
                  strong: ({ children }) => <strong className="font-semibold text-green-800">{children}</strong>,
                  code: ({ children }) => (
                    <code className="bg-green-100 text-green-800 px-1.5 py-0.5 rounded text-sm font-mono">
                      {children}
                    </code>
                  ),
                  blockquote: ({ children }) => (
                    <blockquote className="border-l-4 border-green-300 pl-4 py-2 bg-green-100 rounded-r-lg mb-3">
                      {children}
                    </blockquote>
                  )
                }}
              >
                {summary}
              </ReactMarkdown>
            </div>
          </div>
        )}

        <div className="flex justify-end items-center space-x-2">
          {isAnalysisCopied && (
            <span className="text-xs text-green-600">已复制</span>
          )}
          <Button
            variant="ghost"
            size="icon"
            onClick={() => copyToClipboard(copyText)}
            className="h-8 w-8 text-gray-500 hover:text-gray-900 hover:bg-gray-100 transition-colors"
            title={isAnalysisCopied ? '已复制' : '复制分析内容'}
            aria-label="复制分析内容"
          >
            {isAnalysisCopied ? (
              <Check className="h-4 w-4 text-green-600" />
            ) : (
              <Copy className="h-4 w-4" />
            )}
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className={cn(
      "flex w-full py-4",
      role === 'user' ? '' : 'bg-gray-50/50'
    )}>
      <div className="relative flex w-full flex-col px-4">
        {/* 消息头部 */}
        <div className="flex items-center space-x-3 mb-3">
          <div className={cn(
            "flex h-8 w-8 items-center justify-center rounded-full",
            getRoleColor()
          )}>
            {getRoleIcon()}
          </div>
          <div className="flex items-center space-x-2">
            <span className="font-semibold text-gray-900">{getRoleName()}</span>
            {timestamp && (
              <span className="text-xs text-gray-500 flex items-center">
                <Clock className="h-3 w-3 mr-1" />
                {timestamp}
              </span>
            )}
            {isStreaming && (
              <Badge variant="secondary" className="text-xs">
                <div className="flex items-center space-x-1">
                  <div className="w-1.5 h-1.5 bg-blue-500 rounded-full animate-bounce"></div>
                  <div className="w-1.5 h-1.5 bg-blue-500 rounded-full animate-bounce" style={{ animationDelay: '0.1s' }}></div>
                  <div className="w-1.5 h-1.5 bg-blue-500 rounded-full animate-bounce" style={{ animationDelay: '0.2s' }}></div>
                  <span className="ml-2">正在分析</span>
                </div>
              </Badge>
            )}
          </div>
        </div>

        {/* 消息内容 */}
        <div className="ml-11 space-y-4">
          {structuredData ? (
            renderStructuredContent(structuredData)
          ) : jsonData ? (
            // 如果内容是 JSON，使用 JSON 查看器
            <JsonViewer
              data={jsonData}
              title="JSON 数据"
              maxHeight="400px"
            />
          ) : (
            // 正常的 Markdown 渲染
            <div className="prose prose-sm max-w-none dark:prose-invert">
              <ReactMarkdown
                remarkPlugins={[remarkGfm, remarkMath]}
                components={{
                  p({ children }) {
                    return <p className="mb-2 last:mb-0 leading-relaxed text-gray-700">{children}</p>
                  },
                  h1({ children }) {
                    return <h1 className="text-xl font-bold mb-3 text-gray-900 border-b border-gray-200 pb-2">{children}</h1>
                  },
                  h2({ children }) {
                    return <h2 className="text-lg font-semibold mb-2 text-gray-800 mt-4">{children}</h2>
                  },
                  h3({ children }) {
                    return <h3 className="text-base font-medium mb-2 text-gray-700 mt-3">{children}</h3>
                  },
                  ul({ children }) {
                    return <ul className="list-disc list-inside space-y-1 mb-3 pl-2">{children}</ul>
                  },
                  ol({ children }) {
                    return <ol className="list-decimal list-inside space-y-1 mb-3 pl-2">{children}</ol>
                  },
                  li({ children }) {
                    return <li className="text-gray-700 leading-relaxed">{children}</li>
                  },
                  blockquote({ children }) {
                    return (
                      <blockquote className="border-l-4 border-blue-200 pl-4 py-2 bg-blue-50 rounded-r-lg mb-3 italic">
                        {children}
                      </blockquote>
                    )
                  },
                  table({ children }) {
                    return (
                      <div className="overflow-x-auto mb-3">
                        <table className="min-w-full border border-gray-200 rounded-lg">
                          {children}
                        </table>
                      </div>
                    )
                  },
                  thead({ children }) {
                    return <thead className="bg-gray-50">{children}</thead>
                  },
                  th({ children }) {
                    return <th className="px-3 py-2 text-left text-xs font-semibold text-gray-700 border-b border-gray-200">{children}</th>
                  },
                  td({ children }) {
                    return <td className="px-3 py-2 text-sm text-gray-700 border-b border-gray-100">{children}</td>
                  },
                  strong({ children }) {
                    return <strong className="font-semibold text-gray-900">{children}</strong>
                  },
                  em({ children }) {
                    return <em className="italic text-gray-600">{children}</em>
                  },
                  code({ node, className, children, ...props }) {
                    const match = /language-(\w+)/.exec(className || '')
                    const language = match ? match[1] : ''
                    
                    if (language && typeof children === 'string' && children.includes('\n')) {
                      return (
                        <CodeBlock
                          language={language}
                          value={children.replace(/\n$/, '')}
                        />
                      )
                    }
                    
                    return (
                      <code 
                        className="bg-gray-100 text-gray-800 px-1.5 py-0.5 rounded text-sm font-mono border"
                        {...props}
                      >
                        {children}
                      </code>
                    )
                  }
                }}
              >
                {content}
              </ReactMarkdown>
            </div>
          )}

          {/* 工具调用结果显示 - 时间线风格，简洁展示 */}
          {toolCalls && toolCalls.length > 0 && (
            <div className="space-y-3">
              {toolCalls.map((tool, index) => (
                <div key={index} className="bg-gray-900 rounded-lg overflow-hidden shadow-md border border-gray-700">
                  {/* 工具头部 - Linux终端风格 */}
                  <div className="bg-gradient-to-r from-gray-800 to-gray-700 px-4 py-3 flex items-center justify-between border-b border-gray-600">
                    <div className="flex items-center space-x-3">
                      <div className="flex items-center space-x-1">
                        <div className="w-3 h-3 rounded-full bg-red-500"></div>
                        <div className="w-3 h-3 rounded-full bg-yellow-500"></div>
                        <div className="w-3 h-3 rounded-full bg-green-500"></div>
                      </div>
                      <span className="text-green-400 font-mono text-sm">$</span>
                      <span className="text-white text-sm font-medium">{tool.name}</span>
                    </div>
                    <Badge 
                      variant={tool.status === 'success' ? 'default' : tool.status === 'error' ? 'destructive' : 'secondary'}
                      className="text-xs font-mono"
                    >
                      {tool.status === 'success' ? '成功' : tool.status === 'error' ? '失败' : '执行中'}
                    </Badge>
                  </div>
                  
                  {/* 执行命令 */}
                  {(() => {
                    let command = tool.command || ''
                    if (!command) {
                      if (typeof tool.input === 'string') {
                        command = tool.input
                      } else if (tool.input?.command) {
                        command = tool.input.command
                      } else if (tool.input?.description) {
                        command = tool.input.description
                      } else if (tool.output?.result?.invocation) {
                        command = tool.output.result.invocation
                      } else if (tool.output?.invocation) {
                        command = tool.output.invocation
                      }
                    }
                    
                    return command ? (
                      <div className="px-4 py-3 border-b border-gray-700">
                        <div className="text-green-400 text-xs font-medium mb-2 flex items-center space-x-1">
                          <span>执行命令:</span>
                        </div>
                        <div className="bg-gray-800 rounded-md px-3 py-2 border border-gray-600">
                          <pre className="text-green-300 text-sm font-mono leading-relaxed whitespace-pre-wrap break-all">
                            {command}
                          </pre>
                        </div>
                      </div>
                    ) : null
                  })()}
                  
                  {/* 输出结果 */}
                  {(() => {
                    let output = ''
                    let returnCode = null
                    
                    if (tool.output?.result?.data) {
                      output = tool.output.result.data
                      returnCode = tool.output.result.return_code
                    } else if (tool.output?.data) {
                      output = tool.output.data
                    } else if (typeof tool.output === 'string') {
                      output = tool.output
                    } else if (tool.output) {
                      output = JSON.stringify(tool.output, null, 2)
                    }
                    
                    // 格式化输出内容，改善可读性
                    const formatOutput = (text: string): string => {
                      // 检测是否为YAML格式
                      if (text.includes('apiVersion:') || text.includes('kind:') || text.includes('metadata:')) {
                        // YAML格式：改善缩进和间距
                        const lines = text.split('\n')
                        const formattedLines: string[] = []
                        
                        for (let i = 0; i < lines.length; i++) {
                          const line = lines[i]
                          const nextLine = lines[i + 1]
                          
                          // 为顶级键（如apiVersion, kind, metadata等）前添加间距
                          if (line.match(/^[a-zA-Z][^:]*:/) && !line.startsWith('  ') && formattedLines.length > 0) {
                            formattedLines.push('')
                          }
                          
                          formattedLines.push(line)
                          
                          // 在metadata和spec等大段落后添加额外间距
                          if (line.match(/^(metadata|spec|status):\s*$/) && nextLine && nextLine.startsWith('  ')) {
                            // 不添加额外行，保持紧凑
                          }
                        }
                        
                        return formattedLines.join('\n')
                      }
                      
                      // 检测kubectl事件表格格式
                      if (text.includes('LAST SEEN') || text.includes('TYPE') || text.includes('REASON')) {
                        // 表格格式：保持原有对齐
                        return text
                      }
                      
                      // JSON格式：确保正确缩进
                      try {
                        const parsed = JSON.parse(text)
                        return JSON.stringify(parsed, null, 2)
                      } catch {
                        return text
                      }
                    }
                    
                    return output ? (
                      <div className="px-4 py-3">
                        <div className="flex items-center space-x-2 mb-3">
                          <div className="text-blue-400 text-xs font-medium">执行结果:</div>
                          {returnCode !== null && (
                            <div className={`text-xs px-2 py-0.5 rounded ${
                              returnCode === 0 
                                ? 'bg-green-600 text-green-100' 
                                : 'bg-red-600 text-red-100'
                            }`}>
                              退出码: {returnCode}
                            </div>
                          )}
                        </div>
                        <div className="bg-gray-950 rounded-md border border-gray-700 overflow-hidden">
                          <div className="bg-gray-800 px-3 py-1 border-b border-gray-700">
                            <span className="text-gray-400 text-xs font-mono">输出</span>
                          </div>
                          <pre className="text-gray-200 text-sm font-mono leading-relaxed p-4 whitespace-pre-wrap break-words">
                            {formatOutput(output)}
                          </pre>
                        </div>
                      </div>
                    ) : null
                  })()}
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

const buildStructuredCopyText = (data: HolmesStructuredData): string => {
  const sections: string[] = []

  if (data.planText) {
    sections.push(`分析计划:\n${data.planText}`)
  }

  const allSections = data.taskSections && data.taskSections.length
    ? data.taskSections
    : data.tasks && data.tasks.length
      ? [{ id: 'default', title: '调查任务', toolName: data.toolName || '', tasks: data.tasks, statusText: data.statusText }]
      : []

  if (allSections.length) {
    const blocks = allSections.map(section => {
      const header = section.title || section.toolName || '调查任务'
      const parts: string[] = [`${header}:`]
      if (section.commands && section.commands.length) {
        parts.push(section.commands.map(cmd => `  • ${cmd}`).join('\n'))
      }
      if (section.tasks && section.tasks.length) {
        const tasksText = section.tasks
          .map(task => `  - [${statusSymbol(task.status)}] ${task.content}`)
          .join('\n')
        parts.push(tasksText)
      }
      return parts.join('\n')
    })
    sections.push(blocks.join('\n\n'))
  }

  if (data.progressText) {
    sections.push(`排查进度:\n${data.progressText}`)
  }

  if (data.summary) {
    sections.push(`最终结论:\n${data.summary}`)
  }

  return sections.join('\n\n').trim() || 'HolmesGPT 分析中'
}

const statusSymbol = (status: HolmesTaskItem['status']) => {
  switch (status) {
    case 'completed':
      return '完成'
    case 'in_progress':
      return '进行中'
    default:
      return '待处理'
  }
}
