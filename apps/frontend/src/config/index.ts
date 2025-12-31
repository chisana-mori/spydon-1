export type FrontendConfig = {
  backendBaseUrl: string // 后端根 URL，如 http://localhost:8080 或 http://your-domain/spydon
  apiBaseUrl: string // 自动推导为 backendBaseUrl + /api/v1
  casLoginPath: string
  casLogoutPath: string
  basePath: string
  kiteBaseUrl: string // Kite Dashboard URL，如 http://localhost:18080
}

declare global {
  interface Window {
    __ROBUSTA_RUNTIME_CONFIG__?: Partial<FrontendConfig>
  }
}

// 开发环境下的后端地址 (用于导航跳转)
const defaultBackendHost = process.env.NEXT_PUBLIC_BACKEND_HOST || 'http://localhost:8080'

const defaultConfig: FrontendConfig = {
  backendBaseUrl: '', // API 请求使用相对路径，通过 Next.js rewrites 代理
  apiBaseUrl: '/api/v1', // 相对路径，Next.js 会代理到后端
  casLoginPath: `${defaultBackendHost}/auth/cas/login`,  // CAS 登录需要直接跳转到后端
  casLogoutPath: `${defaultBackendHost}/auth/cas/logout`,
  basePath: '',
  kiteBaseUrl: process.env.NEXT_PUBLIC_KITE_BASE_URL || 'http://localhost:18080',
}

// 从环境变量读取配置
const backendBaseUrlFromEnv = process.env.NEXT_PUBLIC_BACKEND_BASE_URL
const apiBaseUrlFromEnv = process.env.NEXT_PUBLIC_API_BASE_URL || process.env.ROBUSTA_API_BASE_URL

// 构建配置：优先使用显式配置，否则自动推导
const resolvedBackendBaseUrl = backendBaseUrlFromEnv || defaultBackendHost
const resolvedApiBaseUrl = apiBaseUrlFromEnv || `${resolvedBackendBaseUrl.replace(/\/+$/, '')}/api/v1`

const envConfig: Partial<FrontendConfig> = {
  backendBaseUrl: backendBaseUrlFromEnv || undefined,
  apiBaseUrl: apiBaseUrlFromEnv || undefined,
  casLoginPath: process.env.NEXT_PUBLIC_CAS_LOGIN_PATH || undefined,
  casLogoutPath: process.env.NEXT_PUBLIC_CAS_LOGOUT_PATH || undefined,
  basePath: process.env.NEXT_PUBLIC_BASE_PATH || undefined,
  kiteBaseUrl: process.env.NEXT_PUBLIC_KITE_BASE_URL || undefined,
}

const runtimeConfig: Partial<FrontendConfig> =
  (typeof window !== 'undefined' && window.__ROBUSTA_RUNTIME_CONFIG__) || {}

// 合并配置
const mergedConfig = {
  ...defaultConfig,
  ...runtimeConfig,
  ...filterUndefined(envConfig),
}

// 确保 backendBaseUrl 和 apiBaseUrl 保持一致
// 如果只配置了其中一个，自动推导另一个
if (backendBaseUrlFromEnv && !apiBaseUrlFromEnv) {
  mergedConfig.apiBaseUrl = `${backendBaseUrlFromEnv.replace(/\/+$/, '')}/api/v1`
} else if (apiBaseUrlFromEnv && !backendBaseUrlFromEnv) {
  const extracted = apiBaseUrlFromEnv.replace(/\/api\/v1\/?$/, '')
  if (extracted !== apiBaseUrlFromEnv) {
    mergedConfig.backendBaseUrl = extracted
  }
}

export const appConfig: FrontendConfig = mergedConfig

function normalizeBasePath(value: string | undefined): string {
  if (!value) return ''
  const trimmed = value.trim()
  if (!trimmed || trimmed === '/') return ''
  const withLeadingSlash = trimmed.startsWith('/') ? trimmed : `/${trimmed}`
  return withLeadingSlash.replace(/\/+$/, '')
}

appConfig.basePath = normalizeBasePath(appConfig.basePath)

export function resolveConfig(overrides?: Partial<FrontendConfig>): FrontendConfig {
  if (!overrides) {
    return appConfig
  }
  return {
    ...appConfig,
    ...filterUndefined(overrides),
  }
}

function filterUndefined<T extends Record<string, any>>(input: Partial<T>): Partial<T> {
  const output: Partial<T> = {}
  Object.entries(input).forEach(([key, value]) => {
    if (value !== undefined) {
      ; (output as any)[key] = value
    }
  })
  return output
}

export function attachRuntimeConfig(config: Partial<FrontendConfig>) {
  if (typeof window === 'undefined') return

  // 更新 window 对象中的运行时配置
  window.__ROBUSTA_RUNTIME_CONFIG__ = {
    ...(window.__ROBUSTA_RUNTIME_CONFIG__ || {}),
    ...filterUndefined(config),
  }

  // 同时更新 appConfig 对象（运行时动态注入）
  const filteredConfig = filterUndefined(config)
  Object.keys(filteredConfig).forEach((key) => {
    const typedKey = key as keyof FrontendConfig
    if (filteredConfig[typedKey] !== undefined) {
      ; (appConfig as any)[typedKey] = filteredConfig[typedKey]
    }
  })

  // 如果更新了 backendBaseUrl 但没有更新 apiBaseUrl，自动推导
  if (config.backendBaseUrl && !config.apiBaseUrl) {
    const apiBaseUrl = `${config.backendBaseUrl.replace(/\/+$/, '')}/api/v1`
    appConfig.apiBaseUrl = apiBaseUrl
    window.__ROBUSTA_RUNTIME_CONFIG__.apiBaseUrl = apiBaseUrl
  }

  // 规范化 basePath
  if (config.basePath !== undefined) {
    appConfig.basePath = normalizeBasePath(config.basePath)
  }

  console.log('[Config] Runtime config updated:', appConfig)
}

const isAbsoluteUrl = (input: string) => /^https?:\/\//i.test(input)

const BASE_PATH_MANAGED_BY_NEXT = process.env.NEXT_MANAGED_BASE_PATH === 'true'

export function resolveAppPath(target: string, options?: { includeBasePath?: boolean }): any {
  const input = (target || '').trim()
  if (!input) {
    if (options?.includeBasePath || (!BASE_PATH_MANAGED_BY_NEXT && appConfig.basePath)) {
      return appConfig.basePath || '/'
    }
    return '/'
  }
  if (isAbsoluteUrl(input)) {
    return input
  }
  const normalized = input.startsWith('/') ? input : `/${input}`
  const shouldIncludeBase = options?.includeBasePath || (!BASE_PATH_MANAGED_BY_NEXT && !!appConfig.basePath)

  if (shouldIncludeBase) {
    if (!appConfig.basePath) {
      return normalized
    }
    if (normalized === appConfig.basePath || normalized.startsWith(`${appConfig.basePath}/`)) {
      return normalized
    }
    return `${appConfig.basePath}${normalized}`.replace(/\/+$/, '') || '/'
  }
  if (appConfig.basePath) {
    if (normalized === appConfig.basePath) {
      return '/'
    }
    if (normalized.startsWith(`${appConfig.basePath}/`)) {
      const stripped = normalized.slice(appConfig.basePath.length)
      return stripped ? (stripped.startsWith('/') ? stripped : `/${stripped}`) : '/'
    }
  }
  return normalized
}

export function stripAppBasePath(pathname: string): string {
  if (!pathname) return '/'
  if (!appConfig.basePath) return pathname
  if (!pathname.startsWith(appConfig.basePath)) {
    return pathname
  }
  const stripped = pathname.slice(appConfig.basePath.length)
  if (!stripped) return '/'
  return stripped.startsWith('/') ? stripped : `/${stripped}`
}

/**
 * 生成 Kite Dashboard 的链接
 * @param clusterName 集群名称（用于 Kite 的集群切换）
 * @param resource 资源类型，如 'pods', 'deployments', 'services' 等
 * @param namespace 命名空间（可选）
 * @param resourceName 资源名称（可选）
 */
export function getKiteUrl(options?: {
  clusterName?: string
  resource?: string
  namespace?: string
  resourceName?: string
}): string {
  const baseUrl = appConfig.kiteBaseUrl.replace(/\/+$/, '')

  if (!options) {
    return baseUrl
  }

  const { clusterName, resource, namespace, resourceName } = options

  // Kite 的 URL 结构：/{resource}?cluster={cluster}&namespace={namespace}
  // 或者直接跳转到首页让用户选择集群
  const params = new URLSearchParams()

  if (clusterName) {
    params.set('cluster', clusterName)
  }

  if (namespace) {
    params.set('namespace', namespace)
  }

  let path = ''
  if (resource) {
    path = `/${resource}`
    if (resourceName) {
      path += `/${resourceName}`
    }
  }

  const queryString = params.toString()
  return queryString ? `${baseUrl}${path}?${queryString}` : `${baseUrl}${path}`
}

/**
 * 检查 Kite 是否已配置
 */
export function isKiteEnabled(): boolean {
  return !!appConfig.kiteBaseUrl
}
