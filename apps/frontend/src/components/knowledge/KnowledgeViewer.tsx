"use client"

import React, { useState, useEffect } from 'react'
import type { KnowledgeManifest } from '@/types/api'
import { SimpleEditor } from '@/components/tiptap-templates/simple/simple-editor'

interface Props {
  manifest?: KnowledgeManifest | null
}

export function KnowledgeViewer({ manifest }: Props) {
  const [isMounted, setIsMounted] = useState(false)

  useEffect(() => {
    setIsMounted(true)
  }, [])

  if (!manifest) {
    return <div className="text-sm text-muted-foreground">暂无知识库内容</div>
  }

  if (!isMounted) {
    return <div className="text-sm text-muted-foreground">加载中...</div>
  }

  if (!manifest.content?.tiptap && !manifest.content?.markdown) {
    return <div className="text-sm text-muted-foreground">暂无正文（等待编辑发布）</div>
  }

  return (
    <div className="knowledge-viewer-wrapper">
      <style jsx global>{`
        .knowledge-viewer-wrapper {
          width: 100%;
          display: flex;
          justify-content: center;
        }

        .knowledge-viewer-wrapper .simple-editor-wrapper {
          width: 100%;
          display: flex;
          justify-content: center;
          background: transparent;
          border: none;
          box-shadow: none;
        }

        .knowledge-viewer-wrapper .simple-editor-content {
          max-width: 820px;
          margin: 0 auto;
        }

        .knowledge-viewer-wrapper .simple-editor-readonly .simple-editor-content {
          padding: 0;
        }

        .knowledge-viewer-wrapper .simple-editor {
          background: transparent;
          padding: 0;
          min-height: 200px;
        }

        .knowledge-viewer-wrapper .tiptap.ProseMirror {
          text-align: left;
        }
      `}</style>
      <SimpleEditor
        variant="embed"
        embedHeight="auto"
        initialContent={manifest.content.tiptap}
        readOnly={true}
      />
    </div>
  )
}

export default KnowledgeViewer
