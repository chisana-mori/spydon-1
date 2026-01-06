import React from 'react';
import { ScrollArea } from '@/components/ui/scroll-area';
import { useDrain } from './DrainContext';
import { cn } from '@/lib/utils';

export const LogViewer: React.FC<{ className?: string }> = ({ className }) => {
    const { logs } = useDrain();

    return (
        <div className={cn("border rounded-md bg-zinc-950 p-4 font-mono text-xs text-zinc-300", className)}>
            <ScrollArea className="h-full">
                <div className="space-y-1">
                    {/* 日志已按最新在前排序，直接渲染 */}
                    {logs.map((log, i) => (
                        <div key={log.id || i} className="flex gap-2">
                            <span className="text-zinc-500 shrink-0">
                                {typeof log.timestamp === 'string' ? log.timestamp : new Date(log.timestamp).toLocaleTimeString()}
                            </span>
                            <span className={cn(
                                "shrink-0 w-14 font-bold",
                                log.level === 'ERROR' ? "text-red-500" :
                                    log.level === 'WARN' ? "text-yellow-500" :
                                        log.level === 'SUCCESS' ? "text-green-500" :
                                            log.level === 'FOCUS' ? "text-purple-500" :
                                                "text-blue-500"
                            )}>
                                [{log.level}]
                            </span>
                            <span className="break-all whitespace-pre-wrap">
                                {log.message}
                            </span>
                        </div>
                    ))}
                    {logs.length === 0 && (
                        <div className="text-zinc-500 text-center py-4">等待日志...</div>
                    )}
                </div>
            </ScrollArea>
        </div>
    );
};
