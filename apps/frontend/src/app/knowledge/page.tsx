"use client"

import React, { useState, useEffect, Suspense } from 'react'
import Link from 'next/link'
import { useRouter, useSearchParams } from 'next/navigation'
import { resolveAppPath } from '@/config'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { RobustaAPI } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from '@/components/ui/card'
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
  Search,
  Plus,
  Trash2,
  Edit,
  ChevronLeft,
  ChevronRight,
  Clock,
  ShieldAlert,
  ArrowRight,
  BookOpen,
  AlertTriangle
} from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'

// 包装组件以支持 Suspense 边界
export default function KnowledgeHomePage() {
  return (
    <Suspense fallback={<div className="p-6">加载中...</div>}>
      <KnowledgeHomePageContent />
    </Suspense>
  )
}

function KnowledgeHomePageContent() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [keyword, setKeyword] = useState('')
  const [searchKeyword, setSearchKeyword] = useState('')
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [itemToDelete, setItemToDelete] = useState<{ id: string; rule: string } | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  // 监听 URL 参数变化，强制刷新数据
  useEffect(() => {
    const timestamp = searchParams.get('t')
    if (timestamp) {
      queryClient.invalidateQueries({ queryKey: ['kb-list'] })
    }
  }, [searchParams, queryClient])

  const { data, isFetching } = useQuery({
    queryKey: ['kb-list', page, pageSize, searchKeyword],
    queryFn: () => RobustaAPI.listKnowledge({
      page,
      page_size: pageSize,
      rule_name: searchKeyword || undefined
    }),
    staleTime: 0,
  })

  const items = data?.data || []
  const total = data?.pagination?.total || 0
  const totalPages = Math.ceil(total / pageSize)

  const handleSearch = () => {
    setSearchKeyword(keyword)
    setPage(1)
  }

  const handleClearSearch = () => {
    setKeyword('')
    setSearchKeyword('')
    setPage(1)
  }

  const deleteMutation = useMutation({
    mutationFn: (id: string) => RobustaAPI.deleteKnowledge(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['kb-list'] })
      setDeleteDialogOpen(false)
      setItemToDelete(null)
      setDeleteError(null)
    },
    onError: (error: any) => {
      setDeleteError(error?.response?.data?.message || error?.message || "删除知识库条目时出错")
    },
  })

  const handleDeleteClick = (id: string, rule: string) => {
    setItemToDelete({ id, rule })
    setDeleteError(null)
    setDeleteDialogOpen(true)
  }

  const handleConfirmDelete = () => {
    if (itemToDelete) {
      deleteMutation.mutate(itemToDelete.id)
    }
  }

  // Animation variants
  const containerVariants = {
    hidden: { opacity: 0 },
    show: {
      opacity: 1,
      transition: {
        staggerChildren: 0.05
      }
    }
  }

  const itemVariants = {
    hidden: { opacity: 0, y: 20 },
    show: { opacity: 1, y: 0 }
  }

  return (
    <div className="min-h-screen bg-background p-4 md:p-6 space-y-6">
      {/* Header Section */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 px-4 md:px-6">
        <div className="space-y-1">
          <h1 className="text-2xl font-bold tracking-tight flex items-center gap-2">
            <div className="w-9 h-9 bg-primary/10 rounded-lg flex items-center justify-center">
              <BookOpen className="w-5 h-5 text-primary" />
            </div>
            经验指南
          </h1>
          <p className="text-sm text-muted-foreground">
            管理和维护告警处理的最佳实践与知识库
          </p>
        </div>
        <Button
          asChild
          className="h-10 shadow-sm px-4"
        >
          <Link href={resolveAppPath('/knowledge/new')}>
            <Plus className="mr-2 h-4 w-4" />
            新建
          </Link>
        </Button>
      </div>

      {/* Search Section - Using Card Layout as per design rules */}
      <Card className="border-2 border-dashed border-muted-foreground/20 hover:border-primary/50 transition-colors">
        <CardHeader className="pb-4">
          <CardTitle className="text-lg flex items-center gap-2">
            <Search className="w-5 h-5 text-primary" />
            搜索与筛选
          </CardTitle>
          <CardDescription>查找特定的告警规则或知识库条目</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col md:flex-row gap-4">
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-3 h-5 w-5 text-muted-foreground" />
              <Input
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
                placeholder="搜索告警规则名 (例如 KubePodCrashLooping)..."
                className="pl-10 h-11"
                onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
              />
            </div>
            <div className="flex items-center gap-3">
              <Button
                onClick={handleSearch}
                disabled={isFetching}
                className="h-11 px-8"
              >
                {isFetching ? '搜索中...' : '搜索'}
              </Button>
              {searchKeyword && (
                <Button
                  variant="outline"
                  onClick={handleClearSearch}
                  className="h-11"
                >
                  清除
                </Button>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Content Section */}
      <div className="space-y-6">
        <div className="flex items-center justify-between px-1">
          <h2 className="text-lg font-semibold flex items-center gap-2">
            {searchKeyword ? '搜索结果' : '全部条目'}
            <Badge variant="secondary" className="ml-2">
              {total}
            </Badge>
          </h2>
          {totalPages > 1 && (
            <span className="text-sm text-muted-foreground">
              第 {page} / {totalPages} 页
            </span>
          )}
        </div>

        {isFetching ? (
          <div className="grid gap-4">
            {[1, 2, 3].map((i) => (
              <div key={i} className="h-24 rounded-xl bg-muted/50 animate-pulse" />
            ))}
          </div>
        ) : items.length > 0 ? (
          <motion.div
            variants={containerVariants}
            initial="hidden"
            animate="show"
            className="grid gap-4"
          >
            <AnimatePresence mode='wait'>
              {items.map((it) => (
                <motion.div
                  key={it.id}
                  variants={itemVariants}
                  layout
                >
                  <Card className="hover:border-primary/50 transition-all hover:shadow-sm">
                    <CardContent className="p-4 md:p-6 flex flex-col md:flex-row md:items-center gap-6">
                      {/* Icon & Title */}
                      <div className="flex items-start md:items-center gap-4 flex-1 min-w-0">
                        <div className="w-12 h-12 bg-primary/10 rounded-xl flex items-center justify-center flex-shrink-0">
                          <ShieldAlert className="w-6 h-6 text-primary" />
                        </div>
                        <div className="space-y-1.5 min-w-0">
                          <Link href={resolveAppPath(`/knowledge/${it.id}`)} className="block hover:text-primary transition-colors">
                            <h3 className="font-bold text-lg truncate pr-4 font-mono">
                              {it.alert_rule_name}
                            </h3>
                          </Link>
                          <div className="flex items-center gap-4 text-sm text-muted-foreground">
                            <span className="flex items-center gap-1.5">
                              <Badge variant="outline" className="text-xs font-normal">v{it.version}</Badge>
                            </span>
                            <span className="flex items-center gap-1.5">
                              <Clock className="h-3.5 w-3.5" />
                              {it.updated_at ? new Date(it.updated_at).toLocaleDateString('zh-CN') : '-'}
                            </span>
                          </div>
                        </div>
                      </div>

                      {/* Status & Actions */}
                      <div className="flex items-center justify-between md:justify-end gap-6 mt-2 md:mt-0 pl-16 md:pl-0">
                        <Badge
                          variant={it.status === 'published' ? 'default' : 'secondary'}
                          className="capitalize px-3 py-1"
                        >
                          {it.status === 'published' ? '已发布' : '草稿'}
                        </Badge>

                        <div className="flex items-center gap-2">
                          <Button variant="ghost" size="icon" asChild className="h-9 w-9 hover:bg-primary/10 hover:text-primary" title="查看">
                            <Link href={resolveAppPath(`/knowledge/${it.id}`)}>
                              <ArrowRight className="h-4 w-4" />
                            </Link>
                          </Button>

                          <Button variant="ghost" size="icon" asChild className="h-9 w-9 hover:bg-primary/10 hover:text-primary" title="编辑">
                            <Link href={resolveAppPath(`/knowledge/${it.id}/edit`)}>
                              <Edit className="h-4 w-4" />
                            </Link>
                          </Button>

                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-9 w-9 hover:bg-destructive/10 hover:text-destructive"
                            onClick={() => handleDeleteClick(it.id, it.alert_rule_name)}
                            title="删除"
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                </motion.div>
              ))}
            </AnimatePresence>
          </motion.div>
        ) : (
          <Card className="border-dashed">
            <CardContent className="flex flex-col items-center justify-center py-16 text-center">
              <div className="w-16 h-16 bg-muted rounded-full flex items-center justify-center mb-4">
                <Search className="w-8 h-8 text-muted-foreground" />
              </div>
              <h3 className="text-lg font-medium">
                {searchKeyword ? '未找到匹配的条目' : '暂无知识库条目'}
              </h3>
              <p className="text-muted-foreground mt-2 max-w-sm">
                {searchKeyword
                  ? '尝试更换关键词搜索，或者清除搜索条件'
                  : '开始创建您的第一个告警处理经验指南。'}
              </p>
              {!searchKeyword && (
                <Button asChild className="mt-6 h-11">
                  <Link href={resolveAppPath('/knowledge/new')}>
                    创建第一条指南
                  </Link>
                </Button>
              )}
            </CardContent>
          </Card>
        )}

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-center gap-2 pt-6">
            <Button
              variant="outline"
              size="icon"
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page === 1 || isFetching}
              className="h-9 w-9"
            >
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <div className="flex items-center gap-1">
              {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                let pageNum
                if (totalPages <= 5) {
                  pageNum = i + 1
                } else if (page <= 3) {
                  pageNum = i + 1
                } else if (page >= totalPages - 2) {
                  pageNum = totalPages - 4 + i
                } else {
                  pageNum = page - 2 + i
                }
                return (
                  <Button
                    key={pageNum}
                    variant={page === pageNum ? 'default' : 'ghost'}
                    size="sm"
                    onClick={() => setPage(pageNum)}
                    disabled={isFetching}
                    className="w-9 h-9"
                  >
                    {pageNum}
                  </Button>
                )
              })}
            </div>
            <Button
              variant="outline"
              size="icon"
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page === totalPages || isFetching}
              className="h-9 w-9"
            >
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        )}
      </div>

      {/* Delete Dialog - Strictly following the design rules */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent className="max-w-[500px]">
          <AlertDialogHeader className="space-y-3 pb-6">
            <AlertDialogTitle className="text-2xl font-bold flex items-center gap-3">
              <div className="w-8 h-8 bg-destructive/10 rounded-lg flex items-center justify-center">
                <AlertTriangle className="w-4 h-4 text-destructive" />
              </div>
              确认删除
            </AlertDialogTitle>
            <AlertDialogDescription className="text-base">
              确定要删除知识库条目 <span className="font-semibold text-foreground">"{itemToDelete?.rule}"</span> 吗？
              此操作无法撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>

          {deleteError && (
            <div className="rounded-md bg-destructive/15 p-3 text-sm text-destructive mb-4">
              {deleteError}
            </div>
          )}

          <AlertDialogFooter className="flex flex-col sm:flex-row justify-end gap-3 pt-6 border-t">
            <AlertDialogCancel disabled={deleteMutation.isPending} className="h-11 mt-0">取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirmDelete}
              disabled={deleteMutation.isPending}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90 h-11"
            >
              {deleteMutation.isPending ? '删除中...' : '确认删除'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
