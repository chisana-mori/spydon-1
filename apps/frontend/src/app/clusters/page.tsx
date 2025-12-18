'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import Link from 'next/link'
import { resolveAppPath } from '@/config'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
  Server,
  Search,
  RefreshCw,
  Clock,
  CheckCircle,
  XCircle,
  Activity
} from 'lucide-react'
import RobustaAPI from '@/lib/api'
import { formatDistanceToNow } from 'date-fns'
import { zhCN } from 'date-fns/locale'

export default function Clusters() {
  const [page, setPage] = useState(1)
  const [statusFilter, setStatusFilter] = useState('')
  const [searchTerm, setSearchTerm] = useState('')

  // 获取集群列表
  const { data: clustersData, isLoading, refetch } = useQuery({
    queryKey: ['clusters', page, statusFilter, searchTerm],
    queryFn: () => RobustaAPI.getClusters(page, 20, statusFilter || undefined),
    refetchInterval: 30000, // 30秒自动刷新
  })

  const clusters = clustersData?.data || []
  const pagination = clustersData?.pagination

  // 获取状态配置
  const getStatusConfig = (status: string) => {
    switch (status) {
      case 'active':
        return {
          variant: 'secondary' as const,
          label: '活跃',
          icon: CheckCircle,
          color: 'text-green-600'
        }
      case 'inactive':
        return {
          variant: 'destructive' as const,
          label: '离线',
          icon: XCircle,
          color: 'text-red-600'
        }
      case 'maintenance':
        return {
          variant: 'default' as const,
          label: '维护中',
          icon: Activity,
          color: 'text-yellow-600'
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

  // 过滤集群
  const filteredClusters = clusters.filter(cluster => {
    const matchesSearch = !searchTerm ||
      cluster.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      cluster.cluster_id.toLowerCase().includes(searchTerm.toLowerCase())

    return matchesSearch
  })

  return (
    <div className="space-y-6">
      {/* 页面标题 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">集群管理</h1>
          <p className="text-muted-foreground">
            管理和监控所有Kubernetes集群
          </p>
        </div>
        <Button onClick={() => refetch()} variant="outline">
          <RefreshCw className="h-4 w-4 mr-2" />
          刷新
        </Button>
      </div>

      {/* 搜索和过滤 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">搜索和过滤</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-3">
            {/* 搜索框 */}
            <div className="md:col-span-2">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  placeholder="搜索集群名称或ID..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-10"
                />
              </div>
            </div>

            {/* 状态过滤 */}
            <Select
              value={statusFilter || 'all'}
              onValueChange={(value) => setStatusFilter(value === 'all' ? '' : value)}
            >
              <SelectTrigger>
                <SelectValue placeholder="选择状态" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">所有状态</SelectItem>
                <SelectItem value="active">活跃</SelectItem>
                <SelectItem value="inactive">离线</SelectItem>
                <SelectItem value="maintenance">维护中</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      {/* 集群列表 */}
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        {isLoading ? (
          Array.from({ length: 6 }).map((_, i) => (
            <Card key={i}>
              <CardContent className="py-8">
                <div className="text-center">加载中...</div>
              </CardContent>
            </Card>
          ))
        ) : filteredClusters.length === 0 ? (
          <div className="col-span-full">
            <Card>
              <CardContent className="py-8">
                <div className="text-center text-muted-foreground">
                  <Server className="h-12 w-12 mx-auto mb-4 opacity-50" />
                  <p>暂无集群数据</p>
                </div>
              </CardContent>
            </Card>
          </div>
        ) : (
          filteredClusters.map((cluster) => {
            const statusConfig = getStatusConfig(cluster.status)
            const StatusIcon = statusConfig.icon

            return (
              <Card key={cluster.id} className="hover:shadow-md transition-shadow">
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center space-x-2">
                      <StatusIcon className={`h-5 w-5 ${statusConfig.color}`} />
                      <Badge variant={statusConfig.variant}>
                        {statusConfig.label}
                      </Badge>
                    </div>
                  </div>
                  <CardTitle className="text-lg">{cluster.name}</CardTitle>
                  <CardDescription>
                    ID: {cluster.cluster_id}
                  </CardDescription>
                </CardHeader>

                <CardContent className="space-y-4">
                  {cluster.description && (
                    <p className="text-sm text-muted-foreground line-clamp-2">
                      {cluster.description}
                    </p>
                  )}

                  <div className="space-y-2 text-sm">
                    <div className="flex items-center justify-between">
                      <span className="text-muted-foreground">最后心跳:</span>
                      <span className="flex items-center">
                        <Clock className="h-3 w-3 mr-1" />
                        {cluster.last_heartbeat
                          ? formatDistanceToNow(new Date(cluster.last_heartbeat), {
                            addSuffix: true,
                            locale: zhCN
                          })
                          : '无数据'
                        }
                      </span>
                    </div>

                    <div className="flex items-center justify-between">
                      <span className="text-muted-foreground">创建时间:</span>
                      <span>
                        {formatDistanceToNow(new Date(cluster.created_at), {
                          addSuffix: true,
                          locale: zhCN
                        })}
                      </span>
                    </div>
                  </div>

                  <div className="pt-2">
                    <Button asChild variant="outline" className="w-full">
                      <Link href={resolveAppPath(`/clusters/${cluster.cluster_id}`)}>
                        查看详情
                      </Link>
                    </Button>
                  </div>
                </CardContent>
              </Card>
            )
          })
        )}
      </div>

      {/* 分页 */}
      {pagination && pagination.total > pagination.page_size && (
        <div className="flex items-center justify-between">
          <div className="text-sm text-muted-foreground">
            显示 {((pagination.page - 1) * pagination.page_size) + 1} - {Math.min(pagination.page * pagination.page_size, pagination.total)}
            条，共 {pagination.total} 条
          </div>
          <div className="flex space-x-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage(page - 1)}
              disabled={page <= 1}
            >
              上一页
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage(page + 1)}
              disabled={page >= Math.ceil(pagination.total / pagination.page_size)}
            >
              下一页
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
