"use client"

import React, { useEffect, useState, use } from 'react'
import { useRouter } from 'next/navigation'
import { useQuery, useMutation } from '@tanstack/react-query'
import KnowledgeEditor, { type KnowledgeEditorValue } from '@/components/knowledge/KnowledgeEditor'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { RobustaAPI } from '@/lib/api'
import { toast } from 'sonner'

export default function KnowledgeEditPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params)
  const router = useRouter()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['kb', id],
    queryFn: async () => {
      const res = await fetch(`/api/v1/knowledge/${id}?include_manifest=true`, { credentials: 'include' })
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
      toast.success('保存成功')
      await refetch()
    } catch (e: any) {
      toast.error(e?.message || '保存失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handlePublish = async () => {
    setSubmitting(true)
    try {
      // 1) 先保存当前正文到 manifest
      const v = latestEditorValue || {}
      const latestTiptap = v.tiptap ?? manifest?.content?.tiptap ?? { type: 'doc', content: [] }
      await RobustaAPI.updateKnowledge(id, {
        schema: 'kb-manifest@v1',
        articleId: manifest?.articleId || '',
        alertRuleName: v.alertRuleName ?? article?.alert_rule_name ?? '',
        title: v.title ?? article?.title ?? '',
        status: article?.status || 'draft',
        version: article?.version || 1,
        content: { 
          tiptap: latestTiptap
        },
        tags: v.tags ?? (Array.isArray(article?.tags) ? (article?.tags as string[]) : undefined),
      })

      // 2) 再发布
      await RobustaAPI.publishKnowledge(id, '发布')
      toast.success('已发布')
      await refetch()
    } catch (e: any) {
      toast.error(e?.message || '发布失败')
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
        <h1 className="text-2xl font-bold">编辑知识条目</h1>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={() => router.push('/knowledge')}>返回</Button>
          <Button onClick={handlePublish} disabled={submitting || !article}>发布</Button>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{article?.title || '未命名'}</CardTitle>
          <CardDescription>状态：{article?.status} · 版本：{article?.version}</CardDescription>
        </CardHeader>
        <CardContent>
          <KnowledgeEditor
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
