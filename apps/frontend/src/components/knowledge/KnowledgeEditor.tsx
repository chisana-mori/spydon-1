"use client"

import React, { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { toast } from 'sonner'
import { SimpleEditor } from '@/components/tiptap-templates/simple/simple-editor'
import { sreSkillTemplate } from '@/components/tiptap-templates/simple/data/sre-skill-template'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

const PREDEFINED_TAGS = [
  "Kubernetes",
  "Network",
  "Database",
  "JVM",
  "OOM",
  "CPU",
  "Memory",
  "Disk",
  "Application",
  "Security",
]

export interface KnowledgeEditorValue {
  alertRuleName: string
  tags?: string[]
  tiptap: any
  markdown?: string
}

export interface KnowledgeEditorRef {
  submit: () => Promise<void>
}

interface Props {
  value?: Partial<KnowledgeEditorValue>
  onChange?: (value: KnowledgeEditorValue) => void
  onSubmit?: (value: KnowledgeEditorValue) => Promise<void> | void
  submitting?: boolean
  articleId?: string
}

const KnowledgeEditor = React.forwardRef<KnowledgeEditorRef, Props>(({ value, onChange, onSubmit, submitting, articleId }, ref) => {
  const [rule, setRule] = useState(value?.alertRuleName || '')
  const [tags, setTags] = useState<string[]>(value?.tags || [])
  const [isMounted, setIsMounted] = useState(false)
  const [doc, setDoc] = useState<any>(value?.tiptap || sreSkillTemplate)
  const [markdown, setMarkdown] = useState<string>(value?.markdown || '')
  const [initialContent, setInitialContent] = useState<any>(value?.tiptap || sreSkillTemplate)
  const [editorHeight, setEditorHeight] = useState(500)

  React.useEffect(() => {
    setIsMounted(true)

    // 计算编辑器高度：视口高度 - 顶部导航 - 表单字段 - 按钮 - 间距
    const calculateHeight = () => {
      const viewportHeight = window.innerHeight
      // 预留空间：顶部导航(~80px) + 表单字段(~120px) + 按钮(~60px) + 间距(~100px)
      const reservedSpace = 360
      const calculatedHeight = viewportHeight - reservedSpace
      setEditorHeight(Math.max(400, calculatedHeight)) // 最小400px
    }

    calculateHeight()
    window.addEventListener('resize', calculateHeight)
    return () => window.removeEventListener('resize', calculateHeight)
  }, [])

  // 当 value.tiptap 变化时更新初始内容
  React.useEffect(() => {
    if (value?.tiptap) {
      setInitialContent(value.tiptap)
      setDoc(value.tiptap)
    }
  }, [value?.tiptap])

  const handleSave = async () => {
    const json = doc || sreSkillTemplate
    const payload: KnowledgeEditorValue = {
      alertRuleName: rule,
      tags,
      tiptap: json,
      markdown,
    }

    await onSubmit?.(payload)
  }

  // 暴露 submit 方法给父组件
  React.useImperativeHandle(ref, () => ({
    submit: handleSave
  }))

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
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <Label htmlFor="rule">告警规则名</Label>
          <Input id="rule" value={rule} onChange={(e) => setRule(e.target.value)} placeholder="如 KubePodCrashLooping" />
        </div>
        <div>
          <Label htmlFor="tags">标签</Label>
          <Select
            value=""
            onValueChange={(value) => {
              if (!tags.includes(value)) {
                setTags([...tags, value])
              }
            }}
          >
            <SelectTrigger>
              <SelectValue placeholder="选择标签" />
            </SelectTrigger>
            <SelectContent>
              {PREDEFINED_TAGS.map((tag) => (
                <SelectItem key={tag} value={tag}>
                  {tag}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <div className="flex flex-wrap gap-2 mt-2">
            {tags.map((tag) => (
              <div key={tag} className="bg-secondary text-secondary-foreground px-2 py-1 rounded-md text-sm flex items-center gap-1">
                {tag}
                <button
                  onClick={() => setTags(tags.filter((t) => t !== tag))}
                  className="hover:text-destructive"
                >
                  ×
                </button>
              </div>
            ))}
          </div>
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
          embedHeight={editorHeight}
          initialContent={initialContent}
          onUpdate={(json, html, md) => {
            setDoc(json)
            setMarkdown(md)
            onChange?.({
              alertRuleName: rule,
              tags,
              tiptap: json,
              markdown: md,
            })
          }}
        />
      </div>


    </div>
  )
})

KnowledgeEditor.displayName = 'KnowledgeEditor'

export default KnowledgeEditor
