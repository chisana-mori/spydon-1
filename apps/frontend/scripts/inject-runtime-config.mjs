#!/usr/bin/env node
/**
 * 在构建后的 HTML 中注入运行时配置
 * 用于支持容器化部署时动态修改配置
 */
import { readFileSync, writeFileSync, existsSync, readdirSync } from 'fs'
import { join, dirname } from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)

const configPath = join(__dirname, '../config/runtime.json')
const distPath = join(__dirname, '../.next')

function injectConfig() {
  if (!existsSync(configPath)) {
    console.log('⚠ No runtime.json found, skipping injection')
    return
  }

  let config
  try {
    const content = readFileSync(configPath, 'utf-8')
    config = JSON.parse(content)
  } catch (error) {
    console.error('✗ Failed to parse runtime.json:', error.message)
    return
  }

  // 生成注入脚本
  const injectionScript = `
<script>
  window.__ROBUSTA_RUNTIME_CONFIG__ = ${JSON.stringify(config, null, 2)};
</script>`

  console.log('✓ Runtime config injection script prepared')
  console.log('  Note: For production deployment, inject this in your HTML template or use environment variables')
}

injectConfig()
