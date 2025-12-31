"use client"

import React from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { RobustaAPI } from '@/lib/api'
import KnowledgeViewer from '@/components/knowledge/KnowledgeViewer'
import KnowledgeEditor, { type KnowledgeEditorRef, type KnowledgeEditorValue } from '@/components/knowledge/KnowledgeEditor'

import { Button } from '@/components/ui/button'
import { ExternalLink, Calendar, Tag, AlertCircle, BookOpen, Plus } from 'lucide-react'
import Link from 'next/link'
import { resolveAppPath } from '@/config'
import { toast } from 'sonner'

interface Props {
  alertRuleName: string
}

export default function AlertKnowledgePanel({ alertRuleName }: Props) {
  const [isCreating, setIsCreating] = React.useState(false)
  const editorRef = React.useRef<KnowledgeEditorRef>(null)
  const queryClient = useQueryClient()
  const { data: listData, isLoading: isLoadingList } = useQuery({
    queryKey: ['kb-by-rule', alertRuleName],
    queryFn: () => RobustaAPI.queryKnowledgeByRule(alertRuleName, 1),
    enabled: !!alertRuleName,
    staleTime: 30_000,
  })

  const items = (listData as any) || []
  const first = items[0]

  // 获取完整的 manifest 数据
  const { data: detailData, isLoading: isLoadingDetail } = useQuery({
    queryKey: ['kb-detail', first?.id],
    queryFn: () => RobustaAPI.getKnowledgeById(first.id, true),
    enabled: !!first?.id,
    staleTime: 30_000,
  })

  const article = (detailData as any)?.article
  const manifest = (detailData as any)?.manifest

  const isLoading = isLoadingList || isLoadingDetail
  const createGuide = useMutation({
    mutationFn: async (value: KnowledgeEditorValue) => {
      const created = await RobustaAPI.createKnowledge({
        alert_rule_name: value.alertRuleName,
        tags: value.tags,
      })
      const id = created.id
      if (!id) throw new Error('创建失败：缺少ID')

      await RobustaAPI.updateKnowledge(id, {
        schema: 'kb-manifest@v1',
        articleId: '',
        alertRuleName: value.alertRuleName,
        status: 'draft',
        version: 1,
        content: {
          tiptap: value.tiptap,
        },
        markdown: value.markdown,
        tags: value.tags,
      })

      await RobustaAPI.publishKnowledge(id, '发布')
      return id
    },
    onSuccess: async () => {
      toast.success('经验指南已创建并发布')
      setIsCreating(false)
      await queryClient.invalidateQueries({ queryKey: ['kb-by-rule', alertRuleName] })
    },
    onError: (error: any) => {
      toast.error(error?.message || '创建失败')
    },
  })

  const handleInlineSubmit = async (value: KnowledgeEditorValue) => {
    await createGuide.mutateAsync(value)
  }

  return (
    <div className="space-y-4">
      {isLoading ? (
        <div className="space-y-3">
          <div className="h-6 rounded bg-muted/50 animate-pulse w-3/4" />
          <div className="h-4 rounded bg-muted/30 animate-pulse w-1/2" />
          <div className="h-32 rounded bg-muted/30 animate-pulse" />
        </div>
      ) : first && article ? (
        <div className="space-y-4">
          {/* 文章信息 - 居中布局 */}
          <div className="space-y-4 pb-4 border-b">
            {/* 标题和状态 - 居中 */}
            <div className="flex flex-col items-center gap-3">
              <div className="flex items-center gap-3">
                <h3 className="text-xl font-semibold text-foreground">{article.title}</h3>
                <span
                  className={`px-2.5 py-0.5 rounded-full text-xs font-medium whitespace-nowrap ${article.status === 'published'
                    ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                    : 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400'
                    }`}
                >
                  {article.status === 'published' ? '已发布' : '草稿'}
                </span>
              </div>

              {/* 元信息 - 居中 */}
              <div className="flex flex-wrap items-center justify-center gap-x-4 gap-y-2 text-sm">
                <div className="flex items-center gap-1.5">
                  <AlertCircle className="h-4 w-4 text-blue-500" />
                  <span className="text-muted-foreground">规则</span>
                  <span className="px-2.5 py-1 bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400 rounded-md font-mono text-xs font-semibold">
                    {article.alert_rule_name}
                  </span>
                </div>

                <div className="flex items-center gap-1.5">
                  <Calendar className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">更新</span>
                  <span className="font-medium text-foreground">
                    {new Date(article.updated_at).toLocaleDateString('zh-CN')}
                  </span>
                </div>

                <div className="flex items-center gap-1.5">
                  <span className="text-muted-foreground">版本</span>
                  <span className="font-medium text-foreground">v{article.version}</span>
                </div>

                {article.severity && (
                  <div className="flex items-center gap-1.5">
                    <Tag className="h-4 w-4 text-purple-500" />
                    <span className="text-muted-foreground">级别</span>
                    <span
                      className={`px-2.5 py-1 rounded-md text-xs font-semibold ${article.severity === 'critical'
                        ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
                        : article.severity === 'high'
                          ? 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
                          : article.severity === 'medium'
                            ? 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400'
                            : 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
                        }`}
                    >
                      {article.severity}
                    </span>
                  </div>
                )}
              </div>

              {/* 标签 - 居中 */}
              {article.tags && article.tags.length > 0 && (
                <div className="flex items-center justify-center gap-2">
                  <Tag className="h-4 w-4 text-muted-foreground" />
                  <div className="flex flex-wrap justify-center gap-2">
                    {article.tags.map((tag: string, index: number) => (
                      <span
                        key={index}
                        className="px-2.5 py-1 bg-gradient-to-r from-purple-100 to-pink-100 text-purple-700 dark:from-purple-900/30 dark:to-pink-900/30 dark:text-purple-400 rounded-full text-xs font-semibold"
                      >
                        #{tag}
                      </span>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* 内容查看器 */}
          <div className="knowledge-content">
            {manifest ? (
              <KnowledgeViewer manifest={manifest} />
            ) : (
              <div className="text-sm text-muted-foreground py-4">正在加载内容...</div>
            )}
          </div>
        </div>
      ) : isCreating ? (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="text-sm text-muted-foreground">
              为规则 <code className="px-2 py-0.5 bg-muted rounded text-xs font-mono">{alertRuleName}</code> 创建新的经验指南
            </div>
            <div className="flex items-center gap-2">
              <Button variant="ghost" size="sm" onClick={() => setIsCreating(false)} disabled={createGuide.isPending}>
                取消
              </Button>
              <Button size="sm" onClick={() => editorRef.current?.submit()} disabled={createGuide.isPending}>
                {createGuide.isPending ? '保存中...' : '保存并发布'}
              </Button>
            </div>
          </div>
          <div className="border rounded-lg p-4">
            <KnowledgeEditor
              ref={editorRef}
              submitting={createGuide.isPending}
              onSubmit={handleInlineSubmit}
              value={{
                alertRuleName,
              }}
            />
          </div>
        </div>
      ) : (
        <div className="flex flex-col items-center justify-center py-12 space-y-6">
          {/* 图标 */}
          <div className="relative">
            <div className="absolute inset-0 bg-primary/10 rounded-full blur-2xl" />
            <div className="relative bg-gradient-to-br from-primary/20 to-primary/5 p-6 rounded-full">
              <BookOpen className="h-16 w-16 text-primary" />
            </div>
          </div>

          {/* 文字说明 */}
          <div className="text-center space-y-2 max-w-md">
            <h3 className="text-lg font-semibold text-foreground">
              暂无匹配的经验指南
            </h3>
            <p className="text-sm text-muted-foreground">
              当前告警规则 <code className="px-2 py-0.5 bg-muted rounded text-xs font-mono">{alertRuleName}</code> 还没有对应的处理经验
            </p>
          </div>

          {/* 操作按钮 */}
          <div className="flex items-center gap-3">
            <Button variant="default" size="sm" onClick={() => setIsCreating(true)}>
              <Plus className="h-4 w-4 mr-2" />
              创建新指南
            </Button>
            <Button variant="outline" size="sm" asChild>
              <Link href={resolveAppPath('/knowledge') as any}>
                <BookOpen className="h-4 w-4 mr-2" />
                浏览所有指南
              </Link>
            </Button>
          </div>

          {/* 提示信息 */}
          <div className="mt-4 p-4 bg-muted/50 rounded-lg border border-dashed max-w-md">
            <p className="text-xs text-muted-foreground text-center">
              💡 创建经验指南可以帮助团队快速定位和解决类似问题
            </p>
          </div>
        </div>
      )}
    </div>
  )
}
