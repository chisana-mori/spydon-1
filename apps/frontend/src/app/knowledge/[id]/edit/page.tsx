"use client"

import React, { useEffect, useState, use } from 'react'
import { useRouter } from 'next/navigation'
import { resolveAppPath, appConfig } from '@/config'
import { useQuery, useMutation } from '@tanstack/react-query'
import KnowledgeEditor, { type KnowledgeEditorValue } from '@/components/knowledge/KnowledgeEditor'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { RobustaAPI } from '@/lib/api'
import { toast } from 'sonner'

export default function KnowledgeEditPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params)
  const router = useRouter()
  const editorRef = React.useRef<any>(null)

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['kb', id],
    queryFn: async () => {
      const res = await fetch(`${appConfig.apiBaseUrl}/knowledge/${id}?include_manifest=true`, { credentials: 'include' })
      if (!res.ok) throw new Error(await res.text())
      return res.json()
    },
    enabled: !!id,
  })

  const article = data?.data?.article
  const manifest = data?.data?.manifest

  const [submitting, setSubmitting] = useState(false)
  const [latestEditorValue, setLatestEditorValue] = useState<any>(null)

  const handleSave = async (value: KnowledgeEditorValue) => {
    setSubmitting(true)
    try {
      // 1) 先保存内容到 manifest
      await RobustaAPI.updateKnowledge(id, {
        schema: 'kb-manifest@v1',
        articleId: manifest?.articleId || '',
        alertRuleName: value.alertRuleName,
        title: value.title,
        status: article?.status || 'draft',
        version: article?.version || 1,
        content: {
          tiptap: value.tiptap
        },
        tags: value.tags,
      })

      // 2) 直接发布
      await RobustaAPI.publishKnowledge(id, '发布')
      toast.success('已发布')

      // 跳转回列表页并强制刷新
      router.push(`/knowledge?t=${Date.now()}`)
    } catch (e: any) {
      toast.error(e?.message || '保存失败')
    } finally {
      setSubmitting(false)
    }
  }

  if (isLoading) {
    return <div>加载中...</div>
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <Button variant="ghost" size="sm" onClick={() => router.push(resolveAppPath('/knowledge'))}>
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="mr-2">
            <path d="m12 19-7-7 7-7"/>
            <path d="M19 12H5"/>
          </svg>
          返回列表
        </Button>
        <Button onClick={() => editorRef.current?.submit()} disabled={submitting || !article}>
          {submitting ? '发布中...' : '发布'}
        </Button>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Badge
              className={`text-xs ${
                article?.status === 'published'
                  ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400 border-green-200'
                  : 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400 border-yellow-200'
              }`}
            >
              {article?.status === 'published' ? '已发布' : article?.status === 'draft' ? '草稿' : article?.status}
            </Badge>
            <Badge className="text-xs bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400 border-blue-200">
              版本 {article?.version || 1}
            </Badge>
          </div>
        </CardHeader>
        <CardContent>
          <KnowledgeEditor
            ref={editorRef}
            submitting={submitting}
            onSubmit={handleSave}
            onChange={(v) => setLatestEditorValue(v)}
            articleId={id}
            value={{
              title: article?.title,
              alertRuleName: article?.alert_rule_name,
              tags: Array.isArray(article?.tags) ? (article?.tags as string[]) : undefined,
              tiptap: manifest?.content?.tiptap || { type: 'doc', content: [] },
            }}
          />
        </CardContent>
      </Card>
    </div>
  )
}
