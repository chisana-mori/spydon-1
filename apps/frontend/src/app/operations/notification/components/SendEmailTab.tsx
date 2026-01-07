'use client';

import React, { useState, useEffect } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import {
    Send,
    Mail,
    Loader2,
    Eye,
    Users,
    Server,
    ChevronRight,
    Check,
    AlertCircle
} from 'lucide-react';
import { toast } from 'sonner';
import { RobustaAPI } from '@/lib/api';
import {
    EmailTemplate,
    EmailContact,
    PreviewEmailRequest,
    SendEmailRequest,
    AffectedResource
} from '@/types/email';

import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select';
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from '@/components/ui/card';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
    DialogFooter,
} from '@/components/ui/dialog';

// 步骤定义
const steps = [
    { id: 'template', label: '选择模板', icon: Mail },
    { id: 'params', label: '填写参数', icon: Server },
    { id: 'recipients', label: '选择收件人', icon: Users },
    { id: 'preview', label: '预览发送', icon: Send },
];

export function SendEmailTab() {
    const [currentStep, setCurrentStep] = useState(0);
    const [selectedTemplateId, setSelectedTemplateId] = useState<string>('');
    const [params, setParams] = useState<Record<string, any>>({});
    const [selectedCluster, setSelectedCluster] = useState<string>('');
    const [selectedNodes, setSelectedNodes] = useState<string[]>([]);
    const [selectedContacts, setSelectedContacts] = useState<string[]>([]);
    const [customRecipients, setCustomRecipients] = useState<string>('');
    const [previewHtml, setPreviewHtml] = useState<string>('');
    const [previewSubject, setPreviewSubject] = useState<string>('');
    const [affectedResources, setAffectedResources] = useState<AffectedResource[]>([]);
    const [showPreviewDialog, setShowPreviewDialog] = useState(false);

    // 获取模板列表
    const { data: templatesData } = useQuery({
        queryKey: ['email-templates-all'],
        queryFn: () => RobustaAPI.listEmailTemplates(1, 100), // 获取所有启用模板
    });

    const activeTemplates = templatesData?.data.filter(t => t.is_enabled) || [];
    const selectedTemplate = activeTemplates.find(t => t.id.toString() === selectedTemplateId);

    // 获取联系人列表
    const { data: contactsData } = useQuery({
        queryKey: ['email-contacts-all'],
        queryFn: () => RobustaAPI.listEmailContacts(1, 100),
    });

    // 获取集群列表（用于资源参数）
    const { data: clustersData } = useQuery({
        queryKey: ['clusters-all'],
        queryFn: () => RobustaAPI.getClusters(),
    });

    // 获取节点列表（当选择了集群后）
    const { data: nodesData } = useQuery({
        queryKey: ['nodes', selectedCluster],
        queryFn: () => RobustaAPI.getClusterNodes(selectedCluster),
        enabled: !!selectedCluster,
    });

    // 预览邮件 Mutation
    const previewMutation = useMutation({
        mutationFn: (data: PreviewEmailRequest) => RobustaAPI.previewEmail(data),
        onSuccess: (data) => {
            setPreviewSubject(data.subject);
            setPreviewHtml(data.html_body);
            setAffectedResources(data.affected_resources);
            setShowPreviewDialog(true);
        },
        onError: (error: any) => {
            toast.error('生成预览失败: ' + (error.response?.data?.error || error.message));
        }
    });

    // 发送邮件 Mutation
    const sendMutation = useMutation({
        mutationFn: (data: SendEmailRequest) => RobustaAPI.sendEmail(data),
        onSuccess: () => {
            toast.success('邮件发送成功');
            // 重置表单
            setCurrentStep(0);
            setParams({});
            setSelectedCluster('');
            setSelectedNodes([]);
        },
        onError: (error: any) => {
            toast.error('发送失败: ' + (error.response?.data?.error || error.message));
        }
    });

    const handlePreview = () => {
        if (!selectedTemplate) return;

        // 合并收件人
        const recipients = [
            ...selectedContacts,
            ...customRecipients.split(',').map(r => r.trim()).filter(r => r)
        ];

        if (recipients.length === 0) {
            toast.error('请至少选择一个收件人');
            return;
        }

        previewMutation.mutate({
            template_id: parseInt(selectedTemplateId),
            cluster_name: selectedCluster,
            nodes: selectedNodes,
            params: params,
        });
    };

    const handleSend = () => {
        if (!selectedTemplate) return;

        const recipients = [
            ...selectedContacts,
            ...customRecipients.split(',').map(r => r.trim()).filter(r => r)
        ];

        sendMutation.mutate({
            template_id: parseInt(selectedTemplateId),
            cluster_name: selectedCluster,
            nodes: selectedNodes,
            params: params,
            recipients: recipients,
        });
    };

    const toggleContact = (address: string) => {
        if (selectedContacts.includes(address)) {
            setSelectedContacts(selectedContacts.filter(c => c !== address));
        } else {
            setSelectedContacts([...selectedContacts, address]);
        }
    };

    const toggleNode = (node: string) => {
        if (selectedNodes.includes(node)) {
            setSelectedNodes(selectedNodes.filter(n => n !== node));
        } else {
            setSelectedNodes([...selectedNodes, node]);
        }
    };

    // 渲染参数输入控件
    const renderParamInput = (param: any) => {
        switch (param.type) {
            case 'resource':
                // 特殊处理资源类型参数
                return (
                    <div className="space-y-4 border rounded-lg p-4 bg-muted/20">
                        <div className="space-y-2">
                            <label className="text-sm font-medium">选择集群</label>
                            <Select value={selectedCluster} onValueChange={setSelectedCluster}>
                                <SelectTrigger>
                                    <SelectValue placeholder="选择集群" />
                                </SelectTrigger>
                                <SelectContent>
                                    {clustersData?.data?.map((cluster: any) => (
                                        <SelectItem key={cluster.name} value={cluster.name}>
                                            {cluster.name}
                                        </SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>
                        </div>

                        {selectedCluster && (
                            <div className="space-y-2">
                                <label className="text-sm font-medium">选择节点 ({selectedNodes.length})</label>
                                <div className="grid grid-cols-2 md:grid-cols-3 gap-2 max-h-48 overflow-y-auto p-2 border rounded bg-background">
                                    {nodesData?.map((node: any) => (
                                        <div
                                            key={node.metadata.name}
                                            className={`flex items-center space-x-2 p-2 rounded cursor-pointer border ${selectedNodes.includes(node.metadata.name) ? 'border-primary bg-primary/10' : 'border-transparent hover:bg-muted'
                                                }`}
                                            onClick={() => toggleNode(node.metadata.name)}
                                        >
                                            <div className={`w-4 h-4 rounded-full border flex items-center justify-center ${selectedNodes.includes(node.metadata.name) ? 'border-primary bg-primary' : 'border-muted-foreground'}`}>
                                                {selectedNodes.includes(node.metadata.name) && <Check className="w-3 h-3 text-white" />}
                                            </div>
                                            <span className="text-sm truncate" title={node.metadata.name}>
                                                {node.metadata.name}
                                            </span>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        )}
                    </div>
                );
            case 'datetime':
                return (
                    <Input
                        type="datetime-local"
                        value={params[param.key]}
                        onChange={(e) => setParams({ ...params, [param.key]: e.target.value })}
                    />
                );
            case 'select':
                // TODO: 实现通用字典选择
                return (
                    <Select
                        value={params[param.key]}
                        onValueChange={(val) => setParams({ ...params, [param.key]: val })}
                    >
                        <SelectTrigger>
                            <SelectValue placeholder={`选择${param.label}`} />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="option1">选项1</SelectItem>
                            <SelectItem value="option2">选项2</SelectItem>
                        </SelectContent>
                    </Select>
                );
            default:
                return (
                    <Input
                        value={params[param.key] || ''}
                        onChange={(e) => setParams({ ...params, [param.key]: e.target.value })}
                        placeholder={param.placeholder || `请输入${param.label}`}
                    />
                );
        }
    };

    const canProceed = () => {
        switch (currentStep) {
            case 0:
                return !!selectedTemplateId;
            case 1:
                // 简单校验必填参数
                if (!selectedTemplate) return false;
                for (const p of selectedTemplate.params) {
                    if (p.required && p.type !== 'resource' && !params[p.key]) return false;
                }
                // 资源参数校验
                const hasResourceParam = selectedTemplate.params.some(p => p.type === 'resource');
                if (hasResourceParam && (!selectedCluster)) return false;
                return true;
            case 2:
                return selectedContacts.length > 0 || customRecipients.length > 0;
            default:
                return true;
        }
    };

    const nextStep = () => {
        if (currentStep < steps.length - 1 && canProceed()) {
            setCurrentStep(currentStep + 1);
            if (currentStep === 2) {
                // 进入预览页时自动生成预览
                handlePreview();
            }
        }
    };

    const prevStep = () => {
        if (currentStep > 0) {
            setCurrentStep(currentStep - 1);
        }
    };

    return (
        <div className="space-y-6">
            {/* 步骤条 */}
            <div className="flex items-center justify-between px-6 py-6 bg-card rounded-lg border shadow-sm">
                {steps.map((step, index) => {
                    const Icon = step.icon;
                    const isActive = index === currentStep;
                    const isCompleted = index < currentStep;

                    return (
                        <div key={step.id} className="flex flex-col items-center relative z-10 w-24">
                            <div className={`
                                w-10 h-10 rounded-full flex items-center justify-center transition-all duration-300
                                ${isActive ? 'bg-primary text-primary-foreground shadow-md scale-110' :
                                    isCompleted ? 'bg-green-500 text-white' : 'bg-muted text-muted-foreground'}
                            `}>
                                {isCompleted ? <Check className="w-5 h-5" /> : <Icon className="w-5 h-5" />}
                            </div>
                            <span className={`mt-2 text-xs font-medium ${isActive ? 'text-primary' : 'text-muted-foreground'}`}>
                                {step.label}
                            </span>
                        </div>
                    );
                })}
                {/* 进度线 */}
                <div className="absolute top-11 left-0 w-full px-20">
                    {/* 这里用 CSS 可以做更好，暂时简化 */}
                </div>
            </div>

            <Card className="border shadow-sm min-h-[500px]">
                <CardHeader>
                    <CardTitle>{steps[currentStep].label}</CardTitle>
                    <CardDescription>
                        {currentStep === 0 && "选择一个邮件模板开始"}
                        {currentStep === 1 && "填写模板所需的参数"}
                        {currentStep === 2 && "选择接收邮件的人员"}
                        {currentStep === 3 && "确认内容并发送"}
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    {/* 步骤 1: 选择模板 */}
                    {currentStep === 0 && (
                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                            {activeTemplates.map(template => (
                                <div
                                    key={template.id}
                                    onClick={() => setSelectedTemplateId(template.id.toString())}
                                    className={`
                                        cursor-pointer border rounded-lg p-5 space-y-3 transition-all hover:shadow-md
                                        ${selectedTemplateId === template.id.toString()
                                            ? 'border-primary bg-primary/5 ring-1 ring-primary'
                                            : 'border-border hover:border-primary/50'}
                                    `}
                                >
                                    <div className="flex items-center justify-between">
                                        <div className="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-lg text-blue-600 dark:text-blue-400">
                                            <Mail className="w-5 h-5" />
                                        </div>
                                        {selectedTemplateId === template.id.toString() && (
                                            <Check className="w-5 h-5 text-primary" />
                                        )}
                                    </div>
                                    <div>
                                        <h3 className="font-semibold">{template.name}</h3>
                                        <p className="text-sm text-muted-foreground mt-1 line-clamp-2">
                                            {template.title}
                                        </p>
                                    </div>
                                    <div className="flex flex-wrap gap-1 mt-2">
                                        {template.params?.map(p => (
                                            <span key={p.key} className="text-[10px] px-2 py-0.5 bg-muted rounded text-muted-foreground">
                                                {p.label}
                                            </span>
                                        ))}
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}

                    {/* 步骤 2: 填写参数 */}
                    {currentStep === 1 && (
                        <div className="max-w-2xl mx-auto space-y-6">
                            {selectedTemplate?.params.map(param => (
                                <div key={param.key} className="space-y-2">
                                    <label className="text-sm font-medium">
                                        {param.label}
                                        {param.required && <span className="text-red-500 ml-1">*</span>}
                                    </label>
                                    {renderParamInput(param)}
                                </div>
                            ))}
                            {selectedTemplate?.params.length === 0 && (
                                <div className="text-center py-10 text-muted-foreground">
                                    该模板无需参数
                                </div>
                            )}
                        </div>
                    )}

                    {/* 步骤 3: 选择收件人 */}
                    {currentStep === 2 && (
                        <div className="space-y-6">
                            <div className="space-y-2">
                                <label className="text-sm font-medium">预设联系人</label>
                                <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
                                    {contactsData?.data.map((contact: any) => (
                                        <div
                                            key={contact.id}
                                            onClick={() => {
                                                const emails = contact.address.split(',').map((e: string) => e.trim()).filter((e: string) => e);
                                                emails.forEach((e: string) => toggleContact(e));
                                            }}
                                            className={`
                                                cursor-pointer border rounded-lg p-3 flex items-center space-x-3 hover:bg-muted/50
                                                ${contact.address.split(',').some((e: string) => selectedContacts.includes(e.trim())) ? 'border-primary' : ''}
                                            `}
                                        >
                                            <div className="w-8 h-8 rounded-full bg-muted flex items-center justify-center">
                                                <span className="text-xs font-bold">{contact.name[0]}</span>
                                            </div>
                                            <div className="flex-1 min-w-0">
                                                <p className="text-sm font-medium truncate">{contact.name}</p>
                                                <p className="text-xs text-muted-foreground truncate">{contact.address}</p>
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            </div>

                            <div className="space-y-2">
                                <label className="text-sm font-medium">手动输入 (逗号分隔)</label>
                                <Input
                                    value={customRecipients}
                                    onChange={(e) => setCustomRecipients(e.target.value)}
                                    placeholder="example@com, test@com"
                                />
                            </div>

                            <div className="p-4 bg-muted/30 rounded-lg">
                                <h4 className="text-sm font-medium mb-2">已选收件人 ({selectedContacts.length + (customRecipients ? customRecipients.split(',').length : 0)})</h4>
                                <div className="flex flex-wrap gap-2">
                                    {selectedContacts.map(email => (
                                        <span key={email} className="text-xs px-2 py-1 bg-primary/10 text-primary rounded-full flex items-center">
                                            {email}
                                            <button onClick={() => toggleContact(email)} className="ml-1 hover:text-red-500">
                                                <span className="sr-only">移除</span>
                                                ×
                                            </button>
                                        </span>
                                    ))}
                                    {customRecipients && customRecipients.split(',').map((email, i) => email.trim() && (
                                        <span key={`custom-${i}`} className="text-xs px-2 py-1 bg-muted text-foreground rounded-full">
                                            {email}
                                        </span>
                                    ))}
                                </div>
                            </div>
                        </div>
                    )}

                    {/* 步骤 4: 预览 */}
                    {currentStep === 3 && (
                        <div className="space-y-6">
                            <div className="border rounded-lg p-6 bg-card">
                                <h3 className="text-lg font-bold mb-4 border-b pb-2">
                                    {previewSubject || <Loader2 className="animate-spin" />}
                                </h3>

                                {previewMutation.isPending ? (
                                    <div className="flex items-center justify-center py-20">
                                        <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
                                    </div>
                                ) : (
                                    <div
                                        className="prose dark:prose-invert max-w-none"
                                        dangerouslySetInnerHTML={{ __html: previewHtml }}
                                    />
                                )}
                            </div>

                            {/* 受影响资源预览 */}
                            {affectedResources.length > 0 && (
                                <div className="mt-6">
                                    <h4 className="text-sm font-medium mb-3 flex items-center space-x-2">
                                        <AlertCircle className="w-4 h-4" />
                                        <span>将作为附件发送的资源数据</span>
                                    </h4>
                                    <div className="border rounded-lg overflow-hidden">
                                        <table className="w-full text-sm">
                                            <thead className="bg-muted">
                                                <tr>
                                                    <th className="px-3 py-2 text-left">类型</th>
                                                    <th className="px-3 py-2 text-left">名称</th>
                                                    <th className="px-3 py-2 text-left">状态</th>
                                                </tr>
                                            </thead>
                                            <tbody className="divide-y">
                                                {affectedResources.slice(0, 5).map((r, i) => (
                                                    <tr key={i}>
                                                        <td className="px-3 py-2">{r.type}</td>
                                                        <td className="px-3 py-2">{r.name}</td>
                                                        <td className="px-3 py-2">{r.status || '-'}</td>
                                                    </tr>
                                                ))}
                                                {affectedResources.length > 5 && (
                                                    <tr>
                                                        <td colSpan={3} className="px-3 py-2 text-center text-muted-foreground">
                                                            ... 共 {affectedResources.length} 项资源
                                                        </td>
                                                    </tr>
                                                )}
                                            </tbody>
                                        </table>
                                    </div>
                                </div>
                            )}
                        </div>
                    )}
                </CardContent>
                <div className="p-6 border-t bg-muted/10 flex justify-between">
                    <Button
                        variant="outline"
                        onClick={prevStep}
                        disabled={currentStep === 0}
                    >
                        上一步
                    </Button>

                    {currentStep === steps.length - 1 ? (
                        <Button
                            onClick={handleSend}
                            disabled={sendMutation.isPending}
                        >
                            {sendMutation.isPending && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                            发送邮件
                        </Button>
                    ) : (
                        <Button
                            onClick={nextStep}
                            disabled={!canProceed()}
                        >
                            下一步
                            <ChevronRight className="w-4 h-4 ml-2" />
                        </Button>
                    )}
                </div>
            </Card>
        </div>
    );
}
