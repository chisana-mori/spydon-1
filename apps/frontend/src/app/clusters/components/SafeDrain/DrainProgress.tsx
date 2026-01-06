import React from 'react';
import { Progress } from '@/components/ui/progress';
import { useDrain } from './DrainContext';

export const DrainProgress: React.FC = () => {
    const { progress, progressMessage, status } = useDrain();

    return (
        <div className="space-y-2">
            <div className="flex justify-between text-sm">
                <span className="font-medium">
                    {status === 'completed' ? 'Drain Completed' :
                        status === 'failed' ? 'Drain Failed' :
                            status === 'cancelled' ? 'Drain Cancelled' :
                                progressMessage || 'Ready to start'}
                </span>
                <span className="text-muted-foreground">{progress}%</span>
            </div>
            <Progress value={progress} className={
                status === 'failed' ? "bg-red-100 [&>div]:bg-red-500" :
                    status === 'completed' ? "bg-green-100 [&>div]:bg-green-500" :
                        ""
            } />
        </div>
    );
};
