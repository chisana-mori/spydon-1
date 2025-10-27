"use client"

import React, { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { toast } from 'sonner'
import { SimpleEditor } from '@/components/tiptap-templates/simple/simple-editor'

export interface KnowledgeEditorValue {
  title: string
  alertRuleName: string
  tags?: string[]
  tiptap: any
}

interface Props {
  value?: Partial<KnowledgeEditorValue>
  onChange?: (value: KnowledgeEditorValue) => void
  onSubmit?: (value: KnowledgeEditorValue) => Promise<void> | void
  submitting?: boolean
  articleId?: string
}

export default function KnowledgeEditor({ value, onChange, onSubmit, submitting, articleId }: Props) {
  const [title, setTitle] = useState(value?.title || '')
  const [rule, setRule] = useState(value?.alertRuleName || '')
  const [tags, setTags] = useState<string[]>(value?.tags || [])
  const [isMounted, setIsMounted] = useState(false)
  const [doc, setDoc] = useState<any>(value?.tiptap || { type: 'doc', content: [] })
  const [initialContent, setInitialContent] = useState<any>(value?.tiptap || { type: 'doc', content: [] })

  React.useEffect(() => {
    setIsMounted(true)
  }, [])

  // 当 value.tiptap 变化时更新初始内容
  React.useEffect(() => {
    if (value?.tiptap) {
      setInitialContent(value.tiptap)
      setDoc(value.tiptap)
    }
  }, [value?.tiptap])

  const handleSave = async () => {
    const json = doc || { type: 'doc', content: [] }
    const payload: KnowledgeEditorValue = {
      title,
      alertRuleName: rule,
      tags,
      tiptap: json,
    }

    await onSubmit?.(payload)
    toast.success('草稿已保存')
  }

  if (!isMounted) {
    return (
      <div className="space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-10 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
          ))}
        </div>
        <div className="h-[500px] bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
        <div className="h-10 w-20 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Form Fields */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div>
          <Label htmlFor="title">标题</Label>
          <Input id="title" value={title} onChange={(e) => setTitle(e.target.value)} placeholder="请输入标题" />
        </div>
        <div>
          <Label htmlFor="rule">告警规则名</Label>
          <Input id="rule" value={rule} onChange={(e) => setRule(e.target.value)} placeholder="如 KubePodCrashLooping" />
        </div>
        <div>
          <Label htmlFor="tags">标签（逗号分隔）</Label>
          <Input
            id="tags"
            value={(tags || []).join(',')}
            onChange={(e) => setTags(e.target.value.split(',').map((s) => s.trim()).filter(Boolean))}
            placeholder="k8s,pod,network"
          />
        </div>
      </div>

      {/* TipTap Simple Editor - 内嵌模式 */}
      <div
        style={{
          border: '1px solid hsl(var(--border))',
          borderRadius: '0.5rem',
          overflow: 'hidden',
        }}
      >
        <SimpleEditor
          key={JSON.stringify(initialContent)}
          variant="embed"
          embedHeight={500}
          initialContent={initialContent}
          onUpdate={(json) => {
            setDoc(json)
            onChange?.({
              title,
              alertRuleName: rule,
              tags,
              tiptap: json,
            })
          }}
        />
      </div>

      {/* Save Button */}
      <div className="flex items-center gap-3">
        <Button onClick={handleSave} disabled={submitting}>
          {submitting ? '保存中...' : '保存'}
        </Button>
      </div>
    </div>
  )
}
