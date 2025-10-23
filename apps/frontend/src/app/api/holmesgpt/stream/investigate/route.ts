import { NextRequest, NextResponse } from 'next/server'
import { appConfig } from '@/config'

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

    // 项目后端 HolmesGPT 代理地址
    const backendBaseUrl = appConfig.apiBaseUrl.replace(/\/$/, '')
    const upstreamUrl = `${backendBaseUrl}/holmesgpt/stream/investigate`

    console.log('通过后端代理 HolmesGPT 请求 (已添加中文提示):', {
      url: upstreamUrl,
      body: JSON.stringify(body, null, 2)
    })

    // 建立到上游 HolmesGPT 的可中断请求
    const upstreamController = new AbortController()

    // 当客户端取消（例如点击“停止分析”导致 fetch abort）时，联动中断上游请求
    const onClientAbort = () => {
      try {
        upstreamController.abort()
      } catch {}
    }
    request.signal.addEventListener('abort', onClientAbort, { once: true })

    // 向 HolmesGPT 发送流式请求（携带 abort signal）
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'Accept': 'text/event-stream',
      'Cache-Control': 'no-cache',
    }

    const apiToken = process.env.HOLMES_GPT_PROXY_TOKEN || process.env.API_AUTH_TOKEN || ''
    if (apiToken) {
      headers['Authorization'] = `Bearer ${apiToken}`
    }

    const response = await fetch(upstreamUrl, {
      method: 'POST',
      headers,
      body: JSON.stringify(body),
      signal: upstreamController.signal,
    })

    if (!response.ok) {
      console.error('后端 HolmesGPT 代理请求失败:', response.status, response.statusText)
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

    // 创建流式响应（支持 cancel 以联动中断上游）
    const stream = new ReadableStream({
      async start(controller) {
        const reader = response.body!.getReader()
        const decoder = new TextDecoder()
        const encoder = new TextEncoder()

        let buffer = ''
        let currentEventType: string | null = null
        let completionEmitted = false

        const emitEvent = (payload: any, eventType?: string | null) => {
          const type = (eventType ?? '').trim()
          const typeLine = type ? `event: ${type}\n` : ''
          controller.enqueue(
            encoder.encode(`${typeLine}data: ${JSON.stringify(payload)}\n\n`)
          )
        }

        const forwardTransformedEvent = (parsedData: any) => {
          let transformedEvent: { type: string; data: any } | null = null

          if (parsedData?.analysis) {
            transformedEvent = {
              type: 'analysis',
              data: parsedData.analysis,
            }
          } else if (parsedData?.tool_calls) {
            try {
              const calls: any[] = Array.isArray(parsedData.tool_calls) ? parsedData.tool_calls : []
              for (const call of calls) {
                emitEvent(
                  {
                    type: 'analysis',
                    data: call,
                  },
                  currentEventType
                )
              }
              return
            } catch {
              transformedEvent = {
                type: 'analysis',
                data: parsedData.tool_calls,
              }
            }
          } else if (parsedData?.error) {
            transformedEvent = {
              type: 'error',
              data: { message: parsedData.error },
            }
          } else if (
            parsedData?.content ||
            parsedData?.tool_name ||
            parsedData?.todos ||
            parsedData?.params?.todos ||
            parsedData?.result?.data
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

          if (transformedEvent) {
            emitEvent(transformedEvent, currentEventType)
          }
        }

        const processEventData = (eventData: string) => {
          if (eventData === '[DONE]') {
            emitEvent({ type: 'complete', data: {} }, currentEventType)
            completionEmitted = true
            return
          }

          let parsedData: any
          try {
            parsedData = JSON.parse(eventData)
          } catch {
            const sequence = parseJsonSequence(eventData)
            if (sequence.length) {
              sequence.forEach(item => {
                emitEvent(
                  {
                    type: 'analysis',
                    data: item,
                  },
                  currentEventType
                )
              })
              return
            }

            emitEvent(
              {
                type: 'analysis',
                data: eventData,
              },
              currentEventType
            )
            return
          }

          forwardTransformedEvent(parsedData)
        }

        const processBuffer = () => {
          let newlineIndex = buffer.indexOf('\n')
          while (newlineIndex !== -1) {
            let line = buffer.slice(0, newlineIndex)
            buffer = buffer.slice(newlineIndex + 1)

            if (line.endsWith('\r')) {
              line = line.slice(0, -1)
            }

            if (line === '') {
              currentEventType = null
            } else if (line.startsWith(':')) {
              // 注释行，忽略
            } else {
              const eventMatch = line.match(/^event:\s*(.*)$/)
              const dataMatch = line.match(/^data:\s*(.*)$/)

              if (eventMatch) {
                currentEventType = eventMatch[1]
              } else if (dataMatch) {
                processEventData(dataMatch[1])
              } else {
                controller.enqueue(encoder.encode(`${line}\n`))
              }
            }

            newlineIndex = buffer.indexOf('\n')
          }
        }

        try {
          while (true) {
            const { done, value } = await reader.read()

            if (value) {
              buffer += decoder.decode(value, { stream: !done })
              processBuffer()
            }

            if (done) {
              const remaining = decoder.decode()
              if (remaining) {
                buffer += remaining
              }

              if (buffer.length) {
                buffer += '\n'
                processBuffer()
              }

              if (!completionEmitted) {
                emitEvent({ type: 'complete', data: {} })
                completionEmitted = true
              }
              controller.close()
              break
            }
          }
        } catch (error) {
          console.error('流处理错误:', error)
          emitEvent({
            type: 'error',
            data: { message: `流处理失败: ${error}` }
          })
          controller.close()
        } finally {
          try { reader.releaseLock() } catch {}
        }
      },
      cancel(reason) {
        // 前端取消消费时触发：中断上游请求，避免后台继续占用资源
        try { upstreamController.abort() } catch {}
      },
    })

    const nextResponse = new NextResponse(stream, {
      headers: {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache, no-store, must-revalidate',
        'Connection': 'keep-alive',
        'X-Accel-Buffering': 'no',
        'Access-Control-Allow-Origin': '*',
        'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
        'Access-Control-Allow-Headers': 'Content-Type',
      },
    })

    return nextResponse

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
