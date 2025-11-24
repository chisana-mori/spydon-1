import React from 'react'
import { ChatMessage, HolmesStructuredData } from './ChatMessage'
import { parseProgress, sanitizeStructuredData, buildSummarySignature } from './holmesUtils'
import { AnalysisState } from './holmesTypes'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { Button } from '@/components/ui/button'

interface ContentAreaProps {
  analysisState: AnalysisState
  scrollAreaRef: React.RefObject<HTMLDivElement>
  messagesEndRef: React.RefObject<HTMLDivElement>
  handleScroll: () => void
  showScrollToLatest: boolean
  handleScrollToLatest: () => void
  pinnedTasksData?: HolmesStructuredData
  pinnedSummaryData?: HolmesStructuredData
  tasksMessageId: string
  summaryMessageId: string
  onReanalyze: () => void
  isReplayingCache?: boolean
}

const ContentAreaComponent: React.FC<ContentAreaProps> = ({
  analysisState,
  scrollAreaRef,
  messagesEndRef,
  handleScroll,
  showScrollToLatest,
  handleScrollToLatest,
  pinnedTasksData,
  pinnedSummaryData,
  tasksMessageId,
  summaryMessageId,
  onReanalyze,
  isReplayingCache = false
}) => {
  return (
    <div className="flex-1 flex flex-col min-h-0">
      <div ref={scrollAreaRef} onScroll={handleScroll} className="flex-1 overflow-y-auto px-6 py-4 flex flex-col">
        {analysisState.messages.length === 0 ? (
          <div className="flex flex-col items-center justify-center flex-1 text-center p-8">
            <h3 className="text-lg font-medium text-gray-900 mb-2">准备开始智能分析</h3>
            <p className="text-sm text-gray-500 max-w-md leading-relaxed">
              点击触发RCA分析按钮，HolmesGPT 将为您分析告警的根本原因，并提供详细的解决方案和预防措施。
              分析过程包括任务规划、并行调查和结论总结。
            </p>
          </div>
        ) : (
          <div className="space-y-0">
            {(() => {
              const summarySignature = pinnedSummaryData?.summary ? buildSummarySignature(pinnedSummaryData.summary) : null
              const seenContentSignatures = new Set<string>()

              return analysisState.messages.map(message => {
                const sanitized = sanitizeStructuredData(message.structuredData)
                const contentSignature = message.content?.trim() ? buildSummarySignature(message.content) : null

                if (message.role === 'assistant') {
                  const hasToolCalls = Array.isArray(message.toolCalls) && message.toolCalls.length > 0
                  const hasStructured = Boolean(sanitized)
                  const hasContent = Boolean(message.content?.trim())
                  if (!hasToolCalls && !hasStructured && !hasContent) return null
                  if (contentSignature && summarySignature && contentSignature === summarySignature) {
                    if (!hasToolCalls && !hasStructured) return null
                  }
                  if (contentSignature && !hasStructured && !hasToolCalls) {
                    if (seenContentSignatures.has(contentSignature)) return null
                    seenContentSignatures.add(contentSignature)
                  }
                }

                let displayContent = message.content
                if (message.role === 'assistant' && contentSignature && summarySignature && contentSignature === summarySignature) {
                  displayContent = ''
                }

                return (
                  <ChatMessage
                    key={message.id}
                    role={message.role}
                    content={displayContent}
                    timestamp={message.timestamp}
                    isStreaming={message.isStreaming && !isReplayingCache}
                    toolCalls={message.toolCalls || []}
                    structuredData={sanitized}
                  />
                )
              })
            })()}

            {pinnedTasksData && (() => {
              const prog = parseProgress(pinnedTasksData)
              const pinnedIsStreaming = analysisState.status === 'analyzing' && (!pinnedSummaryData) && (prog.total === 0 || prog.completed < prog.total)
              return (
                <ChatMessage
                  key={tasksMessageId}
                  role="assistant"
                  content={''}
                  timestamp={format(new Date(), 'HH:mm:ss', { locale: zhCN })}
                  isStreaming={pinnedIsStreaming && !isReplayingCache}
                  toolCalls={[]}
                  structuredData={pinnedTasksData}
                />
              )
            })()}

            {pinnedSummaryData && (
              <ChatMessage
                key={summaryMessageId}
                role="assistant"
                content={''}
                timestamp={format(new Date(), 'HH:mm:ss', { locale: zhCN })}
                isStreaming={false}
                toolCalls={[]}
                structuredData={pinnedSummaryData}
              />
            )}

            {showScrollToLatest && (
              <div className="sticky bottom-4 flex justify-end">
                <Button size="sm" variant="secondary" className="shadow" onClick={handleScrollToLatest}>回到最新</Button>
              </div>
            )}
            <div ref={messagesEndRef} />
          </div>
        )}
      </div>
      {analysisState.status === 'error' && analysisState.error && (
        <div className="p-4 border-t bg-red-50">
          <div className="flex items-center space-x-2 text-red-600">
            <span className="text-sm font-medium">分析失败</span>
          </div>
          <p className="text-sm text-red-600 mt-1">{analysisState.error}</p>
          <Button onClick={onReanalyze} variant="default" size="sm" className="mt-2 bg-black text-white hover:bg-black/90">重试分析</Button>
        </div>
      )}
    </div>
  )
}

export const ContentArea = React.memo(ContentAreaComponent)
