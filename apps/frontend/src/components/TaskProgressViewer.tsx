'use client'

import React, { useEffect, useState, useRef } from 'react'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Card } from '@/components/ui/card'
import { RobustaAPI } from '@/lib/api'
import { PipelineNodeStatus } from '@/types/pipeline'
import { CheckCircle2, Circle, Loader2, XCircle, SkipForward } from 'lucide-react'
import { cn } from '@/lib/utils'
import { format } from 'date-fns'

interface TaskProgressViewerProps {
    executionId: string
    isOpen: boolean
}

export function TaskProgressViewer({ executionId, isOpen }: TaskProgressViewerProps) {
    const [tasks, setTasks] = useState<PipelineNodeStatus[]>([])
    const [isConnected, setIsConnected] = useState(false)
    const eventSourceRef = useRef<EventSource | null>(null)

    useEffect(() => {
        if (!isOpen || !executionId) {
            closeConnection()
            return
        }

        const url = RobustaAPI.getProgressStreamUrl(executionId)
        console.log(`Connecting to Progress Stream: ${url}`)

        const es = new EventSource(url, { withCredentials: true })
        eventSourceRef.current = es

        es.onopen = () => {
            setIsConnected(true)
        }

        // Listen for "task" events
        es.addEventListener('task', (event) => {
            try {
                const node = JSON.parse(event.data) as PipelineNodeStatus
                updateTask(node)
            } catch (e) {
                console.error("Failed to parse task node", e)
            }
        })

        es.onerror = (err) => {
            console.error('SSE Error:', err)
            setIsConnected(false)
            // Retry logic usually handled by browser or manual reconnect
        }

        return () => {
            closeConnection()
        }
    }, [executionId, isOpen])

    const closeConnection = () => {
        if (eventSourceRef.current) {
            eventSourceRef.current.close()
            eventSourceRef.current = null
            setIsConnected(false)
        }
    }

    const updateTask = (node: PipelineNodeStatus) => {
        setTasks((prev) => {
            // Check if task exists
            const index = prev.findIndex(t => t.task_id === node.task_id)
            if (index !== -1) {
                // Update
                const newTasks = [...prev]
                newTasks[index] = { ...newTasks[index], ...node }
                return newTasks
            } else {
                // Append
                return [...prev, node]
            }
        })
    }

    const getIcon = (status: string) => {
        switch (status) {
            case 'success': return <CheckCircle2 className="w-5 h-5 text-green-500" />
            case 'failed': return <XCircle className="w-5 h-5 text-red-500" />
            case 'running': return <Loader2 className="w-5 h-5 text-blue-500 animate-spin" />
            case 'skipped': return <SkipForward className="w-5 h-5 text-yellow-500" />
            default: return <Circle className="w-5 h-5 text-zinc-600" />
        }
    }

    return (
        <div className="flex flex-col h-full bg-background">
            <div className="flex items-center justify-between px-6 py-3 border-b bg-muted/20">
                <div className="flex items-center gap-2">
                    <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-500 animate-pulse' : 'bg-red-500'}`} />
                    <span className="text-sm font-medium text-muted-foreground">实时任务流</span>
                </div>
                <div className="text-xs text-muted-foreground">
                    {tasks.length} 个任务
                </div>
            </div>

            <ScrollArea className="flex-1 w-full bg-background/50">
                <div className="px-6 py-6 flex flex-col gap-4 relative">
                    {/* Vertical Line */}
                    <div className="absolute left-[39px] top-6 bottom-6 w-[2px] bg-muted z-0" />

                    {tasks.map((task, i) => (
                        <div key={task.task_id} className="flex gap-4 items-start relative z-10 animate-in fade-in slide-in-from-bottom-2 duration-300">
                            <div className="bg-background z-10 mt-1 p-1 rounded-full border border-border">
                                {getIcon(task.status)}
                            </div>
                            <div className="flex-1 min-w-0 bg-card p-4 rounded-lg border border-border/60 shadow-sm hover:shadow-md transition-shadow">
                                <div className="flex justify-between items-start mb-2">
                                    <h4 className="font-semibold text-sm text-foreground truncate pr-2" title={task.task_name}>
                                        {task.task_name}
                                    </h4>
                                    <span className={cn(
                                        "text-[10px] px-2 py-0.5 rounded-full uppercase font-bold tracking-wider border",
                                        task.status === 'success' && "bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-400 dark:border-green-800",
                                        task.status === 'failed' && "bg-red-50 text-red-700 border-red-200 dark:bg-red-900/20 dark:text-red-400 dark:border-red-800",
                                        task.status === 'running' && "bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-900/20 dark:text-blue-400 dark:border-blue-800",
                                        task.status === 'skipped' && "bg-yellow-50 text-yellow-700 border-yellow-200 dark:bg-yellow-900/20 dark:text-yellow-400 dark:border-yellow-800"
                                    )}>
                                        {task.status}
                                    </span>
                                </div>
                                <div className="flex justify-between items-center text-xs text-muted-foreground">
                                    <span className="flex items-center gap-1 font-mono bg-muted/50 px-1.5 py-0.5 rounded">
                                        {task.host || 'localhost'}
                                    </span>
                                    <span>{format(new Date(task.start_time), 'HH:mm:ss')}</span>
                                </div>
                            </div>
                        </div>
                    ))}

                    {tasks.length === 0 && isConnected && (
                        <div className="text-muted-foreground text-center py-10 italic">等待任务开始...</div>
                    )}
                </div>
            </ScrollArea>
        </div>
    )
}
