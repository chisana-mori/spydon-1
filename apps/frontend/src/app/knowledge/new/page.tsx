"use client"

import React, { useState } from 'react'
import { useRouter } from 'next/navigation'
import KnowledgeEditor, { type KnowledgeEditorValue } from '@/components/knowledge/KnowledgeEditor'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { RobustaAPI } from '@/lib/api'
import { toast } from 'sonner'

export default function KnowledgeCreatePage() {
  const router = useRouter()
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async (value: KnowledgeEditorValue) => {
    setSubmitting(true)
    try {
      // 1) 创建条目
      const created = await RobustaAPI.createKnowledge({
        alert_rule_name: value.alertRuleName,
        title: value.title,
        tags: value.tags,
      })
      const id = created.data?.id
      if (!id) throw new Error('创建失败：缺少ID')

      // 2) 保存 manifest（只包含 tiptap 格式）
      await RobustaAPI.updateKnowledge(id, {
        schema: 'kb-manifest@v1',
        articleId: '',
        alertRuleName: value.alertRuleName,
        title: value.title,
        status: 'draft',
        version: 1,
        content: { 
          tiptap: value.tiptap
        },
        tags: value.tags,
      })

      toast.success('已保存草稿')
      router.push(`/knowledge/${id}/edit`)
    } catch (e: any) {
      toast.error(e?.message || '保存失败')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <button
          onClick={() => router.back()}
          className="inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground h-9 px-3"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="mr-2">
            <path d="m12 19-7-7 7-7"/>
            <path d="M19 12H5"/>
          </svg>
          返回
        </button>
      </div>
      <Card>
        <CardHeader>
          <CardTitle>新建知识条目</CardTitle>
          <CardDescription>填写元信息与正文内容，保存后可发布</CardDescription>
        </CardHeader>
        <CardContent>
          <KnowledgeEditor submitting={submitting} onSubmit={handleSubmit} />
        </CardContent>
      </Card>
    </div>
  )
}
