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
import { ShieldAlert } from "lucide-react";
import { DrainProvider, useDrain } from './DrainContext';
import { DrainProgress } from './DrainProgress';
import { LogViewer } from './LogViewer';
import { MigrationsTable } from './MigrationsTable';
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

interface SafeDrainDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    clusterName: string;
    nodeName: string;
}

const SafeDrainContent: React.FC<{ clusterName: string; nodeName: string, onClose: () => void }> = ({ clusterName, nodeName, onClose }) => {
    const { startDrain, cancelDrain, status, drainId, reset } = useDrain();
    const [isStarting, setIsStarting] = useState(false);

    // Auto-start or wait for confirmation?
    // Usually wait for "Confirm" btn.

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

    const isRunning = status === 'running' || status === 'pending' || (status === 'completed' && false);
    const isFinished = status === 'completed' || status === 'failed' || status === 'cancelled';

    return (
        <div className="flex flex-col gap-4 h-full">
            {/* Info / Warning Banner */}
            {!drainId && (
                <Alert>
                    <ShieldAlert className="h-4 w-4" />
                    <AlertTitle>Safe Eviction Mode</AlertTitle>
                    <AlertDescription>
                        This will safely evict pods from <strong>{nodeName}</strong> respecting PDBs.
                        A dedicated lease will be created to prevent concurrent drains.
                    </AlertDescription>
                </Alert>
            )}

            {/* Progress Section */}
            {(drainId) && (
                <div className="p-4 border rounded-lg bg-zinc-50/50 dark:bg-zinc-900/50">
                    <DrainProgress />
                </div>
            )}

            {/* Migrations Table */}
            {drainId && (
                <div className="space-y-2 flex-1 overflow-hidden flex flex-col">
                    <h3 className="text-sm font-medium">Pod Migrations</h3>
                    <div className="flex-1 overflow-auto border rounded-md">
                        <MigrationsTable />
                    </div>
                </div>
            )}

            {/* Logs (Always show or collapsible? Always show is good for monitoring) */}
            {drainId && (
                <div className="h-[200px] shrink-0">
                    <LogViewer className="h-full" />
                </div>
            )}

            <SheetFooter className="mt-auto pt-4">
                {!drainId ? (
                    <>
                        <Button variant="outline" onClick={onClose}>Cancel</Button>
                        <Button variant="destructive" onClick={handleStart} disabled={isStarting}>
                            {isStarting ? "Starting..." : "Start Safe Drain"}
                        </Button>
                    </>
                ) : (
                    <>
                        {isFinished ? (
                            <Button onClick={onClose} variant="outline">Close</Button>
                        ) : (
                            <Button variant="destructive" onClick={handleCancel}>Cancel Drain</Button>
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
            <SheetContent side="right" className="sm:max-w-none w-[calc(100vw-3rem)] sm:w-[calc(100vw-16rem)] overflow-y-auto">
                <SheetHeader>
                    <SheetTitle>Safe Drain: {props.nodeName}</SheetTitle>
                    <SheetDescription>
                        Manage safe eviction of workloads from this node.
                    </SheetDescription>
                </SheetHeader>

                <div className="mt-4 h-[calc(100vh-8rem)]">
                    <DrainProvider>
                        <SafeDrainContent {...props} onClose={() => props.onOpenChange(false)} />
                    </DrainProvider>
                </div>
            </SheetContent>
        </Sheet>
    );
};
