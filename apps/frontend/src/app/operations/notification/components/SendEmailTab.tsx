'use client';

import React, { useState, useMemo, useEffect } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import {
    Send,
    Mail,
    Loader2,
    Users,
    Check,
    AlertCircle,
    LayoutTemplate,
    Sparkles,
    Search,
    ChevronDown,
    X,
    Filter,
    Server,
    ListFilter
} from 'lucide-react';
import { toast } from 'sonner';
import { RobustaAPI } from '@/lib/api';
import {
    PreviewEmailRequest,
    SendEmailRequest,
    AffectedResource
} from '@/types/email';
import { NavyDevice } from '@/types/navy';
import { DeviceSelector } from '@/components/common/DeviceSelector';

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
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog';
import { Badge } from '@/components/ui/badge';
import { Checkbox } from "@/components/ui/checkbox";
import { Textarea } from "@/components/ui/textarea";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from '@/lib/utils';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';

// --- Type Definitions ---
interface ParamDef {
    name: string;
    title: string;
    type: string;
    required?: boolean;
    placeholder?: string;
    value?: string;
}

// --- Helper Components ---

function SearchableSelect<T>({
    items,
    value,
    onChange,
    label,
    placeholder,
    renderItem,
    getItemKey,
    getItemText,
    icon
}: {
    items: T[];
    value: string | undefined;
    onChange: (value: string) => void;
    label: string;
    placeholder?: string;
    renderItem: (item: T) => React.ReactNode;
    getItemKey: (item: T) => string;
    getItemText: (item: T) => string;
    icon?: React.ReactNode;
}) {
    const [open, setOpen] = useState(false);
    const [search, setSearch] = useState('');

    const filteredItems = useMemo(() => {
        if (!search) return items;
        return items.filter(item => getItemText(item).toLowerCase().includes(search.toLowerCase()));
    }, [items, search, getItemText]);

    const selectedItem = useMemo(() => {
        return items.find(i => getItemKey(i) === value);
    }, [items, value, getItemKey]);

    return (
        <div className="space-y-2">
            <label className="text-sm font-medium flex items-center gap-2">
                {icon}
                {label}
                <span className="text-destructive">*</span>
            </label>
            <Popover open={open} onOpenChange={setOpen}>
                <PopoverTrigger asChild>
                    <Button
                        variant="outline"
                        role="combobox"
                        aria-expanded={open}
                        className={cn(
                            "w-full justify-between px-3 text-left font-normal",
                            !value && "text-muted-foreground",
                            "bg-background hover:bg-muted/50"
                        )}
                    >
                        {selectedItem ? (
                            <span className="truncate flex-1">{renderItem(selectedItem)}</span>
                        ) : (
                            <span>{placeholder || "请选择..."}</span>
                        )}
                        <ChevronDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                    </Button>
                </PopoverTrigger>
                <PopoverContent className="w-[var(--radix-popover-trigger-width)] p-0" align="start" side="bottom" avoidCollisions={false}>
                    <div className="flex items-center border-b px-3 py-2">
                        <Search className="mr-2 h-4 w-4 shrink-0 text-muted-foreground opacity-50" />
                        <Input
                            placeholder="搜索..."
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                            className="flex h-9 w-full rounded-md bg-transparent py-3 text-sm outline-none placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50 border-none shadow-none focus-visible:ring-0 focus-visible:ring-offset-0 px-0"
                        />
                    </div>
                    <ScrollArea className="h-60">
                        {filteredItems.length === 0 ? (
                            <div className="py-6 text-center text-sm text-muted-foreground">无匹配结果</div>
                        ) : (
                            <div className="p-1">
                                {filteredItems.map((item) => {
                                    const key = getItemKey(item);
                                    const isSelected = value === key;
                                    return (
                                        <div
                                            key={key}
                                            className={cn(
                                                "relative flex cursor-default select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none transition-colors hover:bg-accent hover:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50",
                                                isSelected && "bg-accent/50 text-accent-foreground"
                                            )}
                                            onClick={() => {
                                                onChange(key);
                                                setOpen(false);
                                            }}
                                        >
                                            <Check
                                                className={cn(
                                                    "mr-2 h-4 w-4",
                                                    isSelected ? "opacity-100" : "opacity-0"
                                                )}
                                            />
                                            {renderItem(item)}
                                        </div>
                                    );
                                })}
                            </div>
                        )}
                    </ScrollArea>
                </PopoverContent>
            </Popover>
        </div>
    );
}

// --- Main Component ---

export function SendEmailTab() {
    // ---------------- State ----------------
    const [selectedTemplateId, setSelectedTemplateId] = useState<string>('');
    const [selectedClusterName, setSelectedClusterName] = useState<string>('');
    const [params, setParams] = useState<Record<string, any>>({});
    const [selectedContacts, setSelectedContacts] = useState<string[]>([]);
    const [customRecipients, setCustomRecipients] = useState<string>('');

    // Node Selection State
    const [selectedNodeNames, setSelectedNodeNames] = useState<Set<string>>(new Set());

    // Preview Dialog
    const [previewHtml, setPreviewHtml] = useState<string>('');
    const [previewSubject, setPreviewSubject] = useState<string>('');
    const [affectedResources, setAffectedResources] = useState<AffectedResource[]>([]);
    const [showPreviewDialog, setShowPreviewDialog] = useState(false);

    // ---------------- Queries ----------------

    // 1. Templates
    const { data: templatesData } = useQuery({
        queryKey: ['email-templates-all'],
        queryFn: () => RobustaAPI.listEmailTemplates(1, 1000),
    });
    const activeTemplates = useMemo(() => templatesData?.data?.filter(t => t.is_enabled) || [], [templatesData]);
    const selectedTemplate = useMemo(() => activeTemplates.find(t => t.id.toString() === selectedTemplateId), [activeTemplates, selectedTemplateId]);

    // 2. Clusters
    const { data: clustersData } = useQuery({
        queryKey: ['clusters-all'],
        queryFn: () => RobustaAPI.getClusters(1, 1000),
    });
    const selectedCluster = useMemo(() => clustersData?.data?.find((c: any) => c.name === selectedClusterName), [clustersData, selectedClusterName]);

    // 3. Contacts
    const { data: contactsData } = useQuery({
        queryKey: ['email-contacts-all'],
        queryFn: () => RobustaAPI.listEmailContacts(1, 100),
    });

    // 4. Devices (Dependent on Cluster)
    const { data: devicesData, isLoading: nodesLoading } = useQuery({
        queryKey: ['cluster-devices', selectedClusterName],
        queryFn: async () => {
            if (!selectedClusterName) return { data: [] };
            const response = await RobustaAPI.queryNavyDevices({
                groups: [{
                    id: 'g1',
                    operator: 'and',
                    blocks: [{
                        id: 'b1',
                        type: 'device',
                        conditionType: 'equal',
                        key: 'cluster',
                        value: selectedClusterName,
                        operator: 'and',
                        isActive: true
                    }]
                }],
                page: 1,
                size: 2000
            });
            return response;
        },
        enabled: !!selectedClusterName,
    });

    const nodesData = useMemo(() => devicesData?.data || [], [devicesData]);

    // ---------------- Logic ----------------

    // Parse params definition from template
    const templateParams = useMemo<ParamDef[]>(() => {
        if (!selectedTemplate?.params?.definitions) return [];
        return selectedTemplate.params.definitions;
    }, [selectedTemplate]);

    // Check if we need to show node selector
    const hasResourceParam = useMemo(() => {
        // Condition: template has a param of type 'resource' OR explicitly named 'nodes'
        return templateParams.some(p => p.type === 'resource' || p.name === 'nodes');
    }, [templateParams]);

    // Helpers
    const toggleContact = (address: string) => {
        if (selectedContacts.includes(address)) {
            setSelectedContacts(selectedContacts.filter(c => c !== address));
        } else {
            setSelectedContacts([...selectedContacts, address]);
        }
    };



    // Construct Payload for Send (Legacy/Direct)
    const constructSendPayload = () => {
        const recipients = [
            ...selectedContacts, // This uses addresses currently, which might be wrong for SendEmailRequest if it expects just strings. Yes it does.
            ...customRecipients.split(/[,\s]+/).map(r => r.trim()).filter(Boolean)
        ];

        // Ensure nodes are mapped correctly (lowercase device ID if applicable)
        const nodesArray = Array.from(selectedNodeNames).map(n => n.toLowerCase());

        const additional = {
            ...params,
            clusterId: selectedCluster?.id,
            nodes: hasResourceParam ? nodesArray : undefined,
        };

        return {
            template_id: parseInt(selectedTemplateId),
            cluster_name: selectedClusterName,
            nodes: nodesArray,
            recipients,
            params: additional,
        };
    };

    // Construct Payload for Build (New Preview)
    const constructBuildPayload = () => {
        // Find the Contact ID for the first selected contact
        // If multiple are selected, we might only be able to use one for this specific backend method
        // Or if 'selectedContacts' contains addresses, we need to find the ID from 'contactsData'
        const contactAddr = selectedContacts[0];
        const contactObj = contactsData?.data?.find((c: any) => c.address.includes(contactAddr));
        const addressId = contactObj ? contactObj.id : 0;

        // Construct 'additional' map
        // User example: {"emailTemplateId":8, "addressId":12, "additional":{"reason":"df", "clusterId":78, "nodes":["..."], "begin":"...", "end":"..."}}

        const nodesArray = Array.from(selectedNodeNames).map(n => n.toLowerCase());

        const additional: Record<string, any> = {
            ...params,
            clusterId: selectedCluster?.id || 0,
            nodes: hasResourceParam ? nodesArray : [],
        };

        // Ensure other params are added
        // The backend expects values in 'additional' map

        return {
            emailTemplateId: parseInt(selectedTemplateId),
            addressId: addressId,
            additional: additional
        };
    };

    // ---------------- Mutations ----------------

    const previewMutation = useMutation({
        mutationFn: (data: any) => RobustaAPI.buildEmail(data),
        onSuccess: (data) => {
            // data matches SendEmailReq structure: { subject, content, addresses, attachFiles... }
            setPreviewSubject(data.subject);
            setPreviewHtml(data.content);
            // We can also extract attachments from data.attachFiles
            // data.attachFiles is []AttachFile { Name, Content (base64) }
            // We'll store it in state if we want to show it, for now we show Subject/Body
            // Convert attachFiles to simpler structure if needed for display

            // NOTE: The previous 'affectedResources' logic was separate. 
            // The BuildEmail method returns the FINAL constructed email including attachments.
            // It does NOT explicitly return affectedResources list unless parsing content or using attachments.
            // We'll reset affectedResources for now as this method is content-centric.
            setAffectedResources([]);

            setShowPreviewDialog(true);
        },
        onError: (error: any) => toast.error('预览生成失败: ' + (error.response?.data?.error || error.message))
    });

    const sendMutation = useMutation({
        mutationFn: (data: SendEmailRequest) => RobustaAPI.sendEmail(data),
        onSuccess: () => {
            toast.success('邮件发送成功');
            // Reset crucial fields
            setParams({});
            setSelectedNodeNames(new Set());
            // Optional: setSelectedTemplateId('');
        },
        onError: (error: any) => toast.error('发送失败: ' + (error.response?.data?.error || error.message))
    });

    const handlePreview = () => {
        if (!selectedTemplateId || !selectedCluster) {
            toast.error("请完善模板和集群选择");
            return;
        }

        if (selectedContacts.length === 0) {
            toast.error("必须选择至少一个预设联系人以获取 addressId");
            return;
        }

        const payload = constructBuildPayload();
        previewMutation.mutate(payload);
    };

    const handleSend = () => {
        if (!selectedTemplateId || !selectedCluster) {
            toast.error("请完善模板和集群选择");
            return;
        }
        const payload = constructSendPayload();
        if (payload.recipients.length === 0) {
            toast.error("请至少填写一个收件人");
            return;
        }
        sendMutation.mutate(payload);
    };

    // ---------------- Render ----------------

    return (
        <div className="w-full mx-auto space-y-8 p-6 pb-20">
            {/* Header */}
            <div className="flex items-center gap-3 border-b pb-4">
                <div className="p-2 bg-primary/10 rounded-lg text-primary">
                    <Send className="w-6 h-6" />
                </div>
                <div>
                    <h2 className="text-xl font-bold">邮件发送</h2>
                    <p className="text-sm text-muted-foreground">基于模板构建并发送通知邮件</p>
                </div>
            </div>

            {/* CARD 1: Basic Configuration (Template & Cluster) */}
            <Card className="shadow-sm hover:shadow-md transition-all border-t-2 border-t-primary/20">
                <CardHeader className="pb-4 border-b">
                    <CardTitle className="text-base flex items-center gap-2">
                        <span className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/10 text-xs font-bold text-primary ring-1 ring-inset ring-primary/20">1</span>
                        基础配置
                    </CardTitle>
                    <CardDescription>选择邮件模板与目标执行集群</CardDescription>
                </CardHeader>
                <CardContent className="pt-6 grid grid-cols-1 md:grid-cols-2 gap-6">
                    <SearchableSelect
                        label="邮件模板"
                        placeholder="搜索并选择模板..."
                        items={activeTemplates}
                        value={selectedTemplateId}
                        onChange={(val) => {
                            setSelectedTemplateId(val);
                            setParams({}); // Reset params on template change
                        }}
                        renderItem={(t) => <span className="font-medium">{t.name}</span>}
                        getItemKey={(t) => t.id.toString()}
                        getItemText={(t) => `${t.name} ${t.title}`}
                        icon={<LayoutTemplate className="w-4 h-4 text-muted-foreground" />}
                    />

                    <SearchableSelect
                        label="目标集群"
                        placeholder="选择执行集群..."
                        items={clustersData?.data || []}
                        value={selectedClusterName}
                        onChange={(val) => {
                            setSelectedClusterName(val);
                            setSelectedNodeNames(new Set()); // Reset nodes on cluster change
                        }}
                        renderItem={(c: any) => (
                            <div className="flex items-center justify-between w-full">
                                <span>{c.name}</span>
                                <Badge variant="secondary" className="text-[10px] h-5">{c.region || 'Region'}</Badge>
                            </div>
                        )}
                        getItemKey={(c: any) => c.name}
                        getItemText={(c: any) => c.name}
                        icon={<Server className="w-4 h-4 text-muted-foreground" />}
                    />
                </CardContent>
            </Card>

            {/* CARD 2: Parameters Configuration (Params + Nodes) */}
            {selectedTemplate && (
                <div className="animate-in fade-in slide-in-from-bottom-4 duration-500 space-y-6">
                    {(templateParams.length > 0 || (hasResourceParam && selectedClusterName)) && (
                        <Card className="shadow-sm hover:shadow-md transition-all border-t-2 border-t-orange-500/20">
                            <CardHeader className="pb-4 border-b">
                                <CardTitle className="text-base flex items-center gap-2">
                                    <span className="flex h-6 w-6 items-center justify-center rounded-full bg-orange-100 text-xs font-bold text-orange-600 ring-1 ring-inset ring-orange-200 dark:bg-orange-900/30 dark:text-orange-300 dark:ring-orange-800">2</span>
                                    参数配置
                                </CardTitle>
                                <CardDescription>配置模板参数及相关资源</CardDescription>
                            </CardHeader>
                            <CardContent className="space-y-8 pt-6">
                                {/* Dynamic Params */}
                                {templateParams.length > 0 && (
                                    <div className="space-y-4">
                                        <div className="flex items-center gap-2 text-sm font-semibold text-muted-foreground border-b pb-2">
                                            <Sparkles className="w-4 h-4" />
                                            模板动态参数
                                        </div>
                                        <div className="grid gap-5">
                                            {templateParams.map((param) => {
                                                if (param.type === 'resource') return null;
                                                return (
                                                    <div key={param.name} className="space-y-2">
                                                        <label className="text-sm font-medium flex items-center gap-1">
                                                            {param.title}
                                                            {param.required && <span className="text-destructive">*</span>}
                                                        </label>
                                                        {param.type === 'datetime' ? (
                                                            <Input
                                                                type="datetime-local"
                                                                value={params[param.name] || ''}
                                                                onChange={(e) => setParams({ ...params, [param.name]: e.target.value })}
                                                            />
                                                        ) : param.type === 'select' ? (
                                                            <Select
                                                                value={params[param.name]}
                                                                onValueChange={(val) => setParams({ ...params, [param.name]: val })}
                                                            >
                                                                <SelectTrigger>
                                                                    <SelectValue placeholder={`选择${param.title}`} />
                                                                </SelectTrigger>
                                                                <SelectContent>
                                                                    <SelectItem value="option1">选项1</SelectItem>
                                                                    <SelectItem value="option2">选项2</SelectItem>
                                                                </SelectContent>
                                                            </Select>
                                                        ) : (
                                                            <Input
                                                                value={params[param.name] || ''}
                                                                onChange={(e) => setParams({ ...params, [param.name]: e.target.value })}
                                                                placeholder={param.placeholder || `请输入${param.title}`}
                                                            />
                                                        )}
                                                    </div>
                                                );
                                            })}
                                        </div>
                                    </div>
                                )}

                                {/* Node Selection */}
                                {hasResourceParam && selectedClusterName && (
                                    <div className="space-y-4 pt-4 border-t">
                                        <div className="flex items-center gap-2 text-sm font-semibold text-muted-foreground border-b pb-2">
                                            <ListFilter className="w-4 h-4" />
                                            节点资源筛选
                                        </div>
                                        <DeviceSelector
                                            devices={nodesData}
                                            selectedCiCodes={selectedNodeNames}
                                            onSelectionChange={setSelectedNodeNames}
                                            isLoading={nodesLoading}
                                        />
                                    </div>
                                )}
                            </CardContent>
                        </Card>
                    )}

                    {/* CARD 3: Recipients */}
                    <Card className="shadow-sm hover:shadow-md transition-all border-t-2 border-t-emerald-500/20">
                        <CardHeader className="pb-4 border-b">
                            <CardTitle className="text-base flex items-center gap-2">
                                <span className="flex h-6 w-6 items-center justify-center rounded-full bg-emerald-100 text-xs font-bold text-emerald-600 ring-1 ring-inset ring-emerald-200 dark:bg-emerald-900/30 dark:text-emerald-300 dark:ring-emerald-800">3</span>
                                收件人设置
                            </CardTitle>
                        </CardHeader>
                        <CardContent className="space-y-4 pt-6">
                            {/* Preset Contacts */}
                            <div className="space-y-2">
                                <label className="text-sm font-medium">预设联系人</label>
                                <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
                                    {contactsData?.data?.map((c: any) => {
                                        const emails = c.address.split(',').map((e: string) => e.trim());
                                        const isActive = emails.some((e: string) => selectedContacts.includes(e));
                                        return (
                                            <div
                                                key={c.id}
                                                onClick={() => emails.forEach((e: string) => toggleContact(e))}
                                                className={cn(
                                                    "cursor-pointer border rounded px-3 py-2 flex items-center gap-2 text-sm transition-all select-none",
                                                    isActive ? "bg-primary/10 border-primary text-primary shadow-sm" : "hover:bg-muted bg-card"
                                                )}
                                            >
                                                <div className="w-5 h-5 rounded-full bg-muted flex items-center justify-center text-[10px] font-bold">
                                                    {c.name[0]}
                                                </div>
                                                <span className="truncate flex-1">{c.name}</span>
                                                {isActive && <Check className="w-3 h-3" />}
                                            </div>
                                        );
                                    })}
                                </div>
                            </div>
                            {/* Manual Input */}
                            <div className="space-y-2">
                                <label className="text-sm font-medium">手动输入 (逗号分隔)</label>
                                <Input
                                    value={customRecipients}
                                    onChange={(e) => setCustomRecipients(e.target.value)}
                                    placeholder="example@domain.com, test@domain.com"
                                />
                            </div>

                            {/* Summary */}
                            <div className="flex flex-wrap gap-2 pt-2">
                                {[...selectedContacts, ...customRecipients.split(/[,\s]+/).filter(Boolean)].map((email, i) => (
                                    <Badge key={i} variant="outline" className="bg-muted/50 pl-2 pr-1 gap-1 group">
                                        {email}
                                        <X
                                            className="w-3 h-3 cursor-pointer hover:text-destructive opacity-50 group-hover:opacity-100"
                                            onClick={() => {
                                                if (selectedContacts.includes(email)) toggleContact(email);
                                            }}
                                        />
                                    </Badge>
                                ))}
                            </div>
                        </CardContent>
                    </Card>

                    {/* Actions */}
                    <div className="flex justify-end gap-3 sticky bottom-0 bg-background/95 backdrop-blur p-4 border-t z-50 shadow-lg rounded-t-lg">
                        <Button variant="outline" size="lg" onClick={handlePreview} disabled={previewMutation.isPending} className="w-40">
                            {previewMutation.isPending && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                            预览邮件
                        </Button>
                        <Button size="lg" onClick={handleSend} disabled={sendMutation.isPending} className="w-40 shadow-lg hover:shadow-primary/20">
                            {sendMutation.isPending ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : <Send className="w-4 h-4 mr-2" />}
                            发送邮件
                        </Button>
                    </div>
                </div>
            )}

            {/* Preview Dialog */}
            <Dialog open={showPreviewDialog} onOpenChange={setShowPreviewDialog}>
                <DialogContent className="max-w-4xl max-h-[80vh] overflow-y-auto">
                    <DialogHeader>
                        <DialogTitle>邮件预览</DialogTitle>
                    </DialogHeader>
                    <div className="space-y-6">
                        <div className="border rounded-lg p-6 bg-card">
                            <h3 className="text-lg font-bold mb-4 border-b pb-2">
                                {previewSubject}
                            </h3>
                            <div
                                className="prose dark:prose-invert max-w-none text-sm"
                                dangerouslySetInnerHTML={{ __html: previewHtml }}
                            />
                        </div>
                        {affectedResources.length > 0 && (
                            <div className="mt-6 border rounded-lg overflow-hidden">
                                <div className="bg-muted px-4 py-2 text-xs font-semibold uppercase text-muted-foreground border-b flex items-center gap-2">
                                    <Server className="w-3 h-3" />
                                    附件资源数据
                                </div>
                                <div className="max-h-60 overflow-y-auto">
                                    <table className="w-full text-sm">
                                        <thead className="bg-muted/50 sticky top-0">
                                            <tr>
                                                <th className="px-3 py-2 text-left">Type</th>
                                                <th className="px-3 py-2 text-left">Name</th>
                                                <th className="px-3 py-2 text-left">Status</th>
                                                <th className="px-3 py-2 text-left">IP</th>
                                            </tr>
                                        </thead>
                                        <tbody className="divide-y">
                                            {affectedResources.map((r, i) => (
                                                <tr key={i} className="hover:bg-muted/50">
                                                    <td className="px-3 py-2">{r.type}</td>
                                                    <td className="px-3 py-2 font-mono text-xs">{r.name}</td>
                                                    <td className="px-3 py-2">{r.status || '-'}</td>
                                                    <td className="px-3 py-2 text-xs">{r.ip || '-'}</td>
                                                </tr>
                                            ))}
                                        </tbody>
                                    </table>
                                </div>
                            </div>
                        )}
                        <div className="flex justify-end pt-4">
                            <Button onClick={() => setShowPreviewDialog(false)}>关闭</Button>
                        </div>
                    </div>
                </DialogContent>
            </Dialog>
        </div>
    );
}
