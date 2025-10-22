'use client'

import { useQuery } from '@tanstack/react-query'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  AlertTriangle,
  Server,
  Activity,
  Clock,
  CheckCircle,
  XCircle,
  AlertCircle,
  TrendingUp,
  Zap,
  Shield,
  Eye,
  BarChart3,
  RefreshCw,
  Layers,
  Settings
} from 'lucide-react'
import { cn } from '@/lib/utils'
import RobustaAPI from '@/lib/api'
import Link from 'next/link'
import { format, formatDistanceToNow } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import {
  ResponsiveContainer,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
} from 'recharts'

export default function Dashboard() {
  // 获取集群概览数据
  const { data: summary, isLoading: summaryLoading } = useQuery({
    queryKey: ['clusters-summary'],
    queryFn: () => RobustaAPI.getClustersSummary(),
    refetchInterval: 30000, // 30秒刷新一次
  })

  // 获取最近告警
  const { data: recentAlerts, isLoading: alertsLoading } = useQuery({
    queryKey: ['recent-alerts'],
    queryFn: () => RobustaAPI.getAlerts(1, 10, { status: 'firing' }),
    refetchInterval: 15000, // 15秒刷新一次
  })

  const summaryData = summary?.data
  const alertsData = recentAlerts?.data || []

  // 获取告警趋势数据
  const { data: alertTrend, isLoading: trendLoading } = useQuery({
    queryKey: ['alert-trend'],
    queryFn: () => RobustaAPI.getAlertTrend(30),
    refetchInterval: 60000,
  })

  const trendData = alertTrend?.data || []
  const trendChartData = trendData.map((point) => ({
    date: format(new Date(point.date), 'MM-dd', { locale: zhCN }),
    count: point.count,
  }))

  // 获取严重级别对应的颜色和图标
  const getSeverityConfig = (severity: string) => {
    switch (severity) {
      case 'critical':
        return { color: 'destructive', icon: XCircle }
      case 'high':
        return { color: 'destructive', icon: AlertCircle }
      case 'medium':
        return { color: 'default', icon: AlertTriangle }
      case 'low':
        return { color: 'secondary', icon: Activity }
      default:
        return { color: 'secondary', icon: Activity }
    }
  }

  const netProfit = summaryData?.total_alerts || 0
  const isProfit = netProfit < 10

  const getRiskBadge = (level: string) => {
    switch (level) {
      case 'low':
        return <Badge variant="secondary" className="bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">
          <CheckCircle className="w-3 h-3 mr-1" />
          低风险
        </Badge>
      case 'medium':
        return <Badge variant="secondary" className="bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200">
          <AlertTriangle className="w-3 h-3 mr-1" />
          中等风险
        </Badge>
      case 'high':
        return <Badge variant="destructive">
          <XCircle className="w-3 h-3 mr-1" />
          高风险
        </Badge>
      default:
        return <Badge variant="outline">未知</Badge>
    }
  }

  return (
    <div className="space-y-6">
      {/* Welcome Header */}
      <div className="flex flex-col space-y-2">
        <h1 className="text-3xl font-bold text-foreground">
          欢迎使用 Robusta Hub! 👋
        </h1>
        <p className="text-muted-foreground">
          这是您的告警管理概览，查看最新的集群状态和告警信息。
        </p>
      </div>

      {/* Risk Status Alert */}
      {(summaryData?.critical_alerts || 0) > 0 && (
        <Card className="border-yellow-200 bg-yellow-50 dark:border-yellow-800 dark:bg-yellow-950">
          <CardContent className="p-4">
            <div className="flex items-center space-x-2">
              <AlertTriangle className="h-5 w-5 text-yellow-600 dark:text-yellow-400" />
              <div>
                <p className="font-medium text-yellow-800 dark:text-yellow-200">
                  风险提醒
                </p>
                <p className="text-sm text-yellow-700 dark:text-yellow-300">
                  当前风险等级：{getRiskBadge('high')}
                  {(summaryData?.critical_alerts || 0) > 0 && (
                    <span className="ml-2">
                      有 {summaryData?.critical_alerts || 0} 个严重告警需要处理
                    </span>
                  )}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Main Content Grid */}
      <div className="grid gap-6 lg:grid-cols-3">
        {/* Stats Grid - Takes up 2 columns to match chart width */}
        <div className="lg:col-span-2">
          <div className="grid gap-6 md:grid-cols-3 mb-6">
            {/* Total Clusters */}
            <Card className="border-2 border-dashed border-muted-foreground/20 hover:border-primary/50 transition-colors">
              <CardHeader className="text-center pb-4">
                <div className="w-12 h-12 bg-blue-50 dark:bg-blue-900/20 rounded-xl flex items-center justify-center mx-auto mb-3">
                  <Server className="w-6 h-6 text-blue-600" />
                </div>
                <CardTitle className="text-lg">集群总数</CardTitle>
                <CardDescription>当前管理的集群数量</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="text-center">
                  <div className="text-3xl font-bold mb-2">
                    {summaryLoading ? (
                      <div className="animate-pulse bg-muted h-8 w-16 rounded mx-auto"></div>
                    ) : (
                      summaryData?.total_clusters || 0
                    )}
                  </div>
                  <p className="text-sm text-muted-foreground">
                    活跃: {summaryData?.active_clusters || 0} 个集群
                  </p>
                </div>
              </CardContent>
            </Card>

            {/* Total Alerts */}
            <Card className="border-2 border-dashed border-muted-foreground/20 hover:border-primary/50 transition-colors">
              <CardHeader className="text-center pb-4">
                <div className="w-12 h-12 bg-orange-50 dark:bg-orange-900/20 rounded-xl flex items-center justify-center mx-auto mb-3">
                  <AlertTriangle className="w-6 h-6 text-orange-600" />
                </div>
                <CardTitle className="text-lg">总告警数</CardTitle>
                <CardDescription>系统中的告警总数</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="text-center">
                  <div className="text-3xl font-bold mb-2">
                    {summaryLoading ? (
                      <div className="animate-pulse bg-muted h-8 w-16 rounded mx-auto"></div>
                    ) : (
                      summaryData?.total_alerts || 0
                    )}
                  </div>
                  <p className="text-sm text-muted-foreground">
                    严重: {summaryData?.critical_alerts || 0} 个
                  </p>
                </div>
              </CardContent>
            </Card>

            {/* Active Alerts */}
            <Card className="border-2 border-dashed border-muted-foreground/20 hover:border-primary/50 transition-colors">
              <CardHeader className="text-center pb-4">
                <div className="w-12 h-12 bg-red-50 dark:bg-red-900/20 rounded-xl flex items-center justify-center mx-auto mb-3">
                  <Activity className="w-6 h-6 text-red-600" />
                </div>
                <CardTitle className="text-lg">活跃告警</CardTitle>
                <CardDescription>需要立即关注的告警</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="text-center">
                  <div className="text-3xl font-bold mb-2">
                    {summaryLoading ? (
                      <div className="animate-pulse bg-muted h-8 w-16 rounded mx-auto"></div>
                    ) : (
                      summaryData?.total_alerts || 0
                    )}
                  </div>
                  <p className="text-sm text-muted-foreground">
                    需要关注
                  </p>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Chart - Now below the stats */}
          <Card className="border-2 border-dashed border-muted-foreground/20 hover:border-primary/50 transition-colors">
            <CardHeader className="pb-4">
              <CardTitle className="text-lg flex items-center gap-2">
                <div className="w-8 h-8 bg-blue-50 dark:bg-blue-900/20 rounded-lg flex items-center justify-center">
                  <BarChart3 className="w-4 h-4 text-blue-600" />
                </div>
                告警趋势
              </CardTitle>
              <CardDescription>
                过去30天的告警变化趋势
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="h-80">
                {trendLoading ? (
                  <div className="h-full flex items-center justify-center">
                    <div className="w-full h-48 bg-muted/30 rounded-lg animate-pulse" />
                  </div>
                ) : trendChartData.length ? (
                  <ResponsiveContainer width="100%" height="100%">
                    <AreaChart data={trendChartData} margin={{ top: 10, right: 20, left: 0, bottom: 0 }}>
                      <defs>
                        <linearGradient id="alertTrend" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%" stopColor="#4F46E5" stopOpacity={0.35} />
                          <stop offset="95%" stopColor="#4F46E5" stopOpacity={0.05} />
                        </linearGradient>
                      </defs>
                      <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#E5E7EB" />
                      <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{ fontSize: 12, fill: '#6B7280' }} interval={4} />
                      <YAxis tickLine={false} axisLine={false} tick={{ fontSize: 12, fill: '#6B7280' }} allowDecimals={false} width={40} />
                      <Tooltip
                        contentStyle={{ borderRadius: 8, border: '1px solid #E5E7EB' }}
                        formatter={(value: number) => [`${value} 条`, '告警量']}
                        labelFormatter={(label: string) => `日期：${label}`}
                      />
                      <Area type="monotone" dataKey="count" stroke="#4F46E5" strokeWidth={2} fill="url(#alertTrend)" />
                    </AreaChart>
                  </ResponsiveContainer>
                ) : (
                  <div className="h-full flex items-center justify-center bg-muted/20 rounded-lg border-2 border-dashed border-muted">
                    <div className="text-center">
                      <BarChart3 className="h-12 w-12 text-muted-foreground mx-auto mb-2" />
                      <p className="text-muted-foreground text-sm">暂无告警数据</p>
                      <p className="text-xs text-muted-foreground">最近30天未记录到新的告警</p>
                    </div>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Recent Alerts - Extended to full height */}
        <Card className="flex flex-col h-full border-2 border-dashed border-muted-foreground/20 hover:border-primary/50 transition-colors">
          <CardHeader className="pb-4">
            <CardTitle className="text-lg flex items-center gap-2">
              <div className="w-8 h-8 bg-red-50 dark:bg-red-900/20 rounded-lg flex items-center justify-center">
                <AlertTriangle className="w-4 h-4 text-red-600" />
              </div>
              最近告警
            </CardTitle>
            <CardDescription>
              最新的告警记录
            </CardDescription>
          </CardHeader>
          <CardContent className="flex-1 flex flex-col">
            <div className="space-y-3 flex-1">
              {alertsLoading ? (
                <div className="space-y-4">
                  {[1, 2, 3, 4, 5].map((i) => (
                    <div key={i} className="animate-pulse">
                      <div className="flex items-start space-x-3">
                        <div className="h-5 w-5 bg-muted rounded mt-0.5"></div>
                        <div className="flex-1">
                          <div className="h-4 bg-muted rounded w-3/4 mb-2"></div>
                          <div className="flex space-x-2">
                            <div className="h-3 bg-muted rounded w-16"></div>
                            <div className="h-3 bg-muted rounded w-20"></div>
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : alertsData.length ? (
                <div className="space-y-3 grid grid-cols-1">
                  {alertsData.slice(0, Math.min(alertsData.length, Math.ceil(360 / 70))).map((alert) => {
                    const severityConfig = getSeverityConfig(alert.severity)
                    const SeverityIcon = severityConfig.icon

                    return (
                      <Link
                        key={alert.id}
                        href={`/alerts/${alert.id}`}
                        className="flex items-center justify-between p-3 rounded-lg border hover:bg-muted/20 hover:border-primary/50 transition-all cursor-pointer group"
                      >
                        <div className="flex items-center space-x-3">
                          <div className={cn(
                            "p-2 rounded-full transition-transform group-hover:scale-110",
                            alert.severity === 'critical' ? 'bg-red-100 text-red-600 dark:bg-red-900 dark:text-red-400' :
                            alert.severity === 'high' ? 'bg-orange-100 text-orange-600 dark:bg-orange-900 dark:text-orange-400' :
                            alert.severity === 'medium' ? 'bg-yellow-100 text-yellow-600 dark:bg-yellow-900 dark:text-yellow-400' :
                            'bg-green-100 text-green-600 dark:bg-green-900 dark:text-green-400'
                          )}>
                            <SeverityIcon className="h-4 w-4" />
                          </div>
                          <div>
                            <p className="font-medium text-sm line-clamp-2 group-hover:text-primary transition-colors">{alert.title}</p>
                            <p className="text-xs text-muted-foreground">
                              {alert.cluster?.name || alert.cluster_id}
                            </p>
                          </div>
                        </div>
                        <div className="text-right">
                          <p className="text-xs text-muted-foreground">
                            {formatDistanceToNow(new Date(alert.created_at), {
                              addSuffix: true,
                              locale: zhCN
                            })}
                          </p>
                        </div>
                      </Link>
                    )
                  })}
                </div>
              ) : (
                <div className="flex-1 flex items-center justify-center text-center py-8">
                  <div>
                    <CheckCircle className="h-12 w-12 text-success mx-auto mb-3" />
                    <p className="text-muted-foreground text-sm">暂无告警记录</p>
                    <p className="text-xs text-muted-foreground mt-1">系统运行正常</p>
                  </div>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      </div>


    </div>
  )
}
