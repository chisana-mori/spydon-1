import { Alert } from '@/types/api'

export interface EnhancedHolmesGPTChatProps {
  alert: Alert
  showCard?: boolean
  cachedResult?: any
  loadingCache?: boolean
}

export interface AnalysisMessage {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  timestamp: string
  isStreaming?: boolean
  toolCalls?: Array<{
    name: string
    input: any
    output?: any
    status: 'pending' | 'success' | 'error'
    command?: string
  }>
  structuredData?: any
}

export interface AnalysisState {
  status: 'idle' | 'analyzing' | 'completed' | 'error'
  messages: AnalysisMessage[]
  error?: string
  totalSteps?: number
  completedSteps?: number
}

export interface StreamProcessor {
  appendChunk: (chunk: string) => boolean
  finalize: (status?: AnalysisState['status']) => void
  processAnalysisData: (payload: any, eventType?: string) => void
  processDataItem: (payload: any, eventType?: string) => void
}

export interface ChatSettings {
  autoScroll: boolean
  soundEnabled: boolean
  showToolCalls: boolean
  language: 'zh-CN' | 'en-US'
}
