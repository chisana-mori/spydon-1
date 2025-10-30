"use client"

import React, { useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { resolveAppPath } from '@/config'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { RobustaAPI } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { ChevronLeft, ChevronRight, Trash2 } from 'lucide-react'
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

export default function KnowledgeHomePage() {
  const router = useRouter()
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [keyword, setKeyword] = useState('')
  const [searchKeyword, setSearchKeyword] = useState('')
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [itemToDelete, setItemToDelete] = useState<{ id: string; title: string } | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  const { data, isFetching } = useQuery({
    queryKey: ['kb-list', page, pageSize, searchKeyword],
    queryFn: () => RobustaAPI.listKnowledge({ 
      page, 
      page_size: pageSize,
      alert_rule_name: searchKeyword || undefined
    }),
    staleTime: 30_000,
  })

  const items = data?.data || []
  const total = data?.total || data?.pagination?.total || 0
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

  const handleDeleteClick = (id: string, title: string) => {
    setItemToDelete({ id, title })
    setDeleteError(null)
    setDeleteDialogOpen(true)
  }

  const handleConfirmDelete = () => {
    if (itemToDelete) {
      deleteMutation.mutate(itemToDelete.id)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">经验指南</h1>
        <Button asChild>
          <Link href={resolveAppPath('/knowledge/new')}>新建条目</Link>
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>搜索与筛选</CardTitle>
          <CardDescription>按告警规则名搜索（例如 KubePodCrashLooping）</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-3">
            <Input 
              value={keyword} 
              onChange={(e) => setKeyword(e.target.value)} 
              placeholder="告警规则名（可选）"
              onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
            />
            <Button onClick={handleSearch} disabled={isFetching}>
              {isFetching ? '搜索中...' : '搜索'}
            </Button>
            {searchKeyword && (
              <Button variant="outline" onClick={handleClearSearch}>
                清除
              </Button>
            )}
          </div>
        </CardContent>
      </Card>

      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">
            {searchKeyword ? `搜索结果` : `全部条目`}
            <span className="ml-2 text-sm font-normal text-muted-foreground">共 {total} 条</span>
          </h2>
          {totalPages > 1 && (
            <div className="text-sm text-muted-foreground">
              第 {page} / {totalPages} 页
            </div>
          )}
        </div>

        {isFetching ? (
          <Card>
            <CardContent className="py-12">
              <div className="text-center text-muted-foreground">加载中...</div>
            </CardContent>
          </Card>
        ) : items.length > 0 ? (
          <>
            <div className="border rounded-lg overflow-hidden bg-white">
              {items.map((it, index) => (
                <div 
                  key={it.id} 
                  className={`flex items-center px-6 py-3 hover:bg-gray-50 transition-colors ${
                    index !== items.length - 1 ? 'border-b border-gray-100' : ''
                  }`}
                >
                  {/* 状态指示器 - 固定宽度 */}
                  <div className="w-16 flex-shrink-0">
                    <span
                      className={`inline-block px-2 py-0.5 rounded-full text-xs font-medium whitespace-nowrap ${
                        it.status === 'published'
                          ? 'bg-green-100 text-green-700'
                          : 'bg-yellow-100 text-yellow-700'
                      }`}
                    >
                      {it.status === 'published' ? '已发布' : '草稿'}
                    </span>
                  </div>

                  {/* 标题 - 弹性宽度 */}
                  <div className="flex-1 min-w-0 px-4">
                    <Link
                      href={resolveAppPath(`/knowledge/${it.id}`)}
                      className="text-base font-medium text-gray-900 hover:text-blue-600 transition-colors block truncate"
                    >
                      {it.title}
                    </Link>
                  </div>

                  {/* 规则名 - 固定宽度 */}
                  <div className="w-64 flex-shrink-0 px-4">
                    <code className="block px-2.5 py-1 bg-orange-50 border border-orange-200 rounded text-orange-700 font-mono text-sm font-medium truncate">
                      {it.alert_rule_name}
                    </code>
                  </div>

                  {/* 版本 - 固定宽度 */}
                  <div className="w-16 flex-shrink-0 text-center">
                    <span className="text-sm text-gray-500">
                      v{it.version}
                    </span>
                  </div>

                  {/* 更新时间 - 固定宽度 */}
                  <div className="w-28 flex-shrink-0 text-center">
                    {it.updated_at && (
                      <span className="text-sm text-gray-500">
                        {new Date(it.updated_at).toLocaleDateString('zh-CN')}
                      </span>
                    )}
                  </div>

                  {/* 操作按钮 - 固定宽度 */}
                  <div className="w-44 flex-shrink-0 flex items-center justify-end gap-2">
                    <Button variant="outline" asChild size="sm" className="h-8">
                      <Link href={resolveAppPath(`/knowledge/${it.id}`)}>查看</Link>
                    </Button>
                    <Button asChild size="sm" className="h-8">
                      <Link href={resolveAppPath(`/knowledge/${it.id}/edit`)}>编辑</Link>
                    </Button>
                    <Button 
                      variant="outline" 
                      size="sm"
                      onClick={() => handleDeleteClick(it.id, it.title)}
                      className="text-destructive hover:text-destructive h-8 w-8 p-0"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>

            {/* 分页控件 */}
            {totalPages > 1 && (
              <div className="flex items-center justify-center gap-2 pt-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page === 1 || isFetching}
                >
                  <ChevronLeft className="h-4 w-4 mr-1" />
                  上一页
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
                        variant={page === pageNum ? 'default' : 'outline'}
                        size="sm"
                        onClick={() => setPage(pageNum)}
                        disabled={isFetching}
                        className="w-9"
                      >
                        {pageNum}
                      </Button>
                    )
                  })}
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page === totalPages || isFetching}
                >
                  下一页
                  <ChevronRight className="h-4 w-4 ml-1" />
                </Button>
              </div>
            )}
          </>
        ) : (
          <Card>
            <CardContent className="py-12">
              <div className="text-center text-muted-foreground">
                {searchKeyword ? '未找到匹配的条目' : '暂无条目，点击右上角新建'}
              </div>
            </CardContent>
          </Card>
        )}
      </div>

      {/* 删除确认对话框 */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>
              确定要删除知识库条目 <span className="font-semibold text-foreground">"{itemToDelete?.title}"</span> 吗？
              此操作无法撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          {deleteError && (
            <div className="rounded-md bg-destructive/15 p-3 text-sm text-destructive">
              {deleteError}
            </div>
          )}
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteMutation.isPending}>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirmDelete}
              disabled={deleteMutation.isPending}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {deleteMutation.isPending ? '删除中...' : '删除'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
