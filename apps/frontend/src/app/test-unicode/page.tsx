'use client'

import React, { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { formatSummaryText } from '@/components/alerts/ChatMessage'

/**
 * Unicode转义序列解码测试页面
 * 用于验证修复是否正确工作
 */
export default function TestUnicodePage() {
  const [testResults, setTestResults] = useState<Array<{
    name: string
    input: string
    output: string
    expected: string
    passed: boolean
  }>>([])

  const runTests = () => {
    const tests = [
      {
        name: '简单Unicode转义',
        input: '\\u95ee\\u9898\\u5206\\u6790\\u62a5\\u544a',
        expected: '问题分析报告'
      },
      {
        name: 'JSON对象中的Unicode转义',
        input: '{"sections": {"\\u95ee\\u9898\\u5206\\u6790\\u62a5\\u544a": "## \\u95ee\\u9898\\u63cf\\u8ff0\\nPod crash-loop-test \\u5728 default \\u547d\\u540d\\u7a7a\\u95f4\\u4e2d\\u51fa\\u73b0\\u4e86 ImagePullBackOff"}}',
        expected: '问题分析报告'
      },
      {
        name: '混合内容',
        input: 'Pod \\u5728 default \\u547d\\u540d\\u7a7a\\u95f4\\u4e2d\\u51fa\\u73b0\\u4e86 ImagePullBackOff',
        expected: 'Pod 在 default 命名空间中出现了 ImagePullBackOff'
      },
      {
        name: '普通字符串(无转义)',
        input: 'This is a normal string',
        expected: 'This is a normal string'
      },
      {
        name: '实际后端响应示例',
        input: '{"sections": {"\\u95ee\\u9898\\u5206\\u6790\\u62a5\\u544a": "## \\u95ee\\u9898\\u63cf\\u8ff0\\n\\n## \\u5173\\u952e\\u53d1\\u73b0\\n\\n1. \\u955c\\u50cf\\u62c9\\u53d6\\u72b6\\u6001\\n2. \\u5bb9\\u5668\\u72b6\\u6001\\n3. \\u5065\\u5eb7\\u68c0\\u67e5\\u914d\\u7f6e\\n\\n## \\u89e3\\u51b3\\u65b9\\u6848\\u6b65\\u9aa4"}}',
        expected: '问题分析报告'
      }
    ]

    const results = tests.map(test => {
      const output = formatSummaryText(test.input)
      const passed = output.includes(test.expected)

      return {
        name: test.name,
        input: test.input,
        output,
        expected: test.expected,
        passed
      }
    })

    setTestResults(results)
  }

  return (
    <div className="container mx-auto p-6 max-w-6xl">
      <Card>
        <CardHeader>
          <CardTitle>Unicode转义序列解码测试</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="flex items-center space-x-4">
            <Button onClick={runTests}>运行测试</Button>
            {testResults.length > 0 && (
              <div className="text-sm">
                <span className="text-green-600 font-semibold">
                  通过: {testResults.filter(r => r.passed).length}
                </span>
                {' / '}
                <span className="text-gray-600">
                  总计: {testResults.length}
                </span>
              </div>
            )}
          </div>

          {testResults.length > 0 && (
            <div className="space-y-4">
              {testResults.map((result, index) => (
                <Card key={index} className={result.passed ? 'border-green-200' : 'border-red-200'}>
                  <CardHeader>
                    <div className="flex items-center justify-between">
                      <CardTitle className="text-base">{result.name}</CardTitle>
                      <span className={`text-sm font-semibold ${result.passed ? 'text-green-600' : 'text-red-600'}`}>
                        {result.passed ? '✓ 通过' : '✗ 失败'}
                      </span>
                    </div>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <div>
                      <div className="text-xs font-semibold text-gray-500 mb-1">输入:</div>
                      <div className="bg-gray-100 p-2 rounded text-xs font-mono break-all">
                        {result.input.substring(0, 200)}
                        {result.input.length > 200 && '...'}
                      </div>
                    </div>

                    <div>
                      <div className="text-xs font-semibold text-gray-500 mb-1">期望包含:</div>
                      <div className="bg-blue-50 p-2 rounded text-sm">
                        {result.expected}
                      </div>
                    </div>

                    <div>
                      <div className="text-xs font-semibold text-gray-500 mb-1">实际输出:</div>
                      <div className={`p-2 rounded text-sm ${result.passed ? 'bg-green-50' : 'bg-red-50'}`}>
                        {result.output.substring(0, 300)}
                        {result.output.length > 300 && '...'}
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}

          <div className="mt-8 p-4 bg-blue-50 rounded-lg">
            <h3 className="text-sm font-semibold text-blue-900 mb-2">测试说明</h3>
            <ul className="text-xs text-blue-800 space-y-1">
              <li>• 测试1: 验证简单的Unicode转义序列解码</li>
              <li>• 测试2: 验证JSON对象中的Unicode转义解码</li>
              <li>• 测试3: 验证混合内容(中英文)的解码</li>
              <li>• 测试4: 验证普通字符串不受影响</li>
              <li>• 测试5: 验证实际后端响应的解码</li>
            </ul>
          </div>

          <div className="mt-4 p-4 bg-yellow-50 rounded-lg">
            <h3 className="text-sm font-semibold text-yellow-900 mb-2">修复内容</h3>
            <ul className="text-xs text-yellow-800 space-y-1">
              <li>• 增强了 <code className="bg-yellow-100 px-1 rounded">decodeEscapedUnicode</code> 函数</li>
              <li>• 更新了 <code className="bg-yellow-100 px-1 rounded">formatSummaryText</code> 函数</li>
              <li>• 支持两种解码方法: JSON.parse 和正则表达式替换</li>
              <li>• 添加了错误处理和日志记录</li>
            </ul>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
