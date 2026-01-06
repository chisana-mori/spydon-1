import React from 'react';
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { useDrain } from './DrainContext';
import { DrainPodMigrationStatus } from '@/types/safe-drain';

const StatusBadge: React.FC<{ status: DrainPodMigrationStatus }> = ({ status }) => {
    const variants: Record<DrainPodMigrationStatus, "default" | "secondary" | "destructive" | "outline"> = {
        pending: "outline",
        evicting: "secondary",
        evicted: "secondary",
        creating: "default",
        completed: "default",
        failed: "destructive",
        ignored: "outline" // e.g. DaemonSets
    };

    const colors: Record<DrainPodMigrationStatus, string> = {
        pending: "text-zinc-500",
        evicting: "bg-blue-100 text-blue-800 hover:bg-blue-100",
        evicted: "bg-purple-100 text-purple-800 hover:bg-purple-100",
        creating: "bg-indigo-100 text-indigo-800 hover:bg-indigo-100",
        completed: "bg-green-100 text-green-800 hover:bg-green-100",
        failed: "bg-red-100 text-red-800 hover:bg-red-100",
        ignored: "text-zinc-400"
    };

    return (
        // Override variant styles with specific colors if needed, or use variant map
        <Badge variant={variants[status]} className={colors[status]}>
            {status.toUpperCase()}
        </Badge>
    );
};

export const MigrationsTable: React.FC = () => {
    const { migrations } = useDrain();

    return (
        <div className="rounded-md border h-[300px] overflow-auto relative">
            <Table>
                    <TableHeader className="sticky top-0 bg-secondary z-10">
                    <TableRow>
                        <TableHead>Pod Name</TableHead>
                        <TableHead>Namespace</TableHead>
                            <TableHead>Status</TableHead>
                            <TableHead>Started</TableHead>
                            <TableHead>Evicted</TableHead>
                            <TableHead>Completed</TableHead>
                            <TableHead className="text-right">Message</TableHead>
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {migrations.length === 0 ? (
                        <TableRow>
                                <TableCell colSpan={7} className="h-24 text-center">
                                No pods migrated yet.
                            </TableCell>
                        </TableRow>
                    ) : (
                        migrations.map((mig) => (
                            <TableRow key={mig.migrationId}>
                                <TableCell className="font-medium">{mig.sourcePod.name}</TableCell>
                                <TableCell>{mig.sourcePod.namespace}</TableCell>
                                <TableCell>
                                    <StatusBadge status={mig.status} />
                                </TableCell>
                                <TableCell className="text-sm text-muted-foreground">
                                    {mig.startTime ? new Date(mig.startTime).toLocaleTimeString() : '-'}
                                </TableCell>
                                <TableCell className="text-sm text-muted-foreground">
                                    {mig.evictionTime ? new Date(mig.evictionTime).toLocaleTimeString() : '-'}
                                </TableCell>
                                <TableCell className="text-sm text-muted-foreground">
                                    {mig.completionTime ? new Date(mig.completionTime).toLocaleTimeString() : '-'}
                                </TableCell>
                                <TableCell className="text-right text-sm text-muted-foreground max-w-[200px] truncate" title={mig.errorMessage}>
                                    {mig.errorMessage || '-'}
                                </TableCell>
                            </TableRow>
                        ))
                    )}
                </TableBody>
            </Table>
        </div>
    );
};
