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
    ListFilter,
    Download,
    Paperclip,
    Settings,
    CheckCircle,
    Eye,
    Edit
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
import { HtmlPreview } from '@/components/common/HtmlPreview';

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
    Sheet,
    SheetContent,
    SheetDescription,
    SheetHeader,
    SheetTitle,
    SheetFooter,
} from '@/components/ui/sheet';
import { Badge } from '@/components/ui/badge';
import { SimpleEditor } from "@/components/tiptap-templates/simple/simple-editor";
import { Checkbox } from "@/components/ui/checkbox";
import { Textarea } from "@/components/ui/textarea";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from '@/lib/utils';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import {
    Command,
    CommandEmpty,
    CommandGroup,
    CommandInput,
    CommandItem,
    CommandList,
} from "@/components/ui/command";
import { Calendar } from "@/components/ui/calendar";
import { Label } from "@/components/ui/label";

// --- Type Definitions ---
interface ParamDef {
    name: string;
    title: string;
    type: string;
    required?: boolean;
    placeholder?: string;
    value?: string;
    defaultTime?: string;
    resourceType?: string;
    options?: { label: string; value: string }[];
}

// --- Helper Components ---

function MultiSearchableSelect<T>({
    items,
    selectedValues,
    onSelectionChange,
    label,
    placeholder,
    renderItem,
    getItemKey,
    getItemText,
    icon
}: {
    items: T[];
    selectedValues: Set<string>;
    onSelectionChange: (values: Set<string>) => void;
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

    const handleSelect = (key: string) => {
        const newSelected = new Set(selectedValues);
        if (newSelected.has(key)) {
            newSelected.delete(key);
        } else {
            newSelected.add(key);
        }
        onSelectionChange(newSelected);
    };

    const selectedCount = selectedValues.size;

    // Get selected items to display their names
    const selectedItems = useMemo(() => {
        return items.filter(item => selectedValues.has(getItemKey(item)));
    }, [items, selectedValues, getItemKey]);

    return (
        <div className="space-y-2">
            <label className="text-sm font-medium flex items-center gap-2">
                {icon}
                {label}
            </label>
            <Popover open={open} onOpenChange={setOpen}>
                <PopoverTrigger asChild>
                    <Button
                        variant="outline"
                        role="combobox"
                        aria-expanded={open}
                        className={cn(
                            "w-full justify-between px-3 text-left font-normal h-9",
                            selectedCount === 0 && "text-muted-foreground",
                            "bg-background hover:bg-muted/50"
                        )}
                    >
                        <div className="flex items-center gap-1.5 truncate flex-1">
                            {selectedCount > 0 ? (
                                <div className="flex items-center gap-1.5 truncate">
                                    <span className="truncate text-sm">
                                        {selectedItems.map(item => getItemText(item)).join(', ')}
                                    </span>
                                    <Badge variant="secondary" className="rounded-sm px-1.5 py-0 font-normal text-xs shrink-0">
                                        {selectedCount}
                                    </Badge>
                                </div>
                            ) : (
                                <span>{placeholder || "请选择..."}</span>
                            )}
                        </div>
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
                                    const isSelected = selectedValues.has(key);
                                    return (
                                        <div
                                            key={key}
                                            className={cn(
                                                "relative flex cursor-default select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none transition-colors hover:bg-accent hover:text-accent-foreground",
                                                isSelected && "bg-accent/50 text-accent-foreground"
                                            )}
                                            onClick={() => handleSelect(key)}
                                        >
                                            <div className={cn(
                                                "mr-2 flex h-4 w-4 items-center justify-center rounded-sm border border-primary",
                                                isSelected ? "bg-primary text-primary-foreground" : "opacity-50 [&_svg]:invisible"
                                            )}>
                                                <Check className={cn("h-4 w-4")} />
                                            </div>
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

// --- Parameter Indicator Component ---

function formatParamValue(value: any, type: string): string {
    if (value === undefined || value === null) return '';
    if (type === 'date' || type === 'datetime') {
        try {
            const d = new Date(value);
            return d.toLocaleString('zh-CN', {
                year: 'numeric',
                month: '2-digit',
                day: '2-digit',
                hour: type === 'datetime' ? '2-digit' : undefined,
                minute: type === 'datetime' ? '2-digit' : undefined
            });
        } catch {
            return String(value);
        }
    }

    if (typeof value === 'object') {
        return JSON.stringify(value);
    }

    const strVal = String(value);
    if (strVal.length > 20) {
        return strVal.substring(0, 20) + '...';
    }

    return strVal;
}

interface ParameterIndicatorProps {
    selectedTemplate: any;
    selectedClusterName: string;
    params: Record<string, any>;
    selectedNodeNames: Set<string>;
    visibleParams: ParamDef[];
    hasResourceParam: boolean;
}

function ParameterIndicator({
    selectedTemplate,
    selectedClusterName,
    params,
    selectedNodeNames,
    visibleParams,
    hasResourceParam
}: ParameterIndicatorProps) {
    // Calculate completion
    const completion = useMemo(() => {
        let total = 0;
        let filled = 0;

        // 1. Template
        total++;
        if (selectedTemplate) filled++;

        // 2. Cluster
        total++;
        if (selectedClusterName) filled++;

        // 3. Params
        if (visibleParams.length > 0) {
            total += visibleParams.length;
            visibleParams.forEach(p => {
                const val = params[p.name];
                if (val !== undefined && val !== '' && val !== null) {
                    filled++;
                }
            });
        }

        // 4. Nodes
        if (hasResourceParam) {
            total++;
            if (selectedNodeNames.size > 0) filled++;
        }

        return total > 0 ? Math.round((filled / total) * 100) : 0;
    }, [selectedTemplate, selectedClusterName, visibleParams, params, hasResourceParam, selectedNodeNames]);

    return (
        <div className="flex flex-col w-full">
            <Card className="shadow-lg border-2 border-primary/20 overflow-hidden bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
                <CardHeader className="bg-primary/5 pb-3 py-3">
                    <CardTitle className="text-sm flex items-center gap-2">
                        <Settings className="w-4 h-4" />
                        配置概览
                    </CardTitle>
                    <div className="mt-2 space-y-1">
                        <div className="flex items-center justify-between text-xs">
                            <span className="text-muted-foreground">完成度</span>
                            <span className={cn("font-medium", completion === 100 && "text-emerald-600")}>{completion}%</span>
                        </div>
                        <div className="h-1.5 bg-muted rounded-full overflow-hidden">
                            <div
                                className={cn(
                                    "h-full transition-all duration-500",
                                    completion === 100 ? "bg-emerald-500" : "bg-primary"
                                )}
                                style={{ width: `${completion}%` }}
                            />
                        </div>
                    </div>
                </CardHeader>
                <CardContent className="space-y-4 pt-4 p-3">
                    {/* 1. Basic Config */}
                    <div className="space-y-2">
                        <div className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">基础信息</div>
                        <div className="bg-muted/30 rounded-md p-2 space-y-2 border">
                            {selectedTemplate ? (
                                <div className="flex items-start gap-2">
                                    <LayoutTemplate className="w-3.5 h-3.5 text-primary mt-0.5 shrink-0" />
                                    <div className="min-w-0">
                                        <div className="text-[10px] text-muted-foreground">模板</div>
                                        <div className="text-xs font-medium truncate leading-tight" title={selectedTemplate.name}>{selectedTemplate.name}</div>
                                    </div>
                                </div>
                            ) : (
                                <div className="flex items-center gap-2 text-muted-foreground italic text-xs">
                                    <AlertCircle className="w-3.5 h-3.5" />
                                    未选择模板
                                </div>
                            )}

                            {selectedClusterName ? (
                                <div className="flex items-start gap-2 pt-1 border-t border-dashed border-muted-foreground/20">
                                    <Server className="w-3.5 h-3.5 text-primary mt-0.5 shrink-0" />
                                    <div className="min-w-0">
                                        <div className="text-[10px] text-muted-foreground">集群</div>
                                        <div className="text-xs font-medium truncate leading-tight">{selectedClusterName}</div>
                                    </div>
                                </div>
                            ) : (
                                <div className="flex items-center gap-2 text-muted-foreground italic text-xs border-t border-dashed border-muted-foreground/20 pt-1">
                                    <AlertCircle className="w-3.5 h-3.5" />
                                    未选择集群
                                </div>
                            )}
                        </div>
                    </div>

                    {/* 2. Params */}
                    {visibleParams.length > 0 && (
                        <div className="space-y-2">
                            <div className="text-xs font-semibold text-muted-foreground uppercase tracking-wider flex items-center justify-between">
                                动态参数
                                <span className="text-[10px] font-normal bg-muted px-1.5 rounded-full">{visibleParams.length}</span>
                            </div>
                            <div className="space-y-1.5">
                                {visibleParams.map(param => {
                                    const value = params[param.name];
                                    const isFilled = value !== undefined && value !== '' && value !== null;
                                    return (
                                        <div key={param.name} className="group">
                                            <div className="flex items-center justify-between mb-0.5">
                                                <span className="text-[11px] text-muted-foreground truncate max-w-[180px]" title={param.title}>{param.title}</span>
                                                {isFilled ? (
                                                    <CheckCircle className="w-3 h-3 text-emerald-500 shrink-0" />
                                                ) : (
                                                    <div className="w-1.5 h-1.5 rounded-full bg-destructive/50 shrink-0" />
                                                )}
                                            </div>
                                            <div className={cn(
                                                "text-xs px-2 py-1.5 rounded border transition-colors break-all",
                                                isFilled
                                                    ? "bg-emerald-50/50 border-emerald-100/50 text-emerald-700 dark:bg-emerald-950/20 dark:border-emerald-900/50 dark:text-emerald-400"
                                                    : "bg-muted/20 border-border/50 text-muted-foreground italic"
                                            )}>
                                                {isFilled ? formatParamValue(value, param.type) : "待输入..."}
                                            </div>
                                        </div>
                                    )
                                })}
                            </div>
                        </div>
                    )}

                    {/* 3. Resources */}
                    {hasResourceParam && (
                        <div className="space-y-2">
                            <div className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">资源选择</div>
                            <div className="bg-muted/30 rounded-md p-2 border">
                                <div className="flex items-center justify-between mb-2">
                                    <span className="text-xs flex items-center gap-1.5">
                                        <ListFilter className="w-3.5 h-3.5" />
                                        节点列表
                                    </span>
                                    <Badge variant={selectedNodeNames.size > 0 ? "default" : "secondary"} className="h-4.5 px-1.5 text-[10px]">
                                        {selectedNodeNames.size}
                                    </Badge>
                                </div>
                                {selectedNodeNames.size > 0 ? (
                                    <div className="flex flex-wrap gap-1">
                                        {Array.from(selectedNodeNames).slice(0, 5).map(node => (
                                            <Badge key={node} variant="outline" className="text-[10px] px-1 h-4 bg-background">
                                                {node}
                                            </Badge>
                                        ))}
                                        {selectedNodeNames.size > 5 && (
                                            <Badge variant="secondary" className="text-[10px] px-1 h-4">
                                                +{selectedNodeNames.size - 5}
                                            </Badge>
                                        )}
                                    </div>
                                ) : (
                                    <div className="text-xs text-muted-foreground italic text-center py-2">
                                        请在左侧筛选添加节点
                                    </div>
                                )}
                            </div>
                        </div>
                    )}
                </CardContent>
            </Card>
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

    // Preview Dialog & State
    const [previewHtml, setPreviewHtml] = useState<string>('');
    const [previewVersion, setPreviewVersion] = useState(0);
    const [previewSubject, setPreviewSubject] = useState<string>('');
    const [finalRecipients, setFinalRecipients] = useState<string[]>([]);
    const [attachFiles, setAttachFiles] = useState<any[]>([]); // Using any for simplicity as per existing usage pattern or define interface
    const [affectedResources, setAffectedResources] = useState<AffectedResource[]>([]);
    const [showPreviewDialog, setShowPreviewDialog] = useState(false);
    const [viewMode, setViewMode] = useState<'preview' | 'edit'>('preview');

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

    // Flattened contacts for MultiSelect.
    // Since one Contact entity has 'address' string which can be comma-separated,
    // we should treat each underlying email as a selectable item OR treat the Contact as a group.
    // The previous logic was: toggleContact(address) where address is a specific email.
    // Let's adapt MultiSearchableSelect to handle 'Contact' entities but when selected, we add all their emails.
    // Or better, let's just list the Contact Entities in the dropdown, and when selected, we add their addresses to the payload.
    // BUT, the 'selectedContacts' state currently holds raw email strings.
    // To make it simple for the UI, let's keep selectedContacts as email strings, but the UI selects Contact Entities.

    // Actually, to support the requirement "contact selection via dropdown",
    // we should list the Contact objects. When one is selected, we consider it "selected".
    // However, the backend needs 'addressId' for preview.
    // And 'recipients' for send.

    // Let's derive 'selectedContactIds' from 'selectedContacts' emails if possible, or key by Contact ID.
    // Current state: selectedContacts is string[] of emails.
    // Let's change strategy: use a Set<string> for selected Contact IDs (as strings) for the UI state,
    // and derive the emails array for the payload.
    // Wait, the existing code: toggleContact(address).
    // Let's refactor:
    // We will select Contact Objects.
    // 'selectedContactValues' will be a Set of Contact IDs (string).

    const [selectedContactIds, setSelectedContactIds] = useState<Set<string>>(new Set());

    // Sync legacy selectedContacts (emails) with selectedContactIds?
    // Or just use selectedContactIds as the source of truth for "Preset Contacts".
    // We also have 'customRecipients'.

    const selectedEmailsFromContacts = useMemo(() => {
        const emails: string[] = [];
        selectedContactIds.forEach(idStr => {
            const contact = contactsData?.data?.find((c: any) => c.id.toString() === idStr);
            if (contact) {
                contact.address.split(',').map((e: string) => emails.push(e.trim()));
            }
        });
        return emails;
    }, [selectedContactIds, contactsData]);

    // We need to keep 'selectedContacts' for compatibility with existing payload construction or update it.
    // The existing 'selectedContacts' was just emails.
    // Let's update 'selectedContacts' to be 'selectedEmailsFromContacts'.

    useEffect(() => {
        // Keep selectedContacts in sync for payload construction
        setSelectedContacts(selectedEmailsFromContacts);
    }, [selectedEmailsFromContacts]);


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

    // Filter out params that should not be displayed in the form
    // (clusterId is handled by global cluster selector, component/resource by DeviceSelector)
    const visibleParams = useMemo(() => {
        return templateParams.filter(param => {
            // Skip component/resource types (handled separately)
            if (param.type === 'component' || param.type === 'resource') return false;
            // Skip clusterId parameters (handled by global cluster selector)
            if (param.name === 'clusterId' || param.name === 'cluster_id') return false;
            return true;
        });
    }, [templateParams]);

    // Check if we need to show node selector
    const hasResourceParam = useMemo(() => {
        // Condition: template has a param of type 'component' OR explicitly named 'nodes'
        // Support both 'component' (new) and 'resource' (legacy) for backward compatibility
        return templateParams.some(p => p.type === 'component' || p.type === 'resource' || p.name === 'nodes');
    }, [templateParams]);

    // Check if parameter configuration card should be shown
    const showParamConfigCard = useMemo(() => {
        return selectedTemplate && (visibleParams.length > 0 || (hasResourceParam && selectedClusterName));
    }, [selectedTemplate, visibleParams.length, hasResourceParam, selectedClusterName]);

    // Construct Payload for Send (Legacy/Direct)
    const constructSendPayload = () => {
        // Use finalRecipients if available (from Preview flow), otherwise fallback to selection
        // Actually, the new flow requires Preview first for final recipients modification.
        // But if we want to call it directly (though UI hides it), we can default to selection.

        let recipients = finalRecipients;
        if (recipients.length === 0) {
            recipients = [
                ...selectedEmailsFromContacts,
                ...customRecipients.split(/[,\s]+/).map(r => r.trim()).filter(Boolean)
            ];
        }

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
            subject: previewSubject,
            body: previewHtml,
            attachFiles: attachFiles // Include attachments in send payload
        };
    };

    // Construct Payload for Build (New Preview)
    const constructBuildPayload = () => {
        // Find the Contact ID for the first selected contact
        // Ideally we should send all selected contact IDs, but the backend preview API might expect one addressId for context?
        // The MailGenReq has 'addressId' (singular). This implies the preview is generated for "a user".
        // We'll just pick the first one.

        let addressId = 0;
        if (selectedContactIds.size > 0) {
            const firstId = Array.from(selectedContactIds)[0];
            addressId = parseInt(firstId);
        }

        const nodesArray = Array.from(selectedNodeNames).map(n => n.toLowerCase());

        const additional: Record<string, any> = {
            ...params,
            clusterId: selectedCluster?.id || 0,
            nodes: hasResourceParam ? nodesArray : [],
        };

        return {
            emailTemplateId: parseInt(selectedTemplateId),
            addressId: addressId,
            additional: additional
        };
    };

    // ---------------- Mutations ----------------

    const previewMutation = useMutation({
        mutationFn: (data: any) => RobustaAPI.previewEmail(data),
        onSuccess: (data: any) => {
            setPreviewSubject(data.subject);

            // Fix: Handle HTML content that might be wrapped in markdown code blocks or escaped
            let cleanHtml = data.content || '';

            // 1. Remove Markdown code blocks if present
            if (cleanHtml.trim().startsWith('```')) {
                cleanHtml = cleanHtml.replace(/^```(html)?\s*/i, '').replace(/\s*```$/, '');
            }

            // 2. Unescape HTML entities if it looks like escaped HTML (e.g. &lt;div&gt;)
            // This prevents the HTML from being rendered as a code block of tags
            if (cleanHtml.includes('&lt;') && cleanHtml.includes('&gt;')) {
                const doc = new DOMParser().parseFromString(cleanHtml, 'text/html');
                const decoded = doc.documentElement.textContent;
                if (decoded) cleanHtml = decoded;
            }

            setPreviewHtml(cleanHtml);
            setAttachFiles(data.attachFiles || []);
            setPreviewVersion(v => v + 1);

            // Populate initial final recipients from current selection + custom
            // Note: The preview response might contain recipients if the backend calculated them,
            // but usually buildEmail just returns content.
            // We'll initialize from our current selection state.
            const currentRecipients = [
                ...selectedEmailsFromContacts,
                ...customRecipients.split(/[,\s]+/).map(r => r.trim()).filter(Boolean)
            ];
            // Start with unique recipients
            setFinalRecipients(Array.from(new Set(currentRecipients)));

            setAffectedResources([]);
            setShowPreviewDialog(true);
            setViewMode('preview'); // Default to preview mode for perfect display
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

        if (selectedContactIds.size === 0) {
            toast.error("请选择至少一个预设联系人以获取预览上下文");
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

    // Toggle contact function for chips
    const toggleContactId = (id: string) => {
        const newSet = new Set(selectedContactIds);
        if (newSet.has(id)) {
            newSet.delete(id);
        } else {
            newSet.add(id);
        }
        setSelectedContactIds(newSet);
    }


    // ---------------- Render ----------------

    return (
        <div className="p-6 pb-20 w-full mx-auto max-w-[1600px]">
            {/* Header */}
            <div className="flex items-center gap-3 border-b pb-4 mb-6">
                <div className="p-2 bg-primary/10 rounded-lg text-primary">
                    <Send className="w-6 h-6" />
                </div>
                <div>
                    <h2 className="text-xl font-bold">邮件发送</h2>
                    <p className="text-sm text-muted-foreground">基于模板构建并发送通知邮件</p>
                </div>
            </div>

            {/* Two Column Layout */}
            <div className="flex gap-6">
                {/* Left Content Area */}
                <div className="flex-1 space-y-8 min-w-0">

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
                                    // Auto-fill default values
                                    const template = activeTemplates.find(t => t.id.toString() === val);
                                    const initialParams: Record<string, any> = {};
                                    if (template?.params?.definitions) {
                                        template.params.definitions.forEach((def: any) => {
                                            // Use 'value' field for default values (supports date, datetime, input, select)
                                            if (def.value) {
                                                initialParams[def.name] = def.value;
                                            }
                                            // Legacy: support defaultTime for backward compatibility
                                            else if (def.type === 'datetime' && def.defaultTime) {
                                                const today = new Date().toISOString().split('T')[0];
                                                // Ensure time format matches datetime-local requirements
                                                let time = def.defaultTime;
                                                if (time.length === 5) time += ':00'; // HH:mm -> HH:mm:ss
                                                initialParams[def.name] = `${today}T${time}`;
                                            }
                                        });
                                    }
                                    setParams(initialParams);
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
                            {(visibleParams.length > 0 || (hasResourceParam && selectedClusterName)) && (
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
                                        {visibleParams.length > 0 && (
                                            <div className="space-y-4">
                                                <div className="flex items-center gap-2 text-sm font-semibold text-muted-foreground border-b pb-2">
                                                    <Sparkles className="w-4 h-4" />
                                                    模板动态参数
                                                </div>
                                                <div className="grid gap-5">
                                                    {visibleParams.map((param) => (
                                                        <div key={param.name} className="space-y-2">
                                                            <label className="text-sm font-medium flex items-center gap-1">
                                                                {param.title}
                                                                {param.required && <span className="text-destructive">*</span>}
                                                            </label>
                                                            {param.type === 'datetime' || param.type === 'date' ? (
                                                                <DateTimePicker
                                                                    value={params[param.name]}
                                                                    onChange={(val) => setParams({ ...params, [param.name]: val })}
                                                                />
                                                            ) : param.type === 'select' ? (
                                                                <EditableCombobox
                                                                    value={params[param.name]}
                                                                    onChange={(val) => setParams({ ...params, [param.name]: val })}
                                                                    options={param.options || []}
                                                                    placeholder={`选择或输入${param.title}`}
                                                                />
                                                            ) : (
                                                                <Input
                                                                    value={params[param.name] || ''}
                                                                    onChange={(e) => setParams({ ...params, [param.name]: e.target.value })}
                                                                    placeholder={param.placeholder || `请输入${param.title}`}
                                                                />
                                                            )}
                                                        </div>
                                                    ))}
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
                                    {/* Preset Contacts - MultiSearchableSelect */}
                                    <MultiSearchableSelect
                                        label="预设联系人 (多选)"
                                        placeholder="搜索并添加联系人..."
                                        items={contactsData?.data || []}
                                        selectedValues={selectedContactIds}
                                        onSelectionChange={setSelectedContactIds}
                                        renderItem={(c: any) => (
                                            <div className="flex flex-col">
                                                <span className="font-medium">{c.name}</span>
                                                <span className="text-xs text-muted-foreground truncate">{c.address}</span>
                                            </div>
                                        )}
                                        getItemKey={(c: any) => c.id.toString()}
                                        getItemText={(c: any) => c.name}
                                        icon={<Users className="w-4 h-4 text-muted-foreground" />}
                                    />

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
                                        {selectedEmailsFromContacts.map((email, i) => (
                                            <Badge key={`c-${i}`} variant="secondary" className="pl-2 pr-1 gap-1">
                                                {email}
                                            </Badge>
                                        ))}
                                        {customRecipients.split(/[,\s]+/).filter(Boolean).map((email, i) => (
                                            <Badge key={`m-${i}`} variant="outline" className="bg-muted/50 pl-2 pr-1 gap-1 group">
                                                {email}
                                                <X
                                                    className="w-3 h-3 cursor-pointer hover:text-destructive opacity-50 group-hover:opacity-100"
                                                    onClick={() => {
                                                        // Manual removal relies on text editing, so we can't easily remove specific manual entry here
                                                        // without parsing/reconstructing the string.
                                                        // But for contacts, we can add X to remove.
                                                        // Let's keep it simple for now.
                                                    }}
                                                />
                                            </Badge>
                                        ))}
                                    </div>
                                </CardContent>
                            </Card>

                            {/* Actions */}
                            {/* Actions */}
                            <div className="flex justify-end gap-3 sticky bottom-0 bg-background/95 backdrop-blur p-4 border-t z-40 shadow-lg rounded-t-lg">
                                <Button
                                    variant="default"
                                    size="lg"
                                    onClick={handlePreview}
                                    disabled={previewMutation.isPending}
                                    className="w-full md:w-48 shadow-lg hover:shadow-primary/20"
                                >
                                    {previewMutation.isPending ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : <Sparkles className="w-4 h-4 mr-2" />}
                                    生成预览 & 确认
                                </Button>
                            </div>
                        </div>
                    )}

                    {/* Preview Sheet (Drawer) */}
                    <Sheet open={showPreviewDialog} onOpenChange={setShowPreviewDialog}>
                        <SheetContent side="right" className="w-full sm:max-w-2xl flex flex-col p-0 gap-0">
                            <SheetHeader className="p-6 border-b bg-muted/20">
                                <SheetTitle className="flex items-center gap-2">
                                    <Mail className="w-5 h-5 text-primary" />
                                    邮件发送确认
                                </SheetTitle>
                                <SheetDescription>
                                    请仔细核对邮件内容、收件人及附件信息，确认无误后发送。
                                </SheetDescription>
                            </SheetHeader>

                            <div className="flex-1 overflow-y-auto p-6 space-y-8">
                                {/* 1. Subject */}
                                <div className="space-y-2">
                                    <label className="text-sm font-medium text-muted-foreground">邮件标题</label>
                                    <Input
                                        value={previewSubject}
                                        onChange={(e) => setPreviewSubject(e.target.value)}
                                        className="font-bold text-lg h-auto py-2"
                                    />
                                </div>

                                {/* 2. Recipients Management */}
                                <div className="space-y-3">
                                    <label className="text-sm font-medium text-muted-foreground flex items-center justify-between">
                                        <span>收件人列表 ({finalRecipients.length})</span>
                                        <span className="text-xs font-normal text-muted-foreground">可在此处增删收件人</span>
                                    </label>

                                    <div className="flex flex-wrap gap-2 p-3 bg-muted/30 rounded-lg border">
                                        {finalRecipients.map((email, i) => (
                                            <Badge key={i} variant="secondary" className="pl-2 pr-1 gap-1 group bg-background border">
                                                {email}
                                                <X
                                                    className="w-3 h-3 cursor-pointer hover:text-destructive opacity-50 group-hover:opacity-100 transition-opacity"
                                                    onClick={() => {
                                                        setFinalRecipients(finalRecipients.filter(r => r !== email));
                                                    }}
                                                />
                                            </Badge>
                                        ))}
                                        {finalRecipients.length === 0 && (
                                            <span className="text-sm text-muted-foreground italic">暂无收件人</span>
                                        )}
                                    </div>

                                    <div className="flex gap-2">
                                        <Input
                                            placeholder="添加额外收件人 (user@example.com)"
                                            className="flex-1"
                                            onKeyDown={(e) => {
                                                if (e.key === 'Enter') {
                                                    const val = e.currentTarget.value.trim();
                                                    if (val && !finalRecipients.includes(val)) {
                                                        setFinalRecipients([...finalRecipients, val]);
                                                        e.currentTarget.value = '';
                                                    }
                                                }
                                            }}
                                        />
                                        <Button variant="secondary" size="icon" onClick={(e) => {
                                            const input = e.currentTarget.previousElementSibling as HTMLInputElement;
                                            const val = input.value.trim();
                                            if (val && !finalRecipients.includes(val)) {
                                                setFinalRecipients([...finalRecipients, val]);
                                                input.value = '';
                                            }
                                        }}>
                                            <Check className="w-4 h-4" />
                                        </Button>
                                    </div>
                                </div>


                                {/* 3. Content Preview */}
                                <div className="space-y-3">
                                    <div className="flex items-center justify-between">
                                        <label className="text-sm font-medium text-muted-foreground">邮件正文预览</label>
                                        <div className="flex items-center p-1 bg-muted rounded-lg border">
                                            <button
                                                onClick={() => setViewMode('preview')}
                                                className={cn(
                                                    "flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-all",
                                                    viewMode === 'preview'
                                                        ? "bg-background text-foreground shadow-sm"
                                                        : "text-muted-foreground hover:text-foreground hover:bg-background/50"
                                                )}
                                            >
                                                <Eye className="w-3.5 h-3.5" />
                                                预览
                                            </button>
                                            <button
                                                onClick={() => setViewMode('edit')}
                                                className={cn(
                                                    "flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-all",
                                                    viewMode === 'edit'
                                                        ? "bg-background text-foreground shadow-sm"
                                                        : "text-muted-foreground hover:text-foreground hover:bg-background/50"
                                                )}
                                            >
                                                <Edit className="w-3.5 h-3.5" />
                                                编辑
                                            </button>
                                        </div>
                                    </div>

                                    <div className="border rounded-lg bg-card shadow-sm overflow-hidden min-h-[400px]">
                                        {viewMode === 'preview' ? (
                                            <HtmlPreview
                                                html={previewHtml}
                                                className="w-full min-h-[400px]"
                                                style={{ height: '500px' }}
                                            />
                                        ) : (
                                            <SimpleEditor
                                                key={previewVersion} // Remount when version changes
                                                initialContent={previewHtml}
                                                onUpdate={(_, html) => setPreviewHtml(html)}
                                                variant="embed"
                                                embedHeight={500}
                                                className="prose dark:prose-invert max-w-none text-sm w-full"
                                            />
                                        )}
                                    </div>
                                </div>

                                {/* 4. Attachments */}
                                {attachFiles.length > 0 && (
                                    <div className="space-y-3">
                                        <label className="text-sm font-medium text-muted-foreground flex items-center gap-2">
                                            <Paperclip className="w-4 h-4" />
                                            附件列表 ({attachFiles.length})
                                        </label>
                                        <div className="grid grid-cols-1 gap-2">
                                            {attachFiles.map((file, i) => (
                                                <div key={i} className="flex items-center justify-between p-3 border rounded-lg bg-card hover:bg-muted/50 transition-colors group">
                                                    <div className="flex items-center gap-3 overflow-hidden">
                                                        <div className="w-8 h-8 rounded bg-primary/10 flex items-center justify-center text-primary font-bold text-xs uppercase shrink-0">
                                                            {file.Name.split('.').pop() || 'FILE'}
                                                        </div>
                                                        <span className="text-sm font-medium truncate">{file.Name}</span>
                                                    </div>
                                                    <Button
                                                        variant="ghost"
                                                        size="sm"
                                                        className="opacity-0 group-hover:opacity-100 transition-opacity"
                                                        onClick={() => {
                                                            // Fix: Ensure download as .xlsx if extension is missing
                                                            let fileName = file.Name;
                                                            if (!fileName.includes('.')) {
                                                                fileName += '.xlsx';
                                                            }

                                                            const blob = base64ToBlob(file.Content);
                                                            const url = URL.createObjectURL(blob);
                                                            const a = document.createElement('a');
                                                            a.href = url;
                                                            a.download = fileName;
                                                            document.body.appendChild(a);
                                                            a.click();
                                                            document.body.removeChild(a);
                                                            URL.revokeObjectURL(url);
                                                        }}
                                                    >
                                                        <Download className="w-4 h-4 mr-2" />
                                                        下载
                                                    </Button>
                                                </div>
                                            ))}
                                        </div>
                                    </div>
                                )}
                            </div>

                            <SheetFooter className="p-6 border-t bg-muted/20 sm:justify-between items-center">
                                <div className="text-xs text-muted-foreground hidden sm:block">
                                    预览无误后请点击发送
                                </div>
                                <div className="flex gap-3 w-full sm:w-auto">
                                    <Button variant="outline" onClick={() => setShowPreviewDialog(false)} className="flex-1 sm:flex-none">
                                        取消
                                    </Button>
                                    <Button
                                        onClick={handleSend}
                                        disabled={sendMutation.isPending || finalRecipients.length === 0}
                                        className="flex-1 sm:flex-none w-32"
                                    >
                                        {sendMutation.isPending ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : <Send className="w-4 h-4 mr-2" />}
                                        确认发送
                                    </Button>
                                </div>
                            </SheetFooter>
                        </SheetContent>
                    </Sheet>
                </div>

                {/* Right Sidebar - Always at bottom */}
                <div className="hidden xl:block w-[300px] shrink-0 self-end">
                    <ParameterIndicator
                        selectedTemplate={selectedTemplate}
                        selectedClusterName={selectedClusterName}
                        params={params}
                        selectedNodeNames={selectedNodeNames}
                        visibleParams={visibleParams}
                        hasResourceParam={hasResourceParam}
                    />
                </div>
            </div>
        </div>
    );
}

// Helper for Base64 download
const base64ToBlob = (base64: string) => {
    const byteCharacters = atob(base64);
    const byteNumbers = new Array(byteCharacters.length);
    for (let i = 0; i < byteCharacters.length; i++) {
        byteNumbers[i] = byteCharacters.charCodeAt(i);
    }
    const byteArray = new Uint8Array(byteNumbers);
    return new Blob([byteArray], { type: 'application/octet-stream' });
};

function EditableCombobox({
    value,
    onChange,
    options = [],
    placeholder
}: {
    value: string | undefined;
    onChange: (val: string) => void;
    options: { label: string; value: string }[];
    placeholder?: string;
}) {
    const [open, setOpen] = useState(false);
    const [searchQuery, setSearchQuery] = useState("");

    // Determine display text
    const matchedOption = options.find(o => o.value === value);
    // If matched, show label. If not matcher (custom value), show value itself.
    const displayText = matchedOption ? matchedOption.label : value;

    // Reset search query when closed
    useEffect(() => {
        if (!open) {
            setSearchQuery("");
        }
    }, [open]);

    const filteredOptions = options.filter(opt =>
        opt.label.toLowerCase().includes(searchQuery.toLowerCase()) ||
        opt.value.toLowerCase().includes(searchQuery.toLowerCase())
    );

    // Check if the current search query exactly matches an existing option (label or value)
    const exactMatch = options.some(o => o.label === searchQuery || o.value === searchQuery);

    const handleSelect = (val: string) => {
        onChange(val);
        setOpen(false);
    };

    return (
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
                    <span className="truncate">{displayText || placeholder || "请选择或输入..."}</span>
                    <ChevronDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                </Button>
            </PopoverTrigger>
            <PopoverContent className="w-[var(--radix-popover-trigger-width)] p-0" align="start">
                <div className="flex flex-col max-h-[300px]">
                    <div className="p-2 border-b">
                        <Input
                            placeholder="搜索或输入新值..."
                            className="h-8"
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                        // Prevent auto-select behavior issues by handling enter in parent or specific handlers if needed
                        />
                    </div>
                    <ScrollArea className="flex-1 overflow-y-auto">
                        <div className="p-1">
                            {/* Option to use custom value if search query exists and isn't an exact match */}
                            {searchQuery && !exactMatch && (
                                <div
                                    className="relative flex cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none hover:bg-accent hover:text-accent-foreground text-blue-600 dark:text-blue-400"
                                    onClick={() => handleSelect(searchQuery)}
                                >
                                    <Check className="mr-2 h-4 w-4 opacity-0" />
                                    使用 "{searchQuery}"
                                </div>
                            )}

                            {filteredOptions.length === 0 && !searchQuery && (
                                <div className="py-6 text-center text-sm text-muted-foreground">暂无选项</div>
                            )}

                            {filteredOptions.length === 0 && searchQuery && exactMatch && (
                                <div className="py-4 text-center text-sm text-muted-foreground">
                                    已存在匹配项
                                </div>
                            )}

                            {filteredOptions.map((opt) => (
                                <div
                                    key={opt.value}
                                    className={cn(
                                        "relative flex cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none hover:bg-accent hover:text-accent-foreground",
                                        value === opt.value && "bg-accent text-accent-foreground"
                                    )}
                                    onClick={() => handleSelect(opt.value)}
                                >
                                    <Check className={cn("mr-2 h-4 w-4", value === opt.value ? "opacity-100" : "opacity-0")} />
                                    {opt.label}
                                </div>
                            ))}
                        </div>
                    </ScrollArea>
                </div>
            </PopoverContent>
        </Popover>
    );
}

export function DateTimePicker({
    value,
    onChange,
}: {
    value?: string
    onChange: (value: string) => void
}) {
    const [open, setOpen] = useState(false)
    const [date, setDate] = useState<Date | undefined>(
        value ? new Date(value) : undefined
    )
    const [timeValue, setTimeValue] = useState<string>(
        value ? extractTime(value) : "00:00:00"
    )

    useEffect(() => {
        if (value) {
            const d = new Date(value);
            if (!isNaN(d.getTime())) {
                setDate(d);
                setTimeValue(extractTime(value));
            }
        }
    }, [value]);

    function extractTime(dateStr: string) {
        const d = new Date(dateStr);
        if (isNaN(d.getTime())) return "00:00:00";
        return d.toTimeString().split(' ')[0];
    }

    const handleDateSelect = (newDate: Date | undefined) => {
        setDate(newDate)
        setOpen(false)
        if (newDate) {
            // combine date and time
            const combined = new Date(newDate);
            const [hours, minutes, seconds] = timeValue.split(':').map(Number);
            combined.setHours(hours || 0);
            combined.setMinutes(minutes || 0);
            combined.setSeconds(seconds || 0);
            onChange(toLocISOString(combined));
        }
    }

    const handleTimeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const newTime = e.target.value;
        setTimeValue(newTime);
        if (date) {
            const combined = new Date(date);
            const [hours, minutes, seconds] = newTime.split(':').map(Number);
            combined.setHours(hours || 0);
            combined.setMinutes(minutes || 0);
            combined.setSeconds(seconds || 0);
            onChange(toLocISOString(combined));
        }
    }

    // Helper to format date to "YYYY-MM-DDTHH:mm:ss" local time, compatible with datetime-local input expectations or backend
    // But standard ISO "YYYY-MM-DDTHH:mm:ss.sssZ" might be what is expected?
    // Previous input type="datetime-local" produces "YYYY-MM-DDTHH:mm".
    // Let's stick to a format that can be parsed back by new Date().
    // And usually backend expects ISO string.
    function toLocISOString(d: Date) {
        const pad = (n: number) => n < 10 ? '0' + n : n;
        return d.getFullYear() + '-' +
            pad(d.getMonth() + 1) + '-' +
            pad(d.getDate()) + 'T' +
            pad(d.getHours()) + ':' +
            pad(d.getMinutes()) + ':' +
            pad(d.getSeconds());
    }

    return (
        <div className="flex gap-4">
            <div className="flex flex-col gap-2">
                <Label className="px-1 text-xs text-muted-foreground">
                    日期
                </Label>
                <Popover open={open} onOpenChange={setOpen}>
                    <PopoverTrigger asChild>
                        <Button
                            variant="outline"
                            className={cn("w-[160px] h-10 justify-between font-normal text-left", !date && "text-muted-foreground")}
                        >
                            {date ? date.toLocaleDateString() : "请选择日期"}
                            <ChevronDown className="h-4 w-4 opacity-50" />
                        </Button>
                    </PopoverTrigger>
                    <PopoverContent className="w-auto p-0" align="start">
                        <Calendar
                            mode="single"
                            selected={date}
                            onSelect={handleDateSelect}
                            initialFocus
                            captionLayout="dropdown"
                            fromYear={2000}
                            toYear={new Date().getFullYear() + 10}
                        />
                    </PopoverContent>
                </Popover>
            </div>
            <div className="flex flex-col gap-2">
                <Label className="px-1 text-xs text-muted-foreground">
                    时间
                </Label>
                <Input
                    type="time"
                    step="1"
                    value={timeValue}
                    onChange={handleTimeChange}
                    className="w-[120px] h-10"
                />
            </div>
        </div>
    )
}
