#!/usr/bin/env node
import { readFileSync, existsSync } from 'fs'
import { fileURLToPath } from 'url'
import { dirname, join } from 'path'

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)

const configPath = join(__dirname, '../config/runtime.json')
const sampleConfigPath = join(__dirname, '../config/runtime.sample.json')

let config = {}

// 优先读取 runtime.json，如果不存在则读取 runtime.sample.json
if (existsSync(configPath)) {
  try {
    const content = readFileSync(configPath, 'utf-8')
    config = JSON.parse(content)
    console.log('✓ Loaded config from runtime.json')
  } catch (error) {
    console.error('✗ Failed to parse runtime.json:', error.message)
    process.exit(1)
  }
} else if (existsSync(sampleConfigPath)) {
  try {
    const content = readFileSync(sampleConfigPath, 'utf-8')
    config = JSON.parse(content)
    console.log('⚠ Using runtime.sample.json (runtime.json not found)')
  } catch (error) {
    console.error('✗ Failed to parse runtime.sample.json:', error.message)
    process.exit(1)
  }
} else {
  console.log('⚠ No config file found, using defaults')
}

// 输出环境变量格式，供 Next.js 使用
if (config.backendBaseUrl) {
  console.log(`NEXT_PUBLIC_BACKEND_BASE_URL=${config.backendBaseUrl}`)
}
if (config.apiBaseUrl) {
  console.log(`NEXT_PUBLIC_API_BASE_URL=${config.apiBaseUrl}`)
}
if (config.casLoginPath) {
  console.log(`NEXT_PUBLIC_CAS_LOGIN_PATH=${config.casLoginPath}`)
}
if (config.casLogoutPath) {
  console.log(`NEXT_PUBLIC_CAS_LOGOUT_PATH=${config.casLogoutPath}`)
}
if (config.basePath) {
  console.log(`NEXT_PUBLIC_BASE_PATH=${config.basePath}`)
}

// 导出配置对象供其他脚本使用
export default config
