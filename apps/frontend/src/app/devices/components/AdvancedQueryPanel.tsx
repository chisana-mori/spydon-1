'use client'

import { useState, useEffect, useCallback } from 'react'
import { useQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import * as React from 'react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Switch } from '@/components/ui/switch'
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from '@/components/ui/dialog'
import {
    Collapsible,
    CollapsibleContent,
    CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
    Plus,
    Trash2,
    Play,
    Save,
    ChevronDown,
    ChevronRight,
    X,
    Loader2,
    Tag,
    AlertTriangle,
    Server,
    Layers,
} from 'lucide-react'
import RobustaAPI from '@/lib/api'
import {
    FilterGroup,
    FilterBlock,
    FilterType,
    ConditionType,
    LogicalOperator,
    NavyFilterOptions,
    CONDITION_TYPES,
    getConditionTypesForFilterType,
    createDefaultFilterBlock,
    createDefaultFilterGroup,
    generateId,
} from '@/types/navy'
import { cn } from '@/lib/utils'

interface AdvancedQueryPanelProps {
    groups: FilterGroup[]
    onChange: (groups: FilterGroup[]) => void
    onExecute: () => void
    isLoading: boolean
    sourceTemplateId?: number
    sourceTemplateName?: string
}

export function AdvancedQueryPanel({
    groups,
    onChange,
    onExecute,
    isLoading,
    sourceTemplateId,
    sourceTemplateName,
}: AdvancedQueryPanelProps) {
    const [expandedGroups, setExpandedGroups] = useState<string[]>(groups.map(g => g.id))
    const [saveDialogOpen, setSaveDialogOpen] = useState(false)
    const [templateName, setTemplateName] = useState('')
    const [templateDesc, setTemplateDesc] = useState('')
    const [isSaving, setIsSaving] = useState(false)

    // 获取筛选选项
    const { data: filterOptions, isLoading: optionsLoading } = useQuery({
        queryKey: ['navy-filter-options'],
        queryFn: () => RobustaAPI.getNavyFilterOptions(),
        staleTime: 5 * 60 * 1000, // 5分钟缓存
    })

    // 添加筛选组
    const addGroup = useCallback(() => {
        const newGroup = createDefaultFilterGroup()
        onChange([...groups, newGroup])
        setExpandedGroups(prev => [...prev, newGroup.id])
    }, [groups, onChange])

    // 删除筛选组
    const removeGroup = useCallback((groupId: string) => {
        if (groups.length <= 1) {
            toast.error('至少需要保留一个筛选组')
            return
        }
        onChange(groups.filter(g => g.id !== groupId))
    }, [groups, onChange])

    // 更新筛选组
    const updateGroup = useCallback((groupId: string, updates: Partial<FilterGroup>) => {
        onChange(groups.map(g => g.id === groupId ? { ...g, ...updates } : g))
    }, [groups, onChange])

    // 添加筛选块
    const addBlock = useCallback((groupId: string, type: FilterType = 'device') => {
        onChange(groups.map(g => {
            if (g.id === groupId) {
                return { ...g, blocks: [...g.blocks, createDefaultFilterBlock(type)] }
            }
            return g
        }))
    }, [groups, onChange])

    // 更新筛选块
    const updateBlock = useCallback((groupId: string, blockId: string, updates: Partial<FilterBlock>) => {
        onChange(groups.map(g => {
            if (g.id === groupId) {
                return {
                    ...g,
                    blocks: g.blocks.map(b => b.id === blockId ? { ...b, ...updates } : b),
                }
            }
            return g
        }))
    }, [groups, onChange])

    // 删除筛选块
    const removeBlock = useCallback((groupId: string, blockId: string) => {
        onChange(groups.map(g => {
            if (g.id === groupId) {
                const newBlocks = g.blocks.filter(b => b.id !== blockId)
                if (newBlocks.length === 0) {
                    newBlocks.push(createDefaultFilterBlock())
                }
                return { ...g, blocks: newBlocks }
            }
            return g
        }))
    }, [groups, onChange])

    // 切换筛选块激活状态
    const toggleBlockActive = useCallback((groupId: string, blockId: string) => {
        onChange(groups.map(g => {
            if (g.id === groupId) {
                return {
                    ...g,
                    blocks: g.blocks.map(b =>
                        b.id === blockId ? { ...b, isActive: !b.isActive } : b
                    ),
                }
            }
            return g
        }))
    }, [groups, onChange])

    // 保存为模板
    const handleSaveTemplate = async () => {
        if (!templateName.trim()) {
            toast.error('请输入模板名称')
            return
        }

        setIsSaving(true)
        try {
            await RobustaAPI.saveNavyTemplate({
                id: sourceTemplateId,
                name: templateName,
                description: templateDesc,
                groups,
            })
            toast.success(sourceTemplateId ? '模板更新成功' : '模板保存成功')
            setSaveDialogOpen(false)
            setTemplateName('')
            setTemplateDesc('')
        } catch (err) {
            toast.error('保存失败')
        } finally {
            setIsSaving(false)
        }
    }

    // 统计激活的条件数
    const activeConditionsCount = groups.reduce((sum, g) =>
        sum + g.blocks.filter((b: any) => {
            const hasKey = b.key || b.Key;
            const active = b.isActive ?? b.is_active ?? b.IsActive ?? true;
            return !!hasKey && active !== false;
        }).length, 0)

    return (
        <div className="space-y-4">
            {/* 操作栏 */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                    <Button onClick={addGroup} variant="outline" size="sm">
                        <Plus className="h-4 w-4 mr-2" />
                        添加筛选组
                    </Button>
                    {sourceTemplateName && (
                        <Badge variant="secondary" className="gap-1">
                            基于模板: {sourceTemplateName}
                        </Badge>
                    )}
                </div>
                <div className="flex items-center gap-2">
                    <span className="text-sm text-muted-foreground">
                        {activeConditionsCount} 条件生效
                    </span>
                    <Dialog open={saveDialogOpen} onOpenChange={setSaveDialogOpen}>
                        <DialogTrigger asChild>
                            <Button variant="outline" size="sm">
                                <Save className="h-4 w-4 mr-2" />
                                保存模板
                            </Button>
                        </DialogTrigger>
                        <DialogContent>
                            <DialogHeader>
                                <DialogTitle>保存查询模板</DialogTitle>
                                <DialogDescription>
                                    将当前查询条件保存为模板，方便下次快速使用
                                </DialogDescription>
                            </DialogHeader>
                            <div className="space-y-4 py-4">
                                <div className="space-y-2">
                                    <Label>模板名称</Label>
                                    <Input
                                        placeholder="如: 生产环境 ARM 服务器"
                                        value={templateName}
                                        onChange={(e) => setTemplateName(e.target.value)}
                                    />
                                </div>
                                <div className="space-y-2">
                                    <Label>模板描述（可选）</Label>
                                    <Input
                                        placeholder="描述这个模板的用途"
                                        value={templateDesc}
                                        onChange={(e) => setTemplateDesc(e.target.value)}
                                    />
                                </div>
                            </div>
                            <DialogFooter>
                                <Button variant="outline" onClick={() => setSaveDialogOpen(false)}>
                                    取消
                                </Button>
                                <Button onClick={handleSaveTemplate} disabled={isSaving}>
                                    {isSaving && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
                                    保存
                                </Button>
                            </DialogFooter>
                        </DialogContent>
                    </Dialog>
                    <Button onClick={onExecute} disabled={isLoading || activeConditionsCount === 0}>
                        {isLoading ? (
                            <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                        ) : (
                            <Play className="h-4 w-4 mr-2" />
                        )}
                        执行查询
                    </Button>
                </div>
            </div>

            {/* 筛选组列表 */}
            <div className="space-y-3">
                {groups.map((group, groupIndex) => (
                    <FilterGroupCard
                        key={group.id}
                        group={group}
                        groupIndex={groupIndex}
                        isExpanded={expandedGroups.includes(group.id)}
                        onToggleExpand={() => {
                            setExpandedGroups(prev =>
                                prev.includes(group.id)
                                    ? prev.filter(id => id !== group.id)
                                    : [...prev, group.id]
                            )
                        }}
                        filterOptions={filterOptions}
                        optionsLoading={optionsLoading}
                        onUpdateGroup={(updates) => updateGroup(group.id, updates)}
                        onRemoveGroup={() => removeGroup(group.id)}
                        onAddBlock={(type) => addBlock(group.id, type)}
                        onUpdateBlock={(blockId, updates) => updateBlock(group.id, blockId, updates)}
                        onRemoveBlock={(blockId) => removeBlock(group.id, blockId)}
                        onToggleBlockActive={(blockId) => toggleBlockActive(group.id, blockId)}
                        showGroupOperator={groupIndex > 0}
                    />
                ))}
            </div>
        </div>
    )
}

// 筛选组卡片
interface FilterGroupCardProps {
    group: FilterGroup
    groupIndex: number
    isExpanded: boolean
    onToggleExpand: () => void
    filterOptions?: NavyFilterOptions
    optionsLoading: boolean
    onUpdateGroup: (updates: Partial<FilterGroup>) => void
    onRemoveGroup: () => void
    onAddBlock: (type: FilterType) => void
    onUpdateBlock: (blockId: string, updates: Partial<FilterBlock>) => void
    onRemoveBlock: (blockId: string) => void
    onToggleBlockActive: (blockId: string) => void
    showGroupOperator: boolean
}

function FilterGroupCard({
    group,
    groupIndex,
    isExpanded,
    onToggleExpand,
    filterOptions,
    optionsLoading,
    onUpdateGroup,
    onRemoveGroup,
    onAddBlock,
    onUpdateBlock,
    onRemoveBlock,
    onToggleBlockActive,
    showGroupOperator,
}: FilterGroupCardProps) {
    const activeCount = group.blocks.filter(b => b.isActive !== false && b.key).length

    return (
        <div className="relative">
            <Card className={cn(
                "border-2 transition-colors",
                isExpanded ? "border-primary/30" : "border-border"
            )}>
                <Collapsible open={isExpanded} onOpenChange={onToggleExpand}>
                    <CardHeader className="py-3 px-4">
                        <div className="flex items-center justify-between">
                            <div className="flex items-center gap-4 flex-1">
                                <CollapsibleTrigger className="flex items-center gap-2 hover:text-primary transition-colors">
                                    {isExpanded ? (
                                        <ChevronDown className="h-4 w-4" />
                                    ) : (
                                        <ChevronRight className="h-4 w-4" />
                                    )}
                                    <span className="text-sm font-semibold">筛选组 {groupIndex + 1}</span>
                                </CollapsibleTrigger>

                                {/* 组逻辑操作符 - 移到头部更显眼的位置 */}
                                <Select
                                    value={group.operator}
                                    onValueChange={(v) => onUpdateGroup({ operator: v as LogicalOperator })}
                                >
                                    <SelectTrigger className={cn(
                                        "h-7 w-[160px] border-0 bg-secondary/50 text-xs font-medium rounded-full",
                                        group.operator === 'and' ? "text-blue-600 bg-blue-50/80 dark:bg-blue-900/40" : "text-amber-600 bg-amber-50/80 dark:bg-amber-900/40"
                                    )}>
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="and">
                                            <div className="flex items-center gap-2">
                                                <Badge variant="outline" className="text-blue-600 bg-blue-50 border-blue-200">AND</Badge>
                                                <span className="text-xs text-muted-foreground">满足所有条件</span>
                                            </div>
                                        </SelectItem>
                                        <SelectItem value="or">
                                            <div className="flex items-center gap-2">
                                                <Badge variant="outline" className="text-amber-600 bg-amber-50 border-amber-200">OR</Badge>
                                                <span className="text-xs text-muted-foreground">满足任一条件</span>
                                            </div>
                                        </SelectItem>
                                    </SelectContent>
                                </Select>

                                <span className="text-xs text-muted-foreground">
                                    {activeCount} 个条件生效
                                </span>
                            </div>

                            <Button
                                variant="ghost"
                                size="icon"
                                className="h-7 w-7 text-muted-foreground hover:text-destructive"
                                onClick={onRemoveGroup}
                            >
                                <Trash2 className="h-4 w-4" />
                            </Button>
                        </div>
                    </CardHeader>

                    <CollapsibleContent>
                        <CardContent className="pt-0 pb-4 px-4 space-y-3">
                            {/* 筛选块列表 */}
                            {group.blocks.map((block, blockIndex) => (
                                <FilterBlockRow
                                    key={block.id}
                                    block={block}
                                    blockIndex={blockIndex}
                                    filterOptions={filterOptions}
                                    optionsLoading={optionsLoading}
                                    onUpdate={(updates) => onUpdateBlock(block.id, updates)}
                                    onRemove={() => onRemoveBlock(block.id)}
                                    onToggleActive={() => onToggleBlockActive(block.id)}
                                    showOperator={blockIndex > 0}
                                />
                            ))}

                            {/* 添加条件按钮 */}
                            <div className="flex items-center gap-2 pt-4 border-t border-border/40 mt-2">
                                <span className="text-xs font-medium text-muted-foreground mr-2">添加条件:</span>
                                <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={() => onAddBlock('device')}
                                    className="gap-2 rounded-full border-dashed border-blue-200 bg-blue-50/50 text-blue-700 hover:bg-blue-100 hover:border-blue-300 dark:bg-blue-900/20 dark:text-blue-300 dark:border-blue-800"
                                >
                                    <Server className="h-3.5 w-3.5" />
                                    设备字段
                                </Button>
                                <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={() => onAddBlock('nodeLabel')}
                                    className="gap-2 rounded-full border-dashed border-emerald-200 bg-emerald-50/50 text-emerald-700 hover:bg-emerald-100 hover:border-emerald-300 dark:bg-emerald-900/20 dark:text-emerald-300 dark:border-emerald-800"
                                >
                                    <Tag className="h-3.5 w-3.5" />
                                    节点标签
                                </Button>
                                <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={() => onAddBlock('taint')}
                                    className="gap-2 rounded-full border-dashed border-orange-200 bg-orange-50/50 text-orange-700 hover:bg-orange-100 hover:border-orange-300 dark:bg-orange-900/20 dark:text-orange-300 dark:border-orange-800"
                                >
                                    <AlertTriangle className="h-3.5 w-3.5" />
                                    节点污点
                                </Button>
                            </div>
                        </CardContent>
                    </CollapsibleContent>
                </Collapsible>
            </Card>
        </div>
    )
}

// 筛选块行
interface FilterBlockRowProps {
    block: FilterBlock
    blockIndex: number
    filterOptions?: NavyFilterOptions
    optionsLoading: boolean
    onUpdate: (updates: Partial<FilterBlock>) => void
    onRemove: () => void
    onToggleActive: () => void
    showOperator: boolean
}

function FilterBlockRow({
    block,
    blockIndex,
    filterOptions,
    optionsLoading,
    onUpdate,
    onRemove,
    onToggleActive,
    showOperator,
}: FilterBlockRowProps) {
    const [fieldValues, setFieldValues] = useState<string[]>([])
    const [loadingValues, setLoadingValues] = useState(false)

    // 获取可用的条件类型
    const availableConditions = getConditionTypesForFilterType(block.type)
    const conditionConfig = CONDITION_TYPES[block.conditionType as ConditionType]

    // 获取可选的键
    const getKeyOptions = () => {
        if (!filterOptions) return []
        switch (block.type) {
            case 'device':
                return filterOptions.deviceFields.map(f => ({ value: f.id, label: f.label }))
            case 'nodeLabel':
                return filterOptions.labelKeys.map(k => ({ value: k, label: k }))
            case 'taint':
                return filterOptions.taintKeys.map(k => ({ value: k, label: k }))
            default:
                return []
        }
    }

    // 当键变更时，加载字段值
    useEffect(() => {
        const loadFieldValues = async () => {
            if (!block.key) {
                setFieldValues([])
                return
            }

            setLoadingValues(true)
            try {
                let values: string[] = []
                if (block.type === 'device') {
                    values = await RobustaAPI.getDeviceFieldValues(block.key)
                } else if (block.type === 'nodeLabel') {
                    values = await RobustaAPI.getLabelValues(block.key)
                } else if (block.type === 'taint') {
                    const taintValues = await RobustaAPI.getTaintValues(block.key)
                    values = taintValues.map(t => t.value)
                }
                setFieldValues(values)
            } catch (err) {
                console.error('加载字段值失败:', err)
            } finally {
                setLoadingValues(false)
            }
        }

        loadFieldValues()
    }, [block.key, block.type])

    const getTypeIcon = () => {
        switch (block.type) {
            case 'device':
                return <Server className="h-3.5 w-3.5" />
            case 'nodeLabel':
                return <Tag className="h-3.5 w-3.5" />
            case 'taint':
                return <AlertTriangle className="h-3.5 w-3.5" />
        }
    }

    // 移除 getTypeBadgeClass 因为不再单独需要


    return (
        <div className={cn(
            "relative flex items-center gap-2 p-2 rounded-xl border transition-all duration-200 group/row",
            block.isActive !== false
                ? "bg-card border-border/50 shadow-sm hover:border-border hover:shadow-md"
                : "bg-muted/20 border-transparent text-muted-foreground grayscale-[0.8]"
        )}>
            {/* 块间逻辑运算符 - 已移除，逻辑由组操作符控制 */}

            {/* 激活开关 */}
            <Switch
                checked={block.isActive !== false}
                onCheckedChange={onToggleActive}
                className="scale-75"
            />

            {/* 键选择 - 合并类型图标和字段选择为一个彩色 Tag Trigger */}
            <Select
                value={block.key}
                onValueChange={(v) => onUpdate({ key: v, value: '' })}
            >
                <SelectTrigger
                    className={cn(
                        "flex-1 min-w-[240px] h-8 border-0 rounded-full font-medium transition-colors ring-offset-background focus:ring-2 focus:ring-ring focus:ring-offset-2",
                        block.type === 'device' && "bg-blue-100 text-blue-700 hover:bg-blue-200 dark:bg-blue-900/40 dark:text-blue-300 dark:hover:bg-blue-900/60",
                        block.type === 'nodeLabel' && "bg-emerald-100 text-emerald-700 hover:bg-emerald-200 dark:bg-emerald-900/40 dark:text-emerald-300 dark:hover:bg-emerald-900/60",
                        block.type === 'taint' && "bg-orange-100 text-orange-700 hover:bg-orange-200 dark:bg-orange-900/40 dark:text-orange-300 dark:hover:bg-orange-900/60"
                    )}
                >
                    <div className="flex items-center gap-1.5 mr-1 overflow-hidden">
                        {getTypeIcon()}
                        <SelectValue placeholder="选择字段" className="truncate" />
                    </div>
                </SelectTrigger>
                <SelectContent>
                    <ScrollArea className="h-[200px]">
                        {getKeyOptions().map(opt => (
                            <SelectItem key={opt.value} value={opt.value}>
                                {opt.label}
                            </SelectItem>
                        ))}
                    </ScrollArea>
                </SelectContent>
            </Select>

            {/* 条件类型 - 仿照参考图样式 */}
            <Select
                value={block.conditionType}
                onValueChange={(v) => onUpdate({ conditionType: v as ConditionType })}
            >
                <SelectTrigger className="w-[120px] h-8 border-0 bg-secondary/20 hover:bg-secondary/40 transition-colors rounded-lg px-2 text-xs focus:ring-0">
                    <SelectValue />
                </SelectTrigger>
                <SelectContent>
                    {availableConditions.map(cond => {
                        const styleMap: Record<string, { symbol: string; bg: string; text: string }> = {
                            equal: { symbol: '=', bg: "bg-sky-100 dark:bg-sky-900/40", text: "text-sky-600 dark:text-sky-400" },
                            notEqual: { symbol: '≠', bg: "bg-red-100 dark:bg-red-900/40", text: "text-red-600 dark:text-red-400" },
                            contains: { symbol: '⊇', bg: "bg-emerald-100 dark:bg-emerald-900/40", text: "text-emerald-600 dark:text-emerald-400" },
                            notContains: { symbol: '⊅', bg: "bg-orange-100 dark:bg-orange-900/40", text: "text-orange-600 dark:text-orange-400" },
                            in: { symbol: '∈', bg: "bg-violet-100 dark:bg-violet-900/40", text: "text-violet-600 dark:text-violet-400" },
                            notIn: { symbol: '∉', bg: "bg-pink-100 dark:bg-pink-900/40", text: "text-pink-600 dark:text-pink-400" },
                            greaterThan: { symbol: '>', bg: "bg-blue-100 dark:bg-blue-900/40", text: "text-blue-600 dark:text-blue-400" },
                            lessThan: { symbol: '<', bg: "bg-blue-100 dark:bg-blue-900/40", text: "text-blue-600 dark:text-blue-400" },
                            exists: { symbol: '∃', bg: "bg-slate-100 dark:bg-slate-800", text: "text-slate-600 dark:text-slate-400" },
                            notExists: { symbol: '∄', bg: "bg-slate-100 dark:bg-slate-800", text: "text-slate-600 dark:text-slate-400" },
                            isEmpty: { symbol: '∅', bg: "bg-slate-100 dark:bg-slate-800", text: "text-slate-600 dark:text-slate-400" },
                            isNotEmpty: { symbol: 'H', bg: "bg-slate-100 dark:bg-slate-800", text: "text-slate-600 dark:text-slate-400" },
                        }

                        const style = styleMap[cond] || { symbol: '?', bg: "bg-gray-100", text: "text-gray-600" }

                        return (
                            <SelectItem key={cond} value={cond} className="text-xs">
                                <div className="flex items-center gap-2.5">
                                    <div className={cn(
                                        "flex items-center justify-center w-5 h-5 rounded-[4px] font-mono text-[10px] font-bold",
                                        style.bg,
                                        style.text
                                    )}>
                                        {style.symbol}
                                    </div>
                                    <span className="text-muted-foreground font-medium">
                                        {CONDITION_TYPES[cond as keyof typeof CONDITION_TYPES]?.label || cond}
                                    </span>
                                </div>
                            </SelectItem>
                        )
                    })}
                </SelectContent>
            </Select>

            {/* 值输入 */}
            {conditionConfig?.requiresValue && (
                <div className="flex-[0_0_50%] min-w-[200px] relative group">
                    {['in', 'notIn', 'contains', 'notContains'].includes(block.conditionType) ? (
                        <TagInput
                            placeholder="输入值 (回车添加, 支持粘贴多行)..."
                            value={Array.isArray(block.value) ? block.value : (typeof block.value === 'string' && block.value ? block.value.split(',') : [])}
                            onChange={(vals) => onUpdate({ value: vals })}
                            options={fieldValues.map(v => ({ label: v, value: v }))}
                            className="bg-background/50 hover:bg-background transition-colors"
                        />
                    ) : (
                        fieldValues.length > 0 ? (
                            <Select
                                value={typeof block.value === 'string' ? block.value : ''}
                                onValueChange={(v) => onUpdate({ value: v })}
                            >
                                <SelectTrigger className="h-8 rounded-full border-input/60 bg-background/50 focus:bg-background transition-colors hover:border-input">
                                    <SelectValue placeholder="选择值..." />
                                </SelectTrigger>
                                <SelectContent>
                                    <ScrollArea className="h-[200px]">
                                        {fieldValues.map(val => (
                                            <SelectItem key={val} value={val}>
                                                {val}
                                            </SelectItem>
                                        ))}
                                    </ScrollArea>
                                </SelectContent>
                            </Select>
                        ) : (
                            <Input
                                value={typeof block.value === 'string' ? block.value : ''}
                                onChange={(e) => onUpdate({ value: e.target.value })}
                                placeholder="输入值..."
                                className="h-8 rounded-full border-input/60 bg-background/50 focus:bg-background transition-colors hover:border-input"
                            />
                        )
                    )}
                </div>
            )}

            <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7 shrink-0 text-muted-foreground/50 hover:text-destructive hover:bg-destructive/10 rounded-full opacity-0 group-hover/row:opacity-100 transition-all"
                onClick={onRemove}
            >
                <X className="h-4 w-4" />
            </Button>
        </div>
    )
}

// 多值标签输入组件
interface TagInputProps {
    value?: string[]
    onChange: (value: string[]) => void
    options?: { label: string; value: string }[]
    placeholder?: string
    className?: string
}

function TagInput({ value = [], onChange, options = [], placeholder, className }: TagInputProps) {
    const [inputValue, setInputValue] = useState('')
    const [open, setOpen] = useState(false)
    const inputRef = React.useRef<HTMLInputElement>(null)

    const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
        if (e.key === 'Enter' || e.key === ',') {
            e.preventDefault()
            addValue()
        } else if (e.key === 'Backspace' && !inputValue && value.length > 0) {
            onChange(value.slice(0, -1))
        }
    }

    const handlePaste = (e: React.ClipboardEvent) => {
        e.preventDefault()
        const pastedData = e.clipboardData.getData('text')
        if (!pastedData) return

        // Split by newline, comma, space, or semicolon
        const splitValues = pastedData.split(/[\n,;\s]+/).map(v => v.trim()).filter(v => v)

        if (splitValues.length > 0) {
            // Filter out duplicates that are already in the value array
            const newValues = splitValues.filter(v => !value.includes(v))
            if (newValues.length > 0) {
                onChange([...value, ...newValues])
            }
            setInputValue('')
        }
    }

    const addValue = (val?: string) => {
        const text = val || inputValue.trim()
        if (text && !value.includes(text)) {
            onChange([...value, text])
            setInputValue('')
        }
    }

    // 过滤选项
    const filteredOptions = options.filter(opt =>
        opt.label.toLowerCase().includes(inputValue.toLowerCase()) &&
        !value.includes(opt.value)
    )

    return (
        <div className={cn(
            "flex flex-wrap items-center gap-1.5 p-1.5 min-h-[36px] rounded-xl border border-input/60 focus-within:ring-2 focus-within:ring-ring focus-within:ring-offset-2",
            className
        )}>
            {value.map((v, i) => (
                <Badge key={i} variant="secondary" className="gap-1 pr-1 font-normal bg-secondary/50">
                    {v}
                    <button
                        className="rounded-full hover:bg-muted p-0.5 transition-colors"
                        onClick={(e) => {
                            e.stopPropagation()
                            onChange(value.filter(item => item !== v))
                        }}
                    >
                        <X className="h-2 w-2" />
                    </button>
                </Badge>
            ))}
            <div className="relative flex-1 min-w-[80px]">
                <input
                    ref={inputRef}
                    className="w-full bg-transparent border-none outline-none text-sm placeholder:text-muted-foreground/70 h-6"
                    placeholder={value.length === 0 ? placeholder : ''}
                    value={inputValue}
                    onChange={(e) => {
                        setInputValue(e.target.value)
                        setOpen(true)
                    }}
                    onFocus={() => setOpen(true)}
                    onBlur={() => setTimeout(() => setOpen(false), 200)}
                    onKeyDown={handleKeyDown}
                    onPaste={handlePaste}
                />

                {/* 建议列表下拉框 */}
                {open && filteredOptions.length > 0 && (
                    <div className="absolute top-full left-0 w-full mt-1 bg-popover text-popover-foreground rounded-lg border shadow-md z-50 max-h-[200px] overflow-auto py-1">
                        {filteredOptions.map(opt => (
                            <div
                                key={opt.value}
                                className="px-2 py-1.5 text-sm cursor-pointer hover:bg-accent hover:text-accent-foreground flex items-center gap-2"
                                onMouseDown={(e) => {
                                    e.preventDefault() // 防止 input blur
                                    addValue(opt.value)
                                }}
                            >
                                <span>{opt.label}</span>
                            </div>
                        ))}
                    </div>
                )}
            </div>
        </div>
    )
}
