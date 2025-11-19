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
  Server,
  AlertTriangle,
  Clock
} from 'lucide-react'
import RobustaAPI from '@/lib/api'
import { formatDistanceToNow, format } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { toast } from 'sonner'
import { RawPayloadViewer } from '@/components/alerts/RawPayloadViewer'
import { AlertAnalysisIntegration } from '@/components/alerts/AlertAnalysisIntegration'
import { ExternalLink } from 'lucide-react'

// 检测文本是否为URL
const isURL = (text: string): boolean => {
  try {
    const url = new URL(text.trim())
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}

// 渲染注释内容，如果是URL则显示为按钮
const renderAnnotationValue = (value: string) => {
  const trimmedValue = String(value).trim()

  if (isURL(trimmedValue)) {
    return (
      <Button
        variant="outline"
        size="sm"
        className="w-full justify-start text-left font-normal hover:bg-blue-50 hover:border-blue-300 transition-colors"
        asChild
      >
        <a href={trimmedValue} target="_blank" rel="noopener noreferrer">
          <ExternalLink className="h-4 w-4 mr-2 flex-shrink-0" />
          <span className="truncate">{trimmedValue}</span>
        </a>
      </Button>
    )
  }

  return (
    <div className="bg-gray-50 p-3 rounded-lg">
      <p className="text-gray-700 text-sm leading-relaxed whitespace-pre-wrap break-words">
        {trimmedValue}
      </p>
    </div>
  )
}

interface AlertDetailPageProps {
  params: Promise<{
    id: string
  }>
}

export default function AlertDetailPage({ params }: AlertDetailPageProps) {
  const { id } = use(params)

  // 获取告警详情
  const { data: alertData, isLoading, isError, error } = useQuery({
    queryKey: ['alert', id],
    queryFn: () => RobustaAPI.getAlert(id!),
    enabled: !!id,
    retry: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  })

  if (isError) {
    return (
      <div className="space-y-6">
        <div className="flex items-center space-x-4">
          <Button variant="ghost" size="sm" asChild>
            <Link href={resolveAppPath('/alerts')}>
              <ArrowLeft className="h-4 w-4 mr-2" />
              返回告警列表
            </Link>
          </Button>
        </div>
        <Card>
          <CardContent className="py-8">
            <div className="text-center text-muted-foreground space-y-2">
              <p>加载告警详情失败。</p>
              <p className="text-xs text-muted-foreground/80">
                {(error as any)?.message || '请稍后重试。'}
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }



  const alert = alertData?.data

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center space-x-4">
          <Button variant="ghost" size="sm" asChild>
            <Link href={resolveAppPath('/alerts')}>
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
            <Link href={resolveAppPath('/alerts')}>
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



  const severityConfig = getSeverityConfig(alert.severity)
  const statusConfig = getStatusConfig(alert.status)

  return (
    <div className="space-y-6">
      {/* 导航 */}
      <div className="flex items-center justify-between">
        <Button variant="ghost" size="sm" asChild>
          <Link href={resolveAppPath('/alerts')}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            返回告警列表
          </Link>
        </Button>
      </div>

      {/* 告警基本信息 - 精美设计 */}
      <div className="bg-gradient-to-br from-white to-gray-50/50 border border-gray-200 rounded-2xl shadow-sm overflow-hidden">
        {/* 顶部状态栏 */}
        <div className="bg-white border-b border-gray-100 px-8 py-6">
          <div className="flex items-start space-x-4">
            <div className={`w-12 h-12 flex-shrink-0 rounded-xl flex items-center justify-center ${
              alert.status === 'firing'
                ? 'bg-red-100 text-red-600'
                : alert.status === 'resolved'
                ? 'bg-green-100 text-green-600'
                : 'bg-gray-100 text-gray-600'
            }`}>
              <AlertTriangle className="h-6 w-6" />
            </div>
            <div className="flex-1 min-w-0">
              <div className="mb-2">
                <div className="flex items-start gap-3 mb-2">
                  <h1
                    className="text-2xl font-bold text-gray-900 break-words flex-1 min-w-0"
                    title={alert.title}
                  >
                    {alert.title}
                  </h1>
                  <div className="flex items-center gap-2 flex-shrink-0">
                    <Badge
                      variant={severityConfig.variant}
                      className={`px-3 py-1 text-sm font-medium whitespace-nowrap ${
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
                      className={`px-3 py-1 text-sm font-medium whitespace-nowrap ${
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
                </div>
              </div>
              <p className="text-gray-600 font-mono text-sm break-all" title={alert.fingerprint}>
                告警指纹: {alert.fingerprint}
              </p>
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
                {Object.entries(alert.annotations)
                  .filter(([key]) => key !== 'enrichment_keys' && key !== 'investigate_uri')
                  .map(([key, value]) => (
                    <div key={key} className="space-y-2">
                      <span className="font-medium text-gray-900 text-sm">{key}:</span>
                      {renderAnnotationValue(String(value))}
                    </div>
                  ))}
              </div>
            </div>
          </div>
        )}
      </div>

      {/* HolmesGPT 智能分析 */}
      <AlertAnalysisIntegration
        alert={alert}
        defaultTab="enhanced"
      />

    </div>
  )
}
