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
