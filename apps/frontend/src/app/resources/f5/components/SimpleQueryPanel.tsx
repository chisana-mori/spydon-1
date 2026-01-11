'use client'

import React, { useState, useCallback } from 'react'
import { Search, RotateCcw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'

interface SimpleQueryPanelProps {
    onSearch: (keyword: string) => void
    isLoading?: boolean
    className?: string
}

/**
 * Parses multi-line input supporting various separators
 */
function parseMultilineInput(input: string): string[] {
    if (!input.trim()) return []
    let normalized = input.replace(/\r\n/g, '\n').replace(/\r/g, '\n')
    normalized = normalized.replace(/，/g, ',').replace(/；/g, ';')
    const lines = normalized.split(/[\n,;]+/)
    return lines.map(line => line.trim()).filter(line => line.length > 0)
}

export function SimpleQueryPanel({ onSearch, isLoading, className }: SimpleQueryPanelProps) {
    const [value, setValue] = useState('')
    const [lastSearchedValue, setLastSearchedValue] = useState('')

    const activeKeywords = parseMultilineInput(lastSearchedValue)

    const handleSearch = useCallback(() => {
        const parsed = parseMultilineInput(value)
        const normalizedKeyword = parsed.join('\n')

        setLastSearchedValue(value)
        onSearch(normalizedKeyword)
    }, [value, onSearch])

    const handleReset = useCallback(() => {
        setValue('')
        setLastSearchedValue('')
        onSearch('')
    }, [onSearch])

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
            e.preventDefault()
            handleSearch()
        }
    }

    return (
        <div className={cn("space-y-4", className)}>
            {/* Input Area Group */}
            <div className="space-y-2">
                <Textarea
                    placeholder="输入关键字搜索 F5 名称、VIP、AppID、Pool成员、集群名称等字段。支持多行查询，每一行一个条件，多个条件之间是 OR 关系。&#10;例如：&#10;192.168.1.100&#10;test-app&#10;prod-cluster"
                    className="min-h-[120px] resize-y text-base p-4 w-full border-slate-200/80 focus-visible:ring-blue-500/50 dark:border-slate-700/50 dark:focus-visible:ring-blue-400/50"
                    value={value}
                    onChange={(e) => setValue(e.target.value)}
                    onKeyDown={handleKeyDown}
                />

                {/* Actions Row */}
                <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                        <Button
                            variant="outline"
                            onClick={handleReset}
                            className="gap-2 bg-background hover:bg-slate-50/80 dark:hover:bg-slate-800/30 border-slate-200/80 dark:border-slate-700/50 transition-all"
                        >
                            <RotateCcw className="h-4 w-4" />
                            重置
                        </Button>
                        <Button
                            onClick={handleSearch}
                            disabled={isLoading}
                            className="gap-2"
                        >
                            <Search className="h-4 w-4" />
                            搜索
                        </Button>
                    </div>

                    <div className="text-xs text-muted-foreground flex items-center gap-1">
                        <span className="inline-block w-3 h-3 rounded-full border border-slate-300/50 text-center leading-[10px] text-[8px] mr-1">i</span>
                        提示：多行查询时，每一行一个条件，多个条件之间是 OR 关系。可以使用 Ctrl+Enter 快捷键执行查询。
                    </div>
                </div>
            </div>

            {/* Keywords Display Area */}
            <div className="flex items-start gap-2 py-2 border-t border-slate-200/50 dark:border-slate-700/50">
                <span className="text-sm font-medium text-muted-foreground shrink-0 mt-1">关键字:</span>
                <div className="flex flex-wrap gap-2">
                    {activeKeywords.length > 0 ? (
                        activeKeywords.map((k, i) => (
                            <Badge key={i} variant="secondary" className="font-normal bg-blue-50/80 text-blue-700 border-blue-200/50 dark:bg-blue-950/30 dark:text-blue-400 dark:border-blue-800/50 px-2 py-0.5 h-6">
                                {k}
                            </Badge>
                        ))
                    ) : (
                        <span className="text-sm text-muted-foreground/50 mt-0.5">(空)</span>
                    )}
                </div>
            </div>
        </div>
    )
}
