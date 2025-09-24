import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Users, UserCog, ShieldCheck, Plus } from 'lucide-react'

export default function UsersPage() {
  return (
    <div className="space-y-8">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-foreground">用户管理</h1>
          <p className="text-muted-foreground mt-2">
            管理平台成员、权限分组与访问控制策略。
          </p>
        </div>
        <div className="flex flex-col sm:flex-row gap-3">
          <Button variant="outline" className="h-11">
            <ShieldCheck className="w-4 h-4 mr-2" />
            权限策略
          </Button>
          <Button className="h-11">
            <Plus className="w-4 h-4 mr-2" />
            新建用户
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <Card className="border-2 border-dashed border-muted-foreground/20 hover:border-primary/50 transition-colors">
          <CardHeader className="text-center pb-4">
            <div className="w-12 h-12 bg-blue-50 dark:bg-blue-900/20 rounded-xl flex items-center justify-center mx-auto mb-3">
              <Users className="w-6 h-6 text-blue-600" />
            </div>
            <CardTitle className="text-lg">成员目录</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-sm text-muted-foreground">
            <p>团队信息同步中，即将开放成员邀请、角色分配与登录审计功能。</p>
          </CardContent>
        </Card>

        <Card className="border-2 border-dashed border-muted-foreground/20 hover:border-primary/50 transition-colors">
          <CardHeader className="text-center pb-4">
            <div className="w-12 h-12 bg-indigo-50 dark:bg-indigo-900/20 rounded-xl flex items-center justify-center mx-auto mb-3">
              <UserCog className="w-6 h-6 text-indigo-600" />
            </div>
            <CardTitle className="text-lg">权限配置</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-sm text-muted-foreground">
            <p>基于角色的权限控制和细粒度操作审计正在重构，敬请期待。</p>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
