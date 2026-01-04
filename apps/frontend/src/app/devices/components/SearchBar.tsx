'use client'

import { useState, useRef, useEffect, useCallback } from 'react'
import { Search, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface SearchBarProps {
    onSearch: (keyword: string, isBatch: boolean) => void
    isLoading?: boolean
    className?: string
}

/**
 * 解析多行输入，支持多种分隔符
 * - 换行符 (\n, \r\n)
 * - 逗号 (中英文)
 * - 分号 (中英文)
 */
function parseMultilineInput(input: string): string[] {
    if (!input.trim()) return []

    // 统一换行符
    let normalized = input.replace(/\r\n/g, '\n').replace(/\r/g, '\n')

    // 替换中文标点
    normalized = normalized.replace(/，/g, ',').replace(/；/g, ';')

    // 按换行符、逗号、分号分割
    const lines = normalized.split(/[\n,;]+/)

    // 清理每行并过滤空行
    const result = lines
        .map(line => line.trim())
        .filter(line => line.length > 0)

    return result
}

export function SearchBar({ onSearch, isLoading, className }: SearchBarProps) {
    const [value, setValue] = useState('')
    const textareaRef = useRef<HTMLTextAreaElement>(null)

    // 解析后的行数
    const parsedLines = parseMultilineInput(value)
    const lineCount = parsedLines.length
    const isBatch = lineCount > 1

    const handleSearch = useCallback(() => {
        if (!value.trim()) return
        // 将解析后的行重新组合成换行符分隔的字符串
        const normalizedKeyword = parsedLines.join('\n')
        onSearch(normalizedKeyword, isBatch)
    }, [value, parsedLines, isBatch, onSearch])

    const handleKeyDown = (e: React.KeyboardEvent) => {
        // Ctrl+Enter 执行搜索
        if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
            e.preventDefault()
            handleSearch()
        }
        // 单行时 Enter 也执行搜索
        if (e.key === 'Enter' && !e.shiftKey && !isBatch && !e.ctrlKey && !e.metaKey) {
            e.preventDefault()
            handleSearch()
        }
        // Escape 清空
        if (e.key === 'Escape') {
            setValue('')
            textareaRef.current?.focus()
        }
    }

    const handleClear = () => {
        setValue('')
        onSearch('', false)
        textareaRef.current?.focus()
    }

    // 自动调整高度
    useEffect(() => {
        const textarea = textareaRef.current
        if (textarea) {
            // 重置高度以获取正确的 scrollHeight
            textarea.style.height = 'auto'
            // 限制最大高度
            const maxHeight = 150
            const newHeight = Math.min(textarea.scrollHeight, maxHeight)
            textarea.style.height = `${Math.max(40, newHeight)}px`
        }
    }, [value])

    return (
        <div className={cn("relative", className)}>
            <div className="flex items-start gap-3">
                {/* 搜索框 */}
                <div className="flex-1 relative rounded-lg border bg-background shadow-sm transition-all focus-within:ring-2 focus-within:ring-ring focus-within:border-primary">
                    {/* 搜索图标 */}
                    <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground pointer-events-none" />

                    {/* 输入区域 */}
                    <textarea
                        ref={textareaRef}
                        placeholder="粘贴多个 IP 或设备编码，每行一个..."
                        className="w-full min-h-[40px] resize-none bg-transparent py-2.5 pl-9 pr-20 text-sm leading-relaxed placeholder:text-muted-foreground focus:outline-none"
                        value={value}
                        onChange={(e) => setValue(e.target.value)}
                        onKeyDown={handleKeyDown}
                        rows={1}
                    />

                    {/* 右侧操作区 - 绝对定位 */}
                    <div className="absolute right-2 top-1.5 flex items-center gap-1.5">
                        {/* 行数显示 */}
                        {lineCount > 0 && (
                            <span className="text-xs text-muted-foreground font-mono tabular-nums">
                                {lineCount} 行
                            </span>
                        )}

                        {/* 清除按钮 */}
                        {value && (
                            <button
                                type="button"
                                className="p-1 text-muted-foreground hover:text-foreground rounded transition-colors"
                                onClick={handleClear}
                            >
                                <X className="h-4 w-4" />
                            </button>
                        )}

                        {/* 搜索按钮 */}
                        <Button
                            size="sm"
                            className="h-7 px-3"
                            onClick={handleSearch}
                            disabled={isLoading || !value.trim()}
                        >
                            搜索
                        </Button>
                    </div>
                </div>
            </div>

            {/* 提示信息 */}
            <p className="text-[11px] text-muted-foreground mt-1.5 ml-0.5 flex items-center gap-2">
                <kbd className="inline-flex items-center rounded border bg-muted px-1.5 py-0.5 font-mono text-[10px] font-medium text-muted-foreground">
                    Ctrl+Enter
                </kbd>
                <span>执行搜索</span>
            </p>
        </div>
    )
}
