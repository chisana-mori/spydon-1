'use client'

import { useEffect, useState } from 'react'
import { attachRuntimeConfig, FrontendConfig } from '@/config'

/**
 * 运行时配置加载器
 * 在客户端加载 __runtime_config__.json 并注入到 window 对象
 * 这个组件应该放在 layout.tsx 中尽早执行
 */
export function RuntimeConfigLoader({ children }: { children: React.ReactNode }) {
    const [loaded, setLoaded] = useState(false)
    const [error, setError] = useState<string | null>(null)

    useEffect(() => {
        // 只在客户端执行
        if (typeof window === 'undefined') {
            setLoaded(true)
            return
        }

        // 如果已经加载过配置，跳过
        if (window.__ROBUSTA_RUNTIME_CONFIG__) {
            console.log('[RuntimeConfig] Already loaded, skipping fetch')
            setLoaded(true)
            return
        }

        const loadConfig = async () => {
            try {
                // 尝试加载运行时配置文件
                const response = await fetch('/__runtime_config__.json', {
                    cache: 'no-store', // 禁用缓存以获取最新配置
                })

                if (!response.ok) {
                    // 配置文件不存在，使用默认配置
                    console.log('[RuntimeConfig] No runtime config found, using defaults')
                    setLoaded(true)
                    return
                }

                const config: Partial<FrontendConfig> = await response.json()
                console.log('[RuntimeConfig] Loaded runtime config:', config)

                // 注入到 window 对象和 appConfig
                attachRuntimeConfig(config)

                setLoaded(true)
            } catch (err) {
                console.error('[RuntimeConfig] Failed to load config:', err)
                setError(err instanceof Error ? err.message : 'Unknown error')
                setLoaded(true) // 即使失败也继续渲染，使用默认配置
            }
        }

        loadConfig()
    }, [])

    // 在配置加载完成前显示加载状态（可选）
    if (!loaded) {
        return null // 或者显示一个最小化的加载指示器
    }

    if (error) {
        console.warn('[RuntimeConfig] Using default config due to error:', error)
    }

    return <>{children}</>
}
