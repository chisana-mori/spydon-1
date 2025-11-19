export const buildCachedSSEChunks = (cached: any): string[] => {
  if (Array.isArray(cached?.stream_chunks) && cached.stream_chunks.length > 0) {
    return cached.stream_chunks
  }
  const events: string[] = []
  const pushEvent = (payload: any) => { events.push(`data: ${JSON.stringify(payload)}\n\n`) }
  if (cached?.analysis) { pushEvent({ type: 'analysis', data: cached.analysis }) }
  const fullText = cached?.metadata?.full_text
  if (fullText) { pushEvent({ type: 'analysis', data: { content: fullText } }) }
  const summary = cached?.metadata?.summary
  if (summary) { pushEvent({ type: 'analysis', data: { content: summary, summary } }) }
  if (events.length) { pushEvent({ type: 'complete', data: {} }); events.push('data: [DONE]\n\n') }
  return events
}

export const createCachedChunkIterable = (chunks: string[]): AsyncIterable<string> => ({
  async *[Symbol.asyncIterator]() {
    for (const chunk of chunks) {
      if (typeof chunk !== 'string') continue
      if (!chunk) continue
      yield chunk
    }
  }
})

export const createResponseChunkIterable = (response: Response) => {
  if (!response.body) { throw new Error('无法读取响应流') }
  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let completed = false
  const iterable: AsyncIterable<string> = {
    async *[Symbol.asyncIterator]() {
      try {
        while (true) {
          const { done, value } = await reader.read()
          if (done) {
            completed = true
            const remaining = decoder.decode()
            if (remaining) { yield remaining }
            break
          }
          if (value) { yield decoder.decode(value, { stream: true }) }
        }
      } finally {
        if (!completed) { try { await reader.cancel() } catch {} }
        try { reader.releaseLock() } catch {}
      }
    }
  }
  const stop = async () => { completed = true; try { await reader.cancel() } catch {} }
  return { iterable, stop }
}

export const consumeSSEChunks = async (
  iterable: AsyncIterable<string>,
  processor: { appendChunk: (chunk: string) => boolean },
  options?: { onStop?: () => void | Promise<void> }
): Promise<void> => {
  for await (const chunk of iterable) {
    if (!chunk) continue
    const shouldStop = processor.appendChunk(chunk)
    if (shouldStop) {
      if (options?.onStop) { try { await options.onStop() } catch {} }
      break
    }
  }
}
