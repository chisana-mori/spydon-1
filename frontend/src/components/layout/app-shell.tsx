'use client'

import { useState, type ReactNode } from 'react'
import { useThemeStore } from '@/stores/themeStore'
import Link from 'next/link'
import type { Route } from 'next'
import { usePathname } from 'next/navigation'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import type { LucideIcon } from 'lucide-react'
import {
  LayoutDashboard,
  AlertTriangle,
  Server,
  Settings,
  Bell,
  Menu,
  X,
  BarChart3,
  Shield,
  Activity,
  Users,
  HelpCircle,
  Moon,
  Sun,
  Monitor,
} from 'lucide-react'

interface AppShellProps {
  children: ReactNode
}

type NavigationItem = {
  name: string
  href: Route
  icon: LucideIcon
  description: string
}

const navigation: NavigationItem[] = [
  {
    name: '仪表盘',
    href: '/',
    icon: LayoutDashboard,
    description: '系统概览',
  },
  {
    name: '告警管理',
    href: '/alerts',
    icon: AlertTriangle,
    description: '告警监控',
  },
  {
    name: '集群管理',
    href: '/clusters',
    icon: Server,
    description: '集群状态',
  },
  {
    name: '分析报告',
    href: '/reports',
    icon: BarChart3,
    description: '数据分析',
  },
  {
    name: '用户管理',
    href: '/users',
    icon: Users,
    description: '权限控制',
  },
]

export function AppShell({ children }: AppShellProps) {
  const { theme, setTheme } = useThemeStore()
  const pathname = usePathname()
  const [sidebarOpen, setSidebarOpen] = useState(false)

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

  const activeNav = navigation.find((item) =>
    item.href === '/' ? pathname === '/' : pathname.startsWith(item.href),
  )

  const renderedNavigation = navigation.map((item) => {
    const isActive = item.href === '/' ? pathname === '/' : pathname.startsWith(item.href)
    const Icon = item.icon

    return (
      <Link
        key={item.name}
        href={item.href}
        className={cn(
          'group flex items-center space-x-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-200 relative',
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
                  Robusta Hub
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
            <div className="px-4 py-3 bg-muted/30 rounded-lg">
              <div className="flex items-center gap-3">
                <div className="w-8 h-8 bg-muted rounded-lg flex items-center justify-center">
                  <Activity className="w-4 h-4 text-muted-foreground" />
                </div>
                <div>
                  <p className="text-sm font-medium text-foreground">系统状态</p>
                  <div className="flex items-center gap-1 text-green-600">
                    <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
                    <span className="text-xs">运行正常</span>
                  </div>
                </div>
              </div>
            </div>
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
            <div className="flex items-center space-x-4">
              <Button
                variant="ghost"
                size="icon"
                className="lg:hidden hover:bg-accent"
                onClick={() => setSidebarOpen(true)}
              >
                <Menu className="h-5 w-5" />
              </Button>
              <div className="hidden md:flex items-center space-x-2 text-sm">
                <span className="text-muted-foreground">Robusta Hub</span>
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

              <Button variant="ghost" size="icon" className="relative">
                <Bell className="h-5 w-5" />
                <span className="absolute -top-1 -right-1 h-3 w-3 bg-critical rounded-full text-xs flex items-center justify-center text-white">
                  3
                </span>
              </Button>

              <div className="flex items-center space-x-3">
                <div className="hidden md:block text-right">
                  <p className="text-sm font-medium text-foreground">管理员</p>
                  <p className="text-xs text-muted-foreground">admin@robusta.com</p>
                </div>
                <div className="h-8 w-8 bg-primary rounded-full flex items-center justify-center">
                  <span className="text-sm font-medium text-primary-foreground">A</span>
                </div>
              </div>
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
