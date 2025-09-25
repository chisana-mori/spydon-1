'use client'

import { HolmesGPTAnalysis } from '@/components/alerts/HolmesGPTAnalysis'

export default function HolmesGPTTestPage() {
  // 模拟告警数据
  const mockAlert = {
    id: 'test-alert-123',
    title: 'ImagePullBackOff: test-failure-pod (default)',
    description: 'Failed to pull at least one image in pod test-failure-pod in namespace default',
    severity: 'critical',
    status: 'firing',
    cluster_id: 'robusta-kind',
    fingerprint: 'b9fb52670652c3605cbfcfd873b6dc5e',
    created_at: new Date().toISOString(),
    starts_at: new Date().toISOString(),
    labels: {
      alertname: 'ImagePullBackOff',
      namespace: 'default',
      pod: 'test-failure-pod',
      container: 'test-container'
    },
    annotations: {
      description: 'Failed to pull at least one image in pod test-failure-pod in namespace default',
      summary: 'Pod test-failure-pod in namespace default has ImagePullBackOff'
    }
  }

  return (
    <div className="container mx-auto py-8 space-y-6">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900 mb-2">HolmesGPT 测试页面</h1>
        <p className="text-gray-600">
          测试 HolmesGPT 智能分析功能的集成效果
        </p>
      </div>

      <div className="bg-white border border-gray-200 rounded-lg p-6 mb-6">
        <h2 className="text-xl font-semibold mb-4">模拟告警信息</h2>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <span className="font-medium">标题:</span> {mockAlert.title}
          </div>
          <div>
            <span className="font-medium">状态:</span> {mockAlert.status}
          </div>
          <div>
            <span className="font-medium">严重级别:</span> {mockAlert.severity}
          </div>
          <div>
            <span className="font-medium">集群:</span> {mockAlert.cluster_id}
          </div>
          <div className="col-span-2">
            <span className="font-medium">描述:</span> {mockAlert.description}
          </div>
        </div>
      </div>

      <HolmesGPTAnalysis alert={mockAlert} />
    </div>
  )
}
