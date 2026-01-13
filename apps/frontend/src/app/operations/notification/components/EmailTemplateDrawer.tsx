'use client';

import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
    Loader2, Plus, Trash2, Mail, FileText, Code, Settings, ChevronRight,
    Server, Package, History, Layers, Rocket, Network,
    ArrowDownCircle, ArrowUpCircle, ArrowRightCircle, CheckCircle2,
    LayoutDashboard, List, Calendar, Box, AlertCircle, Link, Unlink
} from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { RobustaAPI } from '@/lib/api';
import { CreateEmailTemplateRequest, UpdateEmailTemplateRequest, ParamDefinition, TemplateConfig, ParamOption } from '@/types/email';
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
    SheetFooter,
} from '@/components/ui/sheet';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Badge } from '@/components/ui/badge';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { toast } from 'sonner';
import { cn } from '@/lib/utils';

interface EmailTemplateDrawerProps {
    open: boolean;
    onClose: () => void;
    templateId: number | null;
}

const defaultParam: ParamDefinition = {
    name: '',
    title: '',
    type: 'string',
    required: false,
};

const TABLE_OPTIONS = [
    { label: '节点列表 ($nodes)', value: '$nodes', icon: Server, description: '集群节点状态及详情' },
    { label: 'Pod 列表 ($pods)', value: '$pods', icon: Package, description: '运行中的 Pod 实例' },
    { label: '历史 Pod 列表 ($historyPods)', value: '$historyPods', icon: History, description: '已删除或重启的 Pod' },
    { label: '组件列表 ($components)', value: '$components', icon: Layers, description: '系统核心组件状态' },
    { label: 'Deployment 列表 ($deploys)', value: '$deploys', icon: Rocket, description: '无状态应用部署信息' },
    { label: 'Node CIDR ($nodeCidrs)', value: '$nodeCidrs', icon: Network, description: '节点网络网段分配' },
    { label: '待下线 Deployment ($offlineDeploys)', value: '$offlineDeploys', icon: ArrowDownCircle, description: '计划下线的服务' },
    { label: '待上线 Deployment ($onlineDeploys)', value: '$onlineDeploys', icon: ArrowUpCircle, description: '即将上线的服务' },
    { label: '迁移 Deployment ($migrationDeploys)', value: '$migrationDeploys', icon: ArrowRightCircle, description: '正在迁移的服务' },
];

/**
 * Main Container Component
 * Responsible for:
 * 1. Managing Sheet visibility
 * 2. Fetching Data when in Edit mode
 * 3. Normalizing data (handling legacy params array)
 * 4. Rendering the Form component with a specific KEY to force-reset state
 */
export function EmailTemplateDrawer({ open, onClose, templateId }: EmailTemplateDrawerProps) {
    const isEdit = templateId !== null;

    // Fetch existing template for edit mode
    const { data: template, isLoading: isLoadingTemplate } = useQuery({
        queryKey: ['email-template', templateId],
        queryFn: () => RobustaAPI.getEmailTemplate(templateId!),
        enabled: isEdit && open,
    });

    // Normalize Data Logic
    const getInitialData = (): CreateEmailTemplateRequest | null => {
        // New Template
        if (!isEdit) {
            return {
                name: '',
                title: '',
                body: '',
                params: { tables: [], definitions: [] },
                is_enabled: true,
            };
        }

        // Edit Template (Wait for data)
        if (!template) return null;

        // Data Normalization (Handle legacy array vs new object)
        let normalizedParams: TemplateConfig = { tables: [], definitions: [] };
        if (Array.isArray(template.params)) {
            // Legacy: params was []ParamDefinition
            // Map legacy key/label if they exist, otherwise fallback
            normalizedParams.definitions = (template.params as any[]).map(p => ({
                ...p,
                name: p.key || p.name,
                title: p.label || p.title,
                _id: Math.random().toString(36).substr(2, 9)
            })) as ParamDefinition[];
        } else if (template.params) {
            // New: params is TemplateConfig object
            // Handle possibility of receiving 'inputs' from backend if it hasn't migrated data yet
            const rawParams = template.params as any;
            const definitions = rawParams.definitions || rawParams.inputs || [];

            normalizedParams = {
                tables: rawParams.tables || [],
                definitions: definitions.map((p: any) => ({
                    ...p,
                    name: p.name || p.key,   // Support both for transition
                    title: p.title || p.label,
                    defaultTime: p.defaultTime || p.default_time,
                    options: p.options || [],
                    isValueSeparated: p.isValueSeparated || false,
                    _id: Math.random().toString(36).substr(2, 9)
                }))
            };
        }

        return {
            name: template.name,
            title: template.title,
            body: template.body,
            params: normalizedParams,
            is_enabled: template.is_enabled,
        };
    };

    const initialData = getInitialData();

    return (
        <Sheet open={open} onOpenChange={(val) => !val && onClose()}>
            <SheetContent className="sm:max-w-[900px] w-full flex flex-col h-full p-0 gap-0 border-l-0 shadow-2xl bg-background/95 backdrop-blur-xl">
                <SheetHeader className="px-8 py-6 border-b bg-muted/10 relative overflow-hidden">
                    <div className="absolute top-0 right-0 p-12 opacity-5 pointer-events-none">
                        <Mail className="w-64 h-64 text-primary" />
                    </div>
                    <div className="flex items-center gap-5 relative z-10">
                        <motion.div
                            initial={{ scale: 0.8, opacity: 0 }}
                            animate={{ scale: 1, opacity: 1 }}
                            className="p-3.5 bg-primary/10 rounded-2xl ring-1 ring-primary/20 shadow-sm"
                        >
                            <Mail className="w-6 h-6 text-primary" />
                        </motion.div>
                        <div className="space-y-1.5">
                            <SheetTitle className="flex items-center gap-3 text-2xl font-bold tracking-tight">
                                {isEdit ? '编辑邮件模板' : '新建邮件模板'}
                                {initialData && (
                                    <Badge variant={initialData.is_enabled ? 'default' : 'secondary'} className={cn(
                                        "h-6 px-2.5 text-xs font-semibold uppercase tracking-wider shadow-none transition-colors",
                                        initialData.is_enabled
                                            ? "bg-green-500/15 text-green-700 dark:text-green-400 border-green-500/20 hover:bg-green-500/25"
                                            : "bg-red-500/15 text-red-700 dark:text-red-400 border-red-500/20 hover:bg-red-500/25"
                                    )}>
                                        {initialData.is_enabled ? '已启用' : '已禁用'}
                                    </Badge>
                                )}
                            </SheetTitle>
                            <SheetDescription className="text-sm text-muted-foreground/80 font-medium">
                                配置邮件模板的基本信息、HTML 正文和动态参数。
                            </SheetDescription>
                        </div>
                    </div>
                </SheetHeader>

                {isEdit && isLoadingTemplate ? (
                    <div className="flex flex-col items-center justify-center flex-1 space-y-4">
                        <Loader2 className="w-8 h-8 animate-spin text-primary/50" />
                        <p className="text-sm text-muted-foreground animate-pulse">正在加载模板信息...</p>
                    </div>
                ) : (
                    initialData && (
                        <EmailTemplateForm
                            key={`${templateId || 'new'}-${open ? 'open' : 'closed'}`}
                            initialData={initialData}
                            templateId={templateId}
                            onClose={onClose}
                        />
                    )
                )}
            </SheetContent>
        </Sheet>
    );
}

// ----------------------------------------------------------------------
// Form Component
// ----------------------------------------------------------------------

interface EmailTemplateFormProps {
    initialData: CreateEmailTemplateRequest;
    templateId: number | null;
    onClose: () => void;
}

function EmailTemplateForm({ initialData, templateId, onClose }: EmailTemplateFormProps) {
    const queryClient = useQueryClient();
    const isEdit = templateId !== null;
    const [activeTab, setActiveTab] = useState('basic');

    // Local State initialized ONCE from props
    const [formData, setFormData] = useState<CreateEmailTemplateRequest>(initialData);

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

    const isSubmitting = createMutation.isPending || updateMutation.isPending;

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        if (activeTab === 'basic') {
            setActiveTab('params');
            return;
        }

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
        setFormData(prev => ({
            ...prev,
            params: {
                ...prev.params,
                definitions: [...(prev.params.definitions || []), { ...defaultParam, _id: Math.random().toString(36).substr(2, 9) }],
            },
        }));
    };

    const removeParam = (index: number) => {
        setFormData(prev => {
            const newDefs = [...(prev.params.definitions || [])];
            newDefs.splice(index, 1);
            return {
                ...prev,
                params: { ...prev.params, definitions: newDefs },
            };
        });
    };

    const insertParam = (index: number) => {
        setFormData(prev => {
            const newDefs = [...(prev.params.definitions || [])];
            newDefs.splice(index + 1, 0, { ...defaultParam, _id: Math.random().toString(36).substr(2, 9) });
            return {
                ...prev,
                params: { ...prev.params, definitions: newDefs },
            };
        });
    };

    const updateParam = (index: number, field: keyof ParamDefinition, value: any) => {
        setFormData(prev => {
            const newDefs = [...(prev.params.definitions || [])];
            newDefs[index] = { ...newDefs[index], [field]: value };
            return {
                ...prev,
                params: { ...prev.params, definitions: newDefs },
            };
        });
    };

    const toggleTable = (tableValue: string) => {
        setFormData(prev => {
            const currentTables = prev.params.tables || [];
            const isSelected = currentTables.includes(tableValue);
            return {
                ...prev,
                params: {
                    ...prev.params,
                    tables: isSelected
                        ? currentTables.filter(t => t !== tableValue)
                        : [...currentTables, tableValue]
                }
            };
        });
    };

    return (
        <form onSubmit={handleSubmit} className="flex-1 flex flex-col h-full overflow-hidden">
            <Tabs value={activeTab} onValueChange={setActiveTab} className="flex-1 flex flex-col overflow-hidden">
                <div className="px-8 border-b bg-background/50 sticky top-0 z-20 backdrop-blur-sm">
                    <TabsList className="w-full justify-start h-14 bg-transparent p-0 gap-6">
                        {/* ... Tabs Triggers ... */}
                        <TabsTrigger
                            value="basic"
                            className="h-14 rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none px-2 text-sm font-medium text-muted-foreground data-[state=active]:text-primary transition-all duration-300"
                        >
                            <div className="flex items-center gap-2">
                                <FileText className="w-4 h-4" />
                                基本信息
                            </div>
                        </TabsTrigger>
                        <TabsTrigger
                            value="params"
                            className="h-14 rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none px-2 text-sm font-medium text-muted-foreground data-[state=active]:text-primary transition-all duration-300"
                        >
                            <div className="flex items-center gap-2">
                                <Settings className="w-4 h-4" />
                                参数配置
                                <Badge variant="secondary" className="ml-1.5 h-5 px-1.5 min-w-[1.25rem] text-[10px] bg-muted text-muted-foreground group-data-[state=active]:bg-primary/10 group-data-[state=active]:text-primary transition-colors">
                                    {(formData.params?.tables?.length || 0) + (formData.params?.definitions?.length || 0)}
                                </Badge>
                            </div>
                        </TabsTrigger>
                    </TabsList>
                </div>

                <div className="flex-1 overflow-hidden relative">
                    <TabsContent value="basic" className="h-full mt-0 focus-visible:outline-none">
                        <ScrollArea className="h-full">
                            <div className="p-8 space-y-8 max-w-4xl mx-auto">
                                {/* ... Basic Info Fields (Body, Title, etc - Unchanged) ... */}
                                <div className="grid gap-6">
                                    <div className="grid gap-2">
                                        <Label htmlFor="name" className="text-sm font-medium text-muted-foreground">模板名称</Label>
                                        <Input
                                            id="name"
                                            value={formData.name}
                                            onChange={(e) => setFormData(p => ({ ...p, name: e.target.value }))}
                                            placeholder="例如：节点维护通知"
                                            className="h-11 text-base bg-muted/20 border-border/60 hover:border-border focus:border-primary/50 transition-colors"
                                        />
                                    </div>

                                    <div className="grid gap-2">
                                        <div className="flex items-center justify-between">
                                            <Label htmlFor="title" className="text-sm font-medium text-muted-foreground">邮件标题</Label>
                                            <TooltipProvider delayDuration={0}>
                                                <Tooltip>
                                                    <TooltipTrigger asChild>
                                                        <div className="flex items-center gap-1.5 text-xs text-primary/80 cursor-help px-2 py-1 rounded-md bg-primary/5 hover:bg-primary/10 transition-colors">
                                                            <Code className="w-3.5 h-3.5" />
                                                            <span>支持变量</span>
                                                        </div>
                                                    </TooltipTrigger>
                                                    <TooltipContent side="left" className="p-3 text-xs max-w-xs">
                                                        标题支持 Go Template 语法，例如：<br />
                                                        <code className="bg-muted px-1 py-0.5 rounded text-primary">{`{{.cluster}}`}</code> 节点维护通知
                                                    </TooltipContent>
                                                </Tooltip>
                                            </TooltipProvider>
                                        </div>
                                        <Input
                                            id="title"
                                            value={formData.title}
                                            onChange={(e) => setFormData(p => ({ ...p, title: e.target.value }))}
                                            placeholder="请输入邮件主题..."
                                            className="h-11 text-base bg-muted/20 border-border/60 hover:border-border focus:border-primary/50 transition-colors"
                                        />
                                    </div>

                                    <div className="grid gap-2">
                                        <div className="flex items-center justify-between">
                                            <Label htmlFor="body" className="text-sm font-medium text-muted-foreground">HTML 正文</Label>
                                            <Button
                                                type="button"
                                                variant="outline"
                                                size="sm"
                                                onClick={() => {
                                                    try {
                                                        const formatted = formData.body
                                                            .replace(/>\s+</g, '><')
                                                            .replace(/(<([^>]+)>)/g, '\n$1')
                                                            .replace(/^\n/, '')
                                                            .split('\n')
                                                            .filter(line => line.trim())
                                                            .join('\n');
                                                        setFormData(p => ({ ...p, body: formatted }));
                                                        toast.success('HTML 格式化完成');
                                                    } catch {
                                                        toast.error('格式化失败，请检查 HTML 语法');
                                                    }
                                                }}
                                                className="h-7 text-xs"
                                            >
                                                <Code className="w-3 h-3 mr-1" />
                                                格式化 HTML
                                            </Button>
                                        </div>
                                        <div className="relative rounded-xl border border-border/60 bg-muted/20 overflow-hidden focus-within:ring-2 focus-within:ring-primary/20 focus-within:border-primary/50 transition-all">
                                            <Textarea
                                                id="body"
                                                value={formData.body}
                                                onChange={(e) => setFormData(p => ({ ...p, body: e.target.value }))}
                                                placeholder="输入 HTML 模板内容..."
                                                rows={25}
                                                className="font-mono text-sm leading-relaxed p-4 bg-transparent border-0 focus-visible:ring-0 resize-y min-h-[500px]"
                                            />
                                            <div className="absolute bottom-3 right-3 flex items-center gap-2 p-2 bg-background/80 backdrop-blur-md rounded-lg border shadow-sm text-xs text-muted-foreground">
                                                <AlertCircle className="w-3.5 h-3.5 text-primary" />
                                                支持 HTML & Go Template
                                            </div>
                                        </div>
                                    </div>
                                </div>

                                <div className="h-px bg-border/40" />

                                <div className="flex items-center justify-between p-5 rounded-xl border border-border/60 bg-gradient-to-r from-muted/30 to-transparent hover:from-muted/50 transition-all">
                                    <div className="space-y-1.5">
                                        <Label htmlFor="is_enabled" className="text-base font-medium flex items-center gap-2.5">
                                            启用模板
                                            {formData.is_enabled && (
                                                <span className="flex items-center gap-1.5 px-2 py-0.5 rounded-full bg-green-500/10 text-green-700 text-[10px] font-bold uppercase tracking-wider">
                                                    <span className="h-1.5 w-1.5 rounded-full bg-green-500 animate-pulse" />
                                                    Active
                                                </span>
                                            )}
                                        </Label>
                                        <p className="text-sm text-muted-foreground">
                                            控制此邮件模板在系统中的可用性，禁用后将无法通过此模板发送通知。
                                        </p>
                                    </div>
                                    <Switch
                                        id="is_enabled"
                                        checked={formData.is_enabled}
                                        onCheckedChange={(checked) => setFormData(p => ({ ...p, is_enabled: checked }))}
                                        className="scale-110 data-[state=checked]:bg-green-500"
                                    />
                                </div>
                                <div className="h-12" /> {/* Bottom spacer */}
                            </div>
                        </ScrollArea>
                    </TabsContent>

                    <TabsContent value="params" className="h-full mt-0 focus-visible:outline-none">
                        <ScrollArea className="h-full">
                            <div className="p-8 space-y-8 max-w-5xl mx-auto">
                                {/* 1. Tables Section - Unchanged */}
                                <div className="space-y-5">
                                    <div className="space-y-1.5">
                                        <h3 className="text-lg font-semibold flex items-center gap-2 text-foreground/90">
                                            <LayoutDashboard className="w-5 h-5 text-primary" />
                                            附带数据表 (Tables)
                                        </h3>
                                        <p className="text-sm text-muted-foreground">
                                            选择邮件中需要自动生成并附带的数据表格，系统会自动获取上下文数据。
                                        </p>
                                    </div>
                                    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                                        {TABLE_OPTIONS.map((option) => {
                                            const isSelected = (formData.params?.tables || []).includes(option.value);
                                            const Icon = option.icon;
                                            return (
                                                <motion.div
                                                    key={option.value}
                                                    whileHover={{ scale: 1.02 }}
                                                    whileTap={{ scale: 0.98 }}
                                                >
                                                    <div
                                                        onClick={() => toggleTable(option.value)}
                                                        className={cn(
                                                            "relative flex items-start gap-4 px-5 py-4 rounded-xl border-2 cursor-pointer transition-all duration-200 group h-24",
                                                            isSelected
                                                                ? "bg-primary/5 border-primary shadow-sm"
                                                                : "bg-card border-border/40 hover:border-primary/30 hover:bg-muted/30"
                                                        )}
                                                    >
                                                        <div className={cn(
                                                            "p-2 rounded-lg transition-colors",
                                                            isSelected ? "bg-primary/20 text-primary" : "bg-muted text-muted-foreground group-hover:text-primary group-hover:bg-primary/10"
                                                        )}>
                                                            <Icon className="w-5 h-5" />
                                                        </div>
                                                        <div className="space-y-1 flex-1">
                                                            <p className={cn("text-sm font-semibold transition-colors", isSelected ? "text-primary" : "text-foreground")}>
                                                                {option.label.split('(')[0].trim()}
                                                            </p>
                                                            <p className="text-xs text-muted-foreground line-clamp-2 leading-relaxed">
                                                                {option.description}
                                                            </p>
                                                        </div>
                                                        <div className={cn(
                                                            "absolute top-3 right-3 transition-opacity duration-200",
                                                            isSelected ? "opacity-100" : "opacity-0"
                                                        )}>
                                                            <CheckCircle2 className="w-4 h-4 text-primary fill-primary/20" />
                                                        </div>
                                                    </div>
                                                </motion.div>
                                            );
                                        })}
                                    </div>
                                </div>

                                <div className="h-px bg-border/40" />

                                {/* 2. Inputs Section */}
                                <div className="space-y-5">
                                    <div className="flex items-center justify-between">
                                        <div className="space-y-1.5">
                                            <h3 className="text-lg font-semibold flex items-center gap-2 text-foreground/90">
                                                <List className="w-5 h-5 text-primary" />
                                                动态参数定义 (Definitions)
                                            </h3>
                                            <p className="text-sm text-muted-foreground">
                                                定义模板中可使用的动态输入参数，用户发送通知时需要填写这些内容。
                                            </p>
                                        </div>
                                        <Button
                                            type="button"
                                            onClick={addParam}
                                            className="shadow-sm group h-9 bg-primary/90 hover:bg-primary"
                                        >
                                            <Plus className="w-4 h-4 mr-2 group-hover:rotate-90 transition-transform" />
                                            添加参数
                                        </Button>
                                    </div>

                                    <div className="space-y-3 min-h-[200px]">
                                        <AnimatePresence mode="popLayout">
                                            {formData.params?.definitions && formData.params.definitions.length > 0 ? (
                                                formData.params.definitions.map((param, index) => (
                                                    <motion.div
                                                        key={param._id || index}
                                                        layout
                                                        initial={{ opacity: 0, y: 20 }}
                                                        animate={{ opacity: 1, y: 0 }}
                                                        exit={{ opacity: 0, scale: 0.95 }}
                                                        transition={{ duration: 0.2 }}
                                                        className="group relative grid grid-cols-[auto,1fr] gap-3 p-4 pr-10 rounded-xl border border-border/60 bg-gradient-to-br from-card to-muted/20 hover:to-muted/40 transition-all shadow-sm hover:shadow-md"
                                                    >
                                                        <div className="pt-1.5 flex flex-col items-center gap-2">
                                                            <div className="flex items-center justify-center w-7 h-7 rounded-full bg-primary/10 text-primary font-bold text-xs ring-2 ring-background shadow-sm">
                                                                {index + 1}
                                                            </div>
                                                            <div className="w-px h-full bg-border/50 group-last:hidden" />
                                                        </div>

                                                        <div className="space-y-3">
                                                            {/* Delete Button - Positioned Absolute */}
                                                            <div className="absolute top-3 right-3">
                                                                <Button
                                                                    type="button"
                                                                    variant="ghost"
                                                                    size="icon"
                                                                    className="h-7 w-7 text-muted-foreground/60 hover:text-destructive hover:bg-destructive/10"
                                                                    onClick={() => removeParam(index)}
                                                                >
                                                                    <Trash2 className="w-3.5 h-3.5" />
                                                                </Button>
                                                            </div>

                                                            {/* Row 1: Name + Title */}
                                                            <div className="grid grid-cols-2 gap-3">
                                                                <div className="space-y-1">
                                                                    <Label className="text-[11px] text-muted-foreground font-medium">
                                                                        参数名 <span className="text-destructive">*</span>
                                                                    </Label>
                                                                    <Input
                                                                        placeholder="例如: cluster_name"
                                                                        value={param.name}
                                                                        onChange={(e) => updateParam(index, 'name', e.target.value)}
                                                                        className="font-mono text-sm h-8 border-border/60 focus:border-primary/50 bg-background/50"
                                                                    />
                                                                </div>
                                                                <div className="space-y-1">
                                                                    <Label className="text-[11px] text-muted-foreground font-medium">
                                                                        显示标题 <span className="text-destructive">*</span>
                                                                    </Label>
                                                                    <Input
                                                                        placeholder="例如: 集群名称"
                                                                        value={param.title}
                                                                        onChange={(e) => updateParam(index, 'title', e.target.value)}
                                                                        className="h-8 border-border/60 focus:border-primary/50 bg-background/50"
                                                                    />
                                                                </div>
                                                            </div>

                                                            {/* Row 2: Type + Required */}
                                                            <div className="grid grid-cols-2 gap-3">
                                                                <div className="space-y-1">
                                                                    <Label className="text-[11px] text-muted-foreground font-medium">参数类型</Label>
                                                                    <Select
                                                                        value={param.type}
                                                                        onValueChange={(val) => updateParam(index, 'type', val)}
                                                                    >
                                                                        <SelectTrigger className="h-8 border-border/60 bg-background/50">
                                                                            <SelectValue placeholder="选择类型" />
                                                                        </SelectTrigger>
                                                                        <SelectContent>
                                                                            <SelectItem value="string">
                                                                                <div className="flex items-center gap-2">
                                                                                    <FileText className="w-3.5 h-3.5 text-muted-foreground" />
                                                                                    <span>文本输入</span>
                                                                                </div>
                                                                            </SelectItem>
                                                                            <SelectItem value="select">
                                                                                <div className="flex items-center gap-2">
                                                                                    <List className="w-3.5 h-3.5 text-muted-foreground" />
                                                                                    <span>下拉选择</span>
                                                                                </div>
                                                                            </SelectItem>
                                                                            <SelectItem value="date">
                                                                                <div className="flex items-center gap-2">
                                                                                    <Calendar className="w-3.5 h-3.5 text-muted-foreground" />
                                                                                    <span>日期 (年月日)</span>
                                                                                </div>
                                                                            </SelectItem>
                                                                            <SelectItem value="datetime">
                                                                                <div className="flex items-center gap-2">
                                                                                    <Calendar className="w-3.5 h-3.5 text-muted-foreground" />
                                                                                    <span>日期时间 (精确到秒)</span>
                                                                                </div>
                                                                            </SelectItem>
                                                                            <SelectItem value="component">
                                                                                <div className="flex items-center gap-2">
                                                                                    <Server className="w-3.5 h-3.5 text-muted-foreground" />
                                                                                    <span>节点选择</span>
                                                                                </div>
                                                                            </SelectItem>
                                                                        </SelectContent>
                                                                    </Select>
                                                                </div>

                                                                <div className="space-y-1">
                                                                    <Label className="text-[11px] text-muted-foreground font-medium">必填项</Label>
                                                                    <div className="flex items-center justify-between px-3 h-8 rounded-md border border-border/60 bg-background/50">
                                                                        <span className="text-sm font-medium">是否必填</span>
                                                                        <Switch
                                                                            checked={param.required}
                                                                            onCheckedChange={(checked) => updateParam(index, 'required', checked)}
                                                                            className="scale-90"
                                                                        />
                                                                    </div>
                                                                </div>
                                                            </div>

                                                            {/* Dynamic Conditional Fields with Animation */}
                                                            <AnimatePresence>
                                                                {param.type === 'string' && (
                                                                    <motion.div
                                                                        initial={{ opacity: 0, height: 0 }}
                                                                        animate={{ opacity: 1, height: 'auto' }}
                                                                        exit={{ opacity: 0, height: 0 }}
                                                                        className="overflow-hidden"
                                                                    >
                                                                        <div className="bg-muted/30 p-3 rounded-lg border border-border/40 space-y-2">
                                                                            <Label className="text-[11px] font-semibold flex items-center gap-1.5 text-primary">
                                                                                <FileText className="w-3 h-3" />
                                                                                默认值 (可选)
                                                                            </Label>
                                                                            <Input
                                                                                placeholder="例如: 默认文本"
                                                                                value={param.value || ''}
                                                                                onChange={(e) => updateParam(index, 'value', e.target.value)}
                                                                                className="font-mono text-sm h-8 bg-background border-input focus:border-primary transition-all"
                                                                            />
                                                                            <p className="text-[10px] text-muted-foreground">发送时将使用此默认文本值。</p>
                                                                        </div>
                                                                    </motion.div>
                                                                )}
                                                                {param.type === 'select' && (
                                                                    <motion.div
                                                                        initial={{ opacity: 0, height: 0 }}
                                                                        animate={{ opacity: 1, height: 'auto' }}
                                                                        exit={{ opacity: 0, height: 0 }}
                                                                        className="overflow-hidden"
                                                                    >
                                                                        <ParamOptionsEditor
                                                                            options={param.options}
                                                                            isValueSeparated={param.isValueSeparated}
                                                                            onChange={(opts) => updateParam(index, 'options', opts)}
                                                                            onModeChange={(isSep) => updateParam(index, 'isValueSeparated', isSep)}
                                                                        />
                                                                    </motion.div>
                                                                )}
                                                                {param.type === 'datetime' && (
                                                                    <motion.div
                                                                        initial={{ opacity: 0, height: 0 }}
                                                                        animate={{ opacity: 1, height: 'auto' }}
                                                                        exit={{ opacity: 0, height: 0 }}
                                                                        className="overflow-hidden"
                                                                    >
                                                                        <div className="bg-muted/30 p-3 rounded-lg border border-border/40 space-y-3">
                                                                            <Label className="text-[11px] font-semibold flex items-center gap-1.5 text-primary">
                                                                                <Calendar className="w-3 h-3" />
                                                                                默认时间 (可选)
                                                                            </Label>
                                                                            <div className="grid grid-cols-2 gap-3">
                                                                                <div className="space-y-1">
                                                                                    <Label className="text-[10px] text-muted-foreground">日期偏移 (T+N)</Label>
                                                                                    <Input
                                                                                        placeholder="T+0"
                                                                                        value={(param.value || '').split(' ')[0].startsWith('T') ? (param.value || '').split(' ')[0] : 'T+0'}
                                                                                        onChange={(e) => {
                                                                                            const parts = (param.value || '').split(' ');
                                                                                            const currentTimePart = parts.find(p => p.includes(':')) || '';

                                                                                            const datePart = e.target.value.toUpperCase();
                                                                                            const newVal = `${datePart} ${currentTimePart}`.trim();
                                                                                            updateParam(index, 'value', newVal);
                                                                                        }}
                                                                                        className="font-mono text-xs h-8 bg-background border-input focus:border-primary transition-all"
                                                                                    />
                                                                                </div>
                                                                                <div className="space-y-1">
                                                                                    <Label className="text-[10px] text-muted-foreground">具体时间</Label>
                                                                                    <Input
                                                                                        type="time"
                                                                                        step="1"
                                                                                        value={(param.value || '').includes(':') ? ((param.value || '').split(' ').find(p => p.includes(':')) || '') : ''}
                                                                                        onChange={(e) => {
                                                                                            const timePart = e.target.value;
                                                                                            const parts = (param.value || '').split(' ');
                                                                                            const datePart = parts[0].startsWith('T') ? parts[0] : 'T+0';
                                                                                            const newVal = `${datePart} ${timePart}`;
                                                                                            updateParam(index, 'value', newVal.trim());
                                                                                        }}
                                                                                        className="font-mono text-xs h-8 bg-background border-input focus:border-primary transition-all"
                                                                                    />
                                                                                </div>
                                                                            </div>
                                                                            <div className="flex items-start gap-2 text-[10px] text-muted-foreground bg-background/50 p-2 rounded border border-border/20">
                                                                                <AlertCircle className="w-3 h-3 text-primary/70 mt-0.5 shrink-0" />
                                                                                <p>
                                                                                    组合配置：<strong>日期偏移</strong> (T+0为当天) + <strong>具体时间</strong>。<br />
                                                                                    例如：<code>T+1 09:00:00</code> 代表 <strong>次日早上9点</strong>。
                                                                                </p>
                                                                            </div>
                                                                        </div>
                                                                    </motion.div>
                                                                )}
                                                                {param.type === 'date' && (
                                                                    <motion.div
                                                                        initial={{ opacity: 0, height: 0 }}
                                                                        animate={{ opacity: 1, height: 'auto' }}
                                                                        exit={{ opacity: 0, height: 0 }}
                                                                        className="overflow-hidden"
                                                                    >
                                                                        <div className="bg-muted/30 p-3 rounded-lg border border-border/40 space-y-2">
                                                                            <Label className="text-[11px] font-semibold flex items-center gap-1.5 text-primary">
                                                                                <Calendar className="w-3 h-3" />
                                                                                默认日期偏移 (可选)
                                                                            </Label>
                                                                            <Input
                                                                                placeholder="例如: T+0, T+1"
                                                                                value={param.value || ''}
                                                                                onChange={(e) => {
                                                                                    const val = e.target.value.toUpperCase();
                                                                                    updateParam(index, 'value', val);
                                                                                }}
                                                                                className="font-mono text-sm h-8 bg-background border-input focus:border-primary transition-all"
                                                                            />
                                                                            <div className="flex items-start gap-2 text-[10px] text-muted-foreground bg-background/50 p-2 rounded border border-border/20">
                                                                                <AlertCircle className="w-3 h-3 text-primary/70 mt-0.5 shrink-0" />
                                                                                <p>
                                                                                    支持 <strong>T+N</strong> 格式，其中 T 代表当天。<br />
                                                                                    例如：<strong>T+0</strong> 代表当天，<strong>T+1</strong> 代表次日。
                                                                                </p>
                                                                            </div>
                                                                        </div>
                                                                    </motion.div>
                                                                )}
                                                            </AnimatePresence>

                                                            {/* Insert Action */}
                                                            <div className="flex justify-center pt-2">
                                                                <Button
                                                                    type="button"
                                                                    variant="ghost"
                                                                    size="sm"
                                                                    onClick={() => insertParam(index)}
                                                                    className="w-full h-8 text-xs text-muted-foreground/70 hover:text-primary hover:bg-primary/5 dashed-border border-t border-border/30 rounded-none rounded-b-lg"
                                                                >
                                                                    <Plus className="w-3.5 h-3.5 mr-1" />
                                                                    在此位置后添加参数
                                                                </Button>
                                                            </div>
                                                        </div>
                                                    </motion.div>
                                                ))

                                            ) : (
                                                <motion.div
                                                    initial={{ opacity: 0 }}
                                                    animate={{ opacity: 1 }}
                                                    className="flex flex-col items-center justify-center py-16 px-4 rounded-3xl border-2 border-dashed border-muted-foreground/20 bg-muted/5 group hover:bg-muted/10 transition-colors"
                                                >
                                                    <div className="p-4 bg-muted/50 rounded-full mb-4 group-hover:bg-background group-hover:shadow-sm transition-all">
                                                        <Settings className="w-8 h-8 text-muted-foreground/50 group-hover:text-primary/70" />
                                                    </div>
                                                    <p className="text-base font-medium text-foreground mb-1">尚未定义参数</p>
                                                    <p className="text-sm text-muted-foreground mb-6 max-w-xs text-center">
                                                        添加参数后，用户在发送通知时将能够输入动态内容。
                                                    </p>
                                                    <Button variant="outline" onClick={addParam} className="h-9">
                                                        开始添加 <ChevronRight className="w-4 h-4 ml-1 opacity-50" />
                                                    </Button>
                                                </motion.div>
                                            )}
                                        </AnimatePresence>
                                    </div>
                                    <div className="h-12" /> {/* Bottom spacer */}
                                </div>
                            </div>
                        </ScrollArea>
                    </TabsContent>
                </div>

                <SheetFooter className="px-8 py-5 border-t bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 sm:space-x-4 z-20">
                    <Button
                        type="button"
                        variant="outline"
                        onClick={onClose}
                        disabled={isSubmitting}
                        className="w-28 h-10 hover:bg-muted/50"
                    >
                        取消
                    </Button>
                    {activeTab === 'basic' ? (
                        <Button
                            key="btn-next"
                            type="button"
                            onClick={() => setActiveTab('params')}
                            disabled={isSubmitting}
                            className="w-28 h-10 shadow-lg shadow-primary/20 hover:shadow-primary/30 transition-all"
                        >
                            下一步
                            <ChevronRight className="w-4 h-4 ml-1" />
                        </Button>
                    ) : (
                        <Button
                            key="btn-save"
                            type="submit"
                            disabled={isSubmitting}
                            className="w-28 h-10 shadow-lg shadow-primary/20 hover:shadow-primary/30 transition-all"
                        >
                            {isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                            {isEdit ? '保存修改' : '立即创建'}
                        </Button>
                    )}
                </SheetFooter>
            </Tabs>
        </form >
    );
}

// ----------------------------------------------------------------------
// Helper Components
// ----------------------------------------------------------------------

function ParamOptionsEditor({
    options = [],
    isValueSeparated = false,
    onChange,
    onModeChange
}: {
    options?: ParamOption[],
    isValueSeparated?: boolean,
    onChange: (opts: ParamOption[]) => void,
    onModeChange: (isSeparated: boolean) => void
}) {
    const handleAdd = () => {
        onChange([...options, { label: '新选项', value: 'new_option' }]);
    };

    const handleRemove = (idx: number) => {
        const newOpts = [...options];
        newOpts.splice(idx, 1);
        onChange(newOpts);
    };

    const handleUpdate = (idx: number, field: keyof ParamOption, val: string) => {
        const newOpts = [...options];
        const newOpt = { ...newOpts[idx], [field]: val };

        // If NOT separated (i.e. Synced) and we are updating Label, sync Value
        if (!isValueSeparated && field === 'label') {
            newOpt.value = val;
        }

        newOpts[idx] = newOpt;
        onChange(newOpts);
    };

    return (
        <div className="bg-muted/30 p-3 rounded-lg border border-border/40 space-y-3">
            <div className="flex items-center justify-between">
                <Label className="text-[11px] font-semibold flex items-center gap-1.5 text-primary">
                    <List className="w-3 h-3" />
                    选项列表
                </Label>
                <div className="flex items-center gap-2">
                    <TooltipProvider delayDuration={0}>
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <Button
                                    type="button"
                                    variant="ghost"
                                    size="icon"
                                    className={cn("h-6 w-6", !isValueSeparated ? "text-primary bg-primary/10" : "text-muted-foreground")}
                                    onClick={() => onModeChange(!isValueSeparated)}
                                >
                                    {!isValueSeparated ? <Link className="w-3.5 h-3.5" /> : <Unlink className="w-3.5 h-3.5" />}
                                </Button>
                            </TooltipTrigger>
                            <TooltipContent side="left" className="text-xs">
                                {!isValueSeparated ? '键值同步：修改标签自动更新值' : '键值分离：独立编辑标签与值'}
                            </TooltipContent>
                        </Tooltip>
                    </TooltipProvider>
                    <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={handleAdd}
                        className="h-6 text-[10px] px-2"
                    >
                        <Plus className="w-3 h-3 mr-1" />
                        添加
                    </Button>
                </div>
            </div>

            <div className="space-y-2 max-h-[200px] overflow-y-auto p-1">
                {options.length === 0 && (
                    <div className="text-center py-4 text-xs text-muted-foreground bg-background/50 rounded border border-dashed">
                        暂无选项，请添加
                    </div>
                )}
                {options.map((opt, idx) => (
                    <div key={idx} className="flex gap-2 items-start group/row">
                        <div className="flex-1 space-y-1">
                            <Input
                                value={opt.label}
                                onChange={(e) => handleUpdate(idx, 'label', e.target.value)}
                                placeholder="显示文字 (Label)"
                                className="h-8 text-xs bg-background"
                            />
                        </div>

                        {(isValueSeparated) && (
                            <div className="flex-1 space-y-1 animate-in fade-in slide-in-from-left-2 duration-200">
                                <Input
                                    value={opt.value}
                                    onChange={(e) => handleUpdate(idx, 'value', e.target.value)}
                                    placeholder="实际值"
                                    className="h-8 text-xs bg-background font-mono text-muted-foreground"
                                />
                            </div>
                        )}

                        <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            onClick={() => handleRemove(idx)}
                            className="h-8 w-8 text-muted-foreground hover:text-destructive shrink-0 opacity-50 group-hover/row:opacity-100 transition-opacity"
                        >
                            <Trash2 className="w-3.5 h-3.5" />
                        </Button>
                    </div>
                ))}
            </div>
            <p className="text-[10px] text-muted-foreground">定义下拉菜单的候选项 (Label/Value)。</p>
        </div>
    );
}
