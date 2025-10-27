import type { Metadata } from "next"
import "../globals.css";


export const metadata: Metadata = {
  title: "Editor",
}

export default function EditorLayout({
  children,
}: {
  children: React.ReactNode
}) {
  // 完全绕过主布局的 ConditionalShell
  return (
    <html lang="zh-CN" suppressHydrationWarning>
      <head />
      <body style={{ margin: 0, padding: 0, overflow: 'hidden' }}>
        {children}
      </body>
    </html>
  )
}
