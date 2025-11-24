'use client'

import React, { useEffect, useRef } from 'react'
import { Button } from '@/components/ui/button'
import { CheckCircle2, XCircle, Terminal } from 'lucide-react'
import { ApprovalRequest, ApprovalDecision } from './holmesTypes'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import oneDark from 'react-syntax-highlighter/dist/esm/styles/prism/one-dark'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog"

interface ApprovalDialogProps {
    approval: ApprovalRequest
    onDecision: (decision: ApprovalDecision) => void
    disabled?: boolean
}

export const ApprovalDialog: React.FC<ApprovalDialogProps> = ({
    approval,
    onDecision,
    disabled = false
}) => {
    const approveBtnRef = useRef<HTMLButtonElement>(null)

    // 自动聚焦到允许按钮
    useEffect(() => {
        if (!disabled && approveBtnRef.current) {
            // 给一点延迟以确保 Dialog 动画完成
            const timer = setTimeout(() => {
                approveBtnRef.current?.focus()
            }, 100)
            return () => clearTimeout(timer)
        }
    }, [disabled])

    return (
        <Dialog open={true} onOpenChange={(open) => {
            // 如果用户尝试关闭（点击遮罩或按ESC），我们暂时不做处理，或者可以视为拒绝
            // 这里为了安全起见，不做默认处理，强制用户点击按钮
            if (!open) {
                // onDecision('decline') // 可选：点击遮罩视为拒绝
            }
        }}>
            <DialogContent
                className="sm:max-w-4xl bg-gray-900 border-yellow-600/30 shadow-2xl shadow-yellow-600/5"
                showCloseButton={false} // 隐藏默认关闭按钮，强制用户做决策
            >
                <DialogHeader>
                    <DialogTitle className="flex items-center space-x-2 text-yellow-500/90">
                        <Terminal className="h-5 w-5" />
                        <span>待执行命令</span>
                    </DialogTitle>
                    {approval.reason && (
                        <DialogDescription className="text-gray-400 border-l-2 border-yellow-600/20 pl-3 mt-2">
                            {approval.reason}
                        </DialogDescription>
                    )}
                </DialogHeader>

                {/* Terminal Window Look */}
                <div className="relative mt-2 bg-gray-800 rounded-lg overflow-hidden border border-gray-700">
                    {/* Window Controls Decoration */}
                    <div className="flex items-center space-x-1.5 px-4 py-2 bg-gray-800/50 border-b border-gray-800">
                        <div className="w-3 h-3 rounded-full bg-red-500/80"></div>
                        <div className="w-3 h-3 rounded-full bg-yellow-500/80"></div>
                        <div className="w-3 h-3 rounded-full bg-green-500/80"></div>
                        <span className="ml-2 text-xs text-gray-500 font-mono">bash</span>
                    </div>
                    <div className="p-0 overflow-x-auto max-h-[260px]">
                        <SyntaxHighlighter
                            language="bash"
                            style={oneDark}
                            customStyle={{
                                margin: 0,
                                padding: '1rem',
                                background: 'transparent',
                                fontSize: '14px',
                                lineHeight: '1.6',
                            }}
                            codeTagProps={{
                                style: {
                                    fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Consolas, "Liberation Mono", Menlo, monospace',
                                }
                            }}
                            wrapLongLines={true}
                        >
                            {approval.description || '# (命令为空)'}
                        </SyntaxHighlighter>
                    </div>
                </div>

                <DialogFooter className="mt-4 sm:justify-between gap-4">
                    <div className="hidden sm:flex text-xs text-gray-500 items-center">
                        <span className="mr-2">⚠️ 请仔细核对命令内容，确认无误后执行</span>
                    </div>
                    <div className="flex space-x-3 w-full sm:w-auto justify-end">
                        <Button
                            onClick={() => onDecision('decline')}
                            disabled={disabled}
                            variant="ghost"
                            className="text-gray-400 hover:text-red-400 hover:bg-red-950/20 min-w-[100px]"
                        >
                            <XCircle className="h-4 w-4 mr-2" />
                            拒绝
                        </Button>
                        <Button
                            ref={approveBtnRef}
                            onClick={() => onDecision('accept')}
                            disabled={disabled}
                            className="bg-green-600 hover:bg-green-500 text-white border-0 min-w-[100px]"
                        >
                            <CheckCircle2 className="h-4 w-4 mr-2" />
                            放行
                        </Button>
                    </div>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    )
}
