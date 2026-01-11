
"use client"

import { useEffect, useState } from "react"
import { RobustaAPI } from "@/lib/api"
import { OverviewResponse } from "@/types/calico"
import { StatsCards } from "./components/StatsCards"
import { OverviewTable } from "./components/OverviewTable"
import { Button } from "@/components/ui/button"
import { RotateCw, Share2 } from "lucide-react"

export default function CalicoOverviewPage() {
    const [data, setData] = useState<OverviewResponse | null>(null)
    const [isLoading, setIsLoading] = useState(true)
    const [lastUpdated, setLastUpdated] = useState<Date>(new Date())

    const fetchData = async () => {
        setIsLoading(true)
        try {
            const res = await RobustaAPI.getCalicoOverview()
            setData(res)
            setLastUpdated(new Date())
        } catch (error) {
            console.error("Failed to fetch Calico overview", error)
        } finally {
            setIsLoading(false)
        }
    }

    useEffect(() => {
        fetchData()
    }, [])

    return (
        <div className="space-y-8">
            {/* Header */}
            <div className="relative overflow-hidden rounded-2xl bg-gradient-to-br from-primary/5 via-primary/[0.02] to-background border border-primary/10 p-6 sm:p-8">
                {/* Decorative background pattern */}
                <div className="absolute inset-0 opacity-[0.03]" style={{
                    backgroundImage: `radial-gradient(circle at 2px 2px, currentColor 1px, transparent 0)`,
                    backgroundSize: '32px 32px'
                }} />

                {/* Glowing effect */}
                <div className="absolute -right-20 -top-20 h-64 w-64 rounded-full bg-primary/10 blur-3xl" />
                <div className="absolute -left-20 -bottom-20 h-64 w-64 rounded-full bg-primary/5 blur-3xl" />

                <div className="relative flex items-center justify-between">
                    <div className="space-y-2">
                        <div className="flex items-center gap-3">
                            <div className="inline-flex items-center justify-center rounded-xl bg-primary/10 p-2.5 ring-1 ring-primary/20">
                                <Share2 className="h-6 w-6 text-primary" />
                            </div>
                            <h1 className="text-2xl sm:text-3xl font-bold tracking-tight bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text text-transparent">
                                Calico 网络概览
                            </h1>
                            {!isLoading && (
                                <div className="flex items-center gap-1.5 px-3 py-1 rounded-full border border-green-500/20 bg-green-500/10 text-xs font-medium text-green-600 dark:text-green-400 shadow-sm">
                                    <div className="relative">
                                        <div className="h-2 w-2 rounded-full bg-green-500" />
                                        <div className="absolute inset-0 h-2 w-2 rounded-full bg-green-500 animate-ping opacity-75" />
                                    </div>
                                    实时
                                </div>
                            )}
                        </div>
                        <p className="text-sm sm:text-base text-muted-foreground max-w-2xl">
                            所有 Calico 集群的全局可观测性与健康状态监控
                        </p>
                    </div>
                    <div className="flex items-center gap-4">
                        <div className="text-xs text-muted-foreground text-right hidden sm:block">
                            <p className="text-[11px] uppercase tracking-wider font-medium">最后更新</p>
                            <p className="font-mono text-sm font-medium tabular-nums">{lastUpdated.toLocaleTimeString()}</p>
                        </div>
                        <Button
                            variant="outline"
                            size="default"
                            onClick={fetchData}
                            disabled={isLoading}
                            className="shadow-sm hover:shadow hover:bg-primary/5 transition-all duration-200 border-primary/20"
                        >
                            <RotateCw className={`mr-2 h-4 w-4 ${isLoading ? 'animate-spin' : ''}`} />
                            刷新
                        </Button>
                    </div>
                </div>
            </div>

            {/* Stats Cards */}
            <StatsCards stats={data?.stats} isLoading={isLoading} />

            {/* Main Content Area */}
            <div className="grid gap-6 md:grid-cols-1">
                <div className="space-y-4">
                    <div className="flex items-center gap-2">
                        <div className="h-px flex-1 bg-gradient-to-r from-transparent via-border to-transparent" />
                        <h2 className="text-xl font-semibold tracking-tight px-4">跨集群资源总览</h2>
                        <div className="h-px flex-1 bg-gradient-to-r from-transparent via-border to-transparent" />
                    </div>
                    <OverviewTable clusters={data?.clusters || []} isLoading={isLoading} />
                </div>
            </div>
        </div>
    )
}
