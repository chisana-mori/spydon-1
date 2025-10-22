/**
 * Unicode转义序列解码演示
 * 
 * 问题描述:
 * 后端返回的JSON响应中包含Unicode转义序列(如 \u95ee\u9898),
 * 前端需要正确解码这些序列为可读的中文字符。
 * 
 * 示例输入:
 * {"sections": {"\u95ee\u9898\u5206\u6790\u62a5\u544a": "## \u95ee\u9898\u63cf\u8ff0"}}
 * 
 * 期望输出:
 * {"sections": {"问题分析报告": "## 问题描述"}}
 */

/**
 * 解码Unicode转义序列（如 \u95ee\u9898 -> 问题）
 * 处理后端返回的包含Unicode转义的JSON字符串
 */
export const decodeEscapedUnicode = (value: string): string => {
  if (!value || typeof value !== 'string') return value
  
  try {
    // 如果字符串包含Unicode转义序列，尝试解码
    if (value.includes('\\u')) {
      // 方法1: 尝试直接JSON.parse（适用于完整的JSON字符串）
      try {
        const parsed = JSON.parse(value)
        if (typeof parsed === 'string') {
          return parsed
        }
        // 如果解析结果是对象，返回格式化的JSON
        return JSON.stringify(parsed, null, 2)
      } catch {
        // JSON.parse失败，尝试方法2
      }
      
      // 方法2: 使用正则表达式替换Unicode转义序列
      return value.replace(/\\u([0-9a-fA-F]{4})/g, (_match, hex) => {
        return String.fromCharCode(parseInt(hex, 16))
      })
    }
    
    return value
  } catch (error) {
    console.warn('Failed to decode Unicode escapes:', error)
    return value
  }
}

// 演示示例
if (typeof window !== 'undefined') {
  // 仅在浏览器环境中运行演示
  console.group('Unicode解码演示')
  
  // 示例1: 简单的Unicode转义序列
  const example1 = '\\u95ee\\u9898\\u5206\\u6790\\u62a5\\u544a'
  console.log('输入:', example1)
  console.log('输出:', decodeEscapedUnicode(example1))
  console.log('期望: 问题分析报告')
  
  // 示例2: JSON对象中的Unicode转义
  const example2 = '{"sections": {"\\u95ee\\u9898\\u5206\\u6790\\u62a5\\u544a": "## \\u95ee\\u9898\\u63cf\\u8ff0"}}'
  console.log('\n输入:', example2)
  console.log('输出:', decodeEscapedUnicode(example2))
  
  // 示例3: 混合内容
  const example3 = 'Pod \\u5728 default \\u547d\\u540d\\u7a7a\\u95f4\\u4e2d\\u51fa\\u73b0\\u4e86 ImagePullBackOff'
  console.log('\n输入:', example3)
  console.log('输出:', decodeEscapedUnicode(example3))
  console.log('期望: Pod 在 default 命名空间中出现了 ImagePullBackOff')
  
  console.groupEnd()
}

