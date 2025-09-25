'use client'

import React from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { 
  Brain, 
  Loader2, 
  CheckCircle, 
  AlertTriangle,
  Lightbulb,
  Target,
  Shield,
  Zap
} from 'lucide-react'

interface AIAnalysisDisplayProps {
  analysis: string
  isAnalyzing: boolean
  isComplete: boolean
  error?: string | null
  className?: string
}

export function AIAnalysisDisplay({ 
  analysis, 
  isAnalyzing, 
  isComplete, 
  error,
  className 
}: AIAnalysisDisplayProps) {
  // 解析分析内容，提取不同部分
  const parseAnalysis = (text: string) => {
    const sections = {
      summary: '',
      rootCause: '',
      solution: '',
      prevention: '',
      other: ''
    }

    // 简单的关键词匹配来分类内容
    const lines = text.split('\n')
    let currentSection = 'other'
    
    lines.forEach(line => {
      const lowerLine = line.toLowerCase()
      
      if (lowerLine.includes('根因') || lowerLine.includes('原因分析') || lowerLine.includes('问题分析')) {
        currentSection = 'rootCause'
      } else if (lowerLine.includes('解决方案') || lowerLine.includes('修复') || lowerLine.includes('解决')) {
        currentSection = 'solution'
      } else if (lowerLine.includes('预防') || lowerLine.includes('建议') || lowerLine.includes('最佳实践')) {
        currentSection = 'prevention'
      } else if (lowerLine.includes('摘要') || lowerLine.includes('总结')) {
        currentSection = 'summary'
      }
      
      if (line.trim()) {
        sections[currentSection] += line + '\n'
      }
    })

    return sections
  }

  const sections = analysis ? parseAnalysis(analysis) : null

  if (!isAnalyzing && !analysis && !error) {
    return (
      <Card className={`border-2 border-dashed border-gray-200 ${className}`}>
        <CardContent className="flex flex-col items-center justify-center py-12">
          <div className="w-16 h-16 bg-gradient-to-br from-purple-100 to-blue-100 rounded-full flex items-center justify-center mb-4">
            <Brain className="h-8 w-8 text-purple-600" />
          </div>
          <h3 className="text-lg font-semibold text-gray-900 mb-2">AI 智能分析</h3>
          <p className="text-gray-600 text-center max-w-md">
            点击"触发RCA分析"按钮，让 HolmesGPT 为您提供智能的根因分析和解决方案
          </p>
        </CardContent>
      </Card>
    )
  }

  if (error) {
    return (
      <Card className={`border-red-200 bg-red-50 ${className}`}>
        <CardContent className="p-6">
          <div className="flex items-center space-x-3 mb-4">
            <div className="w-10 h-10 bg-red-100 rounded-lg flex items-center justify-center">
              <AlertTriangle className="h-5 w-5 text-red-600" />
            </div>
            <div>
              <h3 className="font-semibold text-red-900">分析失败</h3>
              <p className="text-sm text-red-700">AI 分析过程中遇到问题</p>
            </div>
          </div>
          <div className="bg-red-100 rounded-lg p-4">
            <p className="text-red-800 text-sm">{error}</p>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className={`space-y-6 ${className}`}>
      {/* 分析状态头部 */}
      <Card className="border-purple-200 bg-gradient-to-r from-purple-50 to-blue-50">
        <CardContent className="p-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <div className="w-12 h-12 bg-gradient-to-br from-purple-500 to-blue-600 rounded-xl flex items-center justify-center">
                <Brain className="h-6 w-6 text-white" />
              </div>
              <div>
                <h3 className="text-lg font-semibold text-gray-900">HolmesGPT 智能分析</h3>
                <p className="text-sm text-gray-600">AI 驱动的根因分析和故障排查</p>
              </div>
            </div>
            
            <div className="flex items-center space-x-2">
              {isAnalyzing && (
                <Badge variant="secondary" className="bg-blue-100 text-blue-800 border-blue-200">
                  <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                  分析中...
                </Badge>
              )}
              {isComplete && (
                <Badge variant="secondary" className="bg-green-100 text-green-800 border-green-200">
                  <CheckCircle className="h-3 w-3 mr-1" />
                  完成
                </Badge>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 分析结果展示 */}
      {analysis && (
        <div className="grid gap-6">
          {/* 如果有结构化的分析结果 */}
          {sections && (sections.summary || sections.rootCause || sections.solution || sections.prevention) ? (
            <>
              {/* 问题摘要 */}
              {sections.summary && (
                <Card className="border-blue-200 bg-blue-50">
                  <CardContent className="p-6">
                    <div className="flex items-center space-x-3 mb-4">
                      <div className="w-8 h-8 bg-blue-100 rounded-lg flex items-center justify-center">
                        <Target className="h-4 w-4 text-blue-600" />
                      </div>
                      <h4 className="font-semibold text-blue-900">问题摘要</h4>
                    </div>
                    <div className="prose prose-sm max-w-none text-blue-800">
                      <pre className="whitespace-pre-wrap font-sans leading-relaxed">
                        {sections.summary.trim()}
                      </pre>
                    </div>
                  </CardContent>
                </Card>
              )}

              {/* 根因分析 */}
              {sections.rootCause && (
                <Card className="border-orange-200 bg-orange-50">
                  <CardContent className="p-6">
                    <div className="flex items-center space-x-3 mb-4">
                      <div className="w-8 h-8 bg-orange-100 rounded-lg flex items-center justify-center">
                        <AlertTriangle className="h-4 w-4 text-orange-600" />
                      </div>
                      <h4 className="font-semibold text-orange-900">根因分析</h4>
                    </div>
                    <div className="prose prose-sm max-w-none text-orange-800">
                      <pre className="whitespace-pre-wrap font-sans leading-relaxed">
                        {sections.rootCause.trim()}
                      </pre>
                    </div>
                  </CardContent>
                </Card>
              )}

              {/* 解决方案 */}
              {sections.solution && (
                <Card className="border-green-200 bg-green-50">
                  <CardContent className="p-6">
                    <div className="flex items-center space-x-3 mb-4">
                      <div className="w-8 h-8 bg-green-100 rounded-lg flex items-center justify-center">
                        <Zap className="h-4 w-4 text-green-600" />
                      </div>
                      <h4 className="font-semibold text-green-900">解决方案</h4>
                    </div>
                    <div className="prose prose-sm max-w-none text-green-800">
                      <pre className="whitespace-pre-wrap font-sans leading-relaxed">
                        {sections.solution.trim()}
                      </pre>
                    </div>
                  </CardContent>
                </Card>
              )}

              {/* 预防措施 */}
              {sections.prevention && (
                <Card className="border-purple-200 bg-purple-50">
                  <CardContent className="p-6">
                    <div className="flex items-center space-x-3 mb-4">
                      <div className="w-8 h-8 bg-purple-100 rounded-lg flex items-center justify-center">
                        <Shield className="h-4 w-4 text-purple-600" />
                      </div>
                      <h4 className="font-semibold text-purple-900">预防措施</h4>
                    </div>
                    <div className="prose prose-sm max-w-none text-purple-800">
                      <pre className="whitespace-pre-wrap font-sans leading-relaxed">
                        {sections.prevention.trim()}
                      </pre>
                    </div>
                  </CardContent>
                </Card>
              )}

              {/* 其他信息 */}
              {sections.other && sections.other.trim() && (
                <Card className="border-gray-200 bg-gray-50">
                  <CardContent className="p-6">
                    <div className="flex items-center space-x-3 mb-4">
                      <div className="w-8 h-8 bg-gray-100 rounded-lg flex items-center justify-center">
                        <Lightbulb className="h-4 w-4 text-gray-600" />
                      </div>
                      <h4 className="font-semibold text-gray-900">详细信息</h4>
                    </div>
                    <div className="prose prose-sm max-w-none text-gray-700">
                      <pre className="whitespace-pre-wrap font-sans leading-relaxed">
                        {sections.other.trim()}
                      </pre>
                    </div>
                  </CardContent>
                </Card>
              )}
            </>
          ) : (
            /* 如果没有结构化数据，显示完整分析 */
            <Card className="border-gray-200">
              <CardContent className="p-6">
                <div className="flex items-center space-x-3 mb-4">
                  <div className="w-8 h-8 bg-gray-100 rounded-lg flex items-center justify-center">
                    <Brain className="h-4 w-4 text-gray-600" />
                  </div>
                  <h4 className="font-semibold text-gray-900">分析结果</h4>
                </div>
                <div className="prose prose-sm max-w-none">
                  <pre className="whitespace-pre-wrap font-sans leading-relaxed text-gray-700 bg-gray-50 p-4 rounded-lg">
                    {analysis}
                  </pre>
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      )}

      {/* 分析进行中的占位符 */}
      {isAnalyzing && !analysis && (
        <Card className="border-blue-200 bg-blue-50">
          <CardContent className="p-8">
            <div className="flex flex-col items-center justify-center space-y-4">
              <div className="w-12 h-12 bg-blue-100 rounded-full flex items-center justify-center">
                <Loader2 className="h-6 w-6 text-blue-600 animate-spin" />
              </div>
              <div className="text-center">
                <h4 className="font-semibold text-blue-900 mb-2">AI 正在分析中...</h4>
                <p className="text-sm text-blue-700">
                  HolmesGPT 正在深入分析告警信息，请稍候
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
