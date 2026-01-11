import React, { useState, useMemo, useEffect } from 'react';
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';
import {
    Loader2,
    Search,
    RotateCcw,
    ChevronLeft,
    ChevronRight,
    ChevronsLeft,
    ChevronsRight,
    Star,
    AlertTriangle,
    Tag,
    Check,
    Laptop
} from 'lucide-react';
import { NavyDevice, DeviceFeatureDetails } from "@/types/navy";
import { cn } from '@/lib/utils';
import {
    HoverCard,
    HoverCardContent,
    HoverCardTrigger,
} from "@/components/ui/hover-card";
import { RobustaAPI } from '@/lib/api';

// --- Helper Functions ---
function parseMultilineInput(input: string): string[] {
    if (!input.trim()) return [];
    let normalized = input.replace(/\r\n/g, '\n').replace(/\r/g, '\n');
    normalized = normalized.replace(/，/g, ',').replace(/；/g, ';');
    const lines = normalized.split(/[\n,;]+/).map(l => l.trim()).filter(l => l.length > 0);
    return lines;
}

// --- Special Device Hover Card ---
// Copied from apps/frontend/src/app/tasks/launch/page.tsx
function SpecialDeviceHoverCard({ device }: { device: NavyDevice }) {
    const [featureDetails, setFeatureDetails] = useState<DeviceFeatureDetails | null>(null);
    const [filterOptions, setFilterOptions] = useState<{ labelKeys: string[]; taintKeys: string[] } | null>(null);
    const [isLoading, setIsLoading] = useState(false);
    const [isOpen, setIsOpen] = useState(false);

    useEffect(() => {
        if (isOpen && !featureDetails && !isLoading && device.ci_code) {
            setIsLoading(true);
            Promise.all([
                RobustaAPI.getNavyDeviceFeatures(device.ci_code),
                RobustaAPI.getNavyFilterOptions()
            ])
                .then(([details, options]) => {
                    setFeatureDetails(details);
                    setFilterOptions(options);
                })
                .catch(err => console.error('加载设备特性失败:', err))
                .finally(() => setIsLoading(false));
        }
    }, [isOpen, device.ci_code, featureDetails, isLoading]);

    const managedLabels = featureDetails?.labels?.filter(label => filterOptions?.labelKeys?.includes(label.key)) || [];
    const managedTaints = featureDetails?.taints?.filter(taint => filterOptions?.taintKeys?.includes(taint.key)) || [];
    const hasAnyFeatures = managedLabels.length > 0 || managedTaints.length > 0;

    return (
        <HoverCard open={isOpen} onOpenChange={setIsOpen}>
            <HoverCardTrigger asChild>
                <div className="cursor-help inline-flex">
                    <Star className="h-4 w-4 text-amber-500 fill-amber-500 flex-shrink-0" />
                </div>
            </HoverCardTrigger>
            <HoverCardContent className="w-80 p-0 overflow-hidden" align="start">
                <div className="bg-amber-50 dark:bg-amber-950/30 p-3 border-b border-amber-100 dark:border-amber-900/50 flex items-start gap-3">
                    <div className="p-2 bg-amber-100/50 dark:bg-amber-900/50 rounded-lg shrink-0">
                        <Star className="h-5 w-5 text-amber-600 dark:text-amber-500 fill-amber-600 dark:fill-amber-500" />
                    </div>
                    <div>
                        <h4 className="text-sm font-semibold text-amber-900 dark:text-amber-100 mb-0.5">特殊设备</h4>
                        <p className="text-xs text-amber-700 dark:text-amber-300/80">此设备已被标记为特殊资产</p>
                    </div>
                </div>
                <div className="p-4 space-y-4">
                    <div className="grid grid-cols-2 gap-3">
                        <div className="space-y-1">
                            <span className="text-[10px] uppercase tracking-wider text-muted-foreground font-semibold">机器用途</span>
                            <p className="text-xs font-medium line-clamp-1">{device.group || '-'}</p>
                        </div>
                        <div className="space-y-1">
                            <span className="text-[10px] uppercase tracking-wider text-muted-foreground font-semibold">关联应用</span>
                            <p className="text-xs font-medium line-clamp-1">{device.app_name || '-'}</p>
                        </div>
                    </div>
                    <div className="space-y-3 pt-2 border-t border-border/50">
                        {isLoading ? (
                            <div className="flex items-center gap-2 py-4 justify-center">
                                <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                                <span className="text-xs text-muted-foreground">加载节点特性...</span>
                            </div>
                        ) : hasAnyFeatures ? (
                            <>
                                {managedLabels.length > 0 && (
                                    <div className="space-y-1.5">
                                        <div className="flex items-center gap-1 text-[10px] font-semibold text-muted-foreground">
                                            <Tag className="h-3 w-3" />
                                            <span>受管理标签 ({managedLabels.length})</span>
                                        </div>
                                        <div className="flex flex-wrap gap-1.5">
                                            {managedLabels.map((l, i) => (
                                                <Badge key={i} variant="secondary" className="px-1.5 h-5 text-[10px] bg-blue-50 text-blue-700 border-none">
                                                    {l.key}: {l.value}
                                                </Badge>
                                            ))}
                                        </div>
                                    </div>
                                )}
                                {managedTaints.length > 0 && (
                                    <div className="space-y-1.5">
                                        <div className="flex items-center gap-1 text-[10px] font-semibold text-muted-foreground">
                                            <AlertTriangle className="h-3 w-3" />
                                            <span>受管理污点 ({managedTaints.length})</span>
                                        </div>
                                        <div className="flex flex-wrap gap-1.5">
                                            {managedTaints.map((t, i) => (
                                                <Badge key={i} variant="secondary" className="px-1.5 h-5 text-[10px] bg-orange-50 text-orange-700 border-none">
                                                    {t.key}={t.value}:{t.effect}
                                                </Badge>
                                            ))}
                                        </div>
                                    </div>
                                )}
                            </>
                        ) : (
                            <p className="text-[10px] text-center py-4 text-muted-foreground italic">暂无额外的受管理特性信息</p>
                        )}
                    </div>
                </div>
            </HoverCardContent>
        </HoverCard>
    );
}

interface DeviceSelectorProps {
    devices: NavyDevice[];
    selectedCiCodes: Set<string>;
    onSelectionChange: (selected: Set<string>) => void;
    isLoading?: boolean;
}

export function DeviceSelector({ devices, selectedCiCodes, onSelectionChange, isLoading = false }: DeviceSelectorProps) {
    const [keyword, setKeyword] = useState('');
    const [inputText, setInputText] = useState('');
    const [currentPage, setCurrentPage] = useState(1);
    const [pageSize, setPageSize] = useState(10);

    // Filter Logic
    const processedDevices = useMemo(() => {
        if (!keyword.trim()) return devices;

        const lines = parseMultilineInput(keyword);
        if (lines.length === 0) return devices;

        const result: NavyDevice[] = [];
        const seenIds = new Set<string>();

        lines.forEach(line => {
            const lowerLine = line.toLowerCase();
            const matches = devices.filter(d => {
                const ip = d.ip?.toLowerCase() || '';
                const ci = d.ci_code?.toLowerCase() || '';
                return ip.includes(lowerLine) || ci.includes(lowerLine);
            });

            if (matches.length > 0) {
                const sortedMatches = [...matches].sort((a, b) => {
                    const aExact = (a.ip?.toLowerCase() === lowerLine || a.ci_code?.toLowerCase() === lowerLine) ? 1 : 0;
                    const bExact = (b.ip?.toLowerCase() === lowerLine || b.ci_code?.toLowerCase() === lowerLine) ? 1 : 0;
                    return bExact - aExact;
                });

                sortedMatches.forEach(match => {
                    if (!seenIds.has(match.ci_code)) {
                        result.push({
                            ...match,
                            originalKeyword: line,
                            isExactMatch: (match.ip?.toLowerCase() === lowerLine || match.ci_code?.toLowerCase() === lowerLine)
                        });
                        seenIds.add(match.ci_code);
                    }
                });
            } else {
                // Add missing/virtual device entry
                result.push({
                    id: -Math.random(),
                    ci_code: line, // Treat input as ci_code for display
                    ip: '',
                    status: 'missing',
                    isVirtual: true,
                    isMissing: true,
                    originalKeyword: line,
                    // Mock required fields
                    arch_type: '', idc: '', room: '', cabinet: '', cabinet_no: '', infra_type: '',
                    is_localization: false, net_zone: '', group: '', appid: '', app_name: '',
                    os_create_time: '', cpu: 0, memory: 0, model: '', kvm_ip: '', os: '',
                    company: '', os_name: '', os_issue: '', os_kernel: '', role: '',
                    k8s_status: '',
                    cluster: '', cluster_id: 0, acceptance_time: '', disk_count: 0,
                    disk_detail: '', network_speed: '', is_special: false, feature_count: 0,
                    created_at: '', updated_at: ''
                } as NavyDevice);
            }
        });
        return result;
    }, [devices, keyword]);

    // Pagination
    const totalPages = Math.ceil(processedDevices.length / pageSize);
    const paginatedDevices = useMemo(() => {
        const start = (currentPage - 1) * pageSize;
        return processedDevices.slice(start, start + pageSize);
    }, [processedDevices, currentPage, pageSize]);

    // Reset page on devices/filter change
    useEffect(() => {
        setCurrentPage(1);
    }, [processedDevices.length]);


    // Handlers
    const handleSearch = () => {
        setKeyword(inputText);
    };

    const handleReset = () => {
        setInputText('');
        setKeyword('');
    };

    const toggleSelect = (ciCode: string) => {
        const newSet = new Set(selectedCiCodes);
        if (newSet.has(ciCode)) {
            newSet.delete(ciCode);
        } else {
            newSet.add(ciCode);
        }
        onSelectionChange(newSet);
    };

    const toggleSelectAll = () => {
        const validDevices = processedDevices.filter(d => !d.isVirtual);
        const allSelected = validDevices.length > 0 && validDevices.every(d => selectedCiCodes.has(d.ci_code));

        const newSet = new Set(selectedCiCodes);
        if (allSelected) {
            validDevices.forEach(d => newSet.delete(d.ci_code));
        } else {
            validDevices.forEach(d => newSet.add(d.ci_code));
        }
        onSelectionChange(newSet);
    };

    const getRowClassName = (device: NavyDevice) => {
        if (!device.isVirtual && selectedCiCodes.has(device.ci_code)) return "bg-blue-50/50 dark:bg-blue-900/10 border-blue-200 dark:border-blue-800";
        if (device.isVirtual) return "opacity-60 bg-muted/30";
        if (device.is_special || (device.app_name && device.app_name.trim() !== '')) {
            return "bg-amber-100/40 dark:bg-amber-900/20 hover:bg-amber-100/60 dark:hover:bg-amber-800/30";
        }
        if (device.cluster && device.cluster.trim() !== '') {
            return "bg-emerald-100/30 dark:bg-emerald-950/15 hover:bg-emerald-100/60 dark:hover:bg-emerald-900/30";
        }
        return "hover:bg-muted/50";
    };

    const PaginationControls = () => {
        if (totalPages <= 1) return null;
        return (
            <div className="flex items-center justify-between p-2 mt-2 border-t">
                <div className="text-sm text-muted-foreground">
                    第 {currentPage} 页 / 共 {totalPages} 页
                </div>
                <div className="flex items-center gap-1">
                    <Button variant="outline" size="sm" className="h-8 w-8 p-0" onClick={() => setCurrentPage(1)} disabled={currentPage === 1}><ChevronsLeft className="h-4 w-4" /></Button>
                    <Button variant="outline" size="sm" className="h-8 w-8 p-0" onClick={() => setCurrentPage(c => Math.max(1, c - 1))} disabled={currentPage === 1}><ChevronLeft className="h-4 w-4" /></Button>
                    <Button variant="outline" size="sm" className="h-8 w-8 p-0" onClick={() => setCurrentPage(c => Math.min(totalPages, c + 1))} disabled={currentPage === totalPages}><ChevronRight className="h-4 w-4" /></Button>
                    <Button variant="outline" size="sm" className="h-8 w-8 p-0" onClick={() => setCurrentPage(totalPages)} disabled={currentPage === totalPages}><ChevronsRight className="h-4 w-4" /></Button>
                </div>
            </div>
        );
    };

    return (
        <div className="space-y-4">
            {/* Filter Area */}
            <div className="space-y-2">
                <div className="flex items-center justify-between">
                    <label className="text-sm font-medium">
                        节点筛选 (支持IP/CI_CODE多行输入)
                        {processedDevices.length > 0 && keyword && (
                            <span className="ml-2 text-xs text-muted-foreground font-normal">
                                (匹配规则: 精确匹配优先, 未匹配项显示在底部)
                            </span>
                        )}
                    </label>
                </div>
                <div className="flex gap-2 items-start">
                    <Textarea
                        placeholder="输入IP or CI_CODE, 支持多行批量筛选..."
                        className="flex-1 h-20 min-h-[80px] font-mono text-sm resize-y"
                        value={inputText}
                        onChange={(e) => setInputText(e.target.value)}
                    />
                    <div className="flex flex-col gap-2 shrink-0">
                        <Button onClick={handleSearch} disabled={!inputText.trim() && !keyword.trim()} size="sm" className="h-[38px] px-4">
                            <Search className="w-4 h-4 mr-2" />
                            搜索
                        </Button>
                        <Button variant="outline" onClick={handleReset} disabled={!inputText && !keyword} size="sm" className="h-[38px] px-4">
                            <RotateCcw className="w-4 h-4 mr-2" />
                            重置
                        </Button>
                    </div>
                </div>
            </div>

            {/* Selection Status */}
            <div className="flex items-center justify-between bg-muted/20 px-3 py-2 rounded-lg border">
                <div className="text-xs text-muted-foreground flex items-center gap-2">
                    <span>显示 {processedDevices.length} 个节点</span>
                    {selectedCiCodes.size > 0 && (
                        <Badge variant="secondary" className="text-[10px] font-normal">
                            已选 {selectedCiCodes.size} 个
                        </Badge>
                    )}
                </div>
                <div className="flex items-center gap-2">
                    <select
                        className="h-7 w-20 rounded border border-input bg-transparent px-2 py-1 text-xs shadow-sm ring-offset-background focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                        value={pageSize}
                        onChange={(e) => {
                            setPageSize(Number(e.target.value));
                            setCurrentPage(1);
                        }}
                    >
                        <option value="10">10 / 页</option>
                        <option value="20">20 / 页</option>
                        <option value="50">50 / 页</option>
                        <option value="100">100 / 页</option>
                    </select>
                </div>
            </div>

            {/* Table */}
            <div className="rounded-md border bg-card">
                <Table>
                    <TableHeader className="bg-muted/50">
                        <TableRow className="hover:bg-transparent border-b-border/60">
                            <TableHead className="w-[40px] pl-3">
                                <Checkbox
                                    checked={processedDevices.length > 0 && processedDevices.filter(d => !d.isVirtual).every(d => selectedCiCodes.has(d.ci_code))}
                                    onCheckedChange={toggleSelectAll}
                                    disabled={processedDevices.length === 0}
                                />
                            </TableHead>
                            <TableHead className="w-[200px]">设备ID</TableHead>
                            <TableHead className="w-[120px]">IP</TableHead>
                            <TableHead>IDC/Room</TableHead>
                            <TableHead>App</TableHead>
                            <TableHead className="w-[100px]">状态</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {isLoading ? (
                            <TableRow>
                                <TableCell colSpan={6} className="h-32 text-center">
                                    <div className="flex flex-col items-center justify-center text-muted-foreground">
                                        <Loader2 className="w-6 h-6 animate-spin mb-2" />
                                        <span>加载中...</span>
                                    </div>
                                </TableCell>
                            </TableRow>
                        ) : paginatedDevices.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={6} className="h-32 text-center text-muted-foreground">
                                    暂无数据，请确认集群或筛选条件
                                </TableCell>
                            </TableRow>
                        ) : (
                            paginatedDevices.map((device) => {
                                const isSelected = selectedCiCodes.has(device.ci_code);
                                const isMissing = device.isMissing;
                                return (
                                    <TableRow
                                        key={device.isVirtual ? device.ci_code + Math.random() : device.ci_code}
                                        className={cn(
                                            "transition-colors data-[state=selected]:bg-muted",
                                            getRowClassName(device)
                                        )}
                                        onClick={() => !isMissing && toggleSelect(device.ci_code)}
                                    >
                                        <TableCell className="pl-3 py-2 cursor-pointer">
                                            <Checkbox
                                                checked={isSelected}
                                                onCheckedChange={() => !isMissing && toggleSelect(device.ci_code)}
                                                onClick={(e) => e.stopPropagation()}
                                                disabled={isMissing}
                                                className={cn(isMissing && "opacity-50")}
                                            />
                                        </TableCell>
                                        <TableCell className="py-2 font-mono text-xs">
                                            <div className="flex items-center gap-2">
                                                {device.isVirtual ? (
                                                    <div className="h-4 w-4 shrink-0" />
                                                ) : device.is_special ? (
                                                    <SpecialDeviceHoverCard device={device} />
                                                ) : (
                                                    <Laptop className="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
                                                )}
                                                <div className="min-w-0 flex-1">
                                                    <div className="flex items-center gap-2">
                                                        <p className={cn("truncate", device.isVirtual && "text-muted-foreground italic")}>
                                                            {device.isVirtual ? device.originalKeyword : device.ci_code}
                                                            {device.isExactMatch && !device.isVirtual && device.ci_code?.toLowerCase() === device.originalKeyword?.toLowerCase() && (
                                                                <Check className="inline-block w-3.5 h-3.5 ml-1.5 text-emerald-600 dark:text-emerald-400 align-middle mb-0.5" strokeWidth={3} />
                                                            )}
                                                        </p>
                                                    </div>
                                                </div>
                                                <div className="flex items-center gap-1.5 shrink-0">
                                                    {isMissing && <Badge variant="destructive" className="h-4 px-1 text-[9px] rounded-sm">Missing</Badge>}
                                                </div>
                                            </div>
                                        </TableCell>
                                        <TableCell className="py-2 font-mono text-xs text-muted-foreground">
                                            {device.isVirtual ? (
                                                <span className="text-muted-foreground">-</span>
                                            ) : (
                                                <div className="flex items-center gap-2">
                                                    <code className="text-[13px] font-medium bg-muted/50 px-2 py-0.5 rounded-md border border-border/10">
                                                        {device.ip}
                                                    </code>
                                                    {device.isExactMatch && !device.isVirtual && device.ip?.toLowerCase() === device.originalKeyword?.toLowerCase() && (
                                                        <Check className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400" strokeWidth={3} />
                                                    )}
                                                </div>
                                            )}
                                        </TableCell>
                                        <TableCell className="py-2 text-xs text-muted-foreground">
                                            {isMissing ? '-' : `${device.idc || '-'}/${device.room || '-'}`}
                                        </TableCell>
                                        <TableCell className="py-2 text-xs">
                                            {device.app_name ? (
                                                <Badge variant="outline" className="font-normal border-primary/20 text-primary-foreground bg-primary/80 hover:bg-primary/90">
                                                    {device.app_name}
                                                </Badge>
                                            ) : '-'}
                                        </TableCell>
                                        <TableCell className="py-2">
                                            {isMissing ? (
                                                <span className="text-xs text-muted-foreground">-</span>
                                            ) : (
                                                <Badge
                                                    variant="secondary"
                                                    className={cn(
                                                        "h-5 text-[10px] font-normal px-1.5 border-none",
                                                        device.status?.toLowerCase() === 'active'
                                                            ? "bg-emerald-50 text-emerald-700"
                                                            : "bg-gray-100 text-gray-600"
                                                    )}
                                                >
                                                    {device.status || 'Unknown'}
                                                </Badge>
                                            )}
                                        </TableCell>
                                    </TableRow>
                                );
                            })
                        )}
                    </TableBody>
                </Table>
            </div>
            <PaginationControls />
        </div>
    );
}
