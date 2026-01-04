import React, { useState } from 'react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { TaskLogViewer } from './TaskLogViewer'
import { TaskProgressViewer } from './TaskProgressViewer'
import { FileText, GitGraph, X, Terminal, Activity, ArrowRightFromLine } from 'lucide-react'
import { Button } from './ui/button'
import { SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { cn } from '@/lib/utils'

interface ExecutionLiveViewProps {
    executionId: string
    isOpen: boolean
    onClose?: () => void
}

export function ExecutionLiveView({ executionId, isOpen, onClose }: ExecutionLiveViewProps) {
    const [activeTab, setActiveTab] = useState('logs')

    return (
        <div className="flex flex-col h-full w-full bg-background text-foreground relative overflow-hidden">
            {/* Stunning Header with Glassmorphism */}
            <div className="flex flex-col gap-4 px-6 py-5 border-b border-border/40 bg-background/60 backdrop-blur-xl sticky top-0 z-50">
                <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                        <div className="p-2 rounded-lg bg-primary/10 text-primary">
                            <Activity className="w-5 h-5" />
                        </div>
                        <div>
                            <SheetTitle className="text-xl font-bold tracking-tight">任务执行详情</SheetTitle>
                            <p className="text-xs text-muted-foreground font-mono mt-0.5">ID: #{executionId}</p>
                        </div>
                    </div>

                    <div className="flex items-center gap-2">
                        {/* Close Button */}
                        {onClose && (
                            <Button variant="ghost" size="icon" onClick={onClose} className="h-8 w-8 rounded-full hover:bg-muted">
                                <ArrowRightFromLine className="w-4 h-4 text-muted-foreground" />
                            </Button>
                        )}
                    </div>
                </div>

                {/* Modern Tabs Navigation */}
                <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
                    <TabsList className="w-full justify-start h-auto p-0 bg-transparent border-b border-transparent">
                        <TabsTrigger
                            value="logs"
                            className="relative h-9 rounded-none border-b-2 border-transparent px-4 pb-3 pt-2 font-medium text-muted-foreground shadow-none transition-none data-[state=active]:border-primary data-[state=active]:text-foreground data-[state=active]:shadow-none hover:text-foreground"
                        >
                            <Terminal className="w-4 h-4 mr-2" />
                            控制台日志
                        </TabsTrigger>
                        <TabsTrigger
                            value="visual"
                            className="relative h-9 rounded-none border-b-2 border-transparent px-4 pb-3 pt-2 font-medium text-muted-foreground shadow-none transition-none data-[state=active]:border-primary data-[state=active]:text-foreground data-[state=active]:shadow-none hover:text-foreground"
                        >
                            <GitGraph className="w-4 h-4 mr-2" />
                            可视化流程
                        </TabsTrigger>
                    </TabsList>
                </Tabs>
            </div>

            {/* Content Area */}
            <div className="flex-1 overflow-hidden relative bg-muted/5">
                <div className={cn("h-full w-full absolute inset-0 transition-opacity duration-300", activeTab === 'logs' ? 'opacity-100 z-10' : 'opacity-0 z-0 pointer-events-none')}>
                    <TaskLogViewer executionId={executionId} isOpen={isOpen && activeTab === 'logs'} />
                </div>
                <div className={cn("h-full w-full absolute inset-0 transition-opacity duration-300", activeTab === 'visual' ? 'opacity-100 z-10' : 'opacity-0 z-0 pointer-events-none')}>
                    <TaskProgressViewer executionId={executionId} isOpen={isOpen && activeTab === 'visual'} />
                </div>
            </div>
        </div>
    )
}
