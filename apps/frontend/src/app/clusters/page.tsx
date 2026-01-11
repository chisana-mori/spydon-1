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
import { cn } from "@/lib/utils"
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
  CheckCircle,
  XCircle,
  Activity,
  Calendar,
  Eye,
  Plus,
  Pencil,
  Trash2,
  Copy,
  Settings,
} from 'lucide-react'
import RobustaAPI from '@/lib/api'
import { KiteLink } from '@/components/kite'
import { ClusterDrawer } from './components/ClusterDrawer'
import { InventoryVariablesDialog } from './components/InventoryVariablesDialog'
import { toast } from 'sonner'
import { Cluster } from '@/types/api'
import { useDictionary } from '@/hooks/useDictionary'
import { ClusterNodesSheet } from './components/ClusterNodesSheet'
import { parseDictionaryValues } from '@/lib/api/dictionaries'

// Helper component for Node Count Button
// Helper component for Node Count Button
const CombinedNodeButton = ({
  masterCount,
  etcdCount,
  eventCount,
  onClick
}: {
  masterCount: number
  etcdCount: number
  eventCount: number
  onClick: () => void
}) => {
  if (masterCount === 0 && etcdCount === 0 && eventCount === 0) {
    return <span className="text-muted-foreground/40 text-xs font-mono">-</span>;
  }

  return (
    <Button
      variant="outline"
      size="sm"
      className="h-7 px-2 text-xs font-medium hover:bg-muted/50 transition-all rounded-md gap-2"
      onClick={(e) => {
        e.stopPropagation();
        onClick();
      }}
    >
      <div className="flex items-center gap-1">
        <div className="w-1.5 h-1.5 rounded-full bg-blue-500" />
        <span className={cn(masterCount === 0 && "text-muted-foreground/50")}>{masterCount}</span>
      </div>
      <div className="w-px h-3 bg-border" />
      <div className="flex items-center gap-1">
        <div className="w-1.5 h-1.5 rounded-full bg-purple-500" />
        <span className={cn(etcdCount === 0 && "text-muted-foreground/50")}>{etcdCount}</span>
      </div>
      <div className="w-px h-3 bg-border" />
      <div className="flex items-center gap-1">
        <div className="w-1.5 h-1.5 rounded-full bg-orange-500" />
        <span className={cn(eventCount === 0 && "text-muted-foreground/50")}>{eventCount}</span>
      </div>
    </Button>
  );
};

export default function Clusters() {
  const [page, setPage] = useState(1)
  const [statusFilter, setStatusFilter] = useState('')
  const [searchTerm, setSearchTerm] = useState('')
  const pageSize = 10 // 改为每页 10 条，列表视图更紧凑

  // Dictionaries
  const { items: purposeItems } = useDictionary('purpose')

  // Dialog states
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false)
  const [editingCluster, setEditingCluster] = useState<Cluster | null>(null)
  const [nodesSheetOpen, setNodesSheetOpen] = useState(false)
  const [selectedClusterForNodes, setSelectedClusterForNodes] = useState<Cluster | null>(null)
  const [deletingCluster, setDeletingCluster] = useState<Cluster | null>(null)
  const [isDeleting, setIsDeleting] = useState(false)
  const [inventoryCluster, setInventoryCluster] = useState<Cluster | null>(null)

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
      case 'Running':
        return {
          variant: 'outline' as const,
          label: '运行中',
          icon: CheckCircle,
          className: 'bg-green-500/10 text-green-600 border-green-500/20 hover:bg-green-500/20'
        }
      case 'Offline':
        return {
          variant: 'outline' as const,
          label: '已下线',
          icon: XCircle,
          className: 'bg-red-500/10 text-red-500 border-red-500/20 hover:bg-red-500/20'
        }
      case 'Pending':
        return {
          variant: 'outline' as const,
          label: '维护中',
          icon: Activity,
          className: 'bg-yellow-500/10 text-yellow-600 border-yellow-500/20 hover:bg-yellow-500/20'
        }
      case 'Init':
        return {
          variant: 'outline' as const,
          label: '初始化',
          icon: Server,
          className: 'bg-blue-500/10 text-blue-600 border-blue-500/20 hover:bg-blue-500/20'
        }
      default:
        return {
          variant: 'outline' as const,
          label: status,
          icon: Server,
          className: 'bg-slate-500/10 text-slate-500 border-slate-500/20 hover:bg-slate-500/20'
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
      {/* 页面标题 */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2.5 rounded-xl bg-blue-500/10 text-blue-500 ring-1 ring-blue-500/20">
            <Server className="h-6 w-6" />
          </div>
          <div>
            <h1 className="text-2xl font-bold tracking-tight">集群管理</h1>
            <p className="text-sm text-muted-foreground mt-0.5">
              管理和监控所有Kubernetes集群
            </p>
          </div>
        </div>
        <div className="flex gap-2">
          <Button onClick={() => setIsCreateDialogOpen(true)} className="shadow-lg shadow-primary/20 hover:shadow-xl hover:shadow-primary/30 transition-all">
            <Plus className="h-4 w-4 mr-2" />
            新增集群
          </Button>
          <Button
            onClick={async () => {
              const loadingToast = toast.loading('正在同步集群配置...')
              try {
                const res = await RobustaAPI.syncClusterConfig()
                toast.dismiss(loadingToast)
                toast.success(`同步成功，更新了 ${res.updated_count} 个集群配置`)
                refetch()
              } catch (e) {
                toast.dismiss(loadingToast)
                console.error(e)
                toast.error('同步失败')
              }
            }}
            variant="outline"
            className="shadow-sm hover:shadow"
          >
            <RefreshCw className="h-4 w-4 mr-2" />
            同步配置
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
                  <SelectItem value="Init">初始化</SelectItem>
                  <SelectItem value="Running">运行中</SelectItem>
                  <SelectItem value="Pending">维护中</SelectItem>
                  <SelectItem value="Offline">已下线</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>名称/ID</TableHead>
                <TableHead>用途</TableHead>
                <TableHead>版本</TableHead>
                <TableHead>控制节点</TableHead>
                <TableHead>状态</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                Array.from({ length: 5 }).map((_, i) => (
                  <TableRow key={i}>
                    <TableCell><div className="h-4 w-24 bg-muted animate-pulse rounded" /></TableCell>
                    <TableCell><div className="h-4 w-16 bg-muted animate-pulse rounded" /></TableCell>
                    <TableCell><div className="h-4 w-16 bg-muted animate-pulse rounded" /></TableCell>
                    <TableCell><div className="h-4 w-24 bg-muted animate-pulse rounded" /></TableCell>
                    <TableCell><div className="h-4 w-24 bg-muted animate-pulse rounded" /></TableCell>
                    <TableCell><div className="h-4 w-24 bg-muted animate-pulse rounded" /></TableCell>
                    <TableCell><div className="h-4 w-16 bg-muted animate-pulse rounded" /></TableCell>
                    <TableCell><div className="h-8 w-8 bg-muted animate-pulse rounded float-right" /></TableCell>
                  </TableRow>
                ))
              ) : filteredClusters.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} className="h-24 text-center">
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
                      <TableCell className="font-medium align-top">
                        <div className="flex flex-col">
                          <span className="text-base">{cluster.name}</span>
                          <span className="text-xs text-muted-foreground font-mono">
                            {cluster.cluster_id || '-'}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell className="align-top">
                        <span className="text-sm">{parseDictionaryValues(cluster.purpose, purposeItems)}</span>
                      </TableCell>
                      <TableCell className="align-top">
                        <Badge variant="secondary" className="font-mono text-xs whitespace-nowrap">
                          {cluster.cluster_version || '-'}
                        </Badge>
                      </TableCell>
                      <TableCell className="align-top">
                        <CombinedNodeButton
                          masterCount={cluster.master_ips?.length || 0}
                          etcdCount={cluster.etcd_ips?.length || 0}
                          eventCount={cluster.etcd_event_ips?.length || 0}
                          onClick={() => {
                            setSelectedClusterForNodes(cluster);
                            setNodesSheetOpen(true);
                          }}
                        />
                      </TableCell>
                      <TableCell className="align-top">
                        <Badge variant={statusConfig.variant} className={`flex w-fit items-center gap-1 ${statusConfig.className}`}>
                          <StatusIcon className="h-3 w-3" />
                          {statusConfig.label}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-right align-top">
                        <div className="flex justify-end gap-2 items-center">
                          <KiteLink
                            clusterName={cluster.name}
                            variant="icon"
                          />
                          <Button asChild variant="ghost" size="icon" title="查看详情">
                            <Link href={`/clusters/${cluster.name}`}>
                              <Eye className="h-4 w-4" />
                            </Link>
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            title="参数配置"
                            onClick={() => setInventoryCluster(cluster)}
                          >
                            <Settings className="h-4 w-4" />
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

      {/* 创建/编辑抽屉 */}
      <ClusterDrawer
        open={isCreateDialogOpen || !!editingCluster}
        onClose={() => {
          setIsCreateDialogOpen(false)
          setEditingCluster(null)
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

      {/* Inventory 参数配置对话框 */}
      <InventoryVariablesDialog
        open={!!inventoryCluster}
        onOpenChange={(open) => !open && setInventoryCluster(null)}
        clusterName={inventoryCluster?.name || ''}
        onSuccess={() => {
          toast.success('Inventory 参数已更新')
        }}
      />
      <ClusterNodesSheet
        open={nodesSheetOpen}
        onOpenChange={setNodesSheetOpen}
        cluster={selectedClusterForNodes}
      />
    </div>
  )
}
