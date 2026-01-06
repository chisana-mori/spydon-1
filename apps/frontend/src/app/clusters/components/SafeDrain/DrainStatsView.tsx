import React from 'react';
import { Card, CardContent } from "@/components/ui/card";
import { useDrain } from './DrainContext';
import { CheckCircle2, XCircle, Clock, PlayCircle, Shield, Ban, Box } from "lucide-react";
import { DrainStats } from '@/types/safe-drain';

interface StatCardProps {
    title: string;
    value: number;
    icon: React.ReactNode;
    colorClass: string;
    bgClass: string;
}

const StatCard: React.FC<StatCardProps> = ({ title, value, icon, colorClass, bgClass }) => (
    <Card className={`${bgClass} border-none shadow-sm`}>
        <CardContent className="p-4 flex items-center justify-between">
            <div>
                <p className="text-sm font-medium text-muted-foreground">{title}</p>
                <div className={`text-2xl font-bold ${colorClass}`}>{value}</div>
            </div>
            <div className={`p-2 rounded-full bg-white/50 dark:bg-black/20 ${colorClass}`}>
                {icon}
            </div>
        </CardContent>
    </Card>
);

export const DrainStatsDisplay: React.FC<{ stats: DrainStats }> = ({ stats }) => {
    return (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-4">
            <StatCard
                title="总 Pods"
                value={stats.totalPods}
                icon={<Box className="w-5 h-5" />}
                colorClass="text-zinc-700 dark:text-zinc-300"
                bgClass="bg-zinc-100 dark:bg-zinc-800"
            />
            <StatCard
                title="已迁移"
                value={stats.migratedPods}
                icon={<CheckCircle2 className="w-5 h-5" />}
                colorClass="text-green-600 dark:text-green-400"
                bgClass="bg-green-50 dark:bg-green-900/20"
            />
            <StatCard
                title="迁移失败"
                value={stats.failedPods}
                icon={<XCircle className="w-5 h-5" />}
                colorClass="text-red-600 dark:text-red-400"
                bgClass="bg-red-50 dark:bg-red-900/20"
            />
            <StatCard
                title="已忽略"
                value={stats.ignoredPods}
                icon={<Ban className="w-5 h-5" />}
                colorClass="text-zinc-500"
                bgClass="bg-zinc-50 dark:bg-zinc-900/10"
            />
            <StatCard
                title="等待中"
                value={stats.pendingPods}
                icon={<Clock className="w-5 h-5" />}
                colorClass="text-yellow-600 dark:text-yellow-400"
                bgClass="bg-yellow-50 dark:bg-yellow-900/20"
            />
            <StatCard
                title="迁移中"
                value={stats.migratingPods}
                icon={<PlayCircle className="w-5 h-5" />}
                colorClass="text-blue-600 dark:text-blue-400"
                bgClass="bg-blue-50 dark:bg-blue-900/20"
            />
            <StatCard
                title="已创建 PDB"
                value={stats.pdbCount}
                icon={<Shield className="w-5 h-5" />}
                colorClass="text-indigo-600 dark:text-indigo-400"
                bgClass="bg-indigo-50 dark:bg-indigo-900/20"
            />
        </div>
    );
};

export const DrainStatsView: React.FC = () => {
    const { stats } = useDrain();
    return <DrainStatsDisplay stats={stats} />;
};
