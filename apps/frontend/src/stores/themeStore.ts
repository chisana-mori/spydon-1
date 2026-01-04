import { create } from 'zustand'

export type Theme = 'light' | 'dark' | 'system'

interface ThemeStore {
  theme: Theme
  setTheme: (theme: Theme) => void
}

export const useThemeStore = create<ThemeStore>((set) => ({
  theme: 'system',
  setTheme: (theme) => {
    set({ theme })
    localStorage.setItem('theme', theme)

    if (theme === 'system') {
      const systemTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
      document.documentElement.classList.toggle('dark', systemTheme === 'dark')
    } else {
      document.documentElement.classList.toggle('dark', theme === 'dark')
    }
  },
}))

// Initialize theme from localStorage on app start
if (typeof window !== 'undefined') {
  const storedTheme = localStorage.getItem('theme') as Theme || 'system'

  if (storedTheme === 'system') {
    const systemTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
    document.documentElement.classList.toggle('dark', systemTheme === 'dark')
  } else {
    document.documentElement.classList.toggle('dark', storedTheme === 'dark')
  }

  // Update store with stored theme
  useThemeStore.getState().setTheme(storedTheme)
}
