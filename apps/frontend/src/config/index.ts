export type FrontendConfig = {
  apiBaseUrl: string
  casLoginPath: string
  casLogoutPath: string
}

declare global {
  interface Window {
    __ROBUSTA_RUNTIME_CONFIG__?: Partial<FrontendConfig>
  }
}

const defaultConfig: FrontendConfig = {
  apiBaseUrl: 'http://localhost:8080/api/v1',
  casLoginPath: '/auth/cas/login',
  casLogoutPath: '/auth/cas/logout',
}

const envConfig: Partial<FrontendConfig> = {
  apiBaseUrl:
    process.env.NEXT_PUBLIC_API_BASE_URL ||
    process.env.ROBUSTA_API_BASE_URL ||
    undefined,
  casLoginPath: process.env.NEXT_PUBLIC_CAS_LOGIN_PATH || undefined,
  casLogoutPath: process.env.NEXT_PUBLIC_CAS_LOGOUT_PATH || undefined,
}

const runtimeConfig: Partial<FrontendConfig> =
  (typeof window !== 'undefined' && window.__ROBUSTA_RUNTIME_CONFIG__) || {}

export const appConfig: FrontendConfig = {
  ...defaultConfig,
  ...runtimeConfig,
  ...filterUndefined(envConfig),
}

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
