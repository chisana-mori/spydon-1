import { chromium, FullConfig } from '@playwright/test'

async function globalTeardown(config: FullConfig) {
  console.log('🧹 Starting global E2E test teardown...')

  const browser = await chromium.launch()
  const context = await browser.newContext()
  const page = await context.newPage()

  try {
    // 清理测试数据
    await cleanupTestData(page)

    console.log('✅ Test data cleaned up')
    console.log('✅ Global teardown completed')
  } catch (error) {
    console.error('❌ Global teardown failed:', error)
  } finally {
    await context.close()
    await browser.close()
  }
}

async function cleanupTestData(page) {
  console.log('🧹 Cleaning up test data...')

  try {
    const response = await page.request.post('/api/v1/test/cleanup', {
      data: {
        cleanupTestData: true,
      },
    })

    if (response.status() === 200) {
      console.log('✅ Test data cleaned up successfully')
    }
  } catch (error) {
    console.warn('⚠️ Could not cleanup test data:', error.message)
  }
}

export default globalTeardown
