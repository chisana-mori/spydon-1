import { HolmesStructuredData, HolmesTaskItem, HolmesTaskSection, formatSummaryText } from './ChatMessage'

export function normalizePlainText(text: string): string {
  if (!text) return text
  let result = text
  result = result.replace(/\r\n/g, '\n')
  result = result.replace(/\u000d\u000a/gi, '\n')
  result = result.replace(/\\r\\n/g, '\n')
  result = result.replace(/\\n/g, '\n')
  result = result.replace(/\\t/g, '    ')
  result = result.replace(/\u00a0/g, ' ')
  result = result.replace(/\n{3,}/g, '\n\n')
  return result.trimEnd()
}

export function appendTextChunk(base: string, chunk: string): string {
  const normalizedBase = normalizePlainText(base)
  if (!chunk) return normalizedBase
  const normalizedChunk = normalizePlainText(chunk)
  const trimmedChunk = normalizedChunk.trim()
  if (!trimmedChunk) return base
  const flatChunk = trimmedChunk.replace(/\s+/g, ' ')
  const flatBase = normalizedBase.replace(/\s+/g, ' ')
  if (flatBase.includes(flatChunk)) return normalizedBase
  return normalizedBase ? `${normalizedBase}\n\n${trimmedChunk}` : trimmedChunk
}

export function appendCommands(existing?: string[], incoming?: string[]): string[] | undefined {
  const merged = new Set<string>()
  existing?.forEach(cmd => {
    if (cmd && cmd.trim()) merged.add(cmd.trim())
  })
  incoming?.forEach(cmd => {
    if (cmd && cmd.trim()) merged.add(cmd.trim())
  })
  return merged.size ? Array.from(merged) : undefined
}

export function canonicalizeSummaryFragment(fragment: string): string {
  if (!fragment) return ''
  const normalized = normalizePlainText(fragment)
  return normalized
    .replace(/(^|\n)#+\s*/g, '$1')
    .replace(/(^|\n)[-*]\s+/g, '$1')
    .replace(/(^|\n)\d+\.\s+/g, '$1')
    .replace(/[`*_]/g, '')
    .replace(/\s+/g, ' ')
    .trim()
    .toLowerCase()
}

export function deduplicateSummaryBlocks(text: string): string {
  if (!text) return text
  const parts = text
    .split(/\n{2,}/)
    .map(part => normalizePlainText(part).trim())
    .filter(Boolean)
  const seen = new Set<string>()
  const unique: string[] = []
  parts.forEach(part => {
    const signature = canonicalizeSummaryFragment(part)
    if (signature && !seen.has(signature)) {
      seen.add(signature)
      unique.push(part)
    }
  })
  return unique.join('\n\n')
}

export function buildSummarySignature(text: string): string | null {
  if (!text || typeof text !== 'string') return null
  const canonical = canonicalizeSummaryFragment(text)
  if (!canonical) return null
  return `${canonical.length}:${canonical.substring(0, 100)}`
}

export function collectCommands(payload: any): string[] {
  const commands: string[] = []
  const tryAdd = (value?: any) => {
    if (typeof value === 'string') {
      const trimmed = value.trim()
      if (trimmed && !commands.includes(trimmed)) {
        commands.push(trimmed)
      }
    }
  }
  tryAdd(payload?.invocation)
  tryAdd(payload?.description)
  tryAdd(payload?.command)
  tryAdd(payload?.result?.invocation)
  tryAdd(payload?.result?.description)
  return commands
}

export const normalizeTaskStatus = (status: any): HolmesTaskItem['status'] => {
  const value = String(status ?? '').toLowerCase()
  if (value.includes('progress') || value.includes('running')) return 'in_progress'
  if (value.includes('complete') || value.includes('done') || value.includes('success')) return 'completed'
  return 'pending'
}

export const parseJsonObjectSequence = (raw: string): any[] => {
  const trimmed = raw.trim()
  if (!trimmed) return []
  const result: any[] = []
  let buffer = ''
  let depth = 0
  let inString = false
  let escape = false
  const flushBuffer = () => {
    const candidate = buffer.trim()
    if (!candidate) {
      buffer = ''
      return
    }
    try {
      result.push(JSON.parse(candidate))
    } catch {}
    buffer = ''
  }
  for (let i = 0; i < trimmed.length; i += 1) {
    const char = trimmed[i]
    buffer += char
    if (escape) { escape = false; continue }
    if (char === '\\') { escape = true; continue }
    if (char === '"') { inString = !inString; continue }
    if (!inString) {
      if (char === '{' || char === '[') depth += 1
      if (char === '}' || char === ']') depth -= 1
    }
    if (depth === 0 && !inString) flushBuffer()
  }
  flushBuffer()
  return result
}

export const parseTasksFromStatusText = (statusText: string): HolmesTaskItem[] | null => {
  if (typeof statusText !== 'string') return null
  const tasks: HolmesTaskItem[] = []
  const lines = statusText.split('\n')
  for (const line of lines) {
    const taskMatch = line.match(/\[(.*?)\]\s*\[(\d+)\]\s*(.*)/)
    if (taskMatch) {
      const [, statusSymbol, id, content] = taskMatch
      let status: HolmesTaskItem['status'] = 'pending'
      if (statusSymbol.includes('✓')) status = 'completed'
      else if (statusSymbol.includes('~') || statusSymbol.includes('▶')) status = 'in_progress'
      else status = 'pending'
      tasks.push({ id, content: content.trim(), status })
    }
  }
  return tasks.length > 0 ? tasks : null
}

export const extractTodos = (payload: any): HolmesTaskItem[] | undefined => {
  const candidates = [
    payload?.todos,
    payload?.params?.todos,
    payload?.data?.todos,
    payload?.result?.params?.todos,
    payload?.result?.data ? parseTasksFromStatusText(payload.result.data) : null
  ].filter(item => item && (Array.isArray(item) || typeof item === 'string'))
  if (!candidates.length) return undefined
  const seen = new Map<string, HolmesTaskItem>()
  candidates.forEach(candidate => {
    if (Array.isArray(candidate)) {
      candidate.forEach((item: any, index: number) => {
        if (!item) return
        const content = typeof item.content === 'string' ? item.content.trim() : undefined
        if (!content) return
        const id = String(item.id ?? index)
        const note = typeof item.note === 'string' ? item.note : undefined
        seen.set(id || content, { id, content, status: normalizeTaskStatus(item.status), note })
      })
    } else if (typeof candidate === 'string') {
      const tasks = parseTasksFromStatusText(candidate)
      if (tasks) tasks.forEach(task => { seen.set(task.id, task) })
    }
  })
  return Array.from(seen.values())
}

export const deriveStatusText = (payload: any): string | undefined => {
  const direct = [payload?.status_text, payload?.status, payload?.message]
    .find((value) => typeof value === 'string' && value.trim())
  if (direct) return (direct as string).trim()
  const resultText = typeof payload?.result?.data === 'string' ? payload.result.data : undefined
  if (!resultText) return undefined
  const taskLine = resultText.split('\n').find((line: string) => line.includes('Task Status'))
  return (taskLine || resultText.split('\n')[0]).trim()
}

export const formatSectionsToMarkdown = (sections: Record<string, any>): string | undefined => {
  if (!sections || typeof sections !== 'object') return undefined
  const blocks: string[] = []
  Object.entries(sections).forEach(([rawTitle, rawContent]) => {
    if (rawContent === undefined || rawContent === null) return
    const title = typeof rawTitle === 'string' ? formatSummaryText(rawTitle) : String(rawTitle)
    const content = typeof rawContent === 'string'
      ? formatSummaryText(rawContent)
      : formatSummaryText(JSON.stringify(rawContent, null, 2))
    const normalizedContent = content.trim()
    if (!normalizedContent) return
    blocks.push(`### ${title}\n${normalizedContent}`)
  })
  return blocks.length ? blocks.join('\n\n') : undefined
}

export const looksLikeStructuredSummary = (text: string): boolean => {
  const normalized = normalizePlainText(text)
  if (!normalized) return false
  if (normalized.startsWith('#')) return true
  if (/(问题描述|根本原因|解决方案|预防建议)/.test(normalized)) return true
  const lines = normalized.split('\n')
  if (lines.length >= 4 && normalized.length >= 120) return true
  return false
}

export const extractHolmesStructuredDataFromObject = (payload: any): HolmesStructuredData | undefined => {
  if (!payload || typeof payload !== 'object') return undefined
  const planTextCandidates = [payload.content, payload.result?.message]
    .filter((value) => typeof value === 'string' && value.trim()) as string[]
  let planText: string | undefined = planTextCandidates[0]?.trim()
  const toolName = typeof payload.tool_name === 'string'
    ? payload.tool_name
    : typeof payload.name === 'string'
      ? payload.name
      : undefined
  const tasks = extractTodos(payload)
  const statusText = deriveStatusText(payload)
  let summary: string | undefined
  let progressText: string | undefined
  const summaryParts: string[] = []
  const analysisContent = typeof payload.analysis === 'string' ? payload.analysis.trim() : undefined
  let sectionsSummary: string | undefined
  if (payload.sections && typeof payload.sections === 'object') {
    try { sectionsSummary = formatSectionsToMarkdown(payload.sections) } catch {}
  }
  const isFinalReport = payload.content && typeof payload.content === 'string' &&
    (payload.content.includes('# 问题分析报告') ||
     payload.content.includes('## 问题描述') ||
     payload.content.includes('## 根本原因分析') ||
     payload.content.includes('## 解决方案'))
  if (isFinalReport) {
    if (typeof payload.content === 'string' && payload.content.trim()) {
      summaryParts.push(payload.content.trim())
    }
    planText = undefined
  } else {
    if (sectionsSummary) summaryParts.push(sectionsSummary)
    if ((!planText || /^write\s*\[/i.test(planText)) && tasks && tasks.length) {
      planText = 'HolmesGPT 已生成调查任务清单，以下为建议的调查步骤。'
    }
    if (!planText && typeof payload.content === 'string') {
      planText = payload.content.trim()
    }
    if ((!tasks || tasks.length === 0) && planText && !isFinalReport) {
      progressText = planText
      planText = undefined
    }
  }
  if (analysisContent) {
    const normalizedAnalysis = analysisContent.trim()
    const canonicalAnalysis = canonicalizeSummaryFragment(normalizedAnalysis)
    const canonicalContent = typeof payload.content === 'string'
      ? canonicalizeSummaryFragment(payload.content)
      : null
    if (!canonicalAnalysis || !canonicalContent || canonicalAnalysis !== canonicalContent) {
      summaryParts.push(analysisContent)
    }
  }
  if (summaryParts.length) {
    summary = deduplicateSummaryBlocks(summaryParts.join('\n\n'))
  }
  const commands = collectCommands(payload)
  const hasInfo = Boolean(planText || progressText || (tasks && tasks.length) || commands.length || toolName || statusText || summary)
  if (!hasInfo) return undefined
  const canonicalToolNameRaw = typeof payload.tool_name === 'string'
    ? payload.tool_name
    : typeof payload.name === 'string'
      ? payload.name
      : undefined
  const canonicalToolName = canonicalToolNameRaw?.trim()
  const isTodoWrite = canonicalToolName ? canonicalToolName.toLowerCase().includes('todo') : false
  const sectionId = (() => {
    if (isTodoWrite) return 'todo-write-main'
    if (payload.tool_call_id) return String(payload.tool_call_id)
    if (payload.id) return String(payload.id)
    if (payload.call_id) return String(payload.call_id)
    if (canonicalToolName) return `section-${canonicalToolName.toLowerCase().replace(/[^a-z0-9]/g, '-')}`
    return 'section-default'
  })()
  const statusSummary = extractTaskStatus(payload.result?.data)
  const sectionTitle = (() => {
    if (isTodoWrite) return statusSummary ? `任务更新 · ${statusSummary}` : '任务更新'
    if (canonicalToolName) return `${canonicalToolName} 调用`
    return 'HolmesGPT 调用'
  })()
  const result: HolmesStructuredData = {
    planText,
    tasks,
    toolName,
    statusText: statusSummary || statusText || undefined,
    progressText,
    summary,
    taskSections: ((tasks && tasks.length) || commands.length) ? [{
      id: sectionId,
      title: sectionTitle,
      toolName: toolName || 'HolmesGPT',
      tasks: tasks || [],
      statusText: statusSummary || statusText || undefined,
      commands: commands.length ? commands : undefined,
    }] : undefined,
    raw: payload
  }
  return result
}

export const extractHolmesStructuredData = (payload: any): HolmesStructuredData | undefined => {
  if (!payload) return undefined
  if (typeof payload === 'string') {
    const objects = parseJsonObjectSequence(payload)
    if (!objects.length) return { summary: payload }
    return objects.reduce<HolmesStructuredData | undefined>((acc, item) => {
      const structured = extractHolmesStructuredData(item)
      return mergeHolmesStructuredData(acc, structured)
    }, undefined)
  }
  if (Array.isArray(payload)) {
    return payload.reduce<HolmesStructuredData | undefined>((acc, item) => {
      const structured = extractHolmesStructuredData(item)
      return mergeHolmesStructuredData(acc, structured)
    }, undefined)
  }
  return extractHolmesStructuredDataFromObject(payload)
}

export const mergeHolmesStructuredData = (
  previous?: HolmesStructuredData,
  next?: HolmesStructuredData
): HolmesStructuredData | undefined => {
  if (!previous && !next) return undefined
  if (!previous) return next
  if (!next) return previous
  if (next.summary && !next.planText && !next.tasks && !next.taskSections && !next.toolName) {
    const newSummary = appendTextChunk(previous.summary || '', next.summary)
    return { ...previous, summary: newSummary }
  }
  if (next.progressText && !next.planText && !next.tasks && !next.taskSections && !next.toolName && !next.summary) {
    const newProgress = appendTextChunk((previous as any).progressText || '', next.progressText)
    return { ...previous, progressText: newProgress } as HolmesStructuredData
  }
  const merged: HolmesStructuredData = {
    planText: next.planText || previous.planText,
    toolName: next.toolName || previous.toolName,
    statusText: next.statusText || previous.statusText,
    raw: undefined,
    tasks: []
  }
  const collectRaw = (value?: any) => {
    if (value === undefined) return []
    return Array.isArray(value) ? value : [value]
  }
  const rawCombined = [...collectRaw(previous.raw), ...collectRaw(next.raw)]
  if (rawCombined.length) merged.raw = rawCombined
  const order: string[] = []
  const taskMap = new Map<string, HolmesTaskItem>()
  const registerTask = (task?: HolmesTaskItem) => {
    if (!task) return
    const key = task.id || task.content
    if (!taskMap.has(key)) order.push(key)
    taskMap.set(key, {
      ...task,
      content: normalizePlainText(task.content),
      note: task.note ? normalizePlainText(task.note) : task.note
    })
  }
  previous.tasks?.forEach(registerTask)
  next.tasks?.forEach(registerTask)
  merged.tasks = order.map(key => taskMap.get(key)!).filter(Boolean)
  let summaryText = previous.summary || ''
  if (next.summary) summaryText = appendTextChunk(summaryText, next.summary)
  if (summaryText.trim()) merged.summary = deduplicateSummaryBlocks(summaryText)
  const prevProgress = (previous as any).progressText || ''
  const incomingProgress = next.progressText || ''
  const combinedProgress = incomingProgress ? appendTextChunk(prevProgress, incomingProgress) : prevProgress
  if (combinedProgress && combinedProgress.trim()) (merged as any).progressText = combinedProgress
  const sectionMap = new Map<string, HolmesTaskSection>()
  const sectionOrder: string[] = []
  const registerSection = (section?: HolmesTaskSection) => {
    if (!section) return
    if (!sectionMap.has(section.id)) {
      sectionOrder.push(section.id)
      sectionMap.set(section.id, section)
    } else {
      const existing = sectionMap.get(section.id)!
      const mergedSection: HolmesTaskSection = {
        id: section.id,
        title: section.title || existing.title,
        statusText: section.statusText || existing.statusText,
        toolName: section.toolName || existing.toolName,
        tasks: [],
        commands: appendCommands(existing.commands, section.commands),
      }
      const tMap = new Map<string, HolmesTaskItem>()
      const tOrder: string[] = []
      const addTask = (task: any) => {
        if (!task) return
        const key = task.id || task.content
        if (!tMap.has(key)) tOrder.push(key)
        tMap.set(key, task)
      }
      existing.tasks?.forEach(addTask)
      section.tasks?.forEach(addTask)
      mergedSection.tasks = tOrder.map(key => tMap.get(key)!).filter(Boolean)
      sectionMap.set(section.id, mergedSection)
    }
  }
  previous.taskSections?.forEach(registerSection)
  next.taskSections?.forEach(section => {
    if (!section) return
    if (!section.tasks?.length) return
    if (section.id === 'todo-write-main' && sectionMap.has('todo-write-main')) {
      registerSection(section)
      return
    }
    registerSection(section)
  })
  const sections = sectionOrder.map(id => sectionMap.get(id)!).filter(Boolean)
  if (sections.length > 0) merged.taskSections = sections
  if (!merged.tasks?.length) delete merged.tasks
  if (merged.planText) merged.planText = normalizePlainText(merged.planText)
  if ((merged as any).progressText) (merged as any).progressText = normalizePlainText((merged as any).progressText)
  if (merged.statusText) merged.statusText = normalizePlainText(merged.statusText)
  if (merged.summary) merged.summary = normalizePlainText(merged.summary)
  return merged
}

export const formatTaskListMarkdown = (tasks: HolmesTaskItem[], indent = ''): string => {
  if (!tasks?.length) return ''
  return tasks
    .map(task => {
      const checkbox = task.status === 'completed' ? '[x]' : task.status === 'in_progress' ? '[-]' : '[ ]'
      const lines = [`${indent}- ${checkbox} ${task.content}`]
      if (task.note) lines.push(`${indent}  > 备注：${task.note}`)
      return lines.join('\n')
    })
    .join('\n')
}

export const formatCommandsMarkdown = (commands?: string[]): string => {
  if (!commands || commands.length === 0) return ''
  const blocks: string[] = []
  commands.forEach(cmd => {
    const normalized = (cmd || '').replace(/\r\n/g, '\n').trim()
    if (!normalized) return
    blocks.push('```bash')
    blocks.push(normalized)
    blocks.push('```')
  })
  return blocks.join('\n\n')
}

export const structuredDataToMarkdown = (
  data?: HolmesStructuredData,
  options?: { headingLevel?: number; summaryTitle?: string }
): string | undefined => {
  if (!data) return undefined
  const { headingLevel = 3, summaryTitle = '分析结论' } = options || {}
  const sections: string[] = []
  const heading = (title: string, levelOffset = 0) => {
    const level = Math.min(6, headingLevel + levelOffset)
    return `${'#'.repeat(level)} ${title}`
  }
  if (data.planText) sections.push(`${heading('分析计划')}\n${data.planText}`)
  if (data.progressText) sections.push(`${heading('排查进度')}\n${data.progressText}`)
  if (data.tasks?.length) {
    const tasksMarkdown = formatTaskListMarkdown(data.tasks)
    if (tasksMarkdown) sections.push(`${heading('任务列表')}\n${tasksMarkdown}`)
  }
  if (data.taskSections?.length) {
    const sectionLines: string[] = []
    data.taskSections.forEach(section => {
      if (!section || !section.tasks?.length) return
      sectionLines.push(`${heading(section.title || section.toolName || '任务', 1)}`)
      if (section.statusText) sectionLines.push(`> 状态：${section.statusText}`)
      const tasksMarkdown = formatTaskListMarkdown(section.tasks, '  ')
      if (tasksMarkdown) sectionLines.push(tasksMarkdown)
      if (section.commands?.length) {
        const commandsMarkdown = formatCommandsMarkdown(section.commands)
        if (commandsMarkdown) {
          sectionLines.push(`${heading('执行命令', 2)}`)
          sectionLines.push(commandsMarkdown)
        }
      }
    })
    if (sectionLines.length) sections.push(sectionLines.join('\n'))
  }
  if (data.summary) {
    const formatted = formatSummaryText(data.summary)
    if (formatted.trim()) sections.push(`${heading(summaryTitle)}\n${formatted}`)
  }
  return sections.length ? sections.join('\n\n') : undefined
}

export const extractTaskStatus = (text?: string): string | undefined => {
  if (typeof text !== 'string') return undefined
  const match = text.match(/Task Status\*\*:?:\s*(\d+)\s*completed,\s*(\d+)\s*in progress,\s*(\d+)\s*pending/i)
  if (!match) return undefined
  const [, completed, inProgress, pending] = match
  return `完成 ${completed} · 进行中 ${inProgress} · 待处理 ${pending}`
}

export const parseProgress = (structuredData?: HolmesStructuredData) => {
  if (!structuredData) return { total: 0, completed: 0 }
  const allTasks = [
    ...(structuredData.tasks || []),
    ...(structuredData.taskSections?.flatMap(section => section.tasks) || [])
  ]
  const completed = allTasks.filter(task => task.status === 'completed').length
  const total = allTasks.length
  return { total, completed }
}

export function sanitizeStructuredData(data?: HolmesStructuredData): HolmesStructuredData | undefined {
  if (!data) return data
  let changed = false
  const next: HolmesStructuredData = { ...data }
  const hadSections = Array.isArray(next.taskSections) && next.taskSections.length > 0
  const filtered = hadSections ? next.taskSections!.filter(s => s.id !== 'todo-write-main') : []
  const removedTodo = hadSections && filtered.length !== next.taskSections!.length
  if (removedTodo) { next.taskSections = filtered; changed = true }
  const isTodoLike = removedTodo || (typeof next.toolName === 'string' && next.toolName.toLowerCase().includes('todo'))
  if (isTodoLike) {
    if (next.planText) { next.planText = undefined as any; changed = true }
    if (next.statusText) { next.statusText = undefined as any; changed = true }
    if (next.toolName) { next.toolName = undefined as any; changed = true }
    if (next.tasks && next.tasks.length) { delete (next as any).tasks; changed = true }
  }
  if ((next as any).summary) { delete (next as any).summary; changed = true }
  const hasRenderable = Boolean(
    next.planText ||
    (next as any).progressText ||
    (next.taskSections && next.taskSections.length)
  )
  if (!hasRenderable) return undefined
  return changed ? next : data
}
