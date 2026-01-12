'use client'

import { useState, useEffect, type ReactNode } from 'react'
import { useThemeStore } from '@/stores/themeStore'
import { appConfig, resolveAppPath, stripAppBasePath } from '@/config'
import Link from 'next/link'
import { usePathname, useRouter } from 'next/navigation'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { SystemSettingsDialog } from '@/components/settings/SystemSettingsDialog'
import type { LucideIcon } from 'lucide-react'
import {
  LayoutDashboard,
  AlertTriangle,
  Server,
  Bell,
  Menu,
  X,
  Shield,
  Activity,
  Users,
  BookOpen,
  Moon,
  Sun,
  Monitor,
  Key,
  ChevronDown,
  ChevronRight,
  GitBranch,
  Library,
  Tags,
  AlertCircle,
  Wrench,
  Settings2,
  Database,
  Share2,
} from 'lucide-react'
import { useCurrentUser } from '@/hooks/useCurrentUser'
import { toast } from 'sonner'

const resolveBackendUrl = (path: string) => {
  // 如果是完整的 URL，直接返回
  if (path.startsWith('http://') || path.startsWith('https://')) {
    return path
  }

  // 否则拼接 backendBaseUrl + path
  const backendBase = appConfig.backendBaseUrl.replace(/\/+$/, '')
  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  return `${backendBase}${normalizedPath}`
}

interface AppShellProps {
  children: ReactNode
}

type NavigationItem = {
  name: string
  href?: string
  icon: LucideIcon
  description: string
  children?: NavigationItem[]
}

const navigation: NavigationItem[] = [
  {
    name: '仪表盘',
    href: '/',
    icon: LayoutDashboard,
    description: '系统概览',
  },
  {
    name: '经验指南',
    href: '/knowledge',
    icon: BookOpen,
    description: '经验沉淀与编辑',
  },
  {
    name: '告警管理',
    href: '/alerts',
    icon: AlertTriangle,
    description: '告警监控',
  },
  {
    name: '运维管理',
    icon: Wrench,
    description: '日常运维操作与通知',
    children: [
      {
        name: '变更管理',
        href: '/tasks',
        icon: GitBranch,
        description: '任务与变更执行',
      },
      {
        name: '通知管理',
        href: '/operations/notification',
        icon: Bell,
        description: '邮件发送与配置',
      },
    ],
  },
  {
    name: '集群管理',
    icon: Server,
    description: '集群状态管理',
    children: [
      {
        name: '集群列表',
        href: '/clusters',
        icon: Server,
        description: '集群基础信息',
      },
      {
        name: 'Calico 网络',
        href: '/calico',
        icon: Share2,
        description: 'Calico 网络监控',
      },
    ],
  },
  {
    name: '设备管理',
    href: '/devices',
    icon: Monitor,
    description: '硬件资产管理',
  },
  {
    name: '资源管理',
    icon: Database,
    description: 'F5、存储等资源管理',
    children: [
      {
        name: 'F5管理',
        href: '/resources/f5',
        icon: Server,
        description: 'F5负载均衡器管理',
      },
    ],
  },
  {
    name: '配置管理',
    icon: Library,
    description: '标签、污点与设备应用配置',
    children: [
      {
        name: '标签管理',
        href: '/configuration/labels',
        icon: Tags,
        description: '标签特性管理',
      },
      {
        name: '污点管理',
        href: '/configuration/taints',
        icon: AlertCircle,
        description: '污点特性管理',
      },
      {
        name: '设备应用管理',
        href: '/configuration/device-apps',
        icon: Settings2,
        description: '设备应用类型管理',
      },
    ],
  },
  {
    name: '系统管理',
    icon: Shield,
    description: '系统配置与权限',
    children: [
      {
        name: '字典管理',
        href: '/dictionaries',
        icon: Library,
        description: '系统字典配置',
      },
      {
        name: '用户管理',
        href: '/users',
        icon: Users,
        description: '用户权限管理',
      },
      {
        name: 'API Key管理',
        href: '/apikeys',
        icon: Key,
        description: 'API密钥管理',
      },
    ],
  },
]

export function AppShell({ children }: AppShellProps) {
  const { theme, setTheme } = useThemeStore()
  const pathname = usePathname() || '/'
  const normalizedPathname = stripAppBasePath(pathname)
  const router = useRouter()
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [expandedMenus, setExpandedMenus] = useState<string[]>([])
  const { user, loading: authLoading } = useCurrentUser()

  const userInitial = (user?.name || user?.email || 'U').charAt(0).toUpperCase()

  // 初始化展开状态：如果当前路径匹配某个子菜单，自动展开父菜单
  useEffect(() => {
    navigation.forEach((item) => {
      if (item.children) {
        const hasActiveChild = item.children.some((child) =>
          child.href && (child.href === '/' ? normalizedPathname === '/' : normalizedPathname.startsWith(child.href))
        )
        if (hasActiveChild && !expandedMenus.includes(item.name)) {
          setExpandedMenus((prev) => [...prev, item.name])
        }
      }
    })
  }, [normalizedPathname, expandedMenus])

  const toggleMenu = (menuName: string) => {
    setExpandedMenus((prev) =>
      prev.includes(menuName) ? prev.filter((name) => name !== menuName) : [...prev, menuName]
    )
  }

  // 权限检查：非管理员用户重定向到未授权页面
  useEffect(() => {
    // 跳过加载状态和未授权页面本身
    if (authLoading || normalizedPathname === '/unauthorized') return

    // 如果用户已登录但不是管理员，重定向到未授权页面
    if (user && !user.is_admin) {
      router.push(resolveAppPath('/unauthorized'))
    }
  }, [user, authLoading, normalizedPathname, router])

  useEffect(() => {
    if (typeof window === 'undefined') return
    const handleAuthRequired = () => {
      toast.error('登录状态已失效，正在跳转到登录页面...')
      // 自动跳转到CAS登录页面
      setTimeout(() => {
        const serviceTarget = encodeURIComponent(window.location.href)
        const casLoginUrl = resolveBackendUrl(appConfig.casLoginPath || '/auth/cas/login')
        const separator = casLoginUrl.includes('?') ? '&' : '?'
        window.location.href = `${casLoginUrl}${separator}service=${serviceTarget}`
      }, 1000) // 延迟1秒让用户看到提示
    }
    window.addEventListener('robusta-auth-required', handleAuthRequired)
    return () => {
      window.removeEventListener('robusta-auth-required', handleAuthRequired)
    }
  }, [])

  const resolveCasLoginUrl = () => {
    return resolveBackendUrl(appConfig.casLoginPath || '/auth/cas/login')
  }

  const handleCASLogin = () => {
    if (typeof window === 'undefined') return

    const serviceTarget = encodeURIComponent(window.location.href)
    const casLoginUrl = resolveCasLoginUrl()
    const separator = casLoginUrl.includes('?') ? '&' : '?'
    window.location.href = `${casLoginUrl}${separator}service=${serviceTarget}`
  }

  const handleCASLogout = () => {
    if (typeof window === 'undefined') return
    const redirectTarget = encodeURIComponent(window.location.origin)
    const logoutUrl = resolveBackendUrl(appConfig.casLogoutPath || '/auth/cas/logout')
    const separator = logoutUrl.includes('?') ? '&' : '?'
    window.location.href = `${logoutUrl}${separator}redirect=${redirectTarget}`
  }

  const toggleTheme = () => {
    const themes: Array<'light' | 'dark' | 'system'> = ['light', 'dark', 'system']
    const currentIndex = themes.indexOf(theme)
    const nextTheme = themes[(currentIndex + 1) % themes.length]
    setTheme(nextTheme)
  }

  const getThemeIcon = () => {
    switch (theme) {
      case 'light':
        return <Sun className="h-4 w-4" />
      case 'dark':
        return <Moon className="h-4 w-4" />
      case 'system':
        return <Monitor className="h-4 w-4" />
      default:
        return <Monitor className="h-4 w-4" />
    }
  }

  const activeNav = navigation.find((item) => {
    if (item.href) {
      return item.href === '/' ? normalizedPathname === '/' : normalizedPathname.startsWith(item.href)
    }
    if (item.children) {
      return item.children.some((child) =>
        child.href && (child.href === '/' ? normalizedPathname === '/' : normalizedPathname.startsWith(child.href))
      )
    }
    return false
  })

  const renderedNavigation = navigation.map((item) => {
    const Icon = item.icon
    const isExpanded = expandedMenus.includes(item.name)

    // 如果有子菜单
    if (item.children) {
      const hasActiveChild = item.children.some((child) =>
        child.href && (child.href === '/' ? normalizedPathname === '/' : normalizedPathname.startsWith(child.href))
      )

      return (
        <div key={item.name} className="space-y-1">
          <button
            onClick={() => toggleMenu(item.name)}
            className={cn(
              'group flex items-center justify-between w-full px-3 py-2.5 rounded-lg text-base font-medium transition-all duration-200',
              hasActiveChild
                ? 'bg-primary/5 text-primary'
                : 'text-muted-foreground hover:bg-accent/50 hover:text-foreground',
            )}
          >
            <div className="flex items-center space-x-3">
              <Icon
                className={cn(
                  'h-5 w-5 transition-transform duration-200 flex-shrink-0',
                  hasActiveChild ? 'text-primary' : 'text-muted-foreground group-hover:text-foreground group-hover:scale-110',
                )}
              />
              <span className={cn("font-medium", hasActiveChild ? 'text-primary' : 'text-foreground')}>
                {item.name}
              </span>
            </div>
            {isExpanded ? (
              <ChevronDown className={cn('h-4 w-4', hasActiveChild ? 'text-primary' : 'text-muted-foreground group-hover:text-foreground')} />
            ) : (
              <ChevronRight className={cn('h-4 w-4', hasActiveChild ? 'text-primary' : 'text-muted-foreground group-hover:text-foreground')} />
            )}
          </button>

          <div
            className={cn(
              'overflow-hidden transition-all duration-300 ease-in-out',
              isExpanded ? 'max-h-96 opacity-100' : 'max-h-0 opacity-0',
            )}
          >
            <div className="mt-1 space-y-1">
              {item.children.map((child) => {
                const isActive = child.href && (child.href === '/' ? normalizedPathname === '/' : normalizedPathname.startsWith(child.href))
                const ChildIcon = child.icon
                const childHref = child.href ? resolveAppPath(child.href) : '#'

                return (
                  <Link
                    key={child.name}
                    href={childHref}
                    className={cn(
                      'group flex items-center space-x-3 px-3 py-2 rounded-lg text-sm transition-all duration-200 relative overflow-hidden pl-11',
                      isActive
                        ? 'bg-primary/10 text-primary font-medium'
                        : 'text-muted-foreground hover:bg-accent/30 hover:text-foreground font-normal',
                    )}
                    onClick={() => setSidebarOpen(false)}
                  >
                    {!isActive && (
                      <ChildIcon
                        className={cn(
                          'absolute left-4 h-4 w-4 transition-all duration-200 opacity-50 group-hover:opacity-100',
                        )}
                      />
                    )}
                    {isActive && (
                      <div className="absolute left-4 w-4 flex justify-center">
                        <div className="w-1.5 h-1.5 rounded-full bg-primary" />
                      </div>
                    )}
                    <span className="truncate">
                      {child.name}
                    </span>
                  </Link>
                )
              })}
            </div>
          </div>
        </div>
      )
    }

    // 没有子菜单的普通菜单项
    const isActive = item.href && (item.href === '/' ? normalizedPathname === '/' : normalizedPathname.startsWith(item.href))
    const itemHref = item.href ? resolveAppPath(item.href) : '#'

    return (
      <Link
        key={item.name}
        href={itemHref}
        className={cn(
          'group flex items-center space-x-3 px-3 py-2.5 rounded-lg text-base font-medium transition-all duration-200 relative',
          isActive
            ? 'bg-primary text-primary-foreground shadow-sm'
            : 'text-muted-foreground hover:bg-accent hover:text-foreground hover:shadow-sm',
        )}
        onClick={() => setSidebarOpen(false)}
      >
        <Icon
          className={cn(
            'h-5 w-5 transition-transform duration-200 flex-shrink-0',
            isActive ? 'text-white' : 'text-muted-foreground group-hover:text-foreground group-hover:scale-110',
          )}
        />
        <span className={cn('font-medium', isActive ? 'text-white' : 'text-foreground')}>
          {item.name}
        </span>
      </Link>
    )
  })

  return (
    <div className="flex h-screen bg-background">
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      <aside
        className={cn(
          'fixed inset-y-0 left-0 z-50 w-56 bg-card/95 backdrop-blur-sm border-r border-border/50 transform transition-all duration-300 ease-in-out lg:translate-x-0 shadow-xl lg:shadow-none',
          sidebarOpen ? 'translate-x-0' : '-translate-x-full',
        )}
      >
        <div className="flex h-full flex-col">
          <div className="flex h-16 items-center justify-between px-6 border-b border-border/50">
            <div className="flex items-center space-x-3">
              <div className="w-8 h-8 bg-primary/10 rounded-lg flex items-center justify-center">
                <Shield className="w-4 h-4 text-primary" />
              </div>
              <div>
                <span className="text-lg font-bold text-foreground">
                  Spydon
                </span>
              </div>
            </div>
            <Button
              variant="ghost"
              size="icon"
              className="lg:hidden hover:bg-accent"
              onClick={() => setSidebarOpen(false)}
            >
              <X className="h-5 w-5" />
            </Button>
          </div>

          <nav className="flex-1 px-4 py-4 space-y-1">
            {renderedNavigation}
          </nav>

          <div className="p-4 border-t border-border/50 space-y-3">
            {authLoading ? (
              <div className="px-4 py-3 bg-muted/30 rounded-lg">
                <div className="flex items-center gap-3">
                  <div className="w-8 h-8 bg-muted rounded-lg flex items-center justify-center animate-pulse">
                    <Activity className="w-4 h-4 text-muted-foreground" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-foreground">加载中...</p>
                    <p className="text-xs text-muted-foreground">正在同步账户</p>
                  </div>
                </div>
              </div>
            ) : user ? (
              <div className="space-y-2">
                <div className="px-4 py-3 bg-muted/30 rounded-lg">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 bg-primary rounded-full flex items-center justify-center flex-shrink-0">
                      <span className="text-sm font-medium text-primary-foreground">{userInitial}</span>
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-foreground truncate">
                        {user.name || user.username || user.email}
                      </p>
                      <p className="text-xs text-muted-foreground truncate">{user.email}</p>
                    </div>
                  </div>
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleCASLogout}
                  className="w-full"
                >
                  退出登录
                </Button>
              </div>
            ) : (
              <Button size="sm" onClick={handleCASLogin} className="w-full">
                CAS 登录
              </Button>
            )}
          </div>
        </div>
      </aside>

      {/* Main content */}
      <div className={cn(
        "flex-1 flex flex-col overflow-hidden transition-all duration-300",
        sidebarOpen ? "md:ml-0" : "md:ml-56"
      )}>
        {/* Header */}
        <header className="h-16 border-b border-border bg-card/50 backdrop-blur-sm">
          <div className="flex h-full items-center justify-between px-4 md:px-6">
            <div className="flex items-center gap-4">
              <Button
                variant="ghost"
                size="icon"
                className="lg:hidden hover:bg-accent"
                onClick={() => setSidebarOpen(true)}
              >
                <Menu className="h-5 w-5" />
              </Button>
              <div className="hidden md:flex items-center space-x-2 text-sm">
                <span className="text-muted-foreground">Spydon</span>
                <span className="text-muted-foreground">/</span>
                <span className="font-medium text-foreground">{activeNav?.name ?? '仪表盘'}</span>
              </div>
            </div>

            <div className="flex items-center space-x-2">
              {/* Theme toggle */}
              <Button
                variant="ghost"
                size="icon"
                onClick={toggleTheme}
                className="h-9 w-9"
              >
                {getThemeIcon()}
              </Button>

              {/* System Settings */}
              <SystemSettingsDialog />

              <Button variant="ghost" size="icon" className="relative">
                <Bell className="h-5 w-5" />
                <span className="absolute -top-1 -right-1 h-3 w-3 bg-critical rounded-full text-xs flex items-center justify-center text-white">
                  3
                </span>
              </Button>
            </div>
          </div>
        </header>

        {/* Page content */}
        <main className="flex-1 overflow-auto p-4 md:p-6">
          {children}
        </main>
      </div>
    </div>
  )
}
