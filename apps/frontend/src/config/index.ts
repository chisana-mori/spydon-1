export type FrontendConfig = {
  backendBaseUrl: string // 后端根 URL，如 http://localhost:8080 或 http://your-domain/spydon
  apiBaseUrl: string // 自动推导为 backendBaseUrl + /api/v1
  casLoginPath: string
  casLogoutPath: string
  basePath: string
}

declare global {
  interface Window {
    __ROBUSTA_RUNTIME_CONFIG__?: Partial<FrontendConfig>
  }
}

const defaultBackendBaseUrl = 'http://localhost:8080'

const defaultConfig: FrontendConfig = {
  backendBaseUrl: defaultBackendBaseUrl,
  apiBaseUrl: `${defaultBackendBaseUrl}/api/v1`,
  casLoginPath: '/auth/cas/login',
  casLogoutPath: '/auth/cas/logout',
  basePath: '',
}

// 从环境变量读取配置
const backendBaseUrlFromEnv = process.env.NEXT_PUBLIC_BACKEND_BASE_URL
const apiBaseUrlFromEnv = process.env.NEXT_PUBLIC_API_BASE_URL || process.env.ROBUSTA_API_BASE_URL

// 构建配置：优先使用显式配置，否则自动推导
const resolvedBackendBaseUrl = backendBaseUrlFromEnv || defaultBackendBaseUrl
const resolvedApiBaseUrl = apiBaseUrlFromEnv || `${resolvedBackendBaseUrl.replace(/\/+$/, '')}/api/v1`

const envConfig: Partial<FrontendConfig> = {
  backendBaseUrl: backendBaseUrlFromEnv || undefined,
  apiBaseUrl: apiBaseUrlFromEnv || undefined,
  casLoginPath: process.env.NEXT_PUBLIC_CAS_LOGIN_PATH || undefined,
  casLogoutPath: process.env.NEXT_PUBLIC_CAS_LOGOUT_PATH || undefined,
  basePath: process.env.NEXT_PUBLIC_BASE_PATH || undefined,
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
      ;(output as any)[key] = value
    }
  })
  return output
}

export function attachRuntimeConfig(config: Partial<FrontendConfig>) {
  if (typeof window === 'undefined') return
  window.__ROBUSTA_RUNTIME_CONFIG__ = {
    ...(window.__ROBUSTA_RUNTIME_CONFIG__ || {}),
    ...filterUndefined(config),
  }
}

const isAbsoluteUrl = (input: string) => /^https?:\/\//i.test(input)

const BASE_PATH_MANAGED_BY_NEXT = process.env.NEXT_MANAGED_BASE_PATH === 'true'

export function resolveAppPath(target: string, options?: { includeBasePath?: boolean }): string {
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
