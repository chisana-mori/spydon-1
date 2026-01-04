'use client'

import React, { useEffect, useState, useRef } from 'react'
import Ansi from 'ansi-to-react'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { RobustaAPI } from '@/lib/api'
import { Pause, Play, Download, Loader2 } from 'lucide-react'

interface TaskLogViewerProps {
    executionId: string
    isOpen: boolean
}

export function TaskLogViewer({ executionId, isOpen }: TaskLogViewerProps) {
    const [logs, setLogs] = useState<string[]>([])
    const [isConnected, setIsConnected] = useState(false)
    const [isAutoScroll, setIsAutoScroll] = useState(true)
    const scrollRef = useRef<HTMLDivElement>(null)
    const eventSourceRef = useRef<EventSource | null>(null)

    useEffect(() => {
        if (!isOpen || !executionId) {
            closeConnection()
            return
        }

        const url = RobustaAPI.getLogStreamUrl(executionId)
        console.log(`Connecting to Log Stream: ${url}`)

        const es = new EventSource(url, { withCredentials: true })
        eventSourceRef.current = es

        es.onopen = () => {
            setIsConnected(true)
            setLogs(prev => [...prev, 'System: Connected to log stream...'])
        }

        es.onmessage = (event) => {
            // For raw text stream, we might get data line by line
            // Or if backend sends "log" event
        }

        // Backend sends custom event name "log"
        es.addEventListener('log', (event) => {
            setLogs((prev) => [...prev, event.data])
        })

        es.onerror = (err) => {
            console.error('SSE Error:', err)
            setIsConnected(false)
            setLogs(prev => [...prev, '\nSystem: Connection lost. Reconnecting...'])
            es.close()
            // Optional: Retry logic is usually handled by EventSource automatically unless closed. 
            // If closed here, we might need manual retry or let user trigger.
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

    // Auto-scroll
    useEffect(() => {
        if (isAutoScroll && scrollRef.current) {
            scrollRef.current.scrollIntoView({ behavior: 'smooth' })
        }
    }, [logs, isAutoScroll])

    const handleDownload = () => {
        const blob = new Blob([logs.join('\n')], { type: 'text/plain' })
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `execution-${executionId}.log`
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
        URL.revokeObjectURL(url)
    }

    return (
        <div className="flex flex-col h-full bg-black font-mono text-xs md:text-sm">
            <div className="flex items-center justify-between px-4 py-2 bg-zinc-900 border-b border-zinc-800">
                <div className="flex items-center gap-2">
                    <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-500 animate-pulse' : 'bg-red-500'}`} />
                    <span className="text-zinc-400">Live Console</span>
                </div>
                <div className="flex gap-1">
                    <Button
                        variant="ghost"
                        size="icon"
                        className="h-6 w-6 text-zinc-400 hover:text-white hover:bg-zinc-800"
                        onClick={() => setIsAutoScroll(!isAutoScroll)}
                        title={isAutoScroll ? "Pause Scroll" : "Resume Scroll"}
                    >
                        {isAutoScroll ? <Pause className="h-3 w-3" /> : <Play className="h-3 w-3" />}
                    </Button>
                    <Button
                        variant="ghost"
                        size="icon"
                        className="h-6 w-6 text-zinc-400 hover:text-white hover:bg-zinc-800"
                        onClick={handleDownload}
                        title="Download Logs"
                    >
                        <Download className="h-3 w-3" />
                    </Button>
                </div>
            </div>

            <ScrollArea className="flex-1 w-full bg-black">
                <div className="p-4 flex flex-col gap-0.5 min-h-full">
                    {logs.map((line, i) => (
                        <div key={i} className="break-words whitespace-pre-wrap text-zinc-300 leading-tight">
                            <Ansi>{line}</Ansi>
                        </div>
                    ))}
                    {/* Dummy div for auto-scroll */}
                    <div ref={scrollRef} className="pb-4" />
                </div>
                {!isConnected && logs.length > 0 && (
                    <div className="text-zinc-600 italic px-4 pb-2 text-xs border-t border-zinc-900 pt-2">Stream disconnected.</div>
                )}
            </ScrollArea>
        </div>
    )
}
