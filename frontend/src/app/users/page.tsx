'use client'

import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Users, Search, Shield, ShieldOff, Trash2, RefreshCw } from 'lucide-react'
import RobustaAPI from '@/lib/api'
import { toast } from 'sonner'
import type { User } from '@/types/api'
import { useCurrentUser } from '@/hooks/useCurrentUser'

export default function UsersPage() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [searchInput, setSearchInput] = useState('')
  const [selectedUser, setSelectedUser] = useState<User | null>(null)
  const [actionType, setActionType] = useState<'admin' | 'delete' | null>(null)

  const queryClient = useQueryClient()
  const { user: currentUser } = useCurrentUser()

  // 获取用户列表
  const { data: usersData, isLoading, refetch } = useQuery({
    queryKey: ['users', page, keyword],
    queryFn: () => RobustaAPI.getUsers(page, 20, keyword || undefined),
  })

  // 设置管理员权限
  const setAdminMutation = useMutation({
    mutationFn: ({ userId, isAdmin }: { userId: string; isAdmin: boolean }) =>
      RobustaAPI.setUserAdmin(userId, isAdmin),
    onSuccess: () => {
      toast.success('用户权限更新成功')
      queryClient.invalidateQueries({ queryKey: ['users'] })
      setSelectedUser(null)
      setActionType(null)
    },
    onError: (error: any) => {
      toast.error(error?.response?.data?.error || '更新用户权限失败')
    },
  })

  // 删除用户
  const deleteUserMutation = useMutation({
    mutationFn: (userId: string) => RobustaAPI.deleteUser(userId),
    onSuccess: () => {
      toast.success('用户删除成功')
      queryClient.invalidateQueries({ queryKey: ['users'] })
      setSelectedUser(null)
      setActionType(null)
    },
    onError: (error: any) => {
      toast.error(error?.response?.data?.error || '删除用户失败')
    },
  })

  const handleSearch = () => {
    setKeyword(searchInput)
    setPage(1)
  }

  const handleSetAdmin = (user: User, isAdmin: boolean) => {
    setSelectedUser(user)
    setActionType('admin')
    // 直接执行，不需要确认对话框
    setAdminMutation.mutate({ userId: user.id, isAdmin })
  }

  const handleDeleteUser = (user: User) => {
    setSelectedUser(user)
    setActionType('delete')
  }

  const confirmAction = () => {
    if (!selectedUser) return

    if (actionType === 'delete') {
      deleteUserMutation.mutate(selectedUser.id)
    }
  }

  const users = usersData?.data || []
  const pagination = usersData?.pagination

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-foreground">用户管理</h1>
          <p className="text-muted-foreground mt-2">
            管理平台用户及其管理员权限
          </p>
        </div>
        <Button onClick={() => refetch()} variant="outline">
          <RefreshCw className="w-4 h-4 mr-2" />
          刷新
        </Button>
      </div>

      {/* 搜索栏 */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-muted-foreground w-4 h-4" />
              <Input
                placeholder="搜索用户名、邮箱或姓名..."
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
                className="pl-10"
              />
            </div>
            <Button onClick={handleSearch}>搜索</Button>
          </div>
        </CardContent>
      </Card>

      {/* 用户列表 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Users className="w-5 h-5" />
            用户列表
            {pagination && (
              <span className="text-sm font-normal text-muted-foreground">
                （共 {pagination.total} 个用户）
              </span>
            )}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="text-center py-8 text-muted-foreground">加载中...</div>
          ) : users.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">暂无用户数据</div>
          ) : (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>用户名</TableHead>
                    <TableHead>邮箱</TableHead>
                    <TableHead>姓名</TableHead>
                    <TableHead>权限</TableHead>
                    <TableHead>认证方式</TableHead>
                    <TableHead>最后登录</TableHead>
                    <TableHead className="text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {users.map((user) => (
                    <TableRow key={user.id}>
                      <TableCell className="font-medium">{user.username}</TableCell>
                      <TableCell>{user.email}</TableCell>
                      <TableCell>{user.name || '-'}</TableCell>
                      <TableCell>
                        {user.is_admin ? (
                          <Badge variant="default" className="gap-1">
                            <Shield className="w-3 h-3" />
                            管理员
                          </Badge>
                        ) : (
                          <Badge variant="secondary">普通用户</Badge>
                        )}
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline">{user.provider || 'local'}</Badge>
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {user.last_login_at || '-'}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex justify-end gap-2">
                          {currentUser?.is_admin && (
                            <>
                              {user.is_admin ? (
                                <Button
                                  size="sm"
                                  variant="outline"
                                  onClick={() => handleSetAdmin(user, false)}
                                  disabled={currentUser.id === user.id}
                                  title={currentUser.id === user.id ? '不能取消自己的管理员权限' : '取消管理员'}
                                >
                                  <ShieldOff className="w-4 h-4" />
                                </Button>
                              ) : (
                                <Button
                                  size="sm"
                                  variant="outline"
                                  onClick={() => handleSetAdmin(user, true)}
                                  title="设为管理员"
                                >
                                  <Shield className="w-4 h-4" />
                                </Button>
                              )}
                              <Button
                                size="sm"
                                variant="outline"
                                onClick={() => handleDeleteUser(user)}
                                disabled={currentUser.id === user.id}
                                title={currentUser.id === user.id ? '不能删除自己' : '删除用户'}
                              >
                                <Trash2 className="w-4 h-4" />
                              </Button>
                            </>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}

          {/* 分页 */}
          {pagination && pagination.total > pagination.limit && (
            <div className="flex items-center justify-between mt-4">
              <div className="text-sm text-muted-foreground">
                显示 {(page - 1) * pagination.limit + 1} 到{' '}
                {Math.min(page * pagination.limit, pagination.total)} 条，共 {pagination.total} 条
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage(page - 1)}
                  disabled={page === 1}
                >
                  上一页
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage(page + 1)}
                  disabled={page * pagination.limit >= pagination.total}
                >
                  下一页
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* 删除确认对话框 */}
      <AlertDialog open={actionType === 'delete'} onOpenChange={() => setActionType(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除用户</AlertDialogTitle>
            <AlertDialogDescription>
              确定要删除用户 <strong>{selectedUser?.username}</strong> 吗？此操作无法撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={confirmAction}>确认删除</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
