import { chromium, FullConfig } from '@playwright/test'

async function globalSetup(config: FullConfig) {
  console.log('🚀 Starting global E2E test setup...')

  const browser = await chromium.launch()
  const context = await browser.newContext()
  const page = await context.newPage()

  try {
    // 等待应用启动
    console.log('⏳ Waiting for application to be ready...')
    await page.goto(config.webServer?.url || 'http://localhost:3000')

    // 检查健康检查端点
    const response = await page.goto('/api/health')
    if (response && response.status() !== 200) {
      throw new Error(`Application health check failed: ${response.status()}`)
    }

    console.log('✅ Application is ready for E2E testing')

    // 设置测试数据（如果需要）
    await setupTestData(page)

    console.log('✅ Global setup completed')
  } catch (error) {
    console.error('❌ Global setup failed:', error)
    throw error
  } finally {
    await context.close()
    await browser.close()
  }
}

async function setupTestData(page) {
  // 创建测试用户和数据
  console.log('🔧 Setting up test data...')

  // 这里可以添加创建测试数据的逻辑
  // 例如：创建测试用户、测试告警等

  // 示例：创建测试告警
  try {
    const response = await page.request.post('/api/v1/test/setup', {
      data: {
        createTestData: true,
      },
    })

    if (response.status() === 200) {
      console.log('✅ Test data created successfully')
    }
  } catch (error) {
    console.warn('⚠️ Could not create test data:', error.message)
  }
}

export default globalSetup
