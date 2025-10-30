'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { resolveAppPath } from '@/config'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { ShieldAlert, Mail, LogOut, RefreshCw } from 'lucide-react'
import { useCurrentUser } from '@/hooks/useCurrentUser'
import RobustaAPI from '@/lib/api'
import { toast } from 'sonner'

export default function UnauthorizedPage() {
  const router = useRouter()
  const { user, loading, refresh } = useCurrentUser()

  // 如果用户已经是管理员，重定向到首页
  useEffect(() => {
    if (!loading && user?.is_admin) {
      router.push(resolveAppPath('/'))
    }
  }, [user, loading, router])

  const handleLogout = async () => {
    try {
      await RobustaAPI.logout()
      toast.success('已退出登录')
      // 重定向到CAS登出或首页
      window.location.href = resolveAppPath('/auth/cas/logout', { includeBasePath: true })
    } catch (error) {
      console.error('登出失败:', error)
      toast.error('登出失败，请重试')
    }
  }

  const handleRefresh = async () => {
    await refresh()
    if (user?.is_admin) {
      toast.success('权限已更新，正在跳转...')
      router.push(resolveAppPath('/'))
    } else {
      toast.info('权限未变更，请联系管理员')
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-900 dark:to-slate-800">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto mb-4"></div>
          <p className="text-muted-foreground">加载中...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-900 dark:to-slate-800 p-4">
      <Card className="w-full max-w-2xl shadow-2xl border-2">
        <CardHeader className="text-center pb-4">
          <div className="w-20 h-20 bg-amber-100 dark:bg-amber-900/30 rounded-full flex items-center justify-center mx-auto mb-4">
            <ShieldAlert className="w-10 h-10 text-amber-600 dark:text-amber-500" />
          </div>
          <CardTitle className="text-3xl font-bold">等待管理员授权</CardTitle>
          <CardDescription className="text-base mt-2">
            您的账号已成功登录，但暂时没有访问权限
          </CardDescription>
        </CardHeader>

        <CardContent className="space-y-6">
          {/* 用户信息 */}
          <Alert className="bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-800">
            <AlertDescription className="text-sm">
              <div className="space-y-1">
                <p><strong>用户名：</strong>{user?.username || user?.email}</p>
                <p><strong>邮箱：</strong>{user?.email}</p>
                <p><strong>姓名：</strong>{user?.name || '未设置'}</p>
                <p><strong>当前权限：</strong><span className="text-amber-600 dark:text-amber-500 font-semibold">普通用户（无访问权限）</span></p>
              </div>
            </AlertDescription>
          </Alert>

          {/* 说明信息 */}
          <div className="space-y-4">
            <div className="bg-slate-50 dark:bg-slate-800/50 rounded-lg p-4 space-y-3">
              <h3 className="font-semibold text-lg flex items-center gap-2">
                <span className="w-6 h-6 bg-primary text-primary-foreground rounded-full flex items-center justify-center text-sm">1</span>
                为什么我不能访问系统？
              </h3>
              <p className="text-sm text-muted-foreground ml-8">
                为了保护系统安全，新注册的用户默认没有访问权限。您需要联系系统管理员为您开通访问权限。
              </p>
            </div>

            <div className="bg-slate-50 dark:bg-slate-800/50 rounded-lg p-4 space-y-3">
              <h3 className="font-semibold text-lg flex items-center gap-2">
                <span className="w-6 h-6 bg-primary text-primary-foreground rounded-full flex items-center justify-center text-sm">2</span>
                如何获得访问权限？
              </h3>
              <p className="text-sm text-muted-foreground ml-8">
                请联系系统管理员，提供您的用户名或邮箱，管理员会在用户管理页面为您授予管理员权限。
              </p>
            </div>

            <div className="bg-slate-50 dark:bg-slate-800/50 rounded-lg p-4 space-y-3">
              <h3 className="font-semibold text-lg flex items-center gap-2">
                <span className="w-6 h-6 bg-primary text-primary-foreground rounded-full flex items-center justify-center text-sm">3</span>
                授权后如何访问？
              </h3>
              <p className="text-sm text-muted-foreground ml-8">
                管理员授权后，请点击下方的&quot;刷新权限&quot;按钮，或重新登录系统即可正常访问。
              </p>
            </div>
          </div>

          {/* 联系方式 */}
          <Alert>
            <Mail className="h-4 w-4" />
            <AlertDescription>
              <p className="font-semibold mb-1">需要帮助？</p>
              <p className="text-sm text-muted-foreground">
                请联系系统管理员，说明您需要访问权限。管理员可以在&quot;用户管理&quot;页面为您授权。
              </p>
            </AlertDescription>
          </Alert>

          {/* 操作按钮 */}
          <div className="flex flex-col sm:flex-row gap-3 pt-4">
            <Button 
              onClick={handleRefresh} 
              className="flex-1"
              variant="default"
            >
              <RefreshCw className="w-4 h-4 mr-2" />
              刷新权限状态
            </Button>
            <Button 
              onClick={handleLogout} 
              variant="outline"
              className="flex-1"
            >
              <LogOut className="w-4 h-4 mr-2" />
              退出登录
            </Button>
          </div>

          {/* 提示信息 */}
          <p className="text-xs text-center text-muted-foreground pt-2">
            如果您已获得授权，请点击&quot;刷新权限状态&quot;按钮更新您的访问权限
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
