'use client'

import { useState, useEffect, useCallback } from 'react'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import {
    AlertCircle,
    CheckCircle,
    Loader2,
    Settings,
    FileCode,
    Save,
    RefreshCw,
} from 'lucide-react'
import { toast } from 'sonner'
import RobustaAPI from '@/lib/api'
import { parse as yamlParse } from 'yaml'

interface InventoryVariablesDialogProps {
    open: boolean
    onOpenChange: (open: boolean) => void
    clusterName: string
    onSuccess?: () => void
}

export function InventoryVariablesDialog({
    open,
    onOpenChange,
    clusterName,
    onSuccess,
}: InventoryVariablesDialogProps) {
    const [variables, setVariables] = useState('')
    const [originalVariables, setOriginalVariables] = useState('')
    const [loading, setLoading] = useState(false)
    const [saving, setSaving] = useState(false)
    const [yamlError, setYamlError] = useState<string | null>(null)
    const [hasChanges, setHasChanges] = useState(false)

    // 加载 Inventory 变量
    const loadVariables = useCallback(async () => {
        if (!clusterName) return
        setLoading(true)
        setYamlError(null)
        try {
            const vars = await RobustaAPI.getInventoryVariables(clusterName)
            setVariables(vars)
            setOriginalVariables(vars)
            setHasChanges(false)
        } catch (error: any) {
            console.error('Failed to load inventory variables:', error)
            toast.error(`加载 Inventory 变量失败: ${error?.message || '未知错误'}`)
            // 如果是 404，可能是 Inventory 不存在
            setVariables('')
            setOriginalVariables('')
        } finally {
            setLoading(false)
        }
    }, [clusterName])

    // 当对话框打开时加载数据
    useEffect(() => {
        if (open && clusterName) {
            loadVariables()
        }
    }, [open, clusterName, loadVariables])

    // 检测变更
    useEffect(() => {
        setHasChanges(variables !== originalVariables)
    }, [variables, originalVariables])

    // 验证 YAML
    const validateYaml = useCallback((content: string): boolean => {
        if (!content.trim()) {
            setYamlError(null)
            return true
        }
        try {
            yamlParse(content)
            setYamlError(null)
            return true
        } catch (error: any) {
            setYamlError(error?.message || 'YAML 格式错误')
            return false
        }
    }, [])

    // 处理输入变化
    const handleChange = (value: string) => {
        setVariables(value)
        validateYaml(value)
    }

    // 保存变量
    const handleSave = async () => {
        if (!validateYaml(variables)) {
            toast.error('请先修复 YAML 格式错误')
            return
        }

        setSaving(true)
        try {
            await RobustaAPI.updateInventoryVariables(clusterName, variables)
            toast.success('Inventory 变量已更新')
            setOriginalVariables(variables)
            setHasChanges(false)
            onSuccess?.()
            onOpenChange(false)
        } catch (error: any) {
            console.error('Failed to save inventory variables:', error)
            toast.error(`保存失败: ${error?.message || '未知错误'}`)
        } finally {
            setSaving(false)
        }
    }

    // 重置变更
    const handleReset = () => {
        setVariables(originalVariables)
        setYamlError(null)
    }

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="max-w-2xl max-h-[85vh] flex flex-col">
                <DialogHeader className="pb-4">
                    <div className="flex items-center gap-3">
                        <div className="p-2 rounded-lg bg-primary/10">
                            <Settings className="h-5 w-5 text-primary" />
                        </div>
                        <div>
                            <DialogTitle className="text-lg">
                                Inventory 参数配置
                            </DialogTitle>
                            <DialogDescription className="text-sm">
                                编辑 AWX Inventory <code className="px-1.5 py-0.5 rounded bg-muted font-mono text-xs">{clusterName}</code> 的变量
                            </DialogDescription>
                        </div>
                    </div>
                </DialogHeader>

                <div className="flex-1 overflow-hidden space-y-4">
                    {/* 状态指示器 */}
                    <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                            <FileCode className="h-4 w-4 text-muted-foreground" />
                            <Label className="text-sm font-medium">YAML 格式变量</Label>
                        </div>
                        <div className="flex items-center gap-2">
                            {yamlError ? (
                                <Badge variant="destructive" className="text-xs gap-1">
                                    <AlertCircle className="h-3 w-3" />
                                    格式错误
                                </Badge>
                            ) : variables.trim() ? (
                                <Badge variant="outline" className="text-xs gap-1 text-green-600 border-green-200 bg-green-50">
                                    <CheckCircle className="h-3 w-3" />
                                    格式正确
                                </Badge>
                            ) : null}
                            {hasChanges && (
                                <Badge variant="secondary" className="text-xs">
                                    未保存的修改
                                </Badge>
                            )}
                        </div>
                    </div>

                    {/* YAML 编辑器 */}
                    <div className="relative flex-1">
                        {loading ? (
                            <div className="h-64 border rounded-lg flex items-center justify-center bg-muted/20">
                                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                            </div>
                        ) : (
                            <>
                                <Textarea
                                    value={variables}
                                    onChange={(e) => handleChange(e.target.value)}
                                    placeholder="---&#10;# 输入 YAML 格式的变量&#10;key: value&#10;another_key:&#10;  nested: value"
                                    className="h-64 font-mono text-sm resize-none"
                                    spellCheck={false}
                                />
                                {yamlError && (
                                    <div className="absolute bottom-0 left-0 right-0 p-2 bg-destructive/10 border-t border-destructive/20 rounded-b-lg">
                                        <p className="text-xs text-destructive flex items-center gap-1.5">
                                            <AlertCircle className="h-3.5 w-3.5 shrink-0" />
                                            <span className="truncate">{yamlError}</span>
                                        </p>
                                    </div>
                                )}
                            </>
                        )}
                    </div>

                    {/* 帮助提示 */}
                    <div className="p-3 rounded-lg bg-muted/30 border text-xs text-muted-foreground space-y-1">
                        <p className="font-medium text-foreground">💡 使用说明</p>
                        <ul className="list-disc list-inside space-y-0.5 pl-1">
                            <li>变量将直接写入到 AWX 中同名 Inventory 的变量配置中</li>
                            <li>这些变量在执行任务时会作为全局变量可用</li>
                            <li>格式必须为有效的 YAML，请确保语法正确</li>
                        </ul>
                    </div>
                </div>

                <DialogFooter className="pt-4 gap-2">
                    <Button
                        variant="ghost"
                        onClick={handleReset}
                        disabled={!hasChanges || saving}
                    >
                        <RefreshCw className="h-4 w-4 mr-2" />
                        重置
                    </Button>
                    <Button
                        variant="outline"
                        onClick={() => onOpenChange(false)}
                    >
                        取消
                    </Button>
                    <Button
                        onClick={handleSave}
                        disabled={!hasChanges || !!yamlError || saving}
                    >
                        {saving ? (
                            <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                        ) : (
                            <Save className="h-4 w-4 mr-2" />
                        )}
                        保存
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    )
}
