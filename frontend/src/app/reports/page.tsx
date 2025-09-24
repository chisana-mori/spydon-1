import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { FileChartColumnIncreasing, Sparkles } from 'lucide-react'

export default function ReportsPage() {
  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-bold tracking-tight text-foreground">
          分析报告
        </h1>
        <p className="text-muted-foreground mt-2">
          可视化告警趋势、根因分析表现与集群健康评分将在此呈现。
        </p>
      </div>

      <Card className="bg-muted/30 border-blue-200 dark:border-blue-800">
        <CardHeader className="pb-3">
          <CardTitle className="text-lg flex items-center gap-2">
            <FileChartColumnIncreasing className="w-5 h-5 text-blue-600" />
            智能报告生成中
          </CardTitle>
          <CardDescription>
            自动化分析报告正在重构中，敬请期待。
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-blue-100 dark:bg-blue-900/30 rounded-xl flex items-center justify-center">
                <Sparkles className="w-5 h-5 text-blue-600" />
              </div>
              <div>
                <p className="font-medium">趋势洞察</p>
                <p className="text-sm text-muted-foreground">按告警类型与严重级别生成可操作洞察</p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-indigo-100 dark:bg-indigo-900/30 rounded-xl flex items-center justify-center">
                <Badge variant="outline" className="text-xs">Pro</Badge>
              </div>
              <div>
                <p className="font-medium">自动报告</p>
                <p className="text-sm text-muted-foreground">每周推送多集群健康评分与改进建议</p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
