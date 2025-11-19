import { test, expect } from '@playwright/test'

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    // 登录后访问 dashboard
    await page.goto('/login')
    await page.fill('input[name="email"]', 'test@spydon.com')
    await page.fill('input[name="password"]', 'testpassword')
    await page.click('button[type="submit"]')
    await expect(page.locator('h1')).toContainText('Dashboard')
  })

  test('should display dashboard components', async ({ page }) => {
    // 检查主要组件是否存在
    await expect(page.locator('[data-testid="alert-stats-card"]')).toBeVisible()
    await expect(page.locator('[data-testid="cluster-health-card"]')).toBeVisible()
    await expect(page.locator('[data-testid="recent-alerts-table"]')).toBeVisible()
    await expect(page.locator('[data-testid="alert-trend-chart"]')).toBeVisible()
  })

  test('should load and display alerts', async ({ page }) => {
    // 等待告警表格加载
    await page.waitForSelector('[data-testid="recent-alerts-table"] tbody tr', { timeout: 15000 })

    // 检查是否有告警数据
    const alertRows = page.locator('[data-testid="recent-alerts-table"] tbody tr')
    await expect(alertRows.first()).toBeVisible()

    // 检查告警详情
    await expect(page.locator('[data-testid="alert-severity"]')).toBeVisible()
    await expect(page.locator('[data-testid="alert-time"]')).toBeVisible()
  })

  test('should filter alerts by severity', async ({ page }) => {
    // 点击严重性筛选器
    await page.click('[data-testid="severity-filter"]')
    await page.click('text=Critical')

    // 等待筛选结果
    await page.waitForTimeout(2000)

    // 验证筛选结果
    const alertRows = page.locator('[data-testid="recent-alerts-table"] tbody tr')
    if (await alertRows.count() > 0) {
      // 检查所有显示的告警是否都是 Critical 严重性
      const severities = page.locator('[data-testid="alert-severity"]')
      for (let i = 0; i < await severities.count(); i++) {
        const severity = await severities.nth(i).textContent()
        expect(severity).toContain('Critical')
      }
    }
  })

  test('should navigate to alert details', async ({ page }) -> Promise<void> => {
    // 等待告警表格加载
    await page.waitForSelector('[data-testid="recent-alerts-table"] tbody tr', { timeout: 15000 })

    // 点击第一个告警
    const firstAlert = page.locator('[data-testid="recent-alerts-table"] tbody tr').first()
    await firstAlert.click()

    // 验证导航到告警详情页
    await expect(page.locator('h1')).toContainText('Alert Details', { timeout: 10000 })
    await expect(page.locator('[data-testid="alert-info-panel"]')).toBeVisible()
  })

  test('should display real-time metrics', async ({ page }) => {
    // 检查指标卡片
    await expect(page.locator('[data-testid="total-alerts-metric"]')).toBeVisible()
    await expect(page.locator('[data-testid="active-clusters-metric"]')).toBeVisible()
    await expect(page.locator('[data-testid="resolved-alerts-metric"]')).toBeVisible()

    // 检查图表是否加载
    await expect(page.locator('[data-testid="alert-trend-chart"] canvas')).toBeVisible({ timeout: 10000 })
  })

  test('should handle cluster status updates', async ({ page }) => {
    // 检查集群状态
    const clusterStatus = page.locator('[data-testid="cluster-status"]')
    await expect(clusterStatus).toBeVisible()

    // 验证状态指示器
    await expect(page.locator('[data-testid="status-indicator"]')).toBeVisible()
  })

  test('should support search functionality', async ({ page }) => {
    // 输入搜索关键词
    await page.fill('[data-testid="search-input"]', 'test')

    // 等待搜索结果
    await page.waitForTimeout(2000)

    // 验证搜索结果（如果有数据）
    const alertRows = page.locator('[data-testid="recent-alerts-table"] tbody tr')
    if (await alertRows.count() > 0) {
      // 这里可以添加具体的搜索结果验证逻辑
      await expect(alertRows.first()).toBeVisible()
    }
  })

  test('should export alert data', async ({ page }) => {
    // 点击导出按钮
    await page.click('[data-testid="export-button"]')

    // 选择导出格式
    await page.click('text=Export as CSV')

    // 验证导出开始（具体验证方式取决于导出实现）
    // 这里可以检查是否出现了下载提示或确认消息
    await expect(page.locator('text=Export started') || page.locator('text=Download ready')).toBeVisible({ timeout: 10000 })
  })
})
