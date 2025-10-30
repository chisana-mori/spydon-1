'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { resolveAppPath } from '@/config'
import { useCurrentUser } from '@/hooks/useCurrentUser'

interface RequireAdminProps {
  children: React.ReactNode
  fallback?: React.ReactNode
}

/**
 * 权限保护组件 - 要求用户必须是管理员
 * 如果用户不是管理员，将重定向到未授权页面
 */
export function RequireAdmin({ children, fallback }: RequireAdminProps) {
  const router = useRouter()
  const { user, loading } = useCurrentUser()

  useEffect(() => {
    // 等待用户信息加载完成
    if (loading) return

    // 如果没有用户信息或用户不是管理员，重定向到未授权页面
    if (!user || !user.is_admin) {
      router.push(resolveAppPath('/unauthorized'))
    }
  }, [user, loading, router])

  // 加载中显示fallback或默认加载界面
  if (loading) {
    return (
      fallback || (
        <div className="min-h-screen flex items-center justify-center">
          <div className="text-center">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto mb-4"></div>
            <p className="text-muted-foreground">验证权限中...</p>
          </div>
        </div>
      )
    )
  }

  // 如果不是管理员，显示空白（因为会被重定向）
  if (!user || !user.is_admin) {
    return null
  }

  // 是管理员，显示子组件
  return <>{children}</>
}

/**
 * 页面级权限保护的Hook
 * 用于在页面组件中检查权限
 */
export function useRequireAdmin() {
  const router = useRouter()
  const { user, loading } = useCurrentUser()

  useEffect(() => {
    if (loading) return

    if (!user || !user.is_admin) {
      router.push(resolveAppPath('/unauthorized'))
    }
  }, [user, loading, router])

  return {
    user,
    loading,
    isAdmin: user?.is_admin || false,
  }
}
