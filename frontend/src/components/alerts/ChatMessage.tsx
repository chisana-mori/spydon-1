'use client'

import React, { FC } from 'react'
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
  Terminal,
  Clock,
  AlertTriangle
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { JsonViewer } from './JsonViewer'

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
  }>
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
  sql: '.sql',
  html: '.html',
  css: '.css',
  yaml: '.yaml',
  json: '.json',
  xml: '.xml'
}

const CodeBlock: FC<CodeBlockProps> = ({ language, value }) => {
  const { isCopied, copyToClipboard } = useCopyToClipboard({ timeout: 2000 })

  const downloadAsFile = () => {
    if (typeof window === 'undefined') return
    
    const fileExtension = programmingLanguages[language] || '.txt'
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
    <div className="codeblock relative w-full bg-gray-950 font-mono rounded-lg overflow-hidden">
      <div className="flex w-full items-center justify-between bg-gray-800 px-4 py-2 text-white">
        <span className="text-xs font-medium">{language || 'text'}</span>
        <div className="flex items-center space-x-1">
          <Button
            variant="ghost"
            size="sm"
            className="h-8 w-8 p-0 hover:bg-gray-700"
            onClick={downloadAsFile}
          >
            <Download className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="h-8 w-8 p-0 hover:bg-gray-700"
            onClick={onCopy}
          >
            {isCopied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
          </Button>
        </div>
      </div>
      <SyntaxHighlighter
        language={language}
        style={oneDark}
        customStyle={{
          margin: 0,
          width: '100%',
          background: 'transparent',
          fontSize: '14px'
        }}
      >
        {value}
      </SyntaxHighlighter>
    </div>
  )
}

export const ChatMessage: FC<ChatMessageProps> = ({
  role,
  content,
  timestamp,
  isStreaming = false,
  toolCalls = []
}) => {
  const { copyToClipboard } = useCopyToClipboard({ timeout: 2000 })

  const handleCopy = () => {
    copyToClipboard(content)
  }

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

  const jsonData = isJsonContent(content) ? parseJsonSafely(content) : null

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
          {jsonData ? (
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
                    return <p className="mb-2 last:mb-0 leading-relaxed">{children}</p>
                  },
                  h1({ children }) {
                    return <h1 className="text-xl font-bold mb-3 text-gray-900">{children}</h1>
                  },
                  h2({ children }) {
                    return <h2 className="text-lg font-semibold mb-2 text-gray-800">{children}</h2>
                  },
                  h3({ children }) {
                    return <h3 className="text-base font-medium mb-2 text-gray-700">{children}</h3>
                  },
                  ul({ children }) {
                    return <ul className="list-disc list-inside space-y-1 mb-3">{children}</ul>
                  },
                  ol({ children }) {
                    return <ol className="list-decimal list-inside space-y-1 mb-3">{children}</ol>
                  },
                  li({ children }) {
                    return <li className="text-gray-700">{children}</li>
                  },
                  blockquote({ children }) {
                    return (
                      <blockquote className="border-l-4 border-blue-200 pl-4 py-2 bg-blue-50 rounded-r-lg mb-3">
                        {children}
                      </blockquote>
                    )
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
                        className="bg-gray-100 text-gray-800 px-1.5 py-0.5 rounded text-sm font-mono"
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
        </div>
      </div>
    </div>
  )
}
