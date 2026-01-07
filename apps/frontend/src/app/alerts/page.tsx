'use client'

import { useState, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import Link from 'next/link'
import { resolveAppPath } from '@/config'
import { Button } from '@/components/ui/button'

import { Input } from '@/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'

import {
  AlertTriangle,
  Search,
  Clock,
  Calendar,
  CheckCircle,
  Activity,
  BarChart3,
  ChevronLeft,
  ChevronRight,
  CalendarDays,
  Layers,
  ChevronDown,
  X,
  ExternalLink
} from 'lucide-react'
import { Checkbox } from '@/components/ui/checkbox'
import RobustaAPI from '@/lib/api'
import { format, subDays, addHours, parseISO } from 'date-fns'
import type { AlertFilters } from '@/types/api'

export default function Alerts() {
  const [filters, setFilters] = useState<AlertFilters>({ status: 'firing' })
  const [searchTerm, setSearchTerm] = useState('')
  const [clusterSearch, setClusterSearch] = useState('')
  const [selectedClusters, setSelectedClusters] = useState<string[]>([])
  const [timeRange, setTimeRange] = useState(24) // 默认显示24小时
  const [currentTime, setCurrentTime] = useState(new Date())
  const [useAbsoluteTime, setUseAbsoluteTime] = useState(false)
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')

  // 计算时间范围
  const timeRangeStart = useMemo(() => {
    if (useAbsoluteTime && startDate) {
      return new Date(startDate)
    }
    return subDays(currentTime, timeRange / 24)
  }, [useAbsoluteTime, startDate, currentTime, timeRange])

  const timeRangeEnd = useMemo(() => {
    if (useAbsoluteTime && endDate) {
      return new Date(endDate)
    }
    return currentTime
  }, [useAbsoluteTime, endDate, currentTime])

  // 获取告警列表 - 获取更多数据用于时间线展示
  const { data: alertsData, isLoading } = useQuery({
    queryKey: ['alerts-timeline', filters, searchTerm, timeRange, selectedClusters],
    queryFn: () => RobustaAPI.getAlerts(1, 1000, {
      ...filters,
      keyword: searchTerm || undefined,
      since: timeRangeStart.toISOString(),
    }),
    refetchInterval: 30000, // 30秒自动刷新
  })

  // 获取集群列表用于过滤
  const { data: clustersData } = useQuery({
    queryKey: ['clusters-for-filter'],
    queryFn: () => RobustaAPI.getClusters(1, 100),
  })

  const alerts = useMemo(() => alertsData?.data || [], [alertsData?.data])
  const clusters = clustersData?.data || []

  // 过滤集群列表
  const filteredClusters = useMemo(() => {
    if (!clusterSearch) return clusters;
    return clusters.filter(cluster =>
      cluster.name.toLowerCase().includes(clusterSearch.toLowerCase()) ||
      (cluster.cluster_id && cluster.cluster_id.toLowerCase().includes(clusterSearch.toLowerCase()))
    );
  }, [clusters, clusterSearch]);

  const handleFilterChange = (key: keyof AlertFilters, value: string) => {
    setFilters(prev => ({
      ...prev,
      [key]: value || undefined
    }))
  }

  // 时间线相关计算
  const timelineAlerts = useMemo(() => {
    return alerts
      .filter(alert => {
        const alertTime = parseISO(alert.starts_at || alert.created_at)
        const timeMatch = alertTime >= timeRangeStart && alertTime <= timeRangeEnd
        const clusterMatch = selectedClusters.length === 0 || selectedClusters.includes(alert.cluster_name)
        return timeMatch && clusterMatch
      })
      .sort((a, b) => {
        const timeA = parseISO(a.starts_at || a.created_at)
        const timeB = parseISO(b.starts_at || b.created_at)
        return timeB.getTime() - timeA.getTime() // 改为倒序：最新的在前面
      })
  }, [alerts, timeRangeStart, timeRangeEnd, selectedClusters])

  // 切换集群选择
  const toggleCluster = (clusterId: string) => {
    setSelectedClusters(prev =>
      prev.includes(clusterId)
        ? prev.filter(id => id !== clusterId)
        : [...prev, clusterId]
    )
  }

  // 清空集群选择
  const clearClusters = () => {
    setSelectedClusters([])
    setClusterSearch('')
  }



  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3 mb-6">
        <div className="p-2.5 rounded-xl bg-blue-500/10 text-blue-500 ring-1 ring-blue-500/20">
          <AlertTriangle className="h-6 w-6" />
        </div>
        <div>
          <h1 className="text-2xl font-bold tracking-tight">告警中心</h1>
          <p className="text-sm text-muted-foreground mt-0.5">
            监控和管理系统的所有告警事件
          </p>
        </div>
      </div>

      {/* 顶部筛选栏 */}
      <div className="bg-card border rounded-xl shadow-sm p-4">
        {/* 时间范围选择 */}
        <div className="flex items-center gap-2 mb-4">
          <Calendar className="h-4 w-4 text-muted-foreground" />
          <span className="text-sm text-muted-foreground mr-2">时间范围</span>
          <div className="flex gap-1">
            {[
              { value: 1, label: '最近1小时' },
              { value: 6, label: '最近6小时' },
              { value: 24, label: '最近24小时' },
              { value: 72, label: '最近3天' },
              { value: 168, label: '最近7天' }
            ].map((option) => (
              <Button
                key={option.value}
                variant={!useAbsoluteTime && timeRange === option.value ? "default" : "ghost"}
                size="sm"
                onClick={() => {
                  setUseAbsoluteTime(false)
                  setTimeRange(option.value)
                }}
                className="h-8 text-xs"
              >
                {option.label}
              </Button>
            ))}

            {/* 绝对日期选择 */}
            <Popover>
              <PopoverTrigger asChild>
                <Button
                  variant={useAbsoluteTime ? "default" : "ghost"}
                  size="sm"
                  className="h-8 text-xs flex items-center gap-1"
                >
                  <CalendarDays className="h-3 w-3" />
                  绝对日期
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-80" align="start">
                <div className="space-y-3">
                  <h4 className="font-medium text-sm">选择时间范围</h4>
                  <div className="grid gap-2">
                    <div className="grid grid-cols-3 items-center gap-2">
                      <label htmlFor="start-date" className="text-sm">开始时间</label>
                      <Input
                        id="start-date"
                        type="datetime-local"
                        value={startDate}
                        onChange={(e) => setStartDate(e.target.value)}
                        className="col-span-2 h-8 text-xs"
                      />
                    </div>
                    <div className="grid grid-cols-3 items-center gap-2">
                      <label htmlFor="end-date" className="text-sm">结束时间</label>
                      <Input
                        id="end-date"
                        type="datetime-local"
                        value={endDate}
                        onChange={(e) => setEndDate(e.target.value)}
                        className="col-span-2 h-8 text-xs"
                      />
                    </div>
                    <Button
                      size="sm"
                      onClick={() => {
                        if (startDate && endDate) {
                          setUseAbsoluteTime(true)
                        }
                      }}
                      disabled={!startDate || !endDate}
                      className="h-7 text-xs"
                    >
                      应用时间范围
                    </Button>
                  </div>
                </div>
              </PopoverContent>
            </Popover>
          </div>

          {/* 搜索框 */}
          <div className="flex items-center gap-2 ml-auto">
            <div className="relative">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
              <Input
                placeholder="搜索告警..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="h-8 w-64 text-sm pl-8"
              />
            </div>
          </div>
        </div>

        {/* 筛选标签行 */}
        <div className="flex items-center gap-6 flex-wrap">
          {/* 集群筛选 */}
          <div className="flex items-center gap-2 flex-shrink-0">
            <Layers className="h-4 w-4 text-muted-foreground" />
            <span className="text-sm text-muted-foreground">集群</span>
            <div className="flex items-center gap-1 flex-wrap">
              <Popover>
                <PopoverTrigger asChild>
                  <Button
                    variant="outline"
                    size="sm"
                    className="h-7 px-3 text-xs rounded-full flex items-center gap-2"
                  >
                    {selectedClusters.length === 0
                      ? '所有集群'
                      : `已选 ${selectedClusters.length} 个集群`}
                    <ChevronDown className="h-3 w-3" />
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-72 p-0" align="start">
                  <div className="p-3 space-y-3">
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium">选择集群</span>
                      {selectedClusters.length > 0 && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={clearClusters}
                          className="h-6 px-2 text-xs"
                        >
                          清空
                        </Button>
                      )}
                    </div>
                    <Input
                      placeholder="搜索集群..."
                      className="h-8 text-sm"
                      value={clusterSearch}
                      onChange={(e) => setClusterSearch(e.target.value)}
                    />
                    <div className="max-h-64 overflow-y-auto space-y-1">
                      {filteredClusters.map((cluster) => (
                        <div
                          key={cluster.name}
                          className="flex items-center gap-2 px-2 py-2 rounded hover:bg-muted/50 cursor-pointer text-sm"
                          onClick={() => toggleCluster(cluster.name)}
                        >
                          <Checkbox
                            checked={selectedClusters.includes(cluster.name)}
                            onCheckedChange={() => toggleCluster(cluster.name)}
                            className="pointer-events-none"
                          />
                          <div className="flex-1 min-w-0">
                            <div className="font-medium truncate">{cluster.name}</div>
                            {cluster.cluster_id && (
                              <div className="text-xs text-muted-foreground truncate">{cluster.cluster_id}</div>
                            )}
                          </div>
                        </div>
                      ))}
                      {filteredClusters.length === 0 && clusterSearch && (
                        <div className="px-2 py-4 text-sm text-muted-foreground text-center">
                          未找到匹配的集群
                        </div>
                      )}
                    </div>
                  </div>
                </PopoverContent>
              </Popover>

              {/* 显示已选集群标签 */}
              {selectedClusters.length > 0 && (
                <div className="flex items-center gap-1 flex-wrap">
                  {selectedClusters.slice(0, 2).map((clusterId) => {
                    const cluster = clusters.find(c => c.name === clusterId);
                    return (
                      <div
                        key={clusterId}
                        className="inline-flex items-center gap-1 h-7 px-2 text-xs rounded-full bg-primary text-primary-foreground max-w-[200px]"
                        title={cluster?.name || clusterId}
                      >
                        <span className="truncate">{cluster?.name || clusterId}</span>
                        <button
                          onClick={() => toggleCluster(clusterId)}
                          className="hover:bg-primary-foreground/20 rounded-full p-0.5 flex-shrink-0"
                        >
                          <X className="h-3 w-3" />
                        </button>
                      </div>
                    );
                  })}
                  {selectedClusters.length > 2 && (
                    <span className="text-xs text-muted-foreground flex-shrink-0">
                      +{selectedClusters.length - 2}
                    </span>
                  )}
                </div>
              )}
            </div>
          </div>

          {/* 严重级别筛选 */}
          <div className="flex items-center gap-2">
            <AlertTriangle className="h-4 w-4 text-muted-foreground" />
            <span className="text-sm text-muted-foreground">严重级别</span>
            <div className="flex gap-1">
              {[
                { value: '', label: '所有', color: 'bg-slate-400' },
                { value: 'critical', label: '严重', color: 'bg-red-500' },
                { value: 'high', label: '高', color: 'bg-orange-500' },
                { value: 'medium', label: '中', color: 'bg-yellow-500' },
                { value: 'low', label: '低', color: 'bg-blue-500' }
              ].map((option) => (
                <Button
                  key={option.value}
                  variant={(filters.severity || '') === option.value ? "default" : "ghost"}
                  size="sm"
                  onClick={() => handleFilterChange('severity', option.value)}
                  className={`h-7 px-3 text-xs rounded-full flex items-center gap-1.5`}
                >
                  <div className={`w-2 h-2 rounded-full ${option.color}`} />
                  {option.label}
                </Button>
              ))}
            </div>
          </div>

          {/* 状态筛选 */}
          <div className="flex items-center gap-2">
            <Activity className="h-4 w-4 text-muted-foreground" />
            <span className="text-sm text-muted-foreground">状态</span>
            <div className="flex gap-1">
              {[
                { value: '', label: '所有', color: 'bg-slate-400' },
                { value: 'firing', label: '触发中', color: 'bg-red-500' },
                { value: 'resolved', label: '已解决', color: 'bg-green-500' },
                { value: 'silenced', label: '已静默', color: 'bg-slate-500' }
              ].map((option) => (
                <Button
                  key={option.value}
                  variant={(filters.status || '') === option.value ? "default" : "ghost"}
                  size="sm"
                  onClick={() => handleFilterChange('status', option.value)}
                  className={`h-7 px-3 text-xs rounded-full flex items-center gap-1.5`}
                >
                  <div className={`w-2 h-2 rounded-full ${option.color}`} />
                  {option.label}
                </Button>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* 告警时间线 */}
      <div className="bg-card border rounded-xl shadow-sm overflow-hidden">
        {/* 标题栏 */}
        <div className="flex items-center justify-between p-4 border-b bg-muted/30">
          <div className="flex items-center gap-2">
            <BarChart3 className="h-5 w-5 text-muted-foreground" />
            <h2 className="text-lg font-semibold tracking-tight">告警时间线</h2>
            <span className="text-sm text-muted-foreground">({timelineAlerts.length} 条告警)</span>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setCurrentTime(subDays(currentTime, 1))}
              className="h-8 w-8 p-0"
            >
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <div className="text-sm font-medium text-muted-foreground px-2">
              {format(timeRangeStart, 'MM/dd HH:mm')} - {format(timeRangeEnd, 'MM/dd HH:mm')}
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setCurrentTime(addHours(currentTime, 1))}
              disabled={currentTime >= new Date()}
              className="h-8 w-8 p-0"
            >
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </div>

        {/* 内容区域 */}
        <div className="p-0">
          {isLoading ? (
            <div className="flex items-center justify-center py-16">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
              <span className="ml-2 text-muted-foreground">加载告警数据...</span>
            </div>
          ) : timelineAlerts.length === 0 ? (
            <div className="text-center py-16">
              <div className="bg-green-500/10 h-16 w-16 rounded-full flex items-center justify-center mx-auto mb-4">
                <CheckCircle className="h-8 w-8 text-green-600" />
              </div>
              <h3 className="text-lg font-medium mb-1">当前时间范围内无告警</h3>
              <p className="text-muted-foreground text-sm">系统运行正常，或尝试调整时间范围查看更多数据</p>
            </div>
          ) : (
            <div className="divide-y divide-border">
              {timelineAlerts.map((alert, index) => {
                const getSeverityColor = (severity: string) => {
                  switch (severity) {
                    case 'critical': return 'bg-red-500'
                    case 'high': return 'bg-orange-500'
                    case 'medium': return 'bg-yellow-500'
                    case 'low': return 'bg-blue-500'
                    default: return 'bg-slate-500'
                  }
                }

                const getSeverityBadgeClass = (severity: string) => {
                  switch (severity) {
                    case 'critical': return 'bg-red-500/10 text-red-600 border-red-500/20'
                    case 'high': return 'bg-orange-500/10 text-orange-600 border-orange-500/20'
                    case 'medium': return 'bg-yellow-500/10 text-yellow-600 border-yellow-500/20'
                    case 'low': return 'bg-blue-500/10 text-blue-600 border-blue-500/20'
                    default: return 'bg-slate-500/10 text-slate-600 border-slate-500/20'
                  }
                }

                const getSeverityLabel = (severity: string) => {
                  switch (severity) {
                    case 'critical': return '严重'
                    case 'high': return '高'
                    case 'medium': return '中'
                    case 'low': return '低'
                    default: return severity
                  }
                }

                return (
                  <div
                    key={alert.id}
                    className="flex items-start gap-4 px-6 py-5 hover:bg-muted/30 transition-colors group"
                  >
                    {/* 严重级别指示器 */}
                    <div className={`w-2.5 h-2.5 rounded-full ${getSeverityColor(alert.severity)} flex-shrink-0 mt-2 ring-2 ring-offset-2 ring-transparent group-hover:ring-${getSeverityColor(alert.severity).split('-')[1]}-100 transition-all`} />

                    {/* 主要内容区域 */}
                    <div className="flex-1 min-w-0 space-y-2">
                      <div className="flex items-start justify-between gap-4">
                        {/* 标题 */}
                        <Link
                          href={resolveAppPath(`/alerts/${alert.id}`) as any}
                          className="text-base font-semibold hover:text-primary transition-colors block line-clamp-1"
                        >
                          {alert.title}
                        </Link>

                        {/* 时间 - 移到右上角 */}
                        <div className="flex items-center gap-1.5 text-xs text-muted-foreground whitespace-nowrap pt-1">
                          <Clock className="h-3.5 w-3.5" />
                          <span>
                            {format(parseISO(alert.starts_at || alert.created_at), 'yyyy-MM-dd HH:mm:ss')}
                          </span>
                        </div>
                      </div>

                      {/* 描述信息 */}
                      {alert.description && (
                        <p className="text-sm text-muted-foreground line-clamp-2 leading-relaxed" title={alert.description}>
                          {alert.description}
                        </p>
                      )}

                      {/* 元信息行 */}
                      <div className="flex items-center gap-3 flex-wrap text-sm pt-1">
                        {/* 严重级别徽章 */}
                        <div className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium border ${getSeverityBadgeClass(alert.severity)}`}>
                          {getSeverityLabel(alert.severity)}
                        </div>

                        {/* 状态徽章 */}
                        {alert.status === 'firing' && (
                          <div className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-500/10 text-red-600 border border-red-500/20">
                            触发中
                          </div>
                        )}

                        <span className="text-border h-4 w-px bg-border mx-1"></span>

                        {/* 集群 */}
                        <span className="text-muted-foreground text-xs flex items-center gap-1.5 bg-muted/50 px-2 py-1 rounded">
                          <Layers className="h-3 w-3" />
                          <span className="font-medium text-foreground">{alert.cluster?.name || alert.cluster_name || 'robusta-kind'}</span>
                        </span>

                        {/* Generator URL 按钮 */}
                        {(() => {
                          const generatorUrl = (alert as any).generator_url ||
                            (alert as any).generatorURL ||
                            alert.annotations?.generatorURL ||
                            alert.annotations?.generator_url ||
                            alert.labels?.generatorURL ||
                            alert.labels?.generator_url

                          if (generatorUrl) {
                            return (
                              <>
                                <a
                                  href={generatorUrl}
                                  target="_blank"
                                  rel="noopener noreferrer"
                                  className="inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium text-muted-foreground hover:text-foreground bg-muted/30 hover:bg-muted/50 border border-transparent hover:border-border rounded transition-all"
                                  title={generatorUrl}
                                >
                                  <ExternalLink className="h-3 w-3" />
                                  <span>指标</span>
                                </a>
                              </>
                            )
                          }
                          return null
                        })()}
                      </div>
                    </div>

                    {/* 右侧操作按钮 */}
                    <div className="flex-shrink-0 self-center pl-2">
                      <Link
                        href={resolveAppPath(`/alerts/${alert.id}`) as any}
                        className="inline-flex h-8 items-center justify-center rounded-md border border-input bg-background px-3 text-sm font-medium shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                      >
                        查看详情
                      </Link>
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
