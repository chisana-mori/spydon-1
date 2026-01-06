import React, { useState } from 'react';
import {
    Sheet,
    SheetContent,
    SheetDescription,
    SheetFooter,
    SheetHeader,
    SheetTitle,
} from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { ShieldAlert, Timer } from "lucide-react";
import { DrainProvider, useDrain } from './DrainContext';
import { DrainProgress } from './DrainProgress';
import { LogViewer } from './LogViewer';
import { MigrationsTable } from './MigrationsTable';
import { DrainStatsView } from './DrainStatsView';
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

interface SafeDrainDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    clusterName: string;
    nodeName: string;
}

const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
};

const SafeDrainContent: React.FC<{ clusterName: string; nodeName: string, onClose: () => void }> = ({ clusterName, nodeName, onClose }) => {
    const { startDrain, cancelDrain, status, drainId, migrations, elapsedTime } = useDrain();
    const [isStarting, setIsStarting] = useState(false);

    const handleStart = async () => {
        setIsStarting(true);
        try {
            await startDrain(clusterName, nodeName);
        } catch (e) {
            // handled in context logs
        } finally {
            setIsStarting(false);
        }
    };

    const handleCancel = async () => {
        await cancelDrain();
    };

    const isRunning = status === 'running' || status === 'pending';
    const isFinished = status === 'completed' || status === 'failed' || status === 'cancelled';

    return (
        <div className="flex flex-col gap-4 h-full">
            {/* Info / Warning Banner */}
            {!drainId && (
                <Alert>
                    <ShieldAlert className="h-4 w-4" />
                    <AlertTitle>安全驱逐模式</AlertTitle>
                    <AlertDescription>
                        即将对节点 <strong>{nodeName}</strong> 执行安全驱逐。系统将创建专用 Lease 锁，并严格遵循 PDB 策略进行 Pod 迁移。
                    </AlertDescription>
                </Alert>
            )}

            {/* Header Status & Progress */}
            {drainId && (
                <div className="space-y-4">
                    <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                            <div className={`text-sm font-medium px-2 py-1 rounded-md ${status === 'running' ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300' :
                                    status === 'completed' ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300' :
                                        status === 'failed' ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300' :
                                            'bg-zinc-100 text-zinc-700'
                                }`}>
                                {status === 'running' ? '正在执行' :
                                    status === 'completed' ? '执行完成' :
                                        status === 'failed' ? '执行失败' :
                                            status === 'cancelled' ? '已取消' : status}
                            </div>
                            {isRunning && (
                                <div className="flex items-center text-sm text-muted-foreground ml-2">
                                    <Timer className="w-4 h-4 mr-1" />
                                    {formatTime(elapsedTime)}
                                </div>
                            )}
                        </div>
                    </div>

                    <div className="p-4 border rounded-lg bg-zinc-50/50 dark:bg-zinc-900/50">
                        <DrainProgress />
                    </div>

                    <DrainStatsView />
                </div>
            )}

            {/* Logs Area - Fixed height in middle */}
            {drainId && (
                <div className="h-[180px] shrink-0 border rounded-md overflow-hidden flex flex-col">
                    <div className="px-3 py-2 bg-zinc-100 dark:bg-zinc-800 border-b text-xs font-medium text-muted-foreground flex justify-between items-center">
                        <span>实时日志</span>
                    </div>
                    <div className="flex-1 overflow-hidden">
                        <LogViewer className="h-full border-none rounded-none" />
                    </div>
                </div>
            )}

            {/* Migrations Table - Flex grow */}
            {drainId && (
                <div className="space-y-2 flex-1 overflow-hidden flex flex-col min-h-0">
                    <div className="flex items-center justify-between px-1">
                        <h3 className="text-sm font-medium">迁移详情</h3>
                        <span className="text-xs text-muted-foreground">{migrations.length} 个 Pod</span>
                    </div>
                    <div className="flex-1 overflow-auto border rounded-md bg-white dark:bg-zinc-950">
                        <MigrationsTable migrations={migrations} />
                    </div>
                </div>
            )}

            <SheetFooter className="mt-auto pt-4 shrink-0">
                {!drainId ? (
                    <>
                        <Button variant="outline" onClick={onClose}>取消</Button>
                        <Button variant="destructive" onClick={handleStart} disabled={isStarting}>
                            {isStarting ? "启动中..." : "开始安全驱逐"}
                        </Button>
                    </>
                ) : (
                    <>
                        {isFinished ? (
                            <Button onClick={onClose} variant="outline">关闭</Button>
                        ) : (
                            <Button variant="destructive" onClick={handleCancel}>停止驱逐</Button>
                        )}
                    </>
                )}
            </SheetFooter>
        </div>
    );
};

export const SafeDrainDialog: React.FC<SafeDrainDialogProps> = (props) => {
    return (
        <Sheet open={props.open} onOpenChange={props.onOpenChange}>
            <SheetContent side="right" className="sm:max-w-none w-[calc(100vw-3rem)] sm:w-[calc(100vw-16rem)] overflow-y-auto flex flex-col h-full">
                <SheetHeader className="shrink-0 mb-4">
                    <SheetTitle>安全驱逐: {props.nodeName}</SheetTitle>
                    <SheetDescription>
                        {/* Manage safe eviction of workloads from this node. */}
                    </SheetDescription>
                </SheetHeader>

                <div className="flex-1 overflow-hidden">
                    <DrainProvider>
                        <SafeDrainContent {...props} onClose={() => props.onOpenChange(false)} />
                    </DrainProvider>
                </div>
            </SheetContent>
        </Sheet>
    );
};
