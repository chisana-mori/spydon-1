"use client"

import React from 'react'
import { useQuery } from '@tanstack/react-query'
import { RobustaAPI } from '@/lib/api'
import KnowledgeViewer from '@/components/knowledge/KnowledgeViewer'

import { Button } from '@/components/ui/button'
import { ExternalLink, Calendar, Tag, AlertCircle, BookOpen, Plus } from 'lucide-react'
import Link from 'next/link'

interface Props {
  alertRuleName: string
}

export default function AlertKnowledgePanel({ alertRuleName }: Props) {
  const { data: listData, isLoading: isLoadingList } = useQuery({
    queryKey: ['kb-by-rule', alertRuleName],
    queryFn: () => RobustaAPI.queryKnowledgeByRule(alertRuleName, 1),
    enabled: !!alertRuleName,
    staleTime: 30_000,
  })

  const items = listData?.data || []
  const first = items[0]

  // 获取完整的 manifest 数据
  const { data: detailData, isLoading: isLoadingDetail } = useQuery({
    queryKey: ['kb-detail', first?.id],
    queryFn: () => RobustaAPI.getKnowledgeById(first.id, true),
    enabled: !!first?.id,
    staleTime: 30_000,
  })

  const article = detailData?.data?.article
  const manifest = detailData?.data?.manifest

  const isLoading = isLoadingList || isLoadingDetail

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
            {/* 文章信息 */}
            <div className="space-y-3 pb-4 border-b">
              {/* 右上角查看详情按钮 */}
              <div className="flex items-center justify-end">
                <Button variant="ghost" size="sm" asChild>
                  <Link href={`/knowledge/${first.id}` as any} target="_blank">
                    <ExternalLink className="h-4 w-4 mr-1" />
                    查看详情
                  </Link>
                </Button>
              </div>
              <div className="flex items-start justify-between gap-4">
                <h3 className="text-xl font-semibold text-foreground">{article.title}</h3>
                <span
                  className={`px-2.5 py-0.5 rounded-full text-xs font-medium whitespace-nowrap ${
                    article.status === 'published'
                      ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                      : 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400'
                  }`}
                >
                  {article.status === 'published' ? '已发布' : '草稿'}
                </span>
              </div>

              <div className="flex flex-wrap gap-x-4 gap-y-2 text-sm text-muted-foreground">
                <div className="flex items-center gap-1.5">
                  <AlertCircle className="h-4 w-4" />
                  <span>规则：</span>
                  <code className="px-1.5 py-0.5 bg-muted rounded text-foreground font-mono text-xs">
                    {article.alert_rule_name}
                  </code>
                </div>

                {article.severity && (
                  <div className="flex items-center gap-1.5">
                    <Tag className="h-4 w-4" />
                    <span>级别：</span>
                    <span
                      className={`font-medium ${
                        article.severity === 'critical'
                          ? 'text-red-600 dark:text-red-400'
                          : article.severity === 'high'
                          ? 'text-orange-600 dark:text-orange-400'
                          : article.severity === 'medium'
                          ? 'text-yellow-600 dark:text-yellow-400'
                          : 'text-blue-600 dark:text-blue-400'
                      }`}
                    >
                      {article.severity}
                    </span>
                  </div>
                )}

                <div className="flex items-center gap-1.5">
                  <Calendar className="h-4 w-4" />
                  <span>更新：</span>
                  <span className="font-medium text-foreground">
                    {new Date(article.updated_at).toLocaleDateString('zh-CN')}
                  </span>
                </div>

                <div className="flex items-center gap-1.5">
                  <span>版本：</span>
                  <span className="font-medium text-foreground">v{article.version}</span>
                </div>
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
              <Button variant="default" size="sm" asChild>
                <Link href={`/knowledge/new?rule=${encodeURIComponent(alertRuleName)}`}>
                  <Plus className="h-4 w-4 mr-2" />
                  创建新指南
                </Link>
              </Button>
              <Button variant="outline" size="sm" asChild>
                <Link href="/knowledge">
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

