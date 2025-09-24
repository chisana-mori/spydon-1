'use client'

import Link from 'next/link'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { 
  ArrowLeft, 
  Clock, 
  Server, 
  AlertTriangle,
  Play,
  CheckCircle,
  XCircle,
  Loader2
} from 'lucide-react'
import RobustaAPI from '@/lib/api'
import { formatDistanceToNow, format } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { toast } from 'sonner'

interface AlertDetailPageProps {
  params: {
    id: string
  }
}

export default function AlertDetailPage({ params }: AlertDetailPageProps) {
  const id = params.id
  const queryClient = useQueryClient()

  // 获取告警详情
  const { data: alertData, isLoading } = useQuery({
    queryKey: ['alert', id],
    queryFn: () => RobustaAPI.getAlert(id!),
    enabled: !!id,
  })

  // 获取RCA结果
  const { data: rcaData, isLoading: rcaLoading } = useQuery({
    queryKey: ['rca', id],
    queryFn: () => RobustaAPI.getRCAByAlertId(id!),
    enabled: !!id,
  })

  // 触发RCA分析
  const triggerRCAMutation = useMutation({
    mutationFn: () => RobustaAPI.triggerRCA(id!),
    onSuccess: () => {
      toast.success('RCA分析已触发')
      queryClient.invalidateQueries({ queryKey: ['rca', id] })
    },
    onError: (error: any) => {
      toast.error(`触发RCA失败: ${error.response?.data?.error || error.message}`)
    },
  })

  const alert = alertData?.data
  const rcaRuns = rcaData?.data || []

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center space-x-4">
          <Button variant="ghost" size="sm" asChild>
            <Link href="/alerts">
              <ArrowLeft className="h-4 w-4 mr-2" />
              返回告警列表
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

  if (!alert) {
    return (
      <div className="space-y-6">
        <div className="flex items-center space-x-4">
          <Button variant="ghost" size="sm" asChild>
            <Link href="/alerts">
              <ArrowLeft className="h-4 w-4 mr-2" />
              返回告警列表
            </Link>
          </Button>
        </div>
        <Card>
          <CardContent className="py-8">
            <div className="text-center text-muted-foreground">
              告警不存在或已被删除
            </div>
          </CardContent>
        </Card>
      </div>
    )
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

  // 获取状态配置
  const getStatusConfig = (status: string) => {
    switch (status) {
      case 'firing':
        return { variant: 'destructive' as const, label: '触发中' }
      case 'resolved':
        return { variant: 'secondary' as const, label: '已解决' }
      case 'silenced':
        return { variant: 'outline' as const, label: '已静默' }
      default:
        return { variant: 'outline' as const, label: status }
    }
  }

  // 获取RCA状态配置
  const getRCAStatusConfig = (status: string) => {
    switch (status) {
      case 'completed':
        return { variant: 'secondary' as const, label: '已完成', icon: CheckCircle }
      case 'running':
        return { variant: 'default' as const, label: '运行中', icon: Loader2 }
      case 'failed':
        return { variant: 'destructive' as const, label: '失败', icon: XCircle }
      case 'timeout':
        return { variant: 'destructive' as const, label: '超时', icon: Clock }
      case 'pending':
        return { variant: 'outline' as const, label: '等待中', icon: Clock }
      default:
        return { variant: 'outline' as const, label: status, icon: AlertTriangle }
    }
  }

  const severityConfig = getSeverityConfig(alert.severity)
  const statusConfig = getStatusConfig(alert.status)

  return (
    <div className="space-y-6">
      {/* 导航 */}
      <div className="flex items-center justify-between">
        <Button variant="ghost" size="sm" asChild>
          <Link href="/alerts">
            <ArrowLeft className="h-4 w-4 mr-2" />
            返回告警列表
          </Link>
        </Button>
        
        <Button 
          onClick={() => triggerRCAMutation.mutate()}
          disabled={triggerRCAMutation.isPending}
        >
          {triggerRCAMutation.isPending ? (
            <Loader2 className="h-4 w-4 mr-2 animate-spin" />
          ) : (
            <Play className="h-4 w-4 mr-2" />
          )}
          触发RCA分析
        </Button>
      </div>

      {/* 告警基本信息 - 精美设计 */}
      <div className="bg-gradient-to-br from-white to-gray-50/50 border border-gray-200 rounded-2xl shadow-sm overflow-hidden">
        {/* 顶部状态栏 */}
        <div className="bg-white border-b border-gray-100 px-8 py-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <div className={`w-12 h-12 rounded-xl flex items-center justify-center ${
                alert.status === 'firing'
                  ? 'bg-red-100 text-red-600'
                  : alert.status === 'resolved'
                  ? 'bg-green-100 text-green-600'
                  : 'bg-gray-100 text-gray-600'
              }`}>
                <AlertTriangle className="h-6 w-6" />
              </div>
              <div>
                <div className="flex items-center space-x-3 mb-1">
                  <h1 className="text-2xl font-bold text-gray-900">{alert.title}</h1>
                  <Badge
                    variant={severityConfig.variant}
                    className={`px-3 py-1 text-sm font-medium ${
                      alert.severity === 'critical'
                        ? 'bg-red-100 text-red-800 border-red-200'
                        : alert.severity === 'high'
                        ? 'bg-orange-100 text-orange-800 border-orange-200'
                        : alert.severity === 'medium'
                        ? 'bg-yellow-100 text-yellow-800 border-yellow-200'
                        : 'bg-blue-100 text-blue-800 border-blue-200'
                    }`}
                  >
                    {severityConfig.label}
                  </Badge>
                  <Badge
                    variant={statusConfig.variant}
                    className={`px-3 py-1 text-sm font-medium ${
                      alert.status === 'firing'
                        ? 'bg-red-100 text-red-800 border-red-200'
                        : alert.status === 'resolved'
                        ? 'bg-green-100 text-green-800 border-green-200'
                        : 'bg-gray-100 text-gray-800 border-gray-200'
                    }`}
                  >
                    {statusConfig.label}
                  </Badge>
                </div>
                <p className="text-gray-600 font-mono text-sm">
                  告警指纹: {alert.fingerprint}
                </p>
              </div>
            </div>
          </div>
        </div>

        {/* 主要内容区域 */}
        <div className="px-8 py-6">
          {alert.description && (
            <div className="mb-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-3">描述</h3>
              <div className="bg-blue-50 border border-blue-100 rounded-xl p-4">
                <p className="text-gray-700 leading-relaxed">{alert.description}</p>
              </div>
            </div>
          )}

          {/* 信息网格 - 简化设计 */}
          <div className="grid gap-6 md:grid-cols-2">
            {/* 左侧时间信息 */}
            <div>
              <h3 className="text-lg font-semibold text-gray-900 mb-4">时间信息</h3>
              <div className="space-y-3">
                <div className="flex items-center space-x-3">
                  <div className="w-8 h-8 bg-green-100 rounded-lg flex items-center justify-center">
                    <Clock className="h-4 w-4 text-green-600" />
                  </div>
                  <div className="flex-1">
                    <p className="text-sm font-medium text-gray-900">创建时间</p>
                    <p className="text-sm text-gray-600 font-mono">
                      {format(new Date(alert.created_at), 'yyyy-MM-dd HH:mm:ss')}
                    </p>
                  </div>
                </div>

                {alert.starts_at && (
                  <div className="flex items-center space-x-3">
                    <div className="w-8 h-8 bg-orange-100 rounded-lg flex items-center justify-center">
                      <Clock className="h-4 w-4 text-orange-600" />
                    </div>
                    <div className="flex-1">
                      <p className="text-sm font-medium text-gray-900">开始时间</p>
                      <p className="text-sm text-gray-600 font-mono">
                        {format(new Date(alert.starts_at), 'yyyy-MM-dd HH:mm:ss')}
                      </p>
                    </div>
                  </div>
                )}

                {alert.ends_at && (
                  <div className="flex items-center space-x-3">
                    <div className="w-8 h-8 bg-red-100 rounded-lg flex items-center justify-center">
                      <Clock className="h-4 w-4 text-red-600" />
                    </div>
                    <div className="flex-1">
                      <p className="text-sm font-medium text-gray-900">结束时间</p>
                      <p className="text-sm text-gray-600 font-mono">
                        {format(new Date(alert.ends_at), 'yyyy-MM-dd HH:mm:ss')}
                      </p>
                    </div>
                  </div>
                )}
              </div>
            </div>

            {/* 右侧集群信息 */}
            <div>
              <h3 className="text-lg font-semibold text-gray-900 mb-4">集群信息</h3>
              <div className="space-y-3">
                <div className="flex items-center space-x-3">
                  <div className="w-8 h-8 bg-blue-100 rounded-lg flex items-center justify-center">
                    <Server className="h-4 w-4 text-blue-600" />
                  </div>
                  <div className="flex-1">
                    <p className="text-sm font-medium text-gray-900">集群</p>
                    <p className="text-sm text-gray-600 font-mono">
                      {alert.cluster?.name || alert.cluster_id}
                    </p>
                  </div>
                </div>

                <div className="flex items-center space-x-3">
                  <div className="w-8 h-8 bg-purple-100 rounded-lg flex items-center justify-center">
                    <Clock className="h-4 w-4 text-purple-600" />
                  </div>
                  <div className="flex-1">
                    <p className="text-sm font-medium text-gray-900">持续时间</p>
                    <p className="text-sm text-gray-600">
                      {formatDistanceToNow(new Date(alert.created_at), { locale: zhCN })}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* 标签和注释 */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* 标签 */}
        {alert.labels && Object.keys(alert.labels).length > 0 && (
          <div className="bg-gradient-to-br from-white to-gray-50/50 border border-gray-200 rounded-2xl shadow-sm overflow-hidden">
            <div className="bg-white border-b border-gray-100 px-6 py-4">
              <h3 className="text-lg font-semibold text-gray-900">标签</h3>
            </div>
            <div className="px-6 py-4">
              <div className="space-y-3">
                {Object.entries(alert.labels).map(([key, value]) => (
                  <div key={key} className="flex items-center justify-between py-2 border-b border-gray-100 last:border-b-0">
                    <span className="font-medium text-gray-900 text-sm">{key}:</span>
                    <span className="text-gray-600 text-sm font-mono bg-gray-50 px-2 py-1 rounded">
                      {String(value)}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}

        {/* 注释 */}
        {alert.annotations && Object.keys(alert.annotations).length > 0 && (
          <div className="bg-gradient-to-br from-white to-gray-50/50 border border-gray-200 rounded-2xl shadow-sm overflow-hidden">
            <div className="bg-white border-b border-gray-100 px-6 py-4">
              <h3 className="text-lg font-semibold text-gray-900">注释</h3>
            </div>
            <div className="px-6 py-4">
              <div className="space-y-4">
                {Object.entries(alert.annotations).map(([key, value]) => (
                  <div key={key} className="space-y-2">
                    <span className="font-medium text-gray-900 text-sm">{key}:</span>
                    <div className="bg-gray-50 p-3 rounded-lg">
                      <p className="text-gray-700 text-sm leading-relaxed">
                        {String(value)}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}
      </div>

      {/* RCA分析结果 */}
      <div className="bg-gradient-to-br from-white to-gray-50/50 border border-gray-200 rounded-2xl shadow-sm overflow-hidden">
        <div className="bg-white border-b border-gray-100 px-6 py-4">
          <h3 className="text-lg font-semibold text-gray-900">根因分析 (RCA)</h3>
          <p className="text-sm text-gray-600 mt-1">自动根因分析结果和建议</p>
        </div>
        <div className="px-6 py-4">
          {rcaLoading ? (
            <div className="text-center py-8">
              <Loader2 className="h-8 w-8 animate-spin mx-auto mb-4 text-blue-600" />
              <p className="text-gray-600">加载RCA数据中...</p>
            </div>
          ) : rcaRuns.length === 0 ? (
            <div className="text-center py-12">
              <div className="w-16 h-16 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-4">
                <AlertTriangle className="h-8 w-8 text-gray-400" />
              </div>
              <p className="text-gray-900 font-medium mb-2">暂无RCA分析结果</p>
              <p className="text-sm text-gray-600">点击上方按钮触发分析</p>
            </div>
          ) : (
            <div className="space-y-4">
              {rcaRuns.map((rca) => {
                const rcaStatusConfig = getRCAStatusConfig(rca.status)
                const StatusIcon = rcaStatusConfig.icon

                return (
                  <div key={rca.id} className="border border-gray-200 rounded-lg p-4">
                    <div className="flex items-center justify-between mb-3">
                      <div className="flex items-center space-x-3">
                        <div className={`w-8 h-8 rounded-lg flex items-center justify-center ${
                          rca.status === 'completed'
                            ? 'bg-green-100 text-green-600'
                            : rca.status === 'running'
                            ? 'bg-blue-100 text-blue-600'
                            : rca.status === 'failed'
                            ? 'bg-red-100 text-red-600'
                            : 'bg-gray-100 text-gray-600'
                        }`}>
                          <StatusIcon className={`h-4 w-4 ${
                            rca.status === 'running' ? 'animate-spin' : ''
                          }`} />
                        </div>
                        <Badge
                          variant={rcaStatusConfig.variant}
                          className={`px-3 py-1 text-sm font-medium ${
                            rca.status === 'completed'
                              ? 'bg-green-100 text-green-800 border-green-200'
                              : rca.status === 'running'
                              ? 'bg-blue-100 text-blue-800 border-blue-200'
                              : rca.status === 'failed'
                              ? 'bg-red-100 text-red-800 border-red-200'
                              : 'bg-gray-100 text-gray-800 border-gray-200'
                          }`}
                        >
                          {rcaStatusConfig.label}
                        </Badge>
                      </div>
                      <span className="text-sm text-gray-500 font-mono">
                        {format(new Date(rca.started_at), 'yyyy-MM-dd HH:mm:ss')}
                      </span>
                    </div>

                    {rca.summary && (
                      <div className="mb-3">
                        <h5 className="font-medium text-gray-900 mb-2">分析摘要</h5>
                        <div className="bg-blue-50 p-3 rounded-lg">
                          <p className="text-gray-700 text-sm leading-relaxed">{rca.summary}</p>
                        </div>
                      </div>
                    )}

                    {rca.error_message && (
                      <div className="mb-3">
                        <h5 className="font-medium text-red-900 mb-2">错误信息</h5>
                        <div className="bg-red-50 p-3 rounded-lg">
                          <p className="text-red-700 text-sm leading-relaxed">{rca.error_message}</p>
                        </div>
                      </div>
                    )}
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
