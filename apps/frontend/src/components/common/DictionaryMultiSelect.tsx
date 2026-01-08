"use client"

import * as React from "react"
import { Check, ChevronsUpDown, Loader2, Search, X } from "lucide-react"
import { useQuery } from "@tanstack/react-query"

import { cn } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"
import { getDictionaryItemsByCode } from "@/lib/api/dictionaries"

interface DictionaryMultiSelectProps {
    code: string
    value?: string // 逗号分隔的 keys，如 "APLUS,BPLUS"
    onValueChange: (value: string) => void
    placeholder?: string
    disabled?: boolean
    className?: string
}

/**
 * DictionaryMultiSelect - 字典多选组件
 *
 * 支持多选，选中的值用逗号拼接存储
 * 使用原生 DOM 实现，避免 Radix Popover + cmdk 在 Dialog 内的焦点陷阱冲突问题。
 */
export function DictionaryMultiSelect({
    code,
    value,
    onValueChange,
    placeholder = "请选择...",
    disabled = false,
    className,
}: DictionaryMultiSelectProps) {
    const [open, setOpen] = React.useState(false)
    const [search, setSearch] = React.useState("")
    const [highlightedIndex, setHighlightedIndex] = React.useState(0)

    const containerRef = React.useRef<HTMLDivElement>(null)
    const inputRef = React.useRef<HTMLInputElement>(null)
    const listRef = React.useRef<HTMLDivElement>(null)

    const { data: items = [], isLoading } = useQuery({
        queryKey: ["dictionary", code],
        queryFn: () => getDictionaryItemsByCode(code),
        staleTime: 1000 * 60 * 5,
    })

    // 解析选中的值
    const selectedKeys = React.useMemo(() => {
        if (!value) return []
        return value.split(',').filter(Boolean)
    }, [value])

    const selectedItems = React.useMemo(() => {
        return items.filter((item: any) => selectedKeys.includes(item.key))
    }, [items, selectedKeys])

    // 过滤后的选项
    const filteredItems = React.useMemo(() => {
        if (!search.trim()) return items
        const lowerSearch = search.toLowerCase()
        return items.filter((item: any) =>
            item.key.toLowerCase().includes(lowerSearch) ||
            item.value.toLowerCase().includes(lowerSearch)
        )
    }, [items, search])

    // 重置高亮索引
    React.useEffect(() => {
        setHighlightedIndex(0)
    }, [filteredItems])

    // 点击外部关闭
    React.useEffect(() => {
        if (!open) return

        const handleClickOutside = (e: MouseEvent) => {
            if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
                setOpen(false)
                setSearch("")
            }
        }

        document.addEventListener("mousedown", handleClickOutside)
        return () => document.removeEventListener("mousedown", handleClickOutside)
    }, [open])

    // 打开时聚焦搜索框
    React.useEffect(() => {
        if (open && inputRef.current) {
            setTimeout(() => inputRef.current?.focus(), 10)
        }
    }, [open])

    // 键盘导航
    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (!open) {
            if (e.key === "Enter" || e.key === " " || e.key === "ArrowDown") {
                e.preventDefault()
                setOpen(true)
            }
            return
        }

        switch (e.key) {
            case "ArrowDown":
                e.preventDefault()
                setHighlightedIndex((prev) =>
                    prev < filteredItems.length - 1 ? prev + 1 : prev
                )
                break
            case "ArrowUp":
                e.preventDefault()
                setHighlightedIndex((prev) => (prev > 0 ? prev - 1 : 0))
                break
            case "Enter":
                e.preventDefault()
                if (filteredItems[highlightedIndex]) {
                    handleToggle(filteredItems[highlightedIndex].key)
                }
                break
            case "Escape":
                e.preventDefault()
                setOpen(false)
                setSearch("")
                break
            case "Tab":
                setOpen(false)
                setSearch("")
                break
        }
    }

    const handleToggle = (key: string) => {
        const newSelectedKeys = selectedKeys.includes(key)
            ? selectedKeys.filter(k => k !== key)
            : [...selectedKeys, key]

        onValueChange(newSelectedKeys.join(','))
    }

    const handleRemove = (key: string, e: React.MouseEvent) => {
        e.preventDefault()
        e.stopPropagation()
        const newSelectedKeys = selectedKeys.filter(k => k !== key)
        onValueChange(newSelectedKeys.join(','))
    }

    const handleTriggerClick = (e: React.MouseEvent) => {
        e.preventDefault()
        e.stopPropagation()
        if (!disabled) {
            setOpen((prev) => !prev)
        }
    }

    // 滚动高亮项到可见区域
    React.useEffect(() => {
        if (open && listRef.current) {
            const highlightedEl = listRef.current.querySelector(`[data-index="${highlightedIndex}"]`)
            if (highlightedEl) {
                highlightedEl.scrollIntoView({ block: "nearest" })
            }
        }
    }, [highlightedIndex, open])

    if (isLoading) {
        return (
            <div className={cn("h-9 w-full flex items-center gap-2 px-3 border rounded-md bg-background", className)}>
                <Loader2 className="h-4 w-4 animate-spin text-blue-500" />
                <span className="text-muted-foreground text-sm">加载中...</span>
            </div>
        )
    }

    return (
        <div
            ref={containerRef}
            className="relative w-full"
            onKeyDown={handleKeyDown}
        >
            {/* Trigger Button */}
            <button
                type="button"
                role="combobox"
                aria-expanded={open}
                aria-haspopup="listbox"
                onClick={handleTriggerClick}
                disabled={disabled}
                className={cn(
                    "min-h-9 w-full flex items-center justify-between px-3 py-2",
                    "border rounded-md bg-background text-sm",
                    "hover:bg-accent hover:text-accent-foreground",
                    "focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2",
                    "disabled:cursor-not-allowed disabled:opacity-50",
                    !value && "text-muted-foreground",
                    className
                )}
            >
                {selectedItems.length > 0 ? (
                    <div className="flex items-center gap-1 flex-wrap overflow-hidden">
                        {selectedItems.map((item: any) => (
                            <Badge
                                key={item.key}
                                variant="secondary"
                                className="shrink-0 bg-blue-500/10 text-blue-600 border-blue-500/20 flex items-center gap-1"
                            >
                                {item.key}
                                <X
                                    className="h-3 w-3 cursor-pointer hover:text-blue-800"
                                    onClick={(e) => handleRemove(item.key, e)}
                                />
                            </Badge>
                        ))}
                    </div>
                ) : (
                    <span>{placeholder}</span>
                )}
                <ChevronsUpDown className="ml-auto h-4 w-4 shrink-0 opacity-50" />
            </button>

            {/* Dropdown */}
            {open && (
                <div
                    className={cn(
                        "absolute top-full left-0 w-full mt-1 z-[9999]",
                        "bg-popover border rounded-md shadow-md",
                        "animate-in fade-in-0 zoom-in-95 duration-100"
                    )}
                    onMouseDown={(e) => {
                        e.stopPropagation()
                    }}
                >
                    {/* Search Input */}
                    <div className="flex items-center border-b px-3">
                        <Search className="h-4 w-4 shrink-0 opacity-50 mr-2" />
                        <input
                            ref={inputRef}
                            type="text"
                            placeholder="搜索..."
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                            className={cn(
                                "flex h-10 w-full bg-transparent py-3 text-sm outline-none",
                                "placeholder:text-muted-foreground"
                            )}
                            onMouseDown={(e) => e.stopPropagation()}
                        />
                        {search && (
                            <button
                                type="button"
                                onClick={() => setSearch("")}
                                className="h-4 w-4 shrink-0 opacity-50 hover:opacity-100"
                            >
                                <X className="h-4 w-4" />
                            </button>
                        )}
                    </div>

                    {/* Options List */}
                    <div
                        ref={listRef}
                        role="listbox"
                        className="max-h-[200px] overflow-y-auto p-1"
                    >
                        {filteredItems.length === 0 ? (
                            <div className="py-6 text-center text-sm text-muted-foreground">
                                未找到匹配项
                            </div>
                        ) : (
                            filteredItems.map((item: any, index: number) => {
                                const isSelected = selectedKeys.includes(item.key)
                                return (
                                    <div
                                        key={item.key}
                                        data-index={index}
                                        role="option"
                                        aria-selected={isSelected}
                                        onClick={(e) => {
                                            e.preventDefault()
                                            e.stopPropagation()
                                            handleToggle(item.key)
                                        }}
                                        onMouseEnter={() => setHighlightedIndex(index)}
                                        onMouseDown={(e) => {
                                            e.stopPropagation()
                                        }}
                                        className={cn(
                                            "flex items-center gap-2 px-2 py-2 rounded-sm cursor-pointer text-sm",
                                            "select-none",
                                            index === highlightedIndex && "bg-accent text-accent-foreground",
                                            isSelected && "font-medium"
                                        )}
                                    >
                                        <Check
                                            className={cn(
                                                "h-4 w-4 shrink-0",
                                                isSelected
                                                    ? "text-blue-500 opacity-100"
                                                    : "opacity-0"
                                            )}
                                        />
                                        <Badge
                                            variant="outline"
                                            className="shrink-0 text-xs bg-blue-500/5 text-blue-600 border-blue-500/20"
                                        >
                                            {item.key}
                                        </Badge>
                                        <span className="truncate">{item.value}</span>
                                    </div>
                                )
                            })
                        )}
                    </div>
                </div>
            )}
        </div>
    )
}
