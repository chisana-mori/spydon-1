import { test, expect } from '@playwright/test'

test.describe('Authentication', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
  })

  test('should display login page', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Welcome to Spydon')
    await expect(page.locator('button[type="submit"]')).toBeVisible()
  })

  test('should show validation errors for empty form', async ({ page }) => {
    // 尝试提交空表单
    await page.click('button[type="submit"]')

    // 检查验证错误
    await expect(page.locator('text=Email is required')).toBeVisible()
    await expect(page.locator('text=Password is required')).toBeVisible()
  })

  test('should show error for invalid credentials', async ({ page }) => {
    // 填写无效凭据
    await page.fill('input[name="email"]', 'invalid@example.com')
    await page.fill('input[name="password"]', 'wrongpassword')

    // 提交表单
    await page.click('button[type="submit"]')

    // 检查错误消息
    await expect(page.locator('text=Invalid credentials')).toBeVisible()
  })

  test('should login successfully with valid credentials', async ({ page }) => {
    // 填写有效凭据（测试环境）
    await page.fill('input[name="email"]', 'test@spydon.com')
    await page.fill('input[name="password"]', 'testpassword')

    // 提交表单
    await page.click('button[type="submit"]')

    // 等待登录成功
    await expect(page.locator('h1')).toContainText('Dashboard', { timeout: 10000 })

    // 检查用户菜单是否显示
    await expect(page.locator('[data-testid="user-menu"]')).toBeVisible()
  })

  test('should logout successfully', async ({ page }) => {
    // 首先登录
    await page.fill('input[name="email"]', 'test@spydon.com')
    await page.fill('input[name="password"]', 'testpassword')
    await page.click('button[type="submit"]')
    await expect(page.locator('h1')).toContainText('Dashboard')

    // 点击用户菜单
    await page.click('[data-testid="user-menu"]')

    // 点击登出
    await page.click('text=Logout')

    // 检查是否回到登录页
    await expect(page.locator('h1')).toContainText('Welcome to Spydon')
  })
})
