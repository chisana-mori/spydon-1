'use client'

import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Key, Plus, Trash2, Copy, Check, AlertCircle } from 'lucide-react'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'

const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:8080/api/v1'

interface APIKey {
  id: string
  name: string
  key_prefix: string
  last_used_at?: string
  expires_at?: string
  is_active: boolean
  permissions: string
  created_at: string
}

interface CreateAPIKeyRequest {
  name: string
  expires_in?: number
  permissions: string
}

export default function APIKeysPage() {
  const [apiKeys, setApiKeys] = useState<APIKey[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [showKeyDialog, setShowKeyDialog] = useState(false)
  const [newKeyData, setNewKeyData] = useState<{ key: string; name: string } | null>(null)
  const [copiedKey, setCopiedKey] = useState(false)
  const [formData, setFormData] = useState<CreateAPIKeyRequest>({
    name: '',
    expires_in: undefined,
    permissions: 'read',
  })

  useEffect(() => {
    fetchAPIKeys()
  }, [])

  const fetchAPIKeys = async () => {
    try {
      const response = await fetch(`${API_BASE}/apikeys`, {
        credentials: 'include',
      })

      if (!response.ok) {
        throw new Error('获取API Key列表失败')
      }

      const result = await response.json()
      setApiKeys(result.data || [])
    } catch (error) {
      console.error('获取API Key列表失败:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleCreateAPIKey = async () => {
    if (!formData.name.trim()) {
      alert('请输入API Key名称')
      return
    }

    try {
      const response = await fetch(`${API_BASE}/apikeys`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify(formData),
      })

      if (!response.ok) {
        throw new Error('创建API Key失败')
      }

      const result = await response.json()
      
      // 显示新创建的Key（仅此一次）
      setNewKeyData({
        key: result.key,
        name: result.name,
      })
      setShowCreateDialog(false)
      setShowKeyDialog(true)
      
      // 重置表单
      setFormData({
        name: '',
        expires_in: undefined,
        permissions: 'read',
      })

      // 刷新列表
      fetchAPIKeys()
    } catch (error) {
      console.error('创建API Key失败:', error)
      alert('创建API Key失败')
    }
  }

  const handleDeleteAPIKey = async (id: string) => {
    if (!confirm('确定要删除这个API Key吗？删除后将无法恢复。')) {
      return
    }

    try {
      const response = await fetch(`${API_BASE}/apikeys/${id}`, {
        method: 'DELETE',
        credentials: 'include',
      })

      if (!response.ok) {
        throw new Error('删除API Key失败')
      }

      fetchAPIKeys()
    } catch (error) {
      console.error('删除API Key失败:', error)
      alert('删除API Key失败')
    }
  }

  const copyToClipboard = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text)
      setCopiedKey(true)
      setTimeout(() => setCopiedKey(false), 2000)
    } catch (error) {
      console.error('复制失败:', error)
    }
  }

  const getPermissionLabel = (permission: string) => {
    const labels: Record<string, string> = {
      read: '只读',
      write: '读写',
      admin: '管理员',
    }
    return labels[permission] || permission
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-muted-foreground">加载中...</div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">API Key 管理</h1>
          <p className="text-muted-foreground mt-2">
            管理您的API密钥，用于程序化访问系统
          </p>
        </div>
        <Button onClick={() => setShowCreateDialog(true)}>
          <Plus className="h-4 w-4 mr-2" />
          创建 API Key
        </Button>
      </div>

      {apiKeys.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Key className="h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-muted-foreground text-center">
              还没有API Key
              <br />
              点击上方按钮创建您的第一个API Key
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4">
          {apiKeys.map((apiKey) => (
            <Card key={apiKey.id}>
              <CardHeader>
                <div className="flex items-start justify-between">
                  <div className="space-y-1">
                    <CardTitle className="flex items-center gap-2">
                      <Key className="h-5 w-5" />
                      {apiKey.name}
                    </CardTitle>
                    <CardDescription>
                      <code className="text-xs bg-muted px-2 py-1 rounded">
                        {apiKey.key_prefix}...
                      </code>
                    </CardDescription>
                  </div>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => handleDeleteAPIKey(apiKey.id)}
                  >
                    <Trash2 className="h-4 w-4 text-destructive" />
                  </Button>
                </div>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                  <div>
                    <div className="text-muted-foreground">权限</div>
                    <div className="font-medium">{getPermissionLabel(apiKey.permissions)}</div>
                  </div>
                  <div>
                    <div className="text-muted-foreground">状态</div>
                    <div className="font-medium">
                      {apiKey.is_active ? (
                        <span className="text-green-600">激活</span>
                      ) : (
                        <span className="text-red-600">禁用</span>
                      )}
                    </div>
                  </div>
                  <div>
                    <div className="text-muted-foreground">最后使用</div>
                    <div className="font-medium">
                      {apiKey.last_used_at
                        ? format(new Date(apiKey.last_used_at), 'yyyy-MM-dd HH:mm', { locale: zhCN })
                        : '从未使用'}
                    </div>
                  </div>
                  <div>
                    <div className="text-muted-foreground">创建时间</div>
                    <div className="font-medium">
                      {format(new Date(apiKey.created_at), 'yyyy-MM-dd HH:mm', { locale: zhCN })}
                    </div>
                  </div>
                </div>
                {apiKey.expires_at && (
                  <div className="mt-4 flex items-center gap-2 text-sm text-amber-600">
                    <AlertCircle className="h-4 w-4" />
                    过期时间: {format(new Date(apiKey.expires_at), 'yyyy-MM-dd HH:mm', { locale: zhCN })}
                  </div>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* 创建API Key对话框 */}
      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>创建 API Key</DialogTitle>
            <DialogDescription>
              创建一个新的API密钥用于程序化访问。密钥只会显示一次，请妥善保管。
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="name">名称</Label>
              <Input
                id="name"
                placeholder="例如：生产环境API"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="permissions">权限</Label>
              <Select
                value={formData.permissions}
                onValueChange={(value) => setFormData({ ...formData, permissions: value })}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="read">只读</SelectItem>
                  <SelectItem value="write">读写</SelectItem>
                  <SelectItem value="admin">管理员</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="expires_in">过期时间（可选）</Label>
              <Select
                value={formData.expires_in?.toString() || 'never'}
                onValueChange={(value) =>
                  setFormData({
                    ...formData,
                    expires_in: value === 'never' ? undefined : parseInt(value),
                  })
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="never">永不过期</SelectItem>
                  <SelectItem value="30">30天</SelectItem>
                  <SelectItem value="90">90天</SelectItem>
                  <SelectItem value="180">180天</SelectItem>
                  <SelectItem value="365">1年</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreateDialog(false)}>
              取消
            </Button>
            <Button onClick={handleCreateAPIKey}>创建</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 显示新创建的Key */}
      <Dialog open={showKeyDialog} onOpenChange={setShowKeyDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>API Key 创建成功</DialogTitle>
            <DialogDescription>
              请复制并保存您的API密钥。出于安全考虑，此密钥只会显示一次。
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>名称</Label>
              <div className="font-medium">{newKeyData?.name}</div>
            </div>
            <div className="space-y-2">
              <Label>API Key</Label>
              <div className="flex items-center gap-2">
                <code className="flex-1 bg-muted px-3 py-2 rounded text-sm break-all">
                  {newKeyData?.key}
                </code>
                <Button
                  size="icon"
                  variant="outline"
                  onClick={() => newKeyData && copyToClipboard(newKeyData.key)}
                >
                  {copiedKey ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                </Button>
              </div>
            </div>
            <div className="bg-amber-50 dark:bg-amber-950 border border-amber-200 dark:border-amber-800 rounded-lg p-4">
              <div className="flex items-start gap-2">
                <AlertCircle className="h-5 w-5 text-amber-600 flex-shrink-0 mt-0.5" />
                <div className="text-sm text-amber-800 dark:text-amber-200">
                  <p className="font-medium mb-1">重要提示</p>
                  <p>请立即复制并保存此API密钥。关闭此对话框后，您将无法再次查看完整密钥。</p>
                </div>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button onClick={() => setShowKeyDialog(false)}>我已保存</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

