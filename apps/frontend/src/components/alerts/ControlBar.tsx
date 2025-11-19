import React from 'react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Settings, Play, Loader2, Square, RotateCcw, Download, Clock, Volume2, VolumeX } from 'lucide-react'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { ChatSettings, AnalysisState } from './holmesTypes'

interface ControlBarProps {
  statusIcon: React.ReactNode
  statusText: string
  progressText?: string | null
  showSettings: boolean
  toggleSettings: () => void
  settings: ChatSettings
  setSettings: React.Dispatch<React.SetStateAction<ChatSettings>>
  analysisStatus: AnalysisState['status']
  loadingCache: boolean
  cachedResult?: any
  isReplayingCache: boolean
  onStart: () => void
  onStop: () => void
  onReanalyze: () => void
  onDownload: () => void
}

export const ControlBar: React.FC<ControlBarProps> = ({
  statusIcon,
  statusText,
  progressText,
  showSettings,
  toggleSettings,
  settings,
  setSettings,
  analysisStatus,
  loadingCache,
  cachedResult,
  isReplayingCache,
  onStart,
  onStop,
  onReanalyze,
  onDownload
}) => {
  return (
    <div className="flex-shrink-0 border-b bg-white px-6 py-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          {statusIcon}
          <div>
            <div className="text-lg font-semibold flex items-center space-x-2">
              <span>{statusText}</span>
              {progressText && (
                <Badge variant="outline" className="text-xs">
                  {progressText}
                </Badge>
              )}
            </div>
            <p className="text-sm text-muted-foreground mt-1">基于 Kubernetes 专业知识的智能故障分析</p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          <Button variant="outline" size="sm" onClick={toggleSettings} className="h-8 w-8 p-0">
            <Settings className="h-4 w-4" />
          </Button>

          {cachedResult && analysisStatus === 'completed' && !isReplayingCache && (
            <Badge variant="outline" className="text-xs">
              <Clock className="h-3 w-3 mr-1" />
              {cachedResult?.cached_at ? `缓存结果 (${format(new Date(cachedResult.cached_at), 'MM-dd HH:mm', { locale: zhCN })})` : '缓存结果'}
            </Badge>
          )}

          {analysisStatus === 'idle' && !loadingCache && (
            <Button onClick={onStart} size="sm">
              <Play className="h-4 w-4 mr-2" />
              触发RCA分析
            </Button>
          )}
          {loadingCache && (
            <Button disabled size="sm">
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              加载缓存中...
            </Button>
          )}
          {analysisStatus === 'analyzing' && (
            <Button onClick={onStop} variant="outline" size="sm">
              <Square className="h-4 w-4 mr-2" />
              停止分析
            </Button>
          )}
          {(analysisStatus === 'completed' || analysisStatus === 'error') && (
            <>
              <Button onClick={onReanalyze} variant="default" size="sm" className="bg-black text-white hover:bg-black/90">
                <RotateCcw className="h-4 w-4 mr-2" />
                重新分析
              </Button>
              <Button onClick={onDownload} variant="outline" size="sm">
                <Download className="h-4 w-4 mr-2" />
                下载结果
              </Button>
            </>
          )}
        </div>
      </div>

      {showSettings && (
        <div className="mt-4 p-4 bg-gray-50 rounded-lg border">
          <h4 className="text-sm font-medium mb-3">聊天设置</h4>
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-sm">自动滚动</span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setSettings(prev => ({ ...prev, autoScroll: !prev.autoScroll }))}
                className={settings.autoScroll ? 'bg-blue-50 border-blue-200' : ''}
              >
                {settings.autoScroll ? '已启用' : '已禁用'}
              </Button>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm">完成提示音</span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setSettings(prev => ({ ...prev, soundEnabled: !prev.soundEnabled }))}
                className={settings.soundEnabled ? 'bg-blue-50 border-blue-200' : ''}
              >
                {settings.soundEnabled ? <Volume2 className="h-4 w-4" /> : <VolumeX className="h-4 w-4" />}
              </Button>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm">显示工具调用</span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setSettings(prev => ({ ...prev, showToolCalls: !prev.showToolCalls }))}
                className={settings.showToolCalls ? 'bg-blue-50 border-blue-200' : ''}
              >
                {settings.showToolCalls ? '已启用' : '已禁用'}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
