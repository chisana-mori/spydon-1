'use client'

import React, { useState, useCallback, useRef } from 'react'
import { Search, RotateCcw, Star } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Textarea } from '@/components/ui/textarea'
import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from "@/components/ui/tooltip"
import { cn } from '@/lib/utils'

interface SimpleQueryPanelProps {
    onSearch: (keyword: string, isBatch: boolean) => void
    isLoading?: boolean
    className?: string
    showSpecial?: boolean
    onToggleSpecial?: () => void
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

export function SimpleQueryPanel({ onSearch, isLoading, className, showSpecial, onToggleSpecial }: SimpleQueryPanelProps) {
    const [value, setValue] = useState('')
    const [lastSearchedValue, setLastSearchedValue] = useState('') // To show current active keywords

    // Parse currently entered text (for preview if needed) or just use last searched for the "Keywords" display
    // The image shows "Keywords: (Empty)" which likely reflects the *applied* filter, so we track lastSearchedValue.
    const activeKeywords = parseMultilineInput(lastSearchedValue)

    const handleSearch = useCallback(() => {
        const parsed = parseMultilineInput(value)
        const isBatch = parsed.length > 1
        const normalizedKeyword = parsed.join('\n')

        setLastSearchedValue(value)
        onSearch(normalizedKeyword, isBatch)
    }, [value, onSearch])

    const handleReset = useCallback(() => {
        setValue('')
        setLastSearchedValue('')
        onSearch('', false)
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
                    placeholder="输入关键字搜索设备ID、IP、集群等字段 支持多行查询，每一行一个条件，多个条件之间是 OR 关系 例如：192.168.1.1 192.168.1.2"
                    className="min-h-[120px] resize-y text-base p-4 w-full"
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
                            className="gap-2 bg-background hover:bg-muted"
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
                        {onToggleSpecial && (
                            <TooltipProvider>
                                <Tooltip delayDuration={300}>
                                    <TooltipTrigger asChild>
                                        <Button
                                            variant="ghost"
                                            size="sm"
                                            onClick={onToggleSpecial}
                                            className={cn("gap-1.5 ml-2 font-medium", showSpecial ? "text-amber-600" : "text-muted-foreground")}
                                        >
                                            <Star className={cn("h-4 w-4", showSpecial && "fill-current")} />
                                            特殊设备
                                        </Button>
                                    </TooltipTrigger>
                                    <TooltipContent side="bottom" className="bg-card text-card-foreground border shadow-lg p-3 max-w-[260px]">
                                        <div className="space-y-1.5 text-left">
                                            <p className="font-semibold text-foreground text-sm">筛选特殊设备</p>
                                            <p className="text-muted-foreground text-xs leading-relaxed">
                                                开启后仅显示被标记为"特殊"的设备。这些通常是核心节点、故障排查对象或需重点关注的资产。
                                            </p>
                                        </div>
                                    </TooltipContent>
                                </Tooltip>
                            </TooltipProvider>
                        )}
                    </div>

                    <div className="text-xs text-muted-foreground flex items-center gap-1">
                        <span className="inline-block w-3 h-3 rounded-full border border-muted-foreground/30 text-center leading-[10px] text-[8px] mr-1">i</span>
                        提示：多行查询时，每一行一个条件，多个条件之间是 OR 关系。可以使用 Ctrl+Enter 快捷键执行查询。
                    </div>
                </div>
            </div>

            {/* Keywords Display Area */}
            <div className="flex items-start gap-2 py-2 border-t border-border/40">
                <span className="text-sm font-medium text-muted-foreground shrink-0 mt-1">关键字:</span>
                <div className="flex flex-wrap gap-2">
                    {activeKeywords.length > 0 ? (
                        activeKeywords.map((k, i) => (
                            <Badge key={i} variant="secondary" className="font-normal bg-secondary/50 px-2 py-0.5 h-6">
                                {k}
                            </Badge>
                        ))
                    ) : (
                        <span className="text-sm text-muted-foreground/50 mt-0.5 group-hover:text-muted-foreground transition-colors">(空)</span>
                    )}
                </div>
            </div>
        </div>
    )
}
