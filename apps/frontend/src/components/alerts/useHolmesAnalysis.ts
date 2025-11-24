import { useReducer, useRef, useCallback, useState } from 'react'
import { AnalysisState, AnalysisMessage, ApprovalRequest, ApprovalDecision, StreamProcessor } from './holmesTypes'
import { HolmesStructuredData } from './ChatMessage'
import {
    parseProgress,
    normalizePlainText,
    appendTextChunk,
    deduplicateSummaryBlocks,
    buildSummarySignature,
    extractHolmesStructuredData,
    mergeHolmesStructuredData,
    looksLikeStructuredSummary,
    sanitizeStructuredData
} from './holmesUtils'
import { buildCachedSSEChunks, createCachedChunkIterable, consumeSSEChunks } from './sseHelpers'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { appConfig } from '@/config'

// --- Reducer Types & Logic ---

type Action =
    | { type: 'RESET' }
    | { type: 'SET_STATUS'; payload: AnalysisState['status'] }
    | { type: 'SET_ERROR'; payload: string }
    | { type: 'ADD_MESSAGE'; payload: AnalysisMessage }
    | { type: 'UPDATE_LAST_MESSAGE'; payload: Partial<AnalysisMessage> }
    | { type: 'UPDATE_MESSAGE_BY_ID'; payload: { id: string; updates: Partial<AnalysisMessage> } }
    | { type: 'SET_TOTAL_STEPS'; payload: number }
    | { type: 'SET_COMPLETED_STEPS'; payload: number }

const initialState: AnalysisState = {
    status: 'idle',
    messages: [],
    totalSteps: 0,
    completedSteps: 0,
    error: undefined
}

function analysisReducer(state: AnalysisState, action: Action): AnalysisState {
    switch (action.type) {
        case 'RESET':
            return initialState
        case 'SET_STATUS':
            return { ...state, status: action.payload }
        case 'SET_ERROR':
            return { ...state, error: action.payload, status: 'error' }
        case 'ADD_MESSAGE':
            return { ...state, messages: [...state.messages, action.payload] }
        case 'UPDATE_LAST_MESSAGE': {
            if (state.messages.length === 0) return state
            const lastMsg = state.messages[state.messages.length - 1]
            const updatedMsg = { ...lastMsg, ...action.payload }
            return {
                ...state,
                messages: [...state.messages.slice(0, -1), updatedMsg]
            }
        }
        case 'UPDATE_MESSAGE_BY_ID': {
            return {
                ...state,
                messages: state.messages.map(msg =>
                    msg.id === action.payload.id ? { ...msg, ...action.payload.updates } : msg
                )
            }
        }
        case 'SET_TOTAL_STEPS':
            return { ...state, totalSteps: action.payload }
        case 'SET_COMPLETED_STEPS':
            return { ...state, completedSteps: action.payload }
        default:
            return state
    }
}

// --- Hook Definition ---

export const useHolmesAnalysis = (
    alertId: string,
    options: {
        onPlaySound?: () => void
    } = {}
) => {
    const [state, dispatch] = useReducer(analysisReducer, initialState)

    // Additional state managed outside reducer for specific UI concerns
    const [pinnedTasksData, setPinnedTasksData] = useState<HolmesStructuredData | undefined>(undefined)
    const [pinnedSummaryData, setPinnedSummaryData] = useState<HolmesStructuredData | undefined>(undefined)
    const [connectionId, setConnectionId] = useState<string | null>(null)
    const [pendingApproval, setPendingApproval] = useState<ApprovalRequest | null>(null)
    const [isSubmittingApproval, setIsSubmittingApproval] = useState(false)
    const [isReplayingCache, setIsReplayingCache] = useState(false)

    // Refs for mutable state during streaming
    const wsRef = useRef<WebSocket | null>(null)
    const abortControllerRef = useRef<AbortController | null>(null)
    const allTasksCompletedRef = useRef(false)
    const lastSummarySignatureRef = useRef<string | null>(null)

    // Helper to generate unique IDs
    const messageIdCounter = useRef(0)
    const generateUniqueId = useCallback((prefix: string = 'msg') => {
        messageIdCounter.current++
        return `${prefix}-${Date.now()}-${messageIdCounter.current}-${Math.random().toString(36).slice(2)}`
    }, [])

    const resetState = useCallback(() => {
        dispatch({ type: 'RESET' })
        setPinnedTasksData(undefined)
        setPinnedSummaryData(undefined)
        setConnectionId(null)
        setPendingApproval(null)
        setIsSubmittingApproval(false)
        allTasksCompletedRef.current = false
        lastSummarySignatureRef.current = null
    }, [])

    // --- Stream Processing Logic ---

    const createStreamProcessor = useCallback((timestampFactory: () => string): StreamProcessor => {
        let buffer = ''
        let completed = false
        let lastAiAnswerSignature: string | null = null
        const approvalBuffer: Record<string, Partial<ApprovalRequest>> = {}

        // We need a mutable reference to the current message being built to avoid
        // dispatching on every single character for the same message object reference
        // However, since we use reducer, we dispatch updates.
        // To optimize, we could buffer updates, but for now let's trust React 18 batching.
        // Actually, for the "currentAssistantMessage" logic in the original code,
        // it was keeping a local variable. We can do similar.

        let currentMessageId: string | null = null
        let currentMessageContent = ''
        let currentMessageStructured: any = undefined

        const ensureAssistantMessage = () => {
            if (!currentMessageId) {
                const id = generateUniqueId('assistant')
                currentMessageId = id
                currentMessageContent = ''
                currentMessageStructured = undefined

                dispatch({
                    type: 'ADD_MESSAGE',
                    payload: {
                        id,
                        role: 'assistant',
                        content: '',
                        timestamp: timestampFactory(),
                        isStreaming: true,
                        toolCalls: []
                    }
                })
            }
        }

        const updateCurrentMessage = (updates: Partial<AnalysisMessage>) => {
            if (!currentMessageId) ensureAssistantMessage()

            if (updates.content !== undefined) currentMessageContent = updates.content
            if (updates.structuredData !== undefined) currentMessageStructured = updates.structuredData

            dispatch({
                type: 'UPDATE_LAST_MESSAGE',
                payload: updates
            })

            // Update progress if structured data changed
            if (updates.structuredData) {
                const progress = parseProgress(updates.structuredData)
                dispatch({ type: 'SET_TOTAL_STEPS', payload: progress.total })
                dispatch({ type: 'SET_COMPLETED_STEPS', payload: progress.completed })
            }
        }

        const mergePinnedTasks = (structured?: HolmesStructuredData) => {
            if (!structured) return

            setPinnedTasksData(prev => {
                const merged = mergeHolmesStructuredData(prev, structured)

                if (merged) {
                    const progress = parseProgress(merged)
                    dispatch({ type: 'SET_TOTAL_STEPS', payload: progress.total })
                    dispatch({ type: 'SET_COMPLETED_STEPS', payload: progress.completed })

                    const allTasks = merged.tasks || merged.taskSections?.flatMap(section => section.tasks) || []
                    if (allTasks.length > 0 && allTasks.every(task => task.status === 'completed')) {
                        allTasksCompletedRef.current = true
                    }
                }
                return merged
            })
        }

        const appendSummary = (summary: string, extraStructured?: HolmesStructuredData) => {
            if (!summary) return
            const normalized = normalizePlainText(summary)
            if (!normalized) return
            const deduped = deduplicateSummaryBlocks(normalized)
            if (!deduped.trim()) return

            const signature = buildSummarySignature(deduped)
            if (signature && signature === lastSummarySignatureRef.current) return
            if (signature) lastSummarySignatureRef.current = signature

            setPinnedSummaryData(prev => {
                const merged = mergeHolmesStructuredData(
                    prev,
                    extraStructured ? { ...extraStructured, summary: deduped } : { summary: deduped }
                )
                if (merged?.summary) merged.summary = deduplicateSummaryBlocks(merged.summary)
                return merged
            })
        }

        const removeSummaryFromStructuredData = (data?: HolmesStructuredData): HolmesStructuredData | undefined => {
            if (!data || data.summary === undefined) return data
            const cloned: HolmesStructuredData = { ...data }
            delete (cloned as any).summary
            return cloned
        }

        const processDataItem = (item: any, eventType?: string) => {
            if (!item) return

            // 1. Handle ai_message (intermediate thoughts)
            if (eventType === 'ai_message') {
                const content = item.content || item.data?.content
                if (content && typeof content === 'string') {
                    const normalized = normalizePlainText(content)
                    if (normalized) {
                        if (looksLikeStructuredSummary(normalized)) {
                            appendSummary(normalized)
                            return
                        }

                        const mergedStructured = mergeHolmesStructuredData(currentMessageStructured, { progressText: normalized })
                        updateCurrentMessage({
                            structuredData: mergedStructured,
                            content: appendTextChunk(currentMessageContent, normalized),
                            isStreaming: true
                        })
                    }
                }
                return
            }

            // 2. Handle ai_answer_end (final conclusion)
            if (eventType === 'ai_answer_end') {
                const payload = item && typeof item === 'object' && 'data' in item ? item.data : item
                let combinedStructured: HolmesStructuredData | undefined
                const structuredFromPayload = extractHolmesStructuredData(payload)
                combinedStructured = mergeHolmesStructuredData(combinedStructured, structuredFromPayload)

                let summaryText = ''
                if (structuredFromPayload?.summary) {
                    summaryText = appendTextChunk(summaryText, structuredFromPayload.summary)
                }

                const additionalSummaries: string[] = []
                if (typeof payload === 'string') additionalSummaries.push(payload)
                else if (payload && typeof payload === 'object') {
                    if (typeof payload.analysis === 'string' && payload.analysis.trim()) additionalSummaries.push(payload.analysis.trim())
                    if (typeof payload.content === 'string' && payload.content.trim()) additionalSummaries.push(payload.content.trim())
                }
                if (typeof item === 'string') additionalSummaries.push(item)

                additionalSummaries.forEach(c => summaryText = appendTextChunk(summaryText, c))

                const deduped = deduplicateSummaryBlocks(summaryText)
                const signature = deduped ? buildSummarySignature(deduped) : null

                if (signature && signature === lastAiAnswerSignature) return
                if (signature) lastAiAnswerSignature = signature

                if (deduped.trim()) appendSummary(deduped)

                if (combinedStructured) {
                    const structuredWithoutSummary = removeSummaryFromStructuredData(combinedStructured)
                    const mergedForMessage = removeSummaryFromStructuredData(
                        mergeHolmesStructuredData(currentMessageStructured, structuredWithoutSummary)
                    )
                    updateCurrentMessage({
                        structuredData: mergedForMessage,
                        isStreaming: true
                    })
                }
                return
            }

            // 3. Handle Tool Calls
            if (item.role === 'tool' && item.tool_call_id) {
                if (item.name === 'TodoWrite') {
                    const structured = extractHolmesStructuredData(item)
                    mergePinnedTasks(structured)
                    return
                }

                const toolCall = {
                    name: item.name,
                    input: item.description,
                    output: item.result,
                    status: (item.result?.status === 'success' ? 'success' : item.result?.status === 'error' ? 'error' : 'pending') as 'pending' | 'success' | 'error',
                    command: item.result?.invocation || item.description || ''
                }

                // If current message has content, finish it and start a new one for tool call
                // Or just append tool call to current message?
                // The original logic created a NEW message for the tool call if the current one had content.

                if (currentMessageContent.trim()) {
                    updateCurrentMessage({ isStreaming: false })
                    currentMessageId = null // Force new message creation
                }

                // Create tool message
                const toolMsgId = `tool-${item.tool_call_id || Date.now()}`
                dispatch({
                    type: 'ADD_MESSAGE',
                    payload: {
                        id: toolMsgId,
                        role: 'assistant',
                        content: '',
                        timestamp: timestampFactory(),
                        isStreaming: false,
                        toolCalls: [toolCall]
                    }
                })

                // If we started a new message for this tool, we should probably start a fresh assistant message for subsequent text
                currentMessageId = null
                return
            }

            // 4. Handle standard content
            if (item.content && typeof item.content === 'string') {
                const textContent = normalizePlainText(item.content)
                if (!textContent) return

                if (allTasksCompletedRef.current) {
                    appendSummary(textContent)
                    return
                }

                const structured = extractHolmesStructuredData(item)
                const structuredForMessage = removeSummaryFromStructuredData(structured)

                if (structured?.summary) appendSummary(structured.summary)

                const mergedStructured = mergeHolmesStructuredData(currentMessageStructured, structuredForMessage)
                const shouldAddToContent = !structured?.summary

                updateCurrentMessage({
                    structuredData: mergedStructured,
                    content: shouldAddToContent ? appendTextChunk(currentMessageContent, textContent) : currentMessageContent,
                    isStreaming: true
                })
                return
            }

            // 5. Handle analysis objects
            if (item.type === 'analysis' || item.type === 'analysis_chunk') {
                const payload = item.data ?? item.content
                if (payload && typeof payload === 'object') {
                    const structured = extractHolmesStructuredData(payload)
                    const hasTodo = structured?.taskSections?.some(section => section.id === 'todo-write-main')
                    if (hasTodo) {
                        mergePinnedTasks(structured)
                        return
                    }

                    const structuredForMessage = removeSummaryFromStructuredData(structured)
                    if (structured?.summary) appendSummary(structured.summary)

                    const mergedStructured = mergeHolmesStructuredData(currentMessageStructured, structuredForMessage)
                    updateCurrentMessage({
                        structuredData: mergedStructured,
                        isStreaming: true
                    })
                    return
                }

                if (typeof payload === 'string') {
                    const normalized = normalizePlainText(payload)
                    if (!normalized) return

                    if (looksLikeStructuredSummary(normalized)) {
                        appendSummary(normalized)
                        return
                    }

                    if (allTasksCompletedRef.current) {
                        appendSummary(normalized)
                    } else {
                        const mergedStructured = mergeHolmesStructuredData(currentMessageStructured, { progressText: normalized })
                        updateCurrentMessage({
                            structuredData: mergedStructured,
                            content: appendTextChunk(currentMessageContent, normalized),
                            isStreaming: true
                        })
                    }
                }
            }
        }

        const processAnalysisData = (analysisData: any, eventType?: string) => {
            if (typeof analysisData === 'string') {
                const trimmed = analysisData.trim()

                if (eventType === 'ai_answer_end') {
                    if (trimmed) {
                        const normalized = normalizePlainText(trimmed)
                        allTasksCompletedRef.current = true
                        appendSummary(normalized)

                        // Clear summary from current message to avoid duplication
                        if (currentMessageId) {
                            const withoutSummary = removeSummaryFromStructuredData(currentMessageStructured)
                            updateCurrentMessage({
                                structuredData: withoutSummary,
                                content: '',
                                isStreaming: true
                            })
                        }
                    }
                    return
                }

                if (trimmed.startsWith('{') && trimmed.endsWith('}')) {
                    try {
                        const parsed = JSON.parse(trimmed)
                        if (parsed && typeof parsed === 'object') {
                            processDataItem(parsed, 'ai_answer_end')
                            return
                        }
                    } catch (e) { }
                }

                if (eventType !== 'ai_message' && looksLikeStructuredSummary(trimmed)) {
                    appendSummary(trimmed)
                    return
                }

                if (allTasksCompletedRef.current && eventType !== 'ai_message') {
                    appendSummary(trimmed)
                } else {
                    const mergedStructured = mergeHolmesStructuredData(currentMessageStructured, { progressText: trimmed })
                    updateCurrentMessage({
                        structuredData: mergedStructured,
                        content: appendTextChunk(currentMessageContent, trimmed),
                        isStreaming: true
                    })
                }
                return
            }

            if (Array.isArray(analysisData)) {
                analysisData.forEach(item => processDataItem(item, eventType))
                return
            }

            if (analysisData && typeof analysisData === 'object') {
                processDataItem(analysisData, eventType)
            }
        }

        const markCompleted = () => {
            if (completed) return
            if (currentMessageId) {
                updateCurrentMessage({ isStreaming: false })
            }
            dispatch({ type: 'SET_STATUS', payload: 'completed' })
            completed = true
        }

        let currentEventType: string | null = null

        const appendChunk = (chunk: string): boolean => {
            buffer += chunk
            const lines = buffer.split('\n')
            buffer = lines.pop() || ''
            let stop = false

            for (const line of lines) {
                if (line === '') {
                    currentEventType = null
                    continue
                }
                if (line.startsWith('event: ')) {
                    currentEventType = line.slice(7).trim()
                    continue
                }
                if (!line.startsWith('data: ')) continue

                const rawData = line.slice(6)
                if (!rawData || rawData === '[DONE]') continue

                if (rawData === '{"type":"complete","data":{}}') {
                    markCompleted()
                    stop = true
                    continue
                }

                try {
                    const data = JSON.parse(rawData)

                    if (currentEventType === 'connection_ack' && data.connectionId) {
                        setConnectionId(data.connectionId)
                        continue
                    }

                    if (currentEventType === 'approval_required') {
                        const toolCallId = data.tool_call_id
                        if (toolCallId) {
                            if (!approvalBuffer[toolCallId]) approvalBuffer[toolCallId] = {}

                            if (data.request_id === null || data.request_id === undefined) {
                                if (data.description) approvalBuffer[toolCallId].description = data.description
                            } else {
                                approvalBuffer[toolCallId].requestId = data.request_id
                                approvalBuffer[toolCallId].reason = data.reason
                            }

                            const buffered = approvalBuffer[toolCallId]
                            if (buffered.requestId !== undefined && buffered.requestId !== null && buffered.description) {
                                setPendingApproval({
                                    requestId: buffered.requestId,
                                    toolCallId: toolCallId,
                                    description: buffered.description,
                                    reason: buffered.reason || data.reason || ''
                                })
                                delete approvalBuffer[toolCallId]
                            }
                        }
                        continue
                    }

                    const effectiveEventType = currentEventType || data.type
                    if (data.type === 'analysis' && data.data !== undefined) {
                        processAnalysisData(data.data, effectiveEventType)
                    } else if (Array.isArray(data)) {
                        data.forEach(item => processDataItem(item, effectiveEventType))
                    } else {
                        processDataItem(data, effectiveEventType)
                    }
                } catch (e) {
                    console.error('JSON parse error:', e)
                }
            }
            return stop
        }

        const finalize = (status: AnalysisState['status'] = 'completed') => {
            if (status === 'completed') markCompleted()
            else {
                if (currentMessageId) updateCurrentMessage({ isStreaming: false })
                dispatch({ type: 'SET_STATUS', payload: status })
            }
            if (status === 'completed') completed = true
        }

        return { appendChunk, finalize, processAnalysisData, processDataItem }
    }, [generateUniqueId])

    // --- Actions ---

    const startAnalysis = useCallback(async (alert: any, settings: any) => {
        if (wsRef.current) {
            wsRef.current.close()
            wsRef.current = null
        }

        resetState()

        // Add user message
        dispatch({
            type: 'ADD_MESSAGE',
            payload: {
                id: `user-${Date.now()}`,
                role: 'user',
                content: `请分析告警：${alert.title}\n\n描述：${alert.description || '无描述'}`,
                timestamp: format(new Date(), 'HH:mm:ss', { locale: zhCN })
            }
        })

        dispatch({ type: 'SET_STATUS', payload: 'analyzing' })

        try {
            const backendUrl = appConfig.backendBaseUrl || 'http://localhost:8080'
            const wsUrl = backendUrl.replace(/^http/, 'ws')
            const apiPath = `${wsUrl}/api/v1/holmesgpt/stream/investigate`

            const ws = new WebSocket(apiPath)
            wsRef.current = ws

            const timestampFactory = () => format(new Date(), 'HH:mm:ss', { locale: zhCN })
            const processor = createStreamProcessor(timestampFactory)

            ws.onopen = () => {
                ws.send(JSON.stringify({
                    alert_id: alert.id,
                    depth: 'standard',
                    language: settings.language || 'zh-CN',
                    include_tool_calls: settings.showToolCalls,
                }))
            }

            ws.onmessage = (event) => {
                const stop = processor.appendChunk(event.data)
                if (stop) ws.close()
            }

            ws.onerror = (error) => {
                console.error('WebSocket error:', error)
                dispatch({ type: 'SET_ERROR', payload: '连接发生错误' })
            }

            ws.onclose = (event) => {
                if (event.code !== 1000 && event.code !== 1005) {
                    // Abnormal close
                }
                processor.finalize('completed')
                options.onPlaySound?.()
                wsRef.current = null
            }

        } catch (error: any) {
            dispatch({ type: 'SET_ERROR', payload: error.message || '分析启动失败' })
        }
    }, [resetState, createStreamProcessor, options])

    const stopAnalysis = useCallback(() => {
        if (wsRef.current) {
            wsRef.current.close()
            wsRef.current = null
        }
        if (abortControllerRef.current) {
            abortControllerRef.current.abort()
        }
    }, [])

    const replayCachedResult = useCallback(async (cached: any) => {
        setIsReplayingCache(true)
        resetState()
        dispatch({ type: 'SET_STATUS', payload: 'analyzing' })

        try {
            const chunks = buildCachedSSEChunks(cached)
            if (!chunks.length) {
                dispatch({ type: 'SET_ERROR', payload: '缓存中没有可回放的HolmesGPT数据' })
                return
            }

            const baseDate = cached?.cached_at ? new Date(cached.cached_at) : new Date()
            const timestampFactory = () => format(baseDate, 'HH:mm:ss', { locale: zhCN })
            const processor = createStreamProcessor(timestampFactory)

            await consumeSSEChunks(createCachedChunkIterable(chunks), processor)
            processor.finalize('completed')
            options.onPlaySound?.()
        } catch (error) {
            dispatch({ type: 'SET_ERROR', payload: '加载缓存结果失败' })
        } finally {
            setPendingApproval(null)
            setIsReplayingCache(false)
        }
    }, [resetState, createStreamProcessor, options])

    const sendApprovalDecision = useCallback(async (decision: ApprovalDecision) => {
        if (!connectionId || !pendingApproval) return

        setIsSubmittingApproval(true)
        try {
            const backendUrl = appConfig.backendBaseUrl || 'http://localhost:8080'
            const apiUrl = `${backendUrl}/api/v1/holmesgpt/stream/investigate/send`

            const requestBody = {
                connectionId: connectionId,
                id: parseInt(pendingApproval.requestId, 10) || 0,
                result: { decision }
            }

            const response = await fetch(apiUrl, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(requestBody),
            })

            if (!response.ok) throw new Error(`审批决策发送失败: ${response.status}`)
            setPendingApproval(null)
        } catch (error) {
            console.error('发送审批决策失败:', error)
        } finally {
            setIsSubmittingApproval(false)
        }
    }, [connectionId, pendingApproval])

    return {
        analysisState: state,
        pinnedTasksData,
        pinnedSummaryData,
        pendingApproval,
        isSubmittingApproval,
        isReplayingCache,
        startAnalysis,
        stopAnalysis,
        replayCachedResult,
        sendApprovalDecision
    }
}
