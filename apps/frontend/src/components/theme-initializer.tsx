'use client'

import { useEffect } from 'react'

interface ThemeInitializerProps {
  children?: React.ReactNode
}

export function ThemeInitializer({ children }: ThemeInitializerProps) {
  useEffect(() => {
    const storedTheme = localStorage.getItem('theme') || 'system'

    if (storedTheme === 'system') {
      const systemTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
      document.documentElement.classList.toggle('dark', systemTheme === 'dark')
    } else {
      document.documentElement.classList.toggle('dark', storedTheme === 'dark')
    }
  }, [])

  return <>{children}</>
}
