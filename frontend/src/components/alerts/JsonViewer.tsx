'use client'

import React, { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { 
  ChevronDown, 
  ChevronRight, 
  Copy, 
  Eye, 
  EyeOff,
  Braces,
  Hash,
  Quote,
  CheckCircle2,
  XCircle,
  Clock
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'

interface JsonViewerProps {
  data: any
  title?: string
  className?: string
  maxHeight?: string
}

interface JsonNodeProps {
  data: any
  keyName?: string
  level?: number
  isLast?: boolean
}

const JsonNode: React.FC<JsonNodeProps> = ({ data, keyName, level = 0, isLast = true }) => {
  const [isExpanded, setIsExpanded] = useState(level < 2) // 默认展开前两层
  
  const getValueType = (value: any): string => {
    if (value === null) return 'null'
    if (typeof value === 'boolean') return 'boolean'
    if (typeof value === 'number') return 'number'
    if (typeof value === 'string') return 'string'
    if (Array.isArray(value)) return 'array'
    if (typeof value === 'object') return 'object'
    return 'unknown'
  }

  const getValueIcon = (type: string) => {
    switch (type) {
      case 'object':
        return <Braces className="h-3 w-3 text-blue-500" />
      case 'array':
        return <Hash className="h-3 w-3 text-green-500" />
      case 'string':
        return <Quote className="h-3 w-3 text-yellow-500" />
      case 'number':
        return <Hash className="h-3 w-3 text-purple-500" />
      case 'boolean':
        return data ? <CheckCircle2 className="h-3 w-3 text-green-500" /> : <XCircle className="h-3 w-3 text-red-500" />
      case 'null':
        return <Clock className="h-3 w-3 text-gray-400" />
      default:
        return null
    }
  }

  const getValueColor = (type: string): string => {
    switch (type) {
      case 'string':
        return 'text-green-600'
      case 'number':
        return 'text-blue-600'
      case 'boolean':
        return 'text-purple-600'
      case 'null':
        return 'text-gray-400'
      default:
        return 'text-gray-800'
    }
  }

  const formatValue = (value: any, type: string): string => {
    switch (type) {
      case 'string':
        return `"${value}"`
      case 'null':
        return 'null'
      case 'boolean':
        return value.toString()
      case 'number':
        return value.toString()
      default:
        return String(value)
    }
  }

  const renderValue = (value: any, type: string) => {
    if (type === 'object' && value !== null) {
      const keys = Object.keys(value)
      return (
        <div>
          <div 
            className="flex items-center cursor-pointer hover:bg-gray-50 rounded px-1 py-0.5"
            onClick={() => setIsExpanded(!isExpanded)}
          >
            {isExpanded ? <ChevronDown className="h-3 w-3 mr-1" /> : <ChevronRight className="h-3 w-3 mr-1" />}
            {getValueIcon(type)}
            <span className="ml-1 text-sm font-medium text-gray-700">
              {keyName && <span className="text-blue-700">{keyName}: </span>}
              <span className="text-gray-500">{'{'}</span>
              <span className="text-xs text-gray-400 ml-1">{keys.length} keys</span>
              <span className="text-gray-500">{'}'}</span>
            </span>
          </div>
          {isExpanded && (
            <div className="ml-4 border-l border-gray-200 pl-2 mt-1">
              {keys.map((key, index) => (
                <JsonNode
                  key={key}
                  data={value[key]}
                  keyName={key}
                  level={level + 1}
                  isLast={index === keys.length - 1}
                />
              ))}
            </div>
          )}
        </div>
      )
    }

    if (type === 'array') {
      return (
        <div>
          <div 
            className="flex items-center cursor-pointer hover:bg-gray-50 rounded px-1 py-0.5"
            onClick={() => setIsExpanded(!isExpanded)}
          >
            {isExpanded ? <ChevronDown className="h-3 w-3 mr-1" /> : <ChevronRight className="h-3 w-3 mr-1" />}
            {getValueIcon(type)}
            <span className="ml-1 text-sm font-medium text-gray-700">
              {keyName && <span className="text-blue-700">{keyName}: </span>}
              <span className="text-gray-500">{'['}</span>
              <span className="text-xs text-gray-400 ml-1">{value.length} items</span>
              <span className="text-gray-500">{']'}</span>
            </span>
          </div>
          {isExpanded && (
            <div className="ml-4 border-l border-gray-200 pl-2 mt-1">
              {value.map((item: any, index: number) => (
                <JsonNode
                  key={index}
                  data={item}
                  keyName={`[${index}]`}
                  level={level + 1}
                  isLast={index === value.length - 1}
                />
              ))}
            </div>
          )}
        </div>
      )
    }

    // 基本类型
    return (
      <div className="flex items-center py-0.5">
        {getValueIcon(type)}
        <span className="ml-1 text-sm">
          {keyName && <span className="text-blue-700 font-medium">{keyName}: </span>}
          <span className={cn("font-mono", getValueColor(type))}>
            {formatValue(value, type)}
          </span>
        </span>
      </div>
    )
  }

  const valueType = getValueType(data)
  return renderValue(data, valueType)
}

export const JsonViewer: React.FC<JsonViewerProps> = ({ 
  data, 
  title = "JSON 数据", 
  className,
  maxHeight = "400px"
}) => {
  const [isCollapsed, setIsCollapsed] = useState(false)
  const [showRaw, setShowRaw] = useState(false)
  const { copyToClipboard } = useCopyToClipboard({ timeout: 2000 })

  const handleCopy = () => {
    copyToClipboard(JSON.stringify(data, null, 2))
  }

  // 尝试解析字符串为 JSON
  let parsedData = data
  if (typeof data === 'string') {
    try {
      parsedData = JSON.parse(data)
    } catch {
      // 如果解析失败，保持原始字符串
      parsedData = data
    }
  }

  return (
    <Card className={cn("border border-gray-200", className)}>
      <div className="flex items-center justify-between p-3 bg-gray-50 border-b">
        <div className="flex items-center space-x-2">
          <Braces className="h-4 w-4 text-blue-500" />
          <span className="font-medium text-sm text-gray-700">{title}</span>
          <Badge variant="secondary" className="text-xs">
            {typeof parsedData === 'object' && parsedData !== null
              ? Array.isArray(parsedData) 
                ? `${parsedData.length} items`
                : `${Object.keys(parsedData).length} keys`
              : typeof parsedData
            }
          </Badge>
        </div>
        <div className="flex items-center space-x-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setShowRaw(!showRaw)}
            className="h-7 px-2"
          >
            {showRaw ? <EyeOff className="h-3 w-3" /> : <Eye className="h-3 w-3" />}
            <span className="ml-1 text-xs">{showRaw ? '结构化' : '原始'}</span>
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setIsCollapsed(!isCollapsed)}
            className="h-7 px-2"
          >
            {isCollapsed ? <ChevronRight className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleCopy}
            className="h-7 px-2"
          >
            <Copy className="h-3 w-3" />
          </Button>
        </div>
      </div>
      
      {!isCollapsed && (
        <CardContent className="p-0">
          <div 
            className="overflow-auto p-3 bg-white"
            style={{ maxHeight }}
          >
            {showRaw ? (
              <pre className="text-xs font-mono text-gray-700 whitespace-pre-wrap break-words">
                {JSON.stringify(parsedData, null, 2)}
              </pre>
            ) : (
              <div className="text-sm">
                <JsonNode data={parsedData} />
              </div>
            )}
          </div>
        </CardContent>
      )}
    </Card>
  )
}
