'use client'

import { use } from 'react'
import Link from 'next/link'
import { resolveAppPath } from '@/config'
import { useQuery } from '@tanstack/react-query'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  ArrowLeft,
  Clock,
  Server,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Activity,
  TrendingUp
} from 'lucide-react'
import RobustaAPI from '@/lib/api'
import { formatDistanceToNow, format } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { KiteLink } from '@/components/kite'

interface ClusterDetailPageProps {
  params: Promise<{
    id: string
  }>
}

export default function ClusterDetailPage({ params }: ClusterDetailPageProps) {
  const { id } = use(params)

  // 获取集群详情
  const { data: clusterData, isLoading } = useQuery({
    queryKey: ['cluster', id],
    queryFn: () => RobustaAPI.getCluster(id!),
    enabled: !!id,
  })

  // 获取集群的告警
  const { data: alertsData, isLoading: alertsLoading } = useQuery({
    queryKey: ['cluster-alerts', id],
    queryFn: () => RobustaAPI.getAlerts(1, 20, { cluster_name: id }),
    enabled: !!id,
  })

  // 获取告警统计
  const { data: alertStatsData } = useQuery({
    queryKey: ['alert-stats', id],
    queryFn: () => RobustaAPI.getAlertStats(id),
    enabled: !!id,
  })

  // 获取RCA统计
  const { data: rcaStatsData } = useQuery({
    queryKey: ['rca-stats', id],
    queryFn: () => RobustaAPI.getRCAStats(id),
    enabled: !!id,
  })

  const cluster = clusterData
  const alerts = alertsData?.data || []
  const alertStats = alertStatsData
  const rcaStats = rcaStatsData

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center space-x-4">
          <Button variant="ghost" size="sm" asChild>
            <Link href={resolveAppPath('/clusters')}>
              <ArrowLeft className="h-4 w-4 mr-2" />
              返回集群列表
            </Link>
          </Button>
        </div>
        <Card>
          <CardContent className="py-8">
            <div className="text-center">加载中...</div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!cluster) {
    return (
      <div className="space-y-6">
        <div className="flex items-center space-x-4">
          <Button variant="ghost" size="sm" asChild>
            <Link href={resolveAppPath('/clusters')}>
              <ArrowLeft className="h-4 w-4 mr-2" />
              返回集群列表
            </Link>
          </Button>
        </div>
        <Card>
          <CardContent className="py-8">
            <div className="text-center text-muted-foreground">
              集群不存在或已被删除
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  // 获取状态配置
  const getStatusConfig = (status: string) => {
    switch (status) {
      case 'Running':
        return {
          variant: 'secondary' as const,
          label: '运行中',
          icon: CheckCircle,
          color: 'text-green-600'
        }
      case 'Offline':
        return {
          variant: 'destructive' as const,
          label: '离线',
          icon: XCircle,
          color: 'text-red-600'
        }
      case 'Pending':
        return {
          variant: 'default' as const,
          label: '初始化中',
          icon: Activity,
          color: 'text-yellow-600'
        }
      case 'Init':
        return {
          variant: 'outline' as const,
          label: '初始化',
          icon: Server,
          color: 'text-blue-600'
        }
      default:
        return {
          variant: 'outline' as const,
          label: status,
          icon: Server,
          color: 'text-muted-foreground'
        }
    }
  }

  // 获取严重级别配置
  const getSeverityConfig = (severity: string) => {
    switch (severity) {
      case 'critical':
        return { variant: 'destructive' as const, label: '严重' }
      case 'high':
        return { variant: 'destructive' as const, label: '高' }
      case 'medium':
        return { variant: 'default' as const, label: '中' }
      case 'low':
        return { variant: 'secondary' as const, label: '低' }
      default:
        return { variant: 'secondary' as const, label: severity }
    }
  }

  const statusConfig = getStatusConfig(cluster.status)
  const StatusIcon = statusConfig.icon

  return (
    <div className="space-y-6">
      {/* 导航 */}
      <div className="flex items-center justify-between">
        <Button variant="ghost" size="sm" asChild>
          <Link href={resolveAppPath('/clusters')}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            返回集群列表
          </Link>
        </Button>
        <KiteLink
          clusterName={cluster.name}
          variant="button"
          size="default"
        />
      </div>

      {/* 集群基本信息 - 精美设计 */}
      <div className="bg-gradient-to-br from-white to-gray-50/50 border border-gray-200 rounded-2xl shadow-sm overflow-hidden">
        {/* 顶部状态栏 */}
        <div className="bg-white border-b border-gray-100 px-8 py-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <div className={`w-12 h-12 rounded-xl flex items-center justify-center ${cluster.status === 'Running'
                ? 'bg-green-100 text-green-600'
                : 'bg-gray-100 text-gray-600'
                }`}>
                <StatusIcon className="h-6 w-6" />
              </div>
              <div>
                <div className="flex items-center space-x-3 mb-1">
                  <h1 className="text-2xl font-bold text-gray-900">{cluster.name}</h1>
                  <Badge
                    variant={statusConfig.variant}
                    className={`px-3 py-1 text-sm font-medium ${cluster.status === 'Running'
                      ? 'bg-green-100 text-green-800 border-green-200'
                      : 'bg-gray-100 text-gray-800 border-gray-200'
                      }`}
                  >
                    {statusConfig.label}
                  </Badge>
                </div>
                <p className="text-gray-600 font-mono text-sm">
                  {cluster.cluster_id && `Cluster ID: ${cluster.cluster_id}`}
                </p>
              </div>
            </div>
          </div>
        </div>

        {/* 主要内容区域 */}
        <div className="px-8 py-6">
          {cluster.description && (
            <div className="mb-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-3">描述</h3>
              <div className="bg-blue-50 border border-blue-100 rounded-xl p-4">
                <p className="text-gray-700 leading-relaxed">{cluster.description}</p>
              </div>
            </div>
          )}

          {/* 时间信息网格 */}
          <div className="grid gap-6 md:grid-cols-2">
            {/* 左侧时间信息 */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">时间信息</h3>

              <div className="bg-white border border-gray-200 rounded-xl p-4 hover:shadow-md transition-shadow">
                <div className="flex items-center space-x-3">
                  <div className="w-10 h-10 bg-green-100 rounded-lg flex items-center justify-center">
                    <Clock className="h-5 w-5 text-green-600" />
                  </div>
                  <div className="flex-1">
                    <p className="text-sm font-medium text-gray-900">创建时间</p>
                    <p className="text-sm text-gray-600 font-mono">
                      {format(new Date(cluster.created_at), 'yyyy-MM-dd HH:mm:ss')}
                    </p>
                  </div>
                </div>
              </div>
            </div>

            {/* 右侧状态信息 */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">状态信息</h3>

              <div className="bg-white border border-gray-200 rounded-xl p-4 hover:shadow-md transition-shadow">
                <div className="flex items-center space-x-3">
                  <div className="w-10 h-10 bg-purple-100 rounded-lg flex items-center justify-center">
                    <Activity className="h-5 w-5 text-purple-600" />
                  </div>
                  <div className="flex-1">
                    <p className="text-sm font-medium text-gray-900">更新时间</p>
                    <p className="text-sm text-gray-600 font-mono">
                      {format(new Date(cluster.updated_at), 'yyyy-MM-dd HH:mm:ss')}
                    </p>
                  </div>
                </div>
              </div>

              <div className="bg-white border border-gray-200 rounded-xl p-4 hover:shadow-md transition-shadow">
                <div className="flex items-center space-x-3">
                  <div className="w-10 h-10 bg-orange-100 rounded-lg flex items-center justify-center">
                    <Activity className="h-5 w-5 text-orange-600" />
                  </div>
                  <div className="flex-1">
                    <p className="text-sm font-medium text-gray-900">运行时长</p>
                    <p className="text-sm text-gray-600">
                      {formatDistanceToNow(new Date(cluster.created_at), { locale: zhCN })}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* 统计信息 - 简洁设计 */}
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
        {/* 总告警数 */}
        <div className="bg-white border border-gray-200 rounded-xl p-6 hover:shadow-md transition-all duration-200">
          <div className="flex items-center justify-between mb-4">
            <div className="w-12 h-12 bg-orange-100 rounded-xl flex items-center justify-center">
              <AlertTriangle className="h-6 w-6 text-orange-600" />
            </div>
            <div className="text-right">
              <div className="text-3xl font-bold text-gray-900">
                {alertStats?.total || 0}
              </div>
              <p className="text-sm font-medium text-gray-600">总告警数</p>
            </div>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-sm text-gray-500">24小时内</span>
            <span className="text-sm font-semibold text-orange-600">
              {alertStats?.recent_24h || 0}
            </span>
          </div>
        </div>

        {/* RCA分析 */}
        <div className="bg-white border border-gray-200 rounded-xl p-6 hover:shadow-md transition-all duration-200">
          <div className="flex items-center justify-between mb-4">
            <div className="w-12 h-12 bg-blue-100 rounded-xl flex items-center justify-center">
              <TrendingUp className="h-6 w-6 text-blue-600" />
            </div>
            <div className="text-right">
              <div className="text-3xl font-bold text-gray-900">
                {rcaStats?.total || 0}
              </div>
              <p className="text-sm font-medium text-gray-600">RCA分析</p>
            </div>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-sm text-gray-500">成功率</span>
            <span className="text-sm font-semibold text-blue-600">
              {rcaStats?.success_rate ? `${(rcaStats.success_rate * 100).toFixed(1)}%` : '0%'}
            </span>
          </div>
        </div>

        {/* 严重告警 */}
        <div className="bg-white border border-gray-200 rounded-xl p-6 hover:shadow-md transition-all duration-200">
          <div className="flex items-center justify-between mb-4">
            <div className="w-12 h-12 bg-red-100 rounded-xl flex items-center justify-center">
              <XCircle className="h-6 w-6 text-red-600" />
            </div>
            <div className="text-right">
              <div className="text-3xl font-bold text-red-600">
                {alertStats?.by_severity?.find(s => s.severity === 'critical')?.count || 0}
              </div>
              <p className="text-sm font-medium text-gray-600">严重告警</p>
            </div>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-sm text-gray-500">需要立即处理</span>
            <span className="text-xs px-2 py-1 bg-red-100 text-red-700 rounded-full font-medium">
              严重
            </span>
          </div>
        </div>

        {/* 平均RCA时长 */}
        <div className="bg-white border border-gray-200 rounded-xl p-6 hover:shadow-md transition-all duration-200">
          <div className="flex items-center justify-between mb-4">
            <div className="w-12 h-12 bg-purple-100 rounded-xl flex items-center justify-center">
              <Clock className="h-6 w-6 text-purple-600" />
            </div>
            <div className="text-right">
              <div className="text-3xl font-bold text-gray-900">
                {rcaStats?.avg_duration_seconds
                  ? `${Math.round(rcaStats.avg_duration_seconds)}s`
                  : '0s'
                }
              </div>
              <p className="text-sm font-medium text-gray-600">平均RCA时长</p>
            </div>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-sm text-gray-500">平均分析时间</span>
            <span className="text-xs px-2 py-1 bg-purple-100 text-purple-700 rounded-full font-medium">
              秒
            </span>
          </div>
        </div>
      </div>

      {/* 最近告警 - 精美设计 */}
      <div className="bg-gradient-to-br from-white to-gray-50/50 border border-gray-200 rounded-2xl shadow-sm overflow-hidden">
        {/* 头部 */}
        <div className="bg-white border-b border-gray-100 px-8 py-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <div className="w-10 h-10 bg-yellow-100 rounded-lg flex items-center justify-center">
                <AlertTriangle className="h-5 w-5 text-yellow-600" />
              </div>
              <div>
                <h2 className="text-xl font-bold text-gray-900">最近告警</h2>
                <p className="text-sm text-gray-600">该集群的最新告警信息</p>
              </div>
            </div>
            <Button
              asChild
              className="bg-black text-white hover:bg-gray-800 px-6 py-2 rounded-lg font-medium"
            >
              <Link href={resolveAppPath(`/alerts?cluster_name=${cluster.name}`)}>
                查看全部
              </Link>
            </Button>
          </div>
        </div>

        {/* 内容区域 */}
        <div className="px-8 py-6">
          {alertsLoading ? (
            <div className="flex items-center justify-center py-12">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
              <span className="ml-3 text-gray-600">加载告警数据...</span>
            </div>
          ) : alerts.length === 0 ? (
            <div className="text-center py-12">
              <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
                <CheckCircle className="h-8 w-8 text-green-600" />
              </div>
              <h3 className="text-lg font-semibold text-gray-900 mb-2">暂无告警</h3>
              <p className="text-gray-600">该集群当前运行正常，没有告警信息</p>
            </div>
          ) : (
            <div className="space-y-4">
              {alerts.slice(0, 5).map((alert) => {
                const severityConfig = getSeverityConfig(alert.severity)

                return (
                  <div
                    key={alert.id}
                    className="bg-white border border-gray-200 rounded-xl p-5 hover:shadow-md transition-all duration-200 hover:border-gray-300"
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center space-x-4">
                        {/* 严重级别指示器 */}
                        <div className={`w-3 h-3 rounded-full ${alert.severity === 'critical' ? 'bg-red-500' :
                          alert.severity === 'high' ? 'bg-orange-500' :
                            alert.severity === 'medium' ? 'bg-yellow-500' :
                              'bg-blue-500'
                          }`}></div>

                        <div className="flex-1">
                          <Link
                            href={resolveAppPath(`/alerts/${alert.id}`)}
                            className="text-lg font-semibold text-gray-900 hover:text-blue-600 transition-colors"
                          >
                            {alert.title}
                          </Link>
                          <div className="flex items-center space-x-4 mt-2">
                            <Badge
                              variant={severityConfig.variant}
                              className={`px-3 py-1 text-xs font-medium ${alert.severity === 'critical' ? 'bg-red-100 text-red-800 border-red-200' :
                                alert.severity === 'high' ? 'bg-orange-100 text-orange-800 border-orange-200' :
                                  alert.severity === 'medium' ? 'bg-yellow-100 text-yellow-800 border-yellow-200' :
                                    'bg-blue-100 text-blue-800 border-blue-200'
                                }`}
                            >
                              {severityConfig.label}
                            </Badge>
                            <div className="flex items-center text-sm text-gray-500">
                              <Clock className="h-4 w-4 mr-1" />
                              {formatDistanceToNow(new Date(alert.created_at), {
                                addSuffix: true,
                                locale: zhCN
                              })}
                            </div>
                          </div>
                        </div>
                      </div>

                      <Button
                        asChild
                        variant="outline"
                        size="sm"
                        className="ml-4 px-4 py-2 text-sm font-medium hover:bg-gray-50"
                      >
                        <Link href={resolveAppPath(`/alerts/${alert.id}`)}>
                          查看详情
                        </Link>
                      </Button>
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
