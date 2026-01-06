import React, { useEffect, useRef } from 'react';
import { ScrollArea } from '@/components/ui/scroll-area';
import { useDrain } from './DrainContext';
import { cn } from '@/lib/utils'; // Assuming Shadcn utils

export const LogViewer: React.FC<{ className?: string }> = ({ className }) => {
    const { logs } = useDrain();
    const scrollRef = useRef<HTMLDivElement>(null);
    const bottomRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, [logs]);

    return (
        <div className={cn("border rounded-md bg-zinc-950 p-4 font-mono text-xs text-zinc-300", className)}>
            <ScrollArea className="h-full">
                <div ref={scrollRef} className="space-y-1">
                    {logs.map((log, i) => (
                        <div key={i} className="flex gap-2">
                            <span className="text-zinc-500 shrink-0">
                                {new Date(log.timestamp).toLocaleTimeString()}
                            </span>
                            <span className={cn(
                                "shrink-0 w-12 font-bold",
                                log.level === 'ERROR' ? "text-red-500" :
                                    log.level === 'WARN' ? "text-yellow-500" :
                                        "text-blue-500"
                            )}>
                                [{log.level}]
                            </span>
                            <span className="break-all whitespace-pre-wrap">
                                {log.message}
                            </span>
                        </div>
                    ))}
                    <div ref={bottomRef} />
                </div>
            </ScrollArea>
        </div>
    );
};
