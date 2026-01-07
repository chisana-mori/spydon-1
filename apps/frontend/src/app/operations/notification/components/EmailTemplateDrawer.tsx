'use client';

import React, { useEffect, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Loader2, Plus, Trash2, X } from 'lucide-react';
import { RobustaAPI } from '@/lib/api';
import { CreateEmailTemplateRequest, UpdateEmailTemplateRequest, ParamDefinition } from '@/types/email';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Switch } from '@/components/ui/switch';
import {
    Sheet,
    SheetContent,
    SheetDescription,
    SheetHeader,
    SheetTitle,
} from '@/components/ui/sheet';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select';
import { toast } from 'sonner';

interface EmailTemplateDrawerProps {
    open: boolean;
    onClose: () => void;
    templateId: number | null;
}

const defaultParam: ParamDefinition = {
    key: '',
    label: '',
    type: 'input',
    required: false,
};

export function EmailTemplateDrawer({ open, onClose, templateId }: EmailTemplateDrawerProps) {
    const queryClient = useQueryClient();
    const isEdit = templateId !== null;

    const [formData, setFormData] = useState<CreateEmailTemplateRequest>({
        name: '',
        title: '',
        body: '',
        params: [],
        is_enabled: true,
    });

    // Fetch existing template for edit mode
    const { data: template, isLoading: isLoadingTemplate } = useQuery({
        queryKey: ['email-template', templateId],
        queryFn: () => RobustaAPI.getEmailTemplate(templateId!),
        enabled: isEdit && open,
    });

    // Initialize form data when template loads
    useEffect(() => {
        if (template && isEdit) {
            setFormData({
                name: template.name,
                title: template.title,
                body: template.body,
                params: template.params || [],
                is_enabled: template.is_enabled,
            });
        } else if (!isEdit && open) {
            setFormData({
                name: '',
                title: '',
                body: '',
                params: [],
                is_enabled: true,
            });
        }
    }, [template, isEdit, open]);

    // Create mutation
    const createMutation = useMutation({
        mutationFn: RobustaAPI.createEmailTemplate,
        onSuccess: () => {
            toast.success('邮件模板创建成功');
            queryClient.invalidateQueries({ queryKey: ['email-templates'] });
            onClose();
        },
        onError: (error: any) => {
            toast.error(error?.response?.data?.message || '创建失败');
        },
    });

    // Update mutation
    const updateMutation = useMutation({
        mutationFn: (data: UpdateEmailTemplateRequest) =>
            RobustaAPI.updateEmailTemplate(templateId!, data),
        onSuccess: () => {
            toast.success('邮件模板更新成功');
            queryClient.invalidateQueries({ queryKey: ['email-templates'] });
            queryClient.invalidateQueries({ queryKey: ['email-template', templateId] });
            onClose();
        },
        onError: (error: any) => {
            toast.error(error?.response?.data?.message || '更新失败');
        },
    });

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        if (!formData.name.trim() || !formData.title.trim() || !formData.body.trim()) {
            toast.error('请填写必填字段');
            return;
        }

        if (isEdit) {
            updateMutation.mutate(formData);
        } else {
            createMutation.mutate(formData);
        }
    };

    const addParam = () => {
        setFormData({
            ...formData,
            params: [...(formData.params || []), { ...defaultParam, key: `param_${Date.now()}` }],
        });
    };

    const removeParam = (index: number) => {
        const newParams = [...(formData.params || [])];
        newParams.splice(index, 1);
        setFormData({ ...formData, params: newParams });
    };

    const updateParam = (index: number, field: keyof ParamDefinition, value: any) => {
        const newParams = [...(formData.params || [])];
        newParams[index] = { ...newParams[index], [field]: value };
        setFormData({ ...formData, params: newParams });
    };

    const isSubmitting = createMutation.isPending || updateMutation.isPending;

    return (
        <Sheet open={open} onOpenChange={(val) => !val && onClose()}>
            <SheetContent side="right" className="w-full sm:max-w-2xl overflow-y-auto">
                <SheetHeader>
                    <SheetTitle>{isEdit ? '编辑邮件模板' : '新建邮件模板'}</SheetTitle>
                    <SheetDescription>
                        {isEdit ? '修改邮件模板的基本信息、HTML 正文和参数定义。' : '创建新的邮件模板，用于维护通知。'}
                    </SheetDescription>
                </SheetHeader>

                {isLoadingTemplate ? (
                    <div className="flex items-center justify-center h-40">
                        <Loader2 className="w-6 h-6 animate-spin" />
                    </div>
                ) : (
                    <form onSubmit={handleSubmit} className="space-y-6 mt-6">
                        {/* Basic Info */}
                        <div className="space-y-4">
                            <div className="space-y-2">
                                <Label htmlFor="name">模板名称 *</Label>
                                <Input
                                    id="name"
                                    value={formData.name}
                                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                                    placeholder="例如：节点维护通知"
                                />
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="title">邮件标题 *</Label>
                                <Input
                                    id="title"
                                    value={formData.title}
                                    onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                                    placeholder="支持模板变量，如：{{.cluster}} 节点维护通知"
                                />
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="body">HTML 正文 *</Label>
                                <Textarea
                                    id="body"
                                    value={formData.body}
                                    onChange={(e) => setFormData({ ...formData, body: e.target.value })}
                                    placeholder="输入 HTML 模板内容，支持模板变量..."
                                    rows={15}
                                    className="font-mono text-sm min-h-[400px]"
                                />
                                <p className="text-xs text-muted-foreground">
                                    可用变量：{'{{.cluster}}'}, {'{{.nodes}}'}, {'{{.node_count}}'}, {'{{.affected_resources_table}}'} 等
                                </p>
                            </div>

                            <div className="flex items-center space-x-2">
                                <Switch
                                    id="is_enabled"
                                    checked={formData.is_enabled}
                                    onCheckedChange={(checked) => setFormData({ ...formData, is_enabled: checked })}
                                />
                                <Label htmlFor="is_enabled">启用模板</Label>
                            </div>
                        </div>

                        {/* Parameters */}
                        <div className="space-y-4">
                            <div className="flex items-center justify-between">
                                <Label>参数定义</Label>
                                <Button type="button" variant="outline" size="sm" onClick={addParam}>
                                    <Plus className="w-4 h-4 mr-1" /> 添加参数
                                </Button>
                            </div>

                            {formData.params && formData.params.length > 0 ? (
                                <div className="space-y-3">
                                    {formData.params.map((param, index) => (
                                        <div key={index} className="p-3 border rounded-lg space-y-3 bg-muted/30">
                                            <div className="flex items-center justify-between">
                                                <span className="text-sm font-medium">参数 {index + 1}</span>
                                                <Button
                                                    type="button"
                                                    variant="ghost"
                                                    size="icon"
                                                    className="h-6 w-6"
                                                    onClick={() => removeParam(index)}
                                                >
                                                    <Trash2 className="w-4 h-4 text-destructive" />
                                                </Button>
                                            </div>
                                            <div className="grid grid-cols-2 gap-3">
                                                <Input
                                                    placeholder="键名 (key)"
                                                    value={param.key}
                                                    onChange={(e) => updateParam(index, 'key', e.target.value)}
                                                />
                                                <Input
                                                    placeholder="标签 (label)"
                                                    value={param.label}
                                                    onChange={(e) => updateParam(index, 'label', e.target.value)}
                                                />
                                            </div>
                                            <div className="grid grid-cols-2 gap-3">
                                                <Select
                                                    value={param.type}
                                                    onValueChange={(val) => updateParam(index, 'type', val)}
                                                >
                                                    <SelectTrigger>
                                                        <SelectValue placeholder="类型" />
                                                    </SelectTrigger>
                                                    <SelectContent>
                                                        <SelectItem value="input">文本输入</SelectItem>
                                                        <SelectItem value="select">下拉选择</SelectItem>
                                                        <SelectItem value="datetime">日期时间</SelectItem>
                                                        <SelectItem value="resource">K8s资源</SelectItem>
                                                    </SelectContent>
                                                </Select>
                                                <div className="flex items-center space-x-2">
                                                    <Switch
                                                        checked={param.required}
                                                        onCheckedChange={(checked) => updateParam(index, 'required', checked)}
                                                    />
                                                    <Label className="text-sm">必填</Label>
                                                </div>
                                            </div>
                                            {param.type === 'select' && (
                                                <Input
                                                    placeholder="字典编码 (dictCode)"
                                                    value={param.dictCode || ''}
                                                    onChange={(e) => updateParam(index, 'dictCode', e.target.value)}
                                                />
                                            )}
                                            {param.type === 'resource' && (
                                                <Select
                                                    value={param.resourceType || 'nodes'}
                                                    onValueChange={(val) => updateParam(index, 'resourceType', val)}
                                                >
                                                    <SelectTrigger>
                                                        <SelectValue placeholder="资源类型" />
                                                    </SelectTrigger>
                                                    <SelectContent>
                                                        <SelectItem value="nodes">节点</SelectItem>
                                                        <SelectItem value="pods">Pod</SelectItem>
                                                        <SelectItem value="deployments">Deployment</SelectItem>
                                                    </SelectContent>
                                                </Select>
                                            )}
                                        </div>
                                    ))}
                                </div>
                            ) : (
                                <p className="text-sm text-muted-foreground text-center py-4 border rounded-lg border-dashed">
                                    尚未定义参数，点击上方按钮添加。
                                </p>
                            )}
                        </div>

                        {/* Submit */}
                        <div className="flex justify-end gap-3 pt-4 border-t">
                            <Button type="button" variant="outline" onClick={onClose}>
                                取消
                            </Button>
                            <Button type="submit" disabled={isSubmitting}>
                                {isSubmitting && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                                {isEdit ? '保存更改' : '创建模板'}
                            </Button>
                        </div>
                    </form>
                )}
            </SheetContent>
        </Sheet>
    );
}
