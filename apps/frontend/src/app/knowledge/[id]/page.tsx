"use client"

import React from 'react'
import { useParams, useRouter } from 'next/navigation'
import { useQuery } from '@tanstack/react-query'
import { RobustaAPI } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ArrowLeft, Edit, Calendar, Tag, AlertCircle } from 'lucide-react'
import Link from 'next/link'
import { resolveAppPath } from '@/config'
import { KnowledgeViewer } from '@/components/knowledge/KnowledgeViewer'

export default function KnowledgeViewPage() {
  const params = useParams()
  const router = useRouter()
  const id = params.id as string

  const { data, isLoading, error } = useQuery({
    queryKey: ['knowledge-view', id],
    queryFn: () => RobustaAPI.getKnowledgeById(id, true),
    enabled: !!id,
  })

  const article = data?.data?.article
  const manifest = data?.data?.manifest

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="sm" onClick={() => router.back()}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            返回
          </Button>
        </div>
        <Card>
          <CardContent className="py-12">
            <div className="text-center text-muted-foreground">加载中...</div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (error || !article) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="sm" onClick={() => router.back()}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            返回
          </Button>
        </div>
        <Card>
          <CardContent className="py-12">
            <div className="text-center text-muted-foreground">未找到该条目</div>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6 p-4 md:p-6">
      {/* 头部操作栏 */}
      <div className="flex items-center justify-between">
        <Button variant="ghost" size="sm" onClick={() => router.back()}>
          <ArrowLeft className="h-4 w-4 mr-2" />
          返回列表
        </Button>
        <Button asChild>
          <Link href={resolveAppPath(`/knowledge/${id}/edit`)}>
            <Edit className="h-4 w-4 mr-2" />
            编辑
          </Link>
        </Button>
      </div>

      {/* 文章信息卡片 */}
      <Card>
        <CardHeader className="space-y-4">
          {/* 状态标签 */}
          <div className="flex items-center gap-2">
            <span
              className={`px-3 py-1 rounded-full text-xs font-medium ${article.status === 'published'
                ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                : 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400'
                }`}
            >
              {article.status === 'published' ? '已发布' : '草稿'}
            </span>
            <span className="text-xs text-muted-foreground">版本 v{article.version}</span>
          </div>

          {/* 告警规则名作为标题 */}
          <CardTitle className="text-3xl font-bold font-mono">{article.alert_rule_name}</CardTitle>

          {/* 元信息 */}
          <div className="flex flex-wrap gap-4 text-sm">

            {article.severity && (
              <div className="flex items-center gap-2">
                <Tag className="h-4 w-4 text-purple-500" />
                <span className="text-muted-foreground">严重级别：</span>
                <span
                  className={`px-3 py-1 rounded-md text-xs font-semibold ${article.severity === 'critical'
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

            {article.updated_at && (
              <div className="flex items-center gap-2">
                <Calendar className="h-4 w-4 text-muted-foreground" />
                <span className="text-muted-foreground">更新时间：</span>
                <span className="font-medium text-foreground">
                  {new Date(article.updated_at).toLocaleString('zh-CN')}
                </span>
              </div>
            )}
          </div>

          {/* 标签 */}
          {article.tags && article.tags.length > 0 && (
            <div className="flex items-center gap-2">
              <Tag className="h-4 w-4 text-muted-foreground" />
              <div className="flex flex-wrap gap-2">
                {article.tags.map((tag: string, index: number) => (
                  <span
                    key={index}
                    className="px-3 py-1 bg-gradient-to-r from-purple-100 to-pink-100 text-purple-700 dark:from-purple-900/30 dark:to-pink-900/30 dark:text-purple-400 rounded-full text-xs font-semibold"
                  >
                    #{tag}
                  </span>
                ))}
              </div>
            </div>
          )}
        </CardHeader>
      </Card>

      {/* 内容查看器 */}
      <Card>
        <CardContent className="pt-6">
          {manifest ? (
            <KnowledgeViewer manifest={manifest} />
          ) : (
            <div className="text-center py-12 text-muted-foreground">暂无内容</div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
