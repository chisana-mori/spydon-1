'use client'

import { usePathname } from 'next/navigation'
import { stripAppBasePath } from '@/config'
import { AppShell } from './app-shell'

interface ConditionalShellProps {
  children: React.ReactNode
}

/**
 * 条件渲染的布局包装器
 * 对于特殊页面（如未授权页面），不使用AppShell
 */
export function ConditionalShell({ children }: ConditionalShellProps) {
  const pathname = usePathname()
  const normalizedPathname = stripAppBasePath(pathname || '/')

  // 不需要AppShell的页面列表
  const noShellPages = ['/unauthorized']

  // 如果是特殊页面，直接渲染children
  if (noShellPages.includes(normalizedPathname)) {
    return <>{children}</>
  }

  // 否则使用AppShell包装
  return <AppShell>{children}</AppShell>
}
