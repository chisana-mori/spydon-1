import { NextRequest, NextResponse } from 'next/server'

const parseJsonSequence = (raw: string): any[] => {
  const trimmed = raw.trim()
  if (!trimmed) return []

  const result: any[] = []
  let buffer = ''
  let depth = 0
  let inString = false
  let escape = false

  const flush = () => {
    const candidate = buffer.trim()
    if (!candidate) {
      buffer = ''
      return
    }
    try {
      result.push(JSON.parse(candidate))
    } catch {
      // 忽略无法解析的片段
    }
    buffer = ''
  }

  for (let i = 0; i < trimmed.length; i += 1) {
    const char = trimmed[i]
    buffer += char

    if (escape) {
      escape = false
      continue
    }

    if (char === '\\') {
      escape = true
      continue
    }

    if (char === '"') {
      inString = !inString
      continue
    }

    if (!inString) {
      if (char === '{' || char === '[') depth += 1
      if (char === '}' || char === ']') depth -= 1
    }

    if (depth === 0 && !inString) {
      flush()
    }
  }

  flush()
  return result
}

export async function POST(request: NextRequest) {
  try {
    const body = await request.json()

    // 强化中文提示词
    if (body.description) {
      body.description = `${body.description}

🌟 SYSTEM INSTRUCTION - 系统指令:
You are HolmesGPT, a Kubernetes troubleshooting AI assistant. You MUST respond in Chinese (中文) for this request.
你是 HolmesGPT，一个 Kubernetes 故障排查 AI 助手。你必须用中文回答这个请求。

MANDATORY LANGUAGE REQUIREMENT - 强制语言要求:
- All analysis MUST be in Chinese - 所有分析必须用中文
- All explanations MUST be in Chinese - 所有解释必须用中文
- All recommendations MUST be in Chinese - 所有建议必须用中文
- Technical terms should be explained in Chinese - 技术术语应该用中文解释

请确保你的回答完全使用中文，包括：
1. 问题描述和分析
2. 根本原因分析
3. 解决方案步骤
4. 预防措施建议
5. 技术术语解释`
    }

    // HolmesGPT 服务地址 - 从环境变量获取
    const holmesGPTUrl = process.env.HOLMESGPT_URL || 'http://localhost:8080'

    console.log('代理 HolmesGPT 请求 (已添加中文提示):', {
      url: `${holmesGPTUrl}/api/stream/investigate`,
      body: JSON.stringify(body, null, 2)
    })

    // 向 HolmesGPT 发送流式请求
    const response = await fetch(`${holmesGPTUrl}/api/stream/investigate`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'text/event-stream',
        'Cache-Control': 'no-cache',
      },
      body: JSON.stringify(body),
    })

    if (!response.ok) {
      console.error('HolmesGPT 请求失败:', response.status, response.statusText)
      return NextResponse.json(
        { error: `HolmesGPT 服务错误: ${response.status} ${response.statusText}` },
        { status: response.status }
      )
    }

    if (!response.body) {
      return NextResponse.json(
        { error: 'HolmesGPT 响应体为空' },
        { status: 500 }
      )
    }

    // 创建流式响应
    const stream = new ReadableStream({
      async start(controller) {
        const reader = response.body!.getReader()
        const decoder = new TextDecoder()

        try {
          while (true) {
            const { done, value } = await reader.read()
            
            if (done) {
              // 发送完成事件
              controller.enqueue(
                new TextEncoder().encode('data: {"type":"complete","data":{}}\n\n')
              )
              controller.close()
              break
            }

            const chunk = decoder.decode(value, { stream: true })
            const lines = chunk.split('\n')

            for (const line of lines) {
              if (line.trim() === '') continue
              
              try {
                // 处理 HolmesGPT 的流式响应格式
                if (line.startsWith('data: ')) {
                  const eventData = line.slice(6)
                  
                  if (eventData === '[DONE]') {
                    controller.enqueue(
                      new TextEncoder().encode('data: {"type":"complete","data":{}}\n\n')
                    )
                    continue
                  }

                  // 尝试解析 JSON 数据
                  let parsedData
      try {
        parsedData = JSON.parse(eventData)
      } catch (parseError) {
        const sequence = parseJsonSequence(eventData)
        if (sequence.length) {
          sequence.forEach((item) => {
            const structuredEvent = {
              type: 'analysis',
              data: item,
            }
            controller.enqueue(
              new TextEncoder().encode(`data: ${JSON.stringify(structuredEvent)}\n\n`)
            )
          })
          continue
        }

        const textEvent = {
          type: 'analysis',
          data: eventData,
        }
        controller.enqueue(
          new TextEncoder().encode(`data: ${JSON.stringify(textEvent)}\n\n`)
        )
        continue
      }

                  // 根据 HolmesGPT 的响应格式转换事件类型
                  let transformedEvent

                  if (parsedData.analysis) {
                    transformedEvent = {
                      type: 'analysis',
                      data: parsedData.analysis,
                    }
                  } else if (parsedData.tool_calls) {
                    // 将 tool_calls 数组拆分为多个 analysis 事件，便于前端按顺序消费
                    try {
                      const calls: any[] = Array.isArray(parsedData.tool_calls) ? parsedData.tool_calls : []
                      for (const call of calls) {
                        const perCallEvent = {
                          type: 'analysis',
                          data: call,
                        }
                        controller.enqueue(
                          new TextEncoder().encode(`data: ${JSON.stringify(perCallEvent)}\n\n`)
                        )
                      }
                      // 已逐条发送，跳过默认 transformedEvent 入队
                      continue
                    } catch {
                      // 兜底逻辑：若拆分失败，仍按 analysis 整体发送
                      transformedEvent = {
                        type: 'analysis',
                        data: parsedData.tool_calls,
                      }
                    }
                  } else if (parsedData.error) {
                    transformedEvent = {
                      type: 'error',
                      data: { message: parsedData.error },
                    }
                  } else if (
                    parsedData.content ||
                    parsedData.tool_name ||
                    parsedData.todos ||
                    parsedData.params?.todos ||
                    parsedData.result?.data
                  ) {
                    transformedEvent = {
                      type: 'analysis',
                      data: parsedData,
                    }
                  } else {
                    transformedEvent = {
                      type: 'analysis',
                      data: parsedData,
                    }
                  }

                  controller.enqueue(
                    new TextEncoder().encode(`data: ${JSON.stringify(transformedEvent)}\n\n`)
                  )
                } else {
                  // 非标准格式，直接转发
                  controller.enqueue(new TextEncoder().encode(`${line}\n`))
                }
              } catch (error) {
                console.warn('处理流数据行失败:', error, 'Line:', line)
                // 发送错误但不中断流
                const errorEvent = {
                  type: 'error',
                  data: { message: `数据处理错误: ${error}` }
                }
                controller.enqueue(
                  new TextEncoder().encode(`data: ${JSON.stringify(errorEvent)}\n\n`)
                )
              }
            }
          }
        } catch (error) {
          console.error('流处理错误:', error)
          const errorEvent = {
            type: 'error',
            data: { message: `流处理失败: ${error}` }
          }
          controller.enqueue(
            new TextEncoder().encode(`data: ${JSON.stringify(errorEvent)}\n\n`)
          )
          controller.close()
        } finally {
          reader.releaseLock()
        }
      },
    })

    return new NextResponse(stream, {
      headers: {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache, no-store, must-revalidate',
        'Connection': 'keep-alive',
        'Access-Control-Allow-Origin': '*',
        'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
        'Access-Control-Allow-Headers': 'Content-Type',
      },
    })

  } catch (error) {
    console.error('HolmesGPT 代理错误:', error)
    return NextResponse.json(
      { error: `代理请求失败: ${error}` },
      { status: 500 }
    )
  }
}

export async function OPTIONS() {
  return new NextResponse(null, {
    status: 200,
    headers: {
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type',
    },
  })
}
