'use client'

import React, { useState, useEffect } from 'react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Brain,
  Clock,
  ExternalLink,
  CheckCircle2,
  XCircle,
  Loader2,
  AlertCircle,
  History
} from 'lucide-react'
import { Alert } from '@/types/api'
import AlertKnowledgePanel from '@/components/alerts/AlertKnowledgePanel'
import { appConfig } from '@/config'
import { EnhancedHolmesGPTChat } from './EnhancedHolmesGPTChat'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'

interface AlertAnalysisIntegrationProps {
  alert: Alert
  defaultTab?: 'enhanced' | 'knowledge'
}

export const AlertAnalysisIntegration: React.FC<AlertAnalysisIntegrationProps> = ({
  alert,
  defaultTab = 'enhanced'
}) => {
  const [activeTab, setActiveTab] = useState(defaultTab)
  const [analysisHistory, setAnalysisHistory] = useState<any[]>([])
  const [cachedResult, setCachedResult] = useState<any>(null)
  const [loadingCache, setLoadingCache] = useState(true)
  const [selectedHistoryId, setSelectedHistoryId] = useState<string | null>(null)

  // 获取分析历史记录和缓存
  useEffect(() => {
    const fetchHistoryAndCache = async () => {
      setLoadingCache(true)
      try {
        // 构建 API 路径，包含 basePath（如果配置了）
        const basePath = appConfig.basePath || ''

        // 1. 先获取分析历史列表
        const historyResponse = await fetch(`${basePath}/api/v1/rca/${alert.id}`, {
          credentials: 'include',
        })

        if (historyResponse.ok) {
          const historyData = await historyResponse.json()
          if (historyData.data && Array.isArray(historyData.data)) {
            setAnalysisHistory(historyData.data)
          }
        }

        // 2. 尝试获取缓存的RCA结果（用于自动回放）
        const cacheResponse = await fetch(`${basePath}/api/v1/rca/${alert.id}/cache`, {
          credentials: 'include',
        })

        if (cacheResponse.ok) {
          const cacheData = await cacheResponse.json()
          if (cacheData.cache_hit && cacheData.data) {
            setCachedResult(cacheData.data)
          } else {
            setCachedResult(null)
          }
        } else if (cacheResponse.status === 404) {
          setCachedResult(null)
        }
      } catch (error) {
        // Failed to fetch analysis history or cache
      } finally {
        setLoadingCache(false)
      }
    }

    fetchHistoryAndCache()
  }, [alert.id])

  // 查看历史分析结果
  const handleViewHistory = async (historyId: string) => {
    setSelectedHistoryId(historyId)
    setLoadingCache(true)

    try {
      const basePath = appConfig.basePath || ''
      const response = await fetch(`${basePath}/api/v1/rca/${alert.id}/cache?run_id=${historyId}`, {
        credentials: 'include',
      })

      if (response.ok) {
        const data = await response.json()
        if (data.cache_hit && data.data) {
          setCachedResult(data.data)
        }
      } else if (response.status === 404) {
        setCachedResult({
          run_id: historyId,
          alert_id: alert.id,
          cached_at: new Date().toISOString(),
          metadata: {
            summary: '该次分析尚未生成可回放的缓存数据'
          },
          stream_chunks: []
        })
      }

      window.scrollTo({ top: 0, behavior: 'smooth' })
      setActiveTab('enhanced')
    } catch (error) {
      // Failed to view history
    } finally {
      setLoadingCache(false)
    }
  }

  // 获取状态显示信息
  const getStatusInfo = (status: string) => {
    switch (status) {
      case 'completed':
        return {
          icon: CheckCircle2,
          color: 'text-green-600',
          bgColor: 'bg-green-50',
          borderColor: 'border-green-200',
          label: '已完成',
          badgeVariant: 'default' as const
        }
      case 'running':
        return {
          icon: Loader2,
          color: 'text-blue-600',
          bgColor: 'bg-blue-50',
          borderColor: 'border-blue-200',
          label: '运行中',
          badgeVariant: 'secondary' as const
        }
      case 'failed':
        return {
          icon: XCircle,
          color: 'text-red-600',
          bgColor: 'bg-red-50',
          borderColor: 'border-red-200',
          label: '失败',
          badgeVariant: 'destructive' as const
        }
      default:
        return {
          icon: AlertCircle,
          color: 'text-gray-600',
          bgColor: 'bg-gray-50',
          borderColor: 'border-gray-200',
          label: status,
          badgeVariant: 'outline' as const
        }
    }
  }


  return (
    <div className="space-y-6">
      {/* 提示信息 */}
      {cachedResult && (
        <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 flex items-center space-x-3">
          <CheckCircle2 className="h-5 w-5 text-blue-600 flex-shrink-0" />
          <div className="flex-1">
            <p className="text-sm text-blue-900">
              <span className="font-semibold">已加载历史分析结果</span>
              <span className="ml-2 text-blue-700">
                缓存时间: {format(new Date(cachedResult.cached_at), 'yyyy-MM-dd HH:mm:ss', { locale: zhCN })}
              </span>
            </p>
          </div>
        </div>
      )}

      {/* 分析历史 */}
      {analysisHistory.length > 0 && (
        <Card className="border-2 shadow-sm">
          <CardHeader className="bg-gradient-to-r from-gray-50 to-white">
            <CardTitle className="flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <History className="h-5 w-5 text-blue-600" />
                <span className="text-lg">分析历史</span>
                <Badge variant="secondary" className="ml-2">
                  {analysisHistory.length} 条记录
                </Badge>
              </div>
            </CardTitle>
          </CardHeader>
          <CardContent className="pt-6">
            <div className="space-y-3">
              {analysisHistory.slice(0, 3).map((history, index) => {
                // 使用 completed_at 或 started_at 或 created_at 作为时间戳
                const timestamp = history.completed_at || history.started_at || history.created_at
                const statusInfo = getStatusInfo(history.status)
                const StatusIcon = statusInfo.icon
                const isSelected = selectedHistoryId === history.id

                return (
                  <div
                    key={history.id || index}
                    className={`
                      group relative flex items-center justify-between p-4 rounded-lg border-2 transition-all duration-200
                      ${isSelected ? 'border-blue-500 bg-blue-50' : `${statusInfo.borderColor} ${statusInfo.bgColor} hover:shadow-md hover:border-blue-300`}
                    `}
                  >
                    {/* 左侧内容 */}
                    <div className="flex items-center space-x-4 flex-1">
                      {/* 状态图标 */}
                      <div className={`flex-shrink-0 ${statusInfo.color}`}>
                        <StatusIcon className={`h-5 w-5 ${history.status === 'running' ? 'animate-spin' : ''}`} />
                      </div>

                      {/* 分析信息 */}
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center space-x-2 mb-1">
                          <span className="font-semibold text-gray-900">
                            分析 #{analysisHistory.length - index}
                          </span>
                          <Badge variant={statusInfo.badgeVariant} className="text-xs">
                            {statusInfo.label}
                          </Badge>
                        </div>

                        <div className="flex items-center space-x-4 text-sm text-gray-600">
                          <div className="flex items-center space-x-1">
                            <Clock className="h-3.5 w-3.5" />
                            <span>
                              {timestamp ? format(new Date(timestamp), 'yyyy-MM-dd HH:mm:ss', { locale: zhCN }) : '时间未知'}
                            </span>
                          </div>

                          {history.completed_at && history.started_at && (
                            <div className="text-xs text-gray-500">
                              耗时: {Math.round((new Date(history.completed_at).getTime() - new Date(history.started_at).getTime()) / 1000)}秒
                            </div>
                          )}
                        </div>

                        {/* 摘要信息 */}
                        {history.summary && (
                          <div className="mt-2 text-sm text-gray-700 line-clamp-2">
                            {history.summary}
                          </div>
                        )}
                      </div>
                    </div>

                    {/* 右侧按钮 */}
                    <div className="flex-shrink-0 ml-4">
                      <Button
                        variant={isSelected ? "default" : "outline"}
                        size="sm"
                        onClick={() => handleViewHistory(history.id)}
                        className="group-hover:shadow-sm transition-all"
                      >
                        <ExternalLink className="h-4 w-4 mr-2" />
                        查看结果
                      </Button>
                    </div>
                  </div>
                )
              })}
            </div>
          </CardContent>
        </Card>
      )}

      {/* 分析选项卡 - 图书标签样式 */}
      <div className="min-h-[600px]">
        <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as typeof activeTab)} className="h-full">
          {/* 图书标签样式的选项卡 */}
          <div className="flex items-end space-x-1 border-b border-border">
            <button
              onClick={() => setActiveTab('enhanced')}
              className={`
                relative px-6 py-3 rounded-t-lg font-medium transition-all duration-200
                flex items-center space-x-2
                ${activeTab === 'enhanced'
                  ? 'bg-card text-foreground border-t border-l border-r border-border -mb-px'
                  : 'bg-muted/50 text-muted-foreground hover:bg-muted hover:text-foreground'
                }
              `}
            >
              <Brain className="h-4 w-4" />
              <span>根因分析</span>
              {activeTab === 'enhanced' && (
                <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-primary" />
              )}
            </button>
            <button
              onClick={() => setActiveTab('knowledge')}
              className={`
                relative px-6 py-3 rounded-t-lg font-medium transition-all duration-200
                flex items-center space-x-2
                ${activeTab === 'knowledge'
                  ? 'bg-card text-foreground border-t border-l border-r border-border -mb-px'
                  : 'bg-muted/50 text-muted-foreground hover:bg-muted hover:text-foreground'
                }
              `}
            >
              <History className="h-4 w-4" />
              <span>处理指南</span>
              {activeTab === 'knowledge' && (
                <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-primary" />
              )}
            </button>
          </div>

          {/* 内容区域 */}
          <Card className="rounded-tl-none border-t-0">
            <CardContent className="p-6">
              {activeTab === 'enhanced' && (
                <div className="animate-in fade-in-50 duration-300">
                  <EnhancedHolmesGPTChat
                    alert={alert}
                    showCard={false}
                    cachedResult={cachedResult}
                    loadingCache={loadingCache}
                  />
                </div>
              )}
              {activeTab === 'knowledge' && (
                <div className="animate-in fade-in-50 duration-300 min-h-[400px]">
                  <AlertKnowledgePanel alertRuleName={alert.title} />
                </div>
              )}
            </CardContent>
          </Card>
        </Tabs>
      </div>

    </div>
  )
}
