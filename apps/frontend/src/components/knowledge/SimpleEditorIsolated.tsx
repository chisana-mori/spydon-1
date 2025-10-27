"use client"

import React, { useEffect, useRef } from 'react'
import { createPortal } from 'react-dom'

interface SimpleEditorIsolatedProps {
  height?: string
}

export function SimpleEditorIsolated({ height = '500px' }: SimpleEditorIsolatedProps) {
  const iframeRef = useRef<HTMLIFrameElement>(null)
  const [iframeDocument, setIframeDocument] = React.useState<Document | null>(null)

  useEffect(() => {
    const iframe = iframeRef.current
    if (!iframe) return

    const iframeDoc = iframe.contentDocument || iframe.contentWindow?.document
    if (!iframeDoc) return

    // 写入基础 HTML 结构
    iframeDoc.open()
    iframeDoc.write(`
      <!DOCTYPE html>
      <html>
        <head>
          <meta charset="utf-8">
          <meta name="viewport" content="width=device-width, initial-scale=1">
          <title>TipTap Editor</title>
          <style>
            * {
              margin: 0;
              padding: 0;
              box-sizing: border-box;
            }
            html, body {
              width: 100%;
              height: 100%;
              overflow: hidden;
            }
            #root {
              width: 100%;
              height: 100%;
            }
          </style>
        </head>
        <body>
          <div id="root"></div>
        </body>
      </html>
    `)
    iframeDoc.close()

    // 复制所有样式表到 iframe
    const copyStyles = () => {
      const parentStyles = document.querySelectorAll('style, link[rel="stylesheet"]')
      const iframeHead = iframeDoc.head

      parentStyles.forEach((style) => {
        if (style instanceof HTMLStyleElement) {
          const newStyle = iframeDoc.createElement('style')
          newStyle.textContent = style.textContent
          iframeHead.appendChild(newStyle)
        } else if (style instanceof HTMLLinkElement) {
          const newLink = iframeDoc.createElement('link')
          newLink.rel = 'stylesheet'
          newLink.href = style.href
          iframeHead.appendChild(newLink)
        }
      })
    }

    // 等待 iframe 加载完成
    iframe.onload = () => {
      copyStyles()
      setIframeDocument(iframeDoc)
    }

    // 如果已经加载，直接复制样式
    if (iframeDoc.readyState === 'complete') {
      copyStyles()
      setIframeDocument(iframeDoc)
    }
  }, [])

  return (
    <iframe
      ref={iframeRef}
      style={{
        width: '100%',
        height,
        border: '1px solid var(--border)',
        borderRadius: '0.5rem',
        backgroundColor: 'white',
      }}
      title="TipTap Editor"
    />
  )
}
