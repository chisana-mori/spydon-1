const nextJest = require('next/jest')

const createJestConfig = nextJest({
  // Next.js 应用路径
  dir: './',
})

// Jest 自定义配置
const customJestConfig = {
  // 测试环境设置
  testEnvironment: 'jest-environment-jsdom',

  // 模块路径映射，类似 tsconfig.json 中的 paths
  moduleNameMapping: {
    '^@/(.*)$': '<rootDir>/src/$1',
    '^@components/(.*)$': '<rootDir>/src/components/$1',
    '^@lib/(.*)$': '<rootDir>/src/lib/$1',
    '^@app/(.*)$': '<rootDir>/src/app/$1',
  },

  // 测试文件匹配模式
  testMatch: [
    '<rootDir>/src/**/__tests__/**/*.{js,jsx,ts,tsx}',
    '<rootDir>/src/**/*.{test,spec}.{js,jsx,ts,tsx}',
    '<rootDir>/__tests__/**/*.{js,jsx,ts,tsx}',
  ],

  // 忽略的测试文件和目录
  testPathIgnorePatterns: [
    '<rootDir>/.next/',
    '<rootDir>/node_modules/',
    '<rootDir>/out/',
    '<rootDir>/coverage/',
  ],

  // 转换忽略模式
  transformIgnorePatterns: [
    '/node_modules/(?!(.*\\.mjs$))',
  ],

  // 设置文件
  setupFilesAfterEnv: ['<rootDir>/jest.setup.js'],

  // 代码覆盖率配置
  collectCoverageFrom: [
    'src/**/*.{js,jsx,ts,tsx}',
    '!src/**/*.d.ts',
    '!src/**/*.stories.{js,jsx,ts,tsx}',
    '!src/**/index.ts',
  ],

  // 覆盖率阈值
  coverageThreshold: {
    global: {
      branches: 70,
      functions: 70,
      lines: 70,
      statements: 70,
    },
  },

  // 覆盖率报告格式
  coverageReporters: ['text', 'lcov', 'html'],

  // 输出覆盖率到目录
  coverageDirectory: 'coverage',

  // 在测试失败时显示覆盖率
  collectCoverage: true,

  // Mock 模块
  modulePathIgnorePatterns: ['<rootDir>/dist/'],

  // 测试超时时间 (毫秒)
  testTimeout: 10000,

  // 详细输出
  verbose: true,

  // 清除模拟调用和实例
  clearMocks: true,

  // 恢复所有模拟实现
  restoreMocks: true,
}

// 创建 Jest 配置
module.exports = createJestConfig(customJestConfig)
