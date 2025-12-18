"use client"

import React, { useState, Suspense } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import KnowledgeEditor, { type KnowledgeEditorValue } from '@/components/knowledge/KnowledgeEditor'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { RobustaAPI } from '@/lib/api'
import { toast } from 'sonner'

// 包装组件以支持 Suspense 边界
export default function KnowledgeCreatePage() {
  return (
    <Suspense fallback={<div className="p-6">加载中...</div>}>
      <KnowledgeCreatePageContent />
    </Suspense>
  )
}

function KnowledgeCreatePageContent() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const editorRef = React.useRef<any>(null)
  const [submitting, setSubmitting] = useState(false)

  // 从 URL 参数获取规则名
  const ruleFromUrl = searchParams.get('rule') || ''

  const handleSubmit = async (value: KnowledgeEditorValue) => {
    setSubmitting(true)
    try {
      // 1) 创建条目
      const created = await RobustaAPI.createKnowledge({
        alert_rule_name: value.alertRuleName,
        tags: value.tags,
      })
      const id = created.data?.id
      if (!id) throw new Error('创建失败：缺少ID')

      // 2) 保存 manifest（只包含 tiptap 格式）
      await RobustaAPI.updateKnowledge(id, {
        schema: 'kb-manifest@v1',
        articleId: '',
        alertRuleName: value.alertRuleName,
        status: 'draft',
        version: 1,
        content: {
          tiptap: value.tiptap
        },
        markdown: value.markdown,
        tags: value.tags,
      })

      // 3) 直接发布
      await RobustaAPI.publishKnowledge(id, '发布')

      toast.success('已发布')

      // 跳转到列表页并添加时间戳强制刷新
      router.push(`/knowledge?t=${Date.now()}`)
    } catch (e: any) {
      toast.error(e?.message || '发布失败')
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
            <path d="m12 19-7-7 7-7" />
            <path d="M19 12H5" />
          </svg>
          返回
        </button>
        <Button onClick={() => editorRef.current?.submit()} disabled={submitting}>
          {submitting ? '发布中...' : '发布'}
        </Button>
      </div>
      <Card>
        <CardHeader>
          <CardTitle>新建知识条目</CardTitle>
          <CardDescription>填写元信息与正文内容，点击发布即可创建</CardDescription>
        </CardHeader>
        <CardContent>
          <KnowledgeEditor
            ref={editorRef}
            submitting={submitting}
            onSubmit={handleSubmit}
            value={{
              alertRuleName: ruleFromUrl,
            }}
          />
        </CardContent>
      </Card>
    </div>
  )
}
