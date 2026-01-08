"use client"

import * as React from "react"
import { Copy, Check, BookOpen } from "lucide-react"
import { toast } from "sonner"

import { cn } from "@/lib/utils"
import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from "@/components/ui/tooltip"

interface DictionaryCodeBadgeProps {
    code: string
    className?: string
}

/**
 * 显示字典 code 的徽章组件
 * - Hover 显示美化的提示框
 * - 点击可复制 code 到剪贴板
 */
export function DictionaryCodeBadge({ code, className }: DictionaryCodeBadgeProps) {
    const [copied, setCopied] = React.useState(false)

    const handleCopy = async (e: React.MouseEvent) => {
        e.preventDefault()
        e.stopPropagation()
        try {
            await navigator.clipboard.writeText(code)
            setCopied(true)
            toast.success(`已复制: ${code}`)
            setTimeout(() => setCopied(false), 2000)
        } catch (err) {
            toast.error("复制失败")
        }
    }

    return (
        <TooltipProvider delayDuration={300}>
            <Tooltip>
                <TooltipTrigger asChild>
                    <button
                        type="button"
                        onClick={handleCopy}
                        className={cn(
                            "inline-flex items-center gap-1 text-xs font-mono",
                            "px-1.5 py-0.5 rounded",
                            "bg-muted/80 text-muted-foreground",
                            "hover:bg-blue-500/10 hover:text-blue-600",
                            "transition-colors duration-150 cursor-pointer",
                            "border border-transparent hover:border-blue-500/20",
                            className
                        )}
                    >
                        {copied ? (
                            <Check className="h-3 w-3 text-green-500" />
                        ) : (
                            <BookOpen className="h-3 w-3 opacity-60" />
                        )}
                        <span>{code}</span>
                    </button>
                </TooltipTrigger>
                <TooltipContent
                    side="top"
                    className="z-[1100] bg-popover text-popover-foreground border shadow-lg"
                >
                    <div className="flex flex-col gap-1.5 py-1">
                        <div className="flex items-center gap-2 text-muted-foreground">
                            <BookOpen className="h-3.5 w-3.5 text-blue-500" />
                            <span className="text-xs font-medium">字典编码</span>
                        </div>
                        <div className="flex items-center gap-2 bg-blue-500/10 px-2 py-1 rounded border border-blue-500/20">
                            <code className="text-blue-600 font-mono text-sm font-medium">{code}</code>
                            <Copy className="h-3 w-3 text-blue-400" />
                        </div>
                        <span className="text-xs text-muted-foreground">点击复制到剪贴板</span>
                    </div>
                </TooltipContent>
            </Tooltip>
        </TooltipProvider>
    )
}
