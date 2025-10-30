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
  X
} from 'lucide-react'
import { Checkbox } from '@/components/ui/checkbox'
import RobustaAPI from '@/lib/api'
import { format, subDays, addHours, parseISO } from 'date-fns'
import type { AlertFilters } from '@/types/api'

export default function Alerts() {
  const [filters, setFilters] = useState<AlertFilters>({})
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
      cluster.cluster_id.toLowerCase().includes(clusterSearch.toLowerCase())
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
        const clusterMatch = selectedClusters.length === 0 || selectedClusters.includes(alert.cluster_id)
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
    <div className="space-y-4">
      {/* 顶部筛选栏 - 参照Robusta官方风格 */}
      <div className="bg-white border rounded-lg p-4">
        {/* 时间范围选择 */}
        <div className="flex items-center gap-2 mb-4">
          <Calendar className="h-4 w-4 text-gray-500" />
          <span className="text-sm text-gray-600 mr-2">时间范围</span>
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
                className={`h-8 px-3 text-sm rounded-md ${
                  !useAbsoluteTime && timeRange === option.value
                    ? 'bg-black text-white hover:bg-gray-900'
                    : 'bg-white text-gray-700 border border-gray-200 hover:bg-gray-50'
                }`}
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
                  className={`h-8 px-3 text-sm rounded-md flex items-center gap-1 ${
                    useAbsoluteTime
                      ? 'bg-black text-white hover:bg-gray-900'
                      : 'bg-white text-gray-700 border border-gray-200 hover:bg-gray-50'
                  }`}
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
          <div className="flex items-center gap-2 ml-6">
            <Search className="h-4 w-4 text-gray-500" />
            <Input
              placeholder="搜索告警..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="h-8 w-64 text-sm"
            />
          </div>
        </div>

        {/* 筛选标签行 */}
        <div className="flex items-center gap-6">
          {/* 集群筛选 - 多选搜索下拉 */}
          <div className="flex items-center gap-2">
            <Layers className="h-4 w-4 text-gray-500" />
            <span className="text-sm text-gray-600">集群</span>
            <div className="flex items-center gap-1">
              <Popover>
                <PopoverTrigger asChild>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-7 px-3 text-xs rounded-full bg-gray-100 text-gray-700 hover:bg-gray-200 flex items-center gap-2"
                  >
                    {selectedClusters.length === 0 
                      ? '所有集群'
                      : `已选 ${selectedClusters.length} 个集群`}
                    <ChevronDown className="h-3 w-3" />
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-72 p-0" align="start">
                  <div className="p-3">
                    <div className="flex items-center justify-between mb-2">
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
                      className="h-8 text-sm mb-2"
                      value={clusterSearch}
                      onChange={(e) => setClusterSearch(e.target.value)}
                    />
                    <div className="max-h-64 overflow-y-auto space-y-1">
                      {filteredClusters.map((cluster) => (
                        <div
                          key={cluster.cluster_id}
                          className="flex items-center gap-2 px-2 py-2 rounded hover:bg-gray-100 cursor-pointer"
                          onClick={() => toggleCluster(cluster.cluster_id)}
                        >
                          <Checkbox
                            checked={selectedClusters.includes(cluster.cluster_id)}
                            onCheckedChange={() => toggleCluster(cluster.cluster_id)}
                            className="pointer-events-none"
                          />
                          <div className="flex-1 min-w-0">
                            <div className="text-sm font-medium truncate">{cluster.name}</div>
                            {cluster.cluster_id !== cluster.name && (
                              <div className="text-xs text-gray-500 truncate">{cluster.cluster_id}</div>
                            )}
                          </div>
                        </div>
                      ))}
                      {filteredClusters.length === 0 && clusterSearch && (
                        <div className="px-2 py-4 text-sm text-gray-500 text-center">
                          未找到匹配的集群
                        </div>
                      )}
                    </div>
                  </div>
                </PopoverContent>
              </Popover>
              
              {/* 显示已选集群标签 */}
              {selectedClusters.length > 0 && (
                <div className="flex items-center gap-1">
                  {selectedClusters.slice(0, 2).map((clusterId) => {
                    const cluster = clusters.find(c => c.cluster_id === clusterId);
                    return (
                      <div
                        key={clusterId}
                        className="inline-flex items-center gap-1 h-7 px-2 text-xs rounded-full bg-black text-white"
                      >
                        <span className="max-w-24 truncate">{cluster?.name || clusterId}</span>
                        <button
                          onClick={() => toggleCluster(clusterId)}
                          className="hover:bg-white/20 rounded-full p-0.5"
                        >
                          <X className="h-3 w-3" />
                        </button>
                      </div>
                    );
                  })}
                  {selectedClusters.length > 2 && (
                    <span className="text-xs text-gray-600">
                      +{selectedClusters.length - 2}
                    </span>
                  )}
                </div>
              )}
            </div>
          </div>

          {/* 严重级别筛选 */}
          <div className="flex items-center gap-2">
            <AlertTriangle className="h-4 w-4 text-gray-500" />
            <span className="text-sm text-gray-600">严重级别</span>
            <div className="flex gap-1">
              {[
                { value: '', label: '所有级别', color: 'bg-gray-400' },
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
                  className={`h-7 px-3 text-xs rounded-full flex items-center gap-1.5 ${
                    (filters.severity || '') === option.value
                      ? 'bg-black text-white hover:bg-gray-900'
                      : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                  }`}
                >
                  <div className={`w-2 h-2 rounded-full ${option.color}`} />
                  {option.label}
                </Button>
              ))}
            </div>
          </div>

          {/* 状态筛选 */}
          <div className="flex items-center gap-2">
            <Activity className="h-4 w-4 text-gray-500" />
            <span className="text-sm text-gray-600">状态</span>
            <div className="flex gap-1">
              {[
                { value: '', label: '所有状态', color: 'bg-gray-400' },
                { value: 'firing', label: '触发中', color: 'bg-red-500' },
                { value: 'resolved', label: '已解决', color: 'bg-green-500' },
                { value: 'silenced', label: '已静默', color: 'bg-gray-500' }
              ].map((option) => (
                <Button
                  key={option.value}
                  variant={(filters.status || '') === option.value ? "default" : "ghost"}
                  size="sm"
                  onClick={() => handleFilterChange('status', option.value)}
                  className={`h-7 px-3 text-xs rounded-full flex items-center gap-1.5 ${
                    (filters.status || '') === option.value
                      ? 'bg-black text-white hover:bg-gray-900'
                      : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                  }`}
                >
                  <div className={`w-2 h-2 rounded-full ${option.color}`} />
                  {option.label}
                </Button>
              ))}
            </div>
          </div>
        </div>
      </div>



      {/* 告警时间线 - 参照Robusta官方风格 */}
      <div className="bg-white border rounded-lg">
        {/* 标题栏 */}
        <div className="flex items-center justify-between p-4 border-b">
          <div className="flex items-center gap-2">
            <BarChart3 className="h-5 w-5 text-gray-600" />
            <h2 className="text-lg font-semibold text-gray-900">告警时间线</h2>
            <span className="text-sm text-gray-500">({timelineAlerts.length} 条告警)</span>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setCurrentTime(subDays(currentTime, 1))}
              className="h-8 px-3"
            >
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setCurrentTime(addHours(currentTime, 1))}
              disabled={currentTime >= new Date()}
              className="h-8 px-3"
            >
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </div>

        {/* 内容区域 */}
        <div className="p-4">
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
              <span className="ml-2 text-gray-600">加载告警数据...</span>
            </div>
          ) : timelineAlerts.length === 0 ? (
            <div className="text-center py-12">
              <CheckCircle className="h-16 w-16 text-green-500 mx-auto mb-4" />
              <h3 className="text-lg font-semibold mb-2 text-gray-900">当前时间范围内无告警</h3>
              <p className="text-gray-600">系统运行正常，或尝试调整时间范围查看更多数据</p>
            </div>
          ) : (
            <div className="space-y-4">
              {/* 时间轴 */}
              <div className="flex items-center justify-between text-sm text-gray-500 mb-4">
                <span>{format(timeRangeStart, 'yyyy-MM-dd HH:mm')}</span>
                <span>{format(timeRangeEnd, 'yyyy-MM-dd HH:mm')}</span>
              </div>

              {/* 告警列表 - 参照第二张图片样式 */}
              <div className="space-y-0 border border-gray-200 rounded-lg overflow-hidden bg-white">
                {timelineAlerts.map((alert, index) => {
                  const getSeverityColor = (severity: string) => {
                    switch (severity) {
                      case 'critical': return 'bg-red-500'
                      case 'high': return 'bg-orange-500'
                      case 'medium': return 'bg-yellow-500'
                      case 'low': return 'bg-blue-500'
                      default: return 'bg-gray-500'
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
                      className={`flex items-center justify-between px-6 py-4 hover:bg-gray-50 transition-colors ${
                        index !== timelineAlerts.length - 1 ? 'border-b border-gray-100' : ''
                      }`}
                    >
                      {/* 左侧内容 */}
                      <div className="flex items-center gap-4 flex-1">
                        {/* 严重级别指示器 */}
                        <div className={`w-2.5 h-2.5 rounded-full ${getSeverityColor(alert.severity)} flex-shrink-0`} />

                        {/* 告警信息 */}
                        <div className="flex-1 min-w-0">
                          <Link
                            href={resolveAppPath(`/alerts/${alert.id}`)}
                            className="text-lg font-medium text-gray-900 hover:text-blue-600 transition-colors block mb-1"
                          >
                            {alert.title}
                          </Link>

                          {/* 集群和标签信息 */}
                          <div className="flex items-center gap-3">
                            <span className="text-sm text-gray-600">
                              集群: {alert.cluster?.name || alert.cluster_id || 'robusta-kind'}
                            </span>

                            {/* 严重级别标签 */}
                            <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium text-white ${getSeverityColor(alert.severity)}`}>
                              {getSeverityLabel(alert.severity)}
                            </span>

                            {/* 状态标签 */}
                            {alert.status === 'firing' && (
                              <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-red-100 text-red-800">
                                触发中
                              </span>
                            )}

                            {/* 时间信息 */}
                            <div className="flex items-center gap-1 text-sm text-gray-500">
                              <Clock className="h-4 w-4" />
                              <span>
                                {format(parseISO(alert.starts_at || alert.created_at), 'HH:mm:ss')}
                              </span>
                            </div>
                          </div>
                        </div>
                      </div>

                      {/* 右侧查看详情按钮 */}
                      <Link
                        href={resolveAppPath(`/alerts/${alert.id}`)}
                        className="ml-4 px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 hover:text-gray-900 transition-colors flex-shrink-0"
                      >
                        查看详情
                      </Link>
                    </div>
                  )
                })}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
