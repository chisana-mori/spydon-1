'use client'

import React, { useState, useEffect } from 'react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { 
  Brain, 
  Settings, 
  Clock,
  ExternalLink
} from 'lucide-react'
import { Alert } from '@/types/api'
import { EnhancedHolmesGPTChat } from './EnhancedHolmesGPTChat'
import { RawPayloadViewer } from './RawPayloadViewer'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'

interface AlertAnalysisIntegrationProps {
  alert: Alert
  defaultTab?: 'enhanced' | 'raw'
}

export const AlertAnalysisIntegration: React.FC<AlertAnalysisIntegrationProps> = ({ 
  alert, 
  defaultTab = 'enhanced' 
}) => {
  const [activeTab, setActiveTab] = useState(defaultTab)
  const [analysisHistory, setAnalysisHistory] = useState<any[]>([])

  // 模拟获取分析历史记录
  useEffect(() => {
    // 这里可以调用 API 获取该告警的历史分析记录
    const mockHistory: any[] = []
    setAnalysisHistory(mockHistory)
  }, [alert.id])


  return (
    <div className="space-y-6">

      {/* 分析选项卡 */}
      <div className="min-h-[600px]">
        <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as typeof activeTab)} className="h-full">
          <TabsList className="grid w-full grid-cols-2 mb-4">
            <TabsTrigger value="enhanced" className="flex items-center space-x-2">
              <Brain className="h-4 w-4" />
              <span>增强分析</span>
            </TabsTrigger>
            <TabsTrigger value="raw" className="flex items-center space-x-2">
              <Settings className="h-4 w-4" />
              <span>原始数据</span>
            </TabsTrigger>
          </TabsList>

          <TabsContent value="enhanced" className="flex-1 min-h-0">
            <div className="h-full">
              <EnhancedHolmesGPTChat 
                alert={alert}
                showCard={false}
              />
            </div>
          </TabsContent>

          <TabsContent value="raw" className="h-[500px]">
            <div className="h-full">
              <RawPayloadViewer 
                alertId={alert.id}
                alertTitle={alert.title}
              />
            </div>
          </TabsContent>
        </Tabs>
      </div>

      {/* 分析历史 */}
      {analysisHistory.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              <Clock className="h-5 w-5 text-gray-600" />
              <span>分析历史</span>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {analysisHistory.map((history, index) => (
                <div key={index} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                  <div className="flex items-center space-x-3">
                    <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                    <div>
                      <div className="font-medium">分析 #{index + 1}</div>
                      <div className="text-sm text-gray-600">
                        {format(new Date(history.timestamp), 'yyyy-MM-dd HH:mm', { locale: zhCN })}
                      </div>
                    </div>
                  </div>
                  <Button variant="outline" size="sm">
                    <ExternalLink className="h-4 w-4 mr-2" />
                    查看结果
                  </Button>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

    </div>
  )
}
