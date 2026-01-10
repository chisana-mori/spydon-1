
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
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div className="space-y-1">
                    <div className="flex items-center gap-3">
                        <h1 className="text-2xl font-bold tracking-tight flex items-center gap-2">
                            <Share2 className="h-8 w-8 text-primary" />
                            Calico 网络概览
                        </h1>
                        {!isLoading && (
                            <div className="flex items-center gap-1.5 px-2.5 py-0.5 rounded-full border bg-background text-xs font-medium text-muted-foreground shadow-sm">
                                <div className="h-1.5 w-1.5 rounded-full bg-green-500 animate-pulse" />
                                实时
                            </div>
                        )}
                    </div>
                    <p className="text-muted-foreground">
                        所有 Calico 集群的全局可观测性与健康状态监控
                    </p>
                </div>
                <div className="flex items-center gap-4">
                    <div className="text-xs text-muted-foreground text-right hidden sm:block">
                        <p>最后更新</p>
                        <p className="font-mono font-medium">{lastUpdated.toLocaleTimeString()}</p>
                    </div>
                    <Button variant="outline" size="sm" onClick={fetchData} disabled={isLoading} className="shadow-sm">
                        <RotateCw className={`mr-2 h-4 w-4 ${isLoading ? 'animate-spin' : ''}`} />
                        刷新
                    </Button>
                </div>
            </div>

            {/* Stats Cards */}
            <StatsCards stats={data?.stats} isLoading={isLoading} />

            {/* Main Content Area */}
            <div className="grid gap-6 md:grid-cols-1">
                <div className="space-y-4">
                    <h2 className="text-xl font-semibold tracking-tight">跨集群资源总览</h2>
                    <OverviewTable clusters={data?.clusters || []} isLoading={isLoading} />
                </div>
            </div>
        </div>
    )
}
