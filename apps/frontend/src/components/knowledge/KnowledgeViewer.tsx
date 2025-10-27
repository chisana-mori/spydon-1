"use client"

import React, { useEffect, useState } from 'react'
import type { KnowledgeManifest } from '@/types/api'
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Underline from '@tiptap/extension-underline'
import Link from '@tiptap/extension-link'
import Image from '@tiptap/extension-image'
import TextAlign from '@tiptap/extension-text-align'
import Typography from '@tiptap/extension-typography'
import { common, createLowlight } from 'lowlight'
import CodeBlockLowlight from '@tiptap/extension-code-block-lowlight'

interface Props {
  manifest?: KnowledgeManifest | null
}

let lowlightInstance: any = null

function getLowlight() {
  if (!lowlightInstance) {
    lowlightInstance = createLowlight(common)
  }
  return lowlightInstance
}

export function KnowledgeViewer({ manifest }: Props) {
  const [isMounted, setIsMounted] = useState(false)

  useEffect(() => {
    setIsMounted(true)
  }, [])

  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        codeBlock: false,
      }),
      CodeBlockLowlight.configure({
        lowlight: getLowlight(),
        defaultLanguage: 'plaintext',
      }),
      Underline,
      Link.configure({
        openOnClick: true,
        HTMLAttributes: {
          class: 'text-primary hover:underline',
        },
      }),
      Image.configure({
        inline: false,
        allowBase64: false,
      }),
      TextAlign.configure({
        types: ['heading', 'paragraph'],
      }),
      Typography,
    ],
    content: manifest?.content?.tiptap || { type: 'doc', content: [{ type: 'paragraph' }] },
    editable: false,
    immediatelyRender: false,
    editorProps: {
      attributes: {
        class: 'prose prose-base dark:prose-invert max-w-none focus:outline-none',
      },
    },
  })

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
    <div className="knowledge-viewer">
      <style jsx global>{`
        .knowledge-viewer .ProseMirror {
          padding: 1rem 0;
        }
        
        .knowledge-viewer .ProseMirror h1 {
          font-size: 2em;
          font-weight: 700;
          margin-top: 1.5em;
          margin-bottom: 0.5em;
          line-height: 1.2;
        }
        
        .knowledge-viewer .ProseMirror h2 {
          font-size: 1.5em;
          font-weight: 600;
          margin-top: 1.5em;
          margin-bottom: 0.5em;
          line-height: 1.3;
        }
        
        .knowledge-viewer .ProseMirror h3 {
          font-size: 1.25em;
          font-weight: 600;
          margin-top: 1.5em;
          margin-bottom: 0.5em;
          line-height: 1.4;
        }
        
        .knowledge-viewer .ProseMirror p {
          margin-top: 0.75em;
          margin-bottom: 0.75em;
          line-height: 1.7;
        }
        
        .knowledge-viewer .ProseMirror ul,
        .knowledge-viewer .ProseMirror ol {
          padding-left: 1.5em;
          margin-top: 0.75em;
          margin-bottom: 0.75em;
        }
        
        .knowledge-viewer .ProseMirror li {
          margin-top: 0.25em;
          margin-bottom: 0.25em;
        }
        
        .knowledge-viewer .ProseMirror blockquote {
          border-left: 4px solid hsl(var(--primary));
          padding-left: 1em;
          margin-left: 0;
          margin-top: 1em;
          margin-bottom: 1em;
          font-style: italic;
          color: hsl(var(--muted-foreground));
        }
        
        .knowledge-viewer .ProseMirror pre {
          background: hsl(var(--muted));
          border-radius: 0.5rem;
          padding: 1rem;
          margin-top: 1em;
          margin-bottom: 1em;
          overflow-x: auto;
        }
        
        .knowledge-viewer .ProseMirror code {
          background: hsl(var(--muted));
          padding: 0.2em 0.4em;
          border-radius: 0.25rem;
          font-size: 0.9em;
          font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
        }
        
        .knowledge-viewer .ProseMirror pre code {
          background: transparent;
          padding: 0;
          font-size: 0.875em;
        }
        
        .knowledge-viewer .ProseMirror img {
          max-width: 100%;
          height: auto;
          border-radius: 0.5rem;
          margin-top: 1em;
          margin-bottom: 1em;
        }
        
        .knowledge-viewer .ProseMirror hr {
          border: none;
          border-top: 2px solid hsl(var(--border));
          margin-top: 2em;
          margin-bottom: 2em;
        }
        
        .knowledge-viewer .ProseMirror a {
          color: hsl(var(--primary));
          text-decoration: underline;
          text-decoration-color: hsl(var(--primary) / 0.3);
          transition: text-decoration-color 0.2s;
        }
        
        .knowledge-viewer .ProseMirror a:hover {
          text-decoration-color: hsl(var(--primary));
        }
      `}</style>
      <EditorContent editor={editor} />
    </div>
  )
}

export default KnowledgeViewer


