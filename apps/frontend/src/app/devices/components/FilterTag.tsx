'use client'

import { X } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface FilterTagProps {
    label: string
    value: string
    onRemove: () => void
    variant?: 'default' | 'filter' | 'template'
}

export function FilterTag({ label, value, onRemove, variant = 'default' }: FilterTagProps) {
    return (
        <Badge
            variant="secondary"
            className={cn(
                "gap-1 pr-1 font-normal",
                variant === 'filter' && "bg-blue-50 text-blue-700 border-blue-200",
                variant === 'template' && "bg-purple-50 text-purple-700 border-purple-200"
            )}
        >
            <span className="text-muted-foreground">{label}:</span>
            <span className="font-medium">{value}</span>
            <button
                onClick={(e) => { e.stopPropagation(); onRemove(); }}
                className="ml-1 h-3.5 w-3.5 rounded-full hover:bg-black/10 dark:hover:bg-white/10 flex items-center justify-center transition-colors"
            >
                <X className="h-2.5 w-2.5" />
            </button>
        </Badge>
    )
}

interface ActiveFiltersProps {
    keyword?: string
    onlySpecial?: boolean
    filterCount?: number
    templateName?: string
    onClearKeyword: () => void
    onClearSpecial: () => void
    onClearFilters: () => void
    onClearTemplate: () => void
    onClearAll: () => void
}

export function ActiveFilters({
    keyword,
    onlySpecial,
    filterCount,
    templateName,
    onClearKeyword,
    onClearSpecial,
    onClearFilters,
    onClearTemplate,
    onClearAll,
}: ActiveFiltersProps) {
    const hasAnyFilter = keyword || onlySpecial || (filterCount && filterCount > 0) || templateName

    if (!hasAnyFilter) return null

    return (
        <div className="flex items-center gap-2 flex-wrap">
            <span className="text-xs text-muted-foreground">已应用:</span>

            {keyword && (
                <FilterTag
                    label="搜索"
                    value={keyword.length > 20 ? keyword.slice(0, 20) + '...' : keyword}
                    onRemove={onClearKeyword}
                />
            )}

            {onlySpecial && (
                <FilterTag
                    label="筛选"
                    value="仅特殊设备"
                    onRemove={onClearSpecial}
                    variant="filter"
                />
            )}

            {filterCount && filterCount > 0 && (
                <FilterTag
                    label="高级"
                    value={`${filterCount} 个条件组`}
                    onRemove={onClearFilters}
                    variant="filter"
                />
            )}

            {templateName && (
                <FilterTag
                    label="模板"
                    value={templateName}
                    onRemove={onClearTemplate}
                    variant="template"
                />
            )}

            <Button
                variant="ghost"
                size="sm"
                className="h-6 px-2 text-xs text-muted-foreground"
                onClick={onClearAll}
            >
                清除全部
            </Button>
        </div>
    )
}
