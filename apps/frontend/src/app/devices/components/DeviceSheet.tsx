'use client'

import { useState, useEffect } from 'react'
import {
    Sheet,
    SheetContent,
    SheetHeader,
    SheetTitle,
} from "@/components/ui/sheet"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
    Cpu,
    HardDrive,
    MemoryStick,
    Network,
    Server,
    Tag,
    AlertTriangle,
    Pencil,
    Save,
    X,
    Copy,
    Check,
    Star,
    GalleryVerticalEnd,
    MapPin,
    Globe,
    Shield,
    Database,
    ExternalLink,
    CloudOff,
} from 'lucide-react'
import { NavyDevice, DeviceFeatureDetails } from "@/types/navy"
import { cn } from "@/lib/utils"
import RobustaAPI from '@/lib/api'
import { toast } from 'sonner'

interface DeviceSheetProps {
    device: NavyDevice | null
    open: boolean
    onOpenChange: (open: boolean) => void
    onUpdate?: () => void
}

function InfoRow({ label, value, copyable }: { label: string; value: string | number | null | undefined; copyable?: boolean }) {
    const [copied, setCopied] = useState(false)

    const handleCopy = () => {
        if (value) {
            navigator.clipboard.writeText(String(value))
            setCopied(true)
            setTimeout(() => setCopied(false), 2000)
        }
    }

    return (
        <div className="flex items-start justify-between py-2 border-b border-dashed last:border-0">
            <span className="text-sm text-muted-foreground">{label}</span>
            <div className="flex items-center gap-1">
                <span className="text-sm font-medium text-right">{value || '-'}</span>
                {copyable && value && (
                    <button onClick={handleCopy} className="text-muted-foreground hover:text-foreground">
                        {copied ? <Check className="h-3.5 w-3.5 text-emerald-500" /> : <Copy className="h-3.5 w-3.5" />}
                    </button>
                )}
            </div>
        </div>
    )
}

export function DeviceSheet({ device, open, onOpenChange, onUpdate }: DeviceSheetProps) {
    const [isEditing, setIsEditing] = useState(false)
    const [editGroup, setEditGroup] = useState('')
    const [editRole, setEditRole] = useState('')
    const [isSaving, setIsSaving] = useState(false)
    const [features, setFeatures] = useState<DeviceFeatureDetails | null>(null)
    const [loadingFeatures, setLoadingFeatures] = useState(false)

    useEffect(() => {
        if (device) {
            setEditGroup(device.group || '')
            setEditRole(device.role || '')
            setIsEditing(false)
            loadFeatures(device.id)
        }
    }, [device])

    const loadFeatures = async (deviceId: number) => {
        setLoadingFeatures(true)
        try {
            const data = await RobustaAPI.getNavyDeviceFeatures(String(deviceId))
            setFeatures(data)
        } catch (err) {
            console.error('加载特性失败:', err)
        } finally {
            setLoadingFeatures(false)
        }
    }

    const handleSave = async () => {
        if (!device) return
        setIsSaving(true)
        try {
            if (editGroup !== device.group) {
                await RobustaAPI.updateNavyDeviceGroup(device.id, editGroup)
            }
            if (editRole !== device.role) {
                await RobustaAPI.updateNavyDeviceRole(device.id, editRole)
            }
            toast.success('保存成功')
            setIsEditing(false)
            onUpdate?.()
        } catch (err) {
            toast.error('保存失败')
        } finally {
            setIsSaving(false)
        }
    }

    if (!device) return null

    return (
        <Sheet open={open} onOpenChange={onOpenChange}>
            <SheetContent className="w-[540px] sm:max-w-[600px] p-0 border-l border-border/40 shadow-2xl flex flex-col h-full">
                {/* Modern Header Design */}
                <div className="flex-none p-6 pb-2 bg-gradient-to-b from-muted/50 to-background border-b z-20">
                    <div className="flex items-start justify-between mb-4">
                        <div className="flex gap-4">
                            <div className="h-14 w-14 rounded-xl bg-primary/10 flex items-center justify-center border border-primary/20 shadow-sm flex-shrink-0">
                                <Server className="h-7 w-7 text-primary" />
                            </div>
                            <div className="space-y-1.5">
                                <div className="flex items-center gap-2">
                                    <h2 className="text-xl font-bold tracking-tight font-mono">{device.ci_code}</h2>
                                    {device.is_special && (
                                        <Badge variant="secondary" className="bg-amber-100 text-amber-700 hover:bg-amber-100 border-amber-200 gap-1 px-2 h-5">
                                            <Star className="h-3 w-3 fill-amber-700" />
                                            特殊设备
                                        </Badge>
                                    )}
                                </div>
                                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                                    <div className="flex items-center gap-1.5 bg-muted px-2 py-0.5 rounded-md border">
                                        <Network className="h-3.5 w-3.5" />
                                        <span className="font-mono text-foreground font-medium">{device.ip}</span>
                                    </div>
                                    <span className="text-border">|</span>
                                    <span>{device.idc}</span>
                                    <span className="text-border">|</span>
                                    <span className={cn(
                                        "flex items-center gap-1.5",
                                        device.status === 'online' || device.status === '活跃' ? "text-emerald-600 font-medium" : ""
                                    )}>
                                        <div className={cn(
                                            "h-2 w-2 rounded-full",
                                            device.status === 'online' || device.status === '活跃' ? "bg-emerald-500 animate-pulse" : "bg-muted-foreground"
                                        )} />
                                        {device.status || '未知状态'}
                                    </span>
                                </div>
                            </div>
                        </div>
                        <div className="flex gap-2">
                            {!isEditing ? (
                                <Button variant="ghost" size="icon" onClick={() => setIsEditing(true)} className="h-8 w-8 hover:bg-muted">
                                    <Pencil className="h-4 w-4 text-muted-foreground" />
                                </Button>
                            ) : (
                                <div className="flex items-center gap-2 bg-background shadow-sm border rounded-lg p-1">
                                    <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => setIsEditing(false)}>
                                        <X className="h-4 w-4" />
                                    </Button>
                                    <Separator orientation="vertical" className="h-4" />
                                    <Button size="icon" className="h-7 w-7" onClick={handleSave} disabled={isSaving}>
                                        <Save className="h-3.5 w-3.5" />
                                    </Button>
                                </div>
                            )}
                        </div>
                    </div>
                </div>

                <Tabs defaultValue="overview" className="flex-1 flex flex-col min-h-0 bg-muted/5">
                    <div className="px-6 border-b bg-background">
                        <TabsList className="h-10 w-full justify-start rounded-none bg-transparent p-0 gap-6">
                            <NavItem value="overview" label="设备概览" icon={Server} />
                            <NavItem value="hardware" label="硬件配置" icon={Cpu} />
                            <NavItem value="k8s" label="Kubernetes" icon={GalleryVerticalEnd} />
                        </TabsList>
                    </div>

                    <ScrollArea className="flex-1">
                        <div className="p-6">
                            <TabsContent value="overview" className="mt-0 space-y-6 animate-in fade-in-50 slide-in-from-bottom-2 duration-300">
                                {/* Business Info */}
                                <div className="space-y-3">
                                    <SectionTitle icon={MapPin} title="位置与归属" />
                                    <div className="grid grid-cols-2 gap-3">
                                        <InfoCard label="IDC 机房" value={device.idc} subValue={`${device.room || '-'} / ${device.cabinet || '-'}`} />
                                        <InfoCard label="网络区域" value={device.net_zone} icon={Globe} />
                                        <InfoCard label="业务 AppID" value={device.appid} className="col-span-2" />
                                    </div>
                                </div>

                                {/* Role Info With Edit Support */}
                                <div className="space-y-3">
                                    <SectionTitle icon={Shield} title="角色信息" />
                                    <div className="grid grid-cols-1 gap-3">
                                        {isEditing ? (
                                            <>
                                                <div className="space-y-2 p-3 bg-background border rounded-lg border-dashed border-primary/50">
                                                    <Label className="text-xs text-primary font-medium">机器用途</Label>
                                                    <Input
                                                        value={editGroup}
                                                        onChange={(e) => setEditGroup(e.target.value)}
                                                        className="h-8 shadow-sm"
                                                    />
                                                </div>
                                                <div className="space-y-2 p-3 bg-background border rounded-lg border-dashed border-primary/50">
                                                    <Label className="text-xs text-primary font-medium">集群角色</Label>
                                                    <Input
                                                        value={editRole}
                                                        onChange={(e) => setEditRole(e.target.value)}
                                                        className="h-8 shadow-sm"
                                                    />
                                                </div>
                                            </>
                                        ) : (
                                            <>
                                                <div className="p-3 bg-white dark:bg-card border rounded-lg shadow-sm flex items-center justify-between">
                                                    <div>
                                                        <span className="text-xs text-muted-foreground block mb-1">机器用途</span>
                                                        <span className="font-medium text-sm">{device.group || '未设置'}</span>
                                                    </div>
                                                    <Badge variant="outline" className="font-mono bg-muted/50 text-muted-foreground font-normal">GROUP</Badge>
                                                </div>
                                                <div className="p-3 bg-white dark:bg-card border rounded-lg shadow-sm flex items-center justify-between">
                                                    <div>
                                                        <span className="text-xs text-muted-foreground block mb-1">集群角色</span>
                                                        <span className="font-medium text-sm">{device.role || '未设置'}</span>
                                                    </div>
                                                    <Badge variant="outline" className="font-mono bg-muted/50 text-muted-foreground font-normal">ROLE</Badge>
                                                </div>
                                            </>
                                        )}
                                        {device.cluster && (
                                            <div className="p-3 bg-emerald-50/50 dark:bg-emerald-950/10 border border-emerald-100 dark:border-emerald-900 rounded-lg flex items-center gap-3">
                                                <div className="h-8 w-8 rounded-full bg-emerald-100 dark:bg-emerald-900/30 flex items-center justify-center shrink-0">
                                                    <Database className="h-4 w-4 text-emerald-600 dark:text-emerald-400" />
                                                </div>
                                                <div>
                                                    <span className="text-xs text-emerald-600/80 dark:text-emerald-400/80 block">所属集群</span>
                                                    <span className="font-medium text-sm text-emerald-900 dark:text-emerald-100">{device.cluster}</span>
                                                </div>
                                            </div>
                                        )}
                                    </div>
                                </div>
                            </TabsContent>

                            <TabsContent value="hardware" className="mt-0 space-y-6 animate-in fade-in-50 slide-in-from-bottom-2 duration-300">
                                {/* Core Specs */}
                                <div className="grid grid-cols-2 gap-4">
                                    <div className="relative overflow-hidden rounded-xl border p-4 bg-gradient-to-br from-blue-50/50 to-indigo-50/50 dark:from-blue-950/10 dark:to-indigo-950/10">
                                        <div className="flex items-center gap-2 mb-3">
                                            <div className="p-1.5 bg-blue-100 dark:bg-blue-900/30 rounded-md text-blue-600 dark:text-blue-400">
                                                <Cpu className="h-4 w-4" />
                                            </div>
                                            <span className="text-sm font-medium text-muted-foreground">处理器</span>
                                        </div>
                                        <div className="space-y-1">
                                            <div className="text-2xl font-bold tracking-tight">{device.cpu || '-'} <span className="text-sm font-normal text-muted-foreground">Cores</span></div>
                                            <div className="text-xs text-muted-foreground font-mono bg-background/50 inline-block px-1.5 py-0.5 rounded border">{device.arch_type || 'Unknown'}</div>
                                        </div>
                                    </div>

                                    <div className="relative overflow-hidden rounded-xl border p-4 bg-gradient-to-br from-emerald-50/50 to-teal-50/50 dark:from-emerald-950/10 dark:to-teal-950/10">
                                        <div className="flex items-center gap-2 mb-3">
                                            <div className="p-1.5 bg-emerald-100 dark:bg-emerald-900/30 rounded-md text-emerald-600 dark:text-emerald-400">
                                                <MemoryStick className="h-4 w-4" />
                                            </div>
                                            <span className="text-sm font-medium text-muted-foreground">内存容量</span>
                                        </div>
                                        <div className="space-y-1">
                                            <div className="text-2xl font-bold tracking-tight">{device.memory || '-'} <span className="text-sm font-normal text-muted-foreground">GB</span></div>
                                            <div className="text-xs text-muted-foreground">DDR4 ECC (Est.)</div>
                                        </div>
                                    </div>
                                </div>

                                <div className="space-y-4">
                                    <SectionTitle icon={HardDrive} title="存储与系统" />
                                    <div className="border rounded-xl divide-y bg-card overflow-hidden text-sm">
                                        <DetailRow label="设备型号" value={device.model} />
                                        <DetailRow label="操作系统" value={device.os_name} sub={device.os_kernel} />
                                        <DetailRow label="磁盘概览" value={device.disk_count} sub={device.disk_detail} />
                                        <DetailRow label="网卡速率" value={device.network_speed} />
                                        <DetailRow label="国产化设备" value={device.is_localization ? 'Yes' : 'No'} />
                                    </div>
                                </div>

                                {device.kvm_ip && (
                                    <div className="p-4 rounded-xl border border-blue-100 bg-blue-50/30 flex items-center justify-between">
                                        <div className="flex items-center gap-3">
                                            <div className="h-8 w-8 rounded-full bg-blue-100 text-blue-600 flex items-center justify-center">
                                                <ExternalLink className="h-4 w-4" />
                                            </div>
                                            <div>
                                                <p className="text-sm font-medium text-blue-900">带外管理 (KVM)</p>
                                                <p className="text-xs text-blue-700 font-mono">{device.kvm_ip}</p>
                                            </div>
                                        </div>
                                        <Button size="sm" variant="outline" className="h-8 bg-white hover:bg-blue-50 text-blue-700 border-blue-200" onClick={() => window.open(`https://${device.kvm_ip}`, '_blank')}>
                                            打开控制台
                                        </Button>
                                    </div>
                                )}
                            </TabsContent>

                            <TabsContent value="k8s" className="mt-0 space-y-6 animate-in fade-in-50 slide-in-from-bottom-2 duration-300">
                                {loadingFeatures ? (
                                    <div className="flex flex-col items-center justify-center py-12 space-y-3">
                                        <div className="animate-spin h-6 w-6 border-2 border-primary border-t-transparent rounded-full" />
                                        <p className="text-sm text-muted-foreground">正在从 Kubernetes 获取实时数据...</p>
                                    </div>
                                ) : features ? (
                                    <div className="space-y-6">
                                        <div className="space-y-3">
                                            <div className="flex items-center justify-between">
                                                <SectionTitle icon={Tag} title="节点标签 (Labels)" />
                                                <Badge variant="secondary" className="font-mono text-[10px]">{features.labels?.length || 0}</Badge>
                                            </div>
                                            {features.labels && features.labels.length > 0 ? (
                                                <div className="grid grid-cols-1 gap-2">
                                                    {features.labels.map((label, i) => (
                                                        <div key={i} className="flex flex-col sm:flex-row sm:items-center justify-between p-3 rounded-lg border bg-card/50 hover:bg-card hover:shadow-sm transition-all text-sm gap-2">
                                                            <span className="font-mono text-muted-foreground tracking-tight text-xs break-all">{label.key}</span>
                                                            <Separator orientation="horizontal" className="sm:hidden w-full" />
                                                            <code className="font-mono text-foreground bg-muted/50 px-1.5 py-0.5 rounded text-xs break-all sm:text-right">{label.value}</code>
                                                        </div>
                                                    ))}
                                                </div>
                                            ) : (
                                                <EmptyState text="暂无标签数据" />
                                            )}
                                        </div>

                                        <div className="space-y-3">
                                            <div className="flex items-center justify-between">
                                                <SectionTitle icon={AlertTriangle} title="节点污点 (Taints)" />
                                                <Badge variant="secondary" className="font-mono text-[10px]">{features.taints?.length || 0}</Badge>
                                            </div>
                                            {features.taints && features.taints.length > 0 ? (
                                                <div className="space-y-2">
                                                    {features.taints.map((taint, i) => (
                                                        <div key={i} className="flex items-center justify-between p-3 rounded-lg border border-orange-100 bg-orange-50/50 dark:border-orange-900/50 dark:bg-orange-950/20">
                                                            <div className="flex flex-col">
                                                                <span className="font-medium text-sm text-orange-900 dark:text-orange-100">{taint.key}</span>
                                                                {taint.value && <span className="text-xs text-orange-700/70 dark:text-orange-400/70 mt-0.5">{taint.value}</span>}
                                                            </div>
                                                            <Badge variant="outline" className="border-orange-200 text-orange-600 bg-white/50">{taint.effect}</Badge>
                                                        </div>
                                                    ))}
                                                </div>
                                            ) : (
                                                <EmptyState text="无污点" />
                                            )}
                                        </div>
                                    </div>
                                ) : (
                                    <div className="rounded-xl border border-dashed p-8 text-center flex flex-col items-center justify-center bg-muted/20">
                                        <CloudOff className="h-10 w-10 text-muted-foreground/30 mb-3" />
                                        <p className="text-sm font-medium text-foreground">无法获取 K8s 数据</p>
                                        <p className="text-xs text-muted-foreground mt-1 max-w-[200px]">该设备可能未加入集群或 API 服务暂时不可用</p>
                                    </div>
                                )}
                            </TabsContent>
                        </div>
                    </ScrollArea>
                </Tabs>
            </SheetContent>
        </Sheet>
    )
}

// Subcomponents for cleaner code
function NavItem({ value, label, icon: Icon }: { value: string; label: string; icon: any }) {
    return (
        <TabsTrigger
            value={value}
            className="group flex items-center gap-2 rounded-none border-b-2 border-transparent px-1 py-2 text-sm font-medium text-muted-foreground data-[state=active]:border-primary data-[state=active]:text-foreground transition-all"
        >
            <Icon className="h-4 w-4 text-muted-foreground group-data-[state=active]:text-primary mb-0.5" />
            {label}
        </TabsTrigger>
    )
}

function SectionTitle({ icon: Icon, title }: { icon: any; title: string }) {
    return (
        <h4 className="flex items-center gap-2 text-sm font-semibold text-foreground/90">
            <div className="p-1 rounded bg-muted/80">
                <Icon className="h-3.5 w-3.5" />
            </div>
            {title}
        </h4>
    )
}

function InfoCard({ label, value, subValue, icon: Icon, className }: { label: string; value: string | number | null | undefined; subValue?: string; icon?: any; className?: string }) {
    return (
        <div className={cn("p-3 rounded-xl border bg-card shadow-sm hover:shadow-md transition-shadow", className)}>
            <div className="flex items-center gap-2 mb-1.5 text-xs text-muted-foreground">
                {Icon && <Icon className="h-3 w-3" />}
                {label}
            </div>
            <div className="font-medium text-sm truncate" title={String(value)}>{value || '-'}</div>
            {subValue && <div className="text-xs text-muted-foreground mt-0.5 truncate" title={subValue}>{subValue}</div>}
        </div>
    )
}

function DetailRow({ label, value, sub }: { label: string; value: React.ReactNode; sub?: string }) {
    return (
        <div className="flex items-center justify-between p-3 first:pt-3 last:pb-3 hover:bg-muted/30 transition-colors">
            <span className="text-muted-foreground shrink-0">{label}</span>
            <div className="text-right overflow-hidden">
                <div className="font-medium truncate pl-4">{value || '-'}</div>
                {sub && <div className="text-xs text-muted-foreground mt-0.5 truncate pl-4" title={sub}>{sub}</div>}
            </div>
        </div>
    )
}

function EmptyState({ text }: { text: string }) {
    return (
        <div className="text-center py-6 border rounded-lg bg-muted/10 border-dashed">
            <p className="text-xs text-muted-foreground">{text}</p>
        </div>
    )
}
