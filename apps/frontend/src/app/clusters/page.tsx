'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import Link from 'next/link'
import { resolveAppPath } from '@/config'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import {
  Server,
  Search,
  RefreshCw,
  Clock,
  CheckCircle,
  XCircle,
  Activity,
  Calendar,
  Eye,
  Plus,
  Pencil,
  Trash2
} from 'lucide-react'
import RobustaAPI from '@/lib/api'
import { formatDistanceToNow } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { KiteLink } from '@/components/kite'
import { ClusterDialog } from './components/ClusterDialog'
import { toast } from 'sonner'
import { Cluster } from '@/types/api'

export default function Clusters() {
  const [page, setPage] = useState(1)
  const [statusFilter, setStatusFilter] = useState('')
  const [searchTerm, setSearchTerm] = useState('')
  const pageSize = 10 // 改为每页 10 条，列表视图更紧凑

  // Dialog states
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false)
  const [editingCluster, setEditingCluster] = useState<Cluster | null>(null)
  const [deletingCluster, setDeletingCluster] = useState<Cluster | null>(null)
  const [isDeleting, setIsDeleting] = useState(false)

  // 获取集群列表
  const { data: clustersData, isLoading, refetch } = useQuery({
    queryKey: ['clusters', page, statusFilter, searchTerm],
    queryFn: () => RobustaAPI.getClusters(page, pageSize, statusFilter || undefined),
    refetchInterval: 30000, // 30秒自动刷新
  })

  const clusters = clustersData?.data || []
  const pagination = clustersData?.pagination

  // 获取状态配置
  const getStatusConfig = (status: string) => {
    switch (status) {
      case 'active':
        return {
          variant: 'outline' as const,
          label: '活跃',
          icon: CheckCircle,
          className: 'bg-emerald-50 text-emerald-700 border-emerald-200 hover:bg-emerald-50'
        }
      case 'inactive':
        return {
          variant: 'outline' as const,
          label: '离线',
          icon: XCircle,
          className: 'bg-rose-50 text-rose-700 border-rose-200 hover:bg-rose-50'
        }
      case 'maintenance':
        return {
          variant: 'outline' as const,
          label: '维护中',
          icon: Activity,
          className: 'bg-amber-50 text-amber-700 border-amber-200 hover:bg-amber-50'
        }
      default:
        return {
          variant: 'outline' as const,
          label: status,
          icon: Server,
          className: 'bg-slate-50 text-slate-700 border-slate-200 hover:bg-slate-50'
        }
    }
  }

  // 过滤集群
  const filteredClusters = clusters.filter(cluster => {
    const matchesSearch = !searchTerm ||
      cluster.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      (cluster.cluster_id && cluster.cluster_id.toLowerCase().includes(searchTerm.toLowerCase()))

    return matchesSearch
  })

  const handleDelete = async () => {
    if (!deletingCluster) return

    setIsDeleting(true)
    try {
      await RobustaAPI.deleteCluster(deletingCluster.name)
      toast.success('集群删除成功')
      refetch()
    } catch (error) {
      console.error(error)
      toast.error('删除失败')
    } finally {
      setIsDeleting(false)
      setDeletingCluster(null)
    }
  }

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
        <div className="flex gap-2">
          <Button onClick={() => setIsCreateDialogOpen(true)}>
            <Plus className="h-4 w-4 mr-2" />
            新增集群
          </Button>
          <Button onClick={() => refetch()} variant="outline">
            <RefreshCw className="h-4 w-4 mr-2" />
            刷新
          </Button>
        </div>
      </div>

      {/* 搜索和过滤 */}
      <Card>
        <CardHeader className="py-4">
          <div className="flex flex-col md:flex-row gap-4 items-center justify-between">
            <CardTitle className="text-lg hidden md:block">集群列表</CardTitle>
            <div className="flex flex-1 w-full md:w-auto gap-4 items-center">
              {/* 搜索框 */}
              <div className="relative flex-1 md:max-w-sm">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  placeholder="搜索集群名称或ID..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-10"
                />
              </div>

              {/* 状态过滤 */}
              <Select
                value={statusFilter || 'all'}
                onValueChange={(value) => setStatusFilter(value === 'all' ? '' : value)}
              >
                <SelectTrigger className="w-[150px]">
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
          </div>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>名称</TableHead>
                <TableHead>ID</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>最后心跳</TableHead>
                <TableHead>创建时间</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                Array.from({ length: 5 }).map((_, i) => (
                  <TableRow key={i}>
                    <TableCell><div className="h-4 w-24 bg-muted animate-pulse rounded" /></TableCell>
                    <TableCell><div className="h-4 w-32 bg-muted animate-pulse rounded" /></TableCell>
                    <TableCell><div className="h-4 w-16 bg-muted animate-pulse rounded" /></TableCell>
                    <TableCell><div className="h-4 w-20 bg-muted animate-pulse rounded" /></TableCell>
                    <TableCell><div className="h-4 w-20 bg-muted animate-pulse rounded" /></TableCell>
                    <TableCell><div className="h-8 w-8 bg-muted animate-pulse rounded float-right" /></TableCell>
                  </TableRow>
                ))
              ) : filteredClusters.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="h-24 text-center">
                    <div className="flex flex-col items-center justify-center text-muted-foreground">
                      <Server className="h-8 w-8 mb-2 opacity-50" />
                      <p>暂无集群数据</p>
                    </div>
                  </TableCell>
                </TableRow>
              ) : (
                filteredClusters.map((cluster) => {
                  const statusConfig = getStatusConfig(cluster.status)
                  const StatusIcon = statusConfig.icon

                  return (
                    <TableRow key={cluster.id}>
                      <TableCell className="font-medium">
                        <div className="flex flex-col">
                          <span className="text-base">{cluster.name}</span>
                          {cluster.description && (
                            <span className="text-xs text-muted-foreground truncate max-w-[200px]" title={cluster.description}>
                              {cluster.description}
                            </span>
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="font-mono text-xs text-muted-foreground">
                        {cluster.cluster_id || '-'}
                      </TableCell>
                      <TableCell>
                        <Badge variant={statusConfig.variant} className={`flex w-fit items-center gap-1 ${statusConfig.className}`}>
                          <StatusIcon className="h-3 w-3" />
                          {statusConfig.label}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center text-sm text-muted-foreground">
                          <Clock className="mr-2 h-4 w-4" />
                          {cluster.last_heartbeat
                            ? formatDistanceToNow(new Date(cluster.last_heartbeat), {
                              addSuffix: true,
                              locale: zhCN
                            })
                            : '无数据'
                          }
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center text-sm text-muted-foreground">
                          <Calendar className="mr-2 h-4 w-4" />
                          {formatDistanceToNow(new Date(cluster.created_at), {
                            addSuffix: true,
                            locale: zhCN
                          })}
                        </div>
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex justify-end gap-2 items-center">
                          <KiteLink
                            clusterName={cluster.name}
                            variant="icon"
                          />
                          <Button asChild variant="ghost" size="icon" title="查看详情">
                            <Link href={resolveAppPath(`/clusters/${cluster.name}`)}>
                              <Eye className="h-4 w-4" />
                            </Link>
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            title="编辑"
                            onClick={() => setEditingCluster(cluster)}
                          >
                            <Pencil className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            title="删除"
                            className="text-destructive hover:text-destructive"
                            onClick={() => setDeletingCluster(cluster)}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                })
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* 分页 */}
      {pagination && (
        <div className="flex items-center justify-between px-2">
          <div className="text-sm text-muted-foreground">
            显示 {pagination.total === 0 ? 0 : ((pagination.page - 1) * pagination.page_size) + 1} - {Math.min(pagination.page * pagination.page_size, pagination.total)}
            条，共 {pagination.total} 条
          </div>
          <div className="flex space-x-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage(page - 1)}
              disabled={page <= 1 || isLoading}
            >
              上一页
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage(page + 1)}
              disabled={page >= Math.ceil(pagination.total / pagination.page_size) || isLoading}
            >
              下一页
            </Button>
          </div>
        </div>
      )}

      {/* 创建/编辑对话框 */}
      <ClusterDialog
        open={isCreateDialogOpen || !!editingCluster}
        onOpenChange={(open) => {
          if (!open) {
            setIsCreateDialogOpen(false)
            setEditingCluster(null)
          } else {
            // Opening handling if needed
          }
        }}
        cluster={editingCluster}
        onSuccess={() => {
          refetch()
        }}
      />

      {/* 删除确认对话框 */}
      <AlertDialog open={!!deletingCluster} onOpenChange={(open) => !open && setDeletingCluster(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除集群?</AlertDialogTitle>
            <AlertDialogDescription>
              您正在删除集群 <strong>{deletingCluster?.name}</strong>。此操作无法撤销，将会删除该集群的所有历史数据和告警信息。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={(e) => {
                e.preventDefault()
                handleDelete()
              }}
              disabled={isDeleting}
            >
              {isDeleting ? '删除中...' : '确认删除'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
