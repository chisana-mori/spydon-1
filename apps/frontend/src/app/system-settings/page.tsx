'use client';

import React, { useEffect, useState } from 'react';
import { Settings as SettingsIcon, Loader2, Save, RefreshCw, CheckCircle2, AlertCircle } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { listSettings, updateSetting } from '@/lib/api/settings';
import { AutoRCAConfig, SETTING_KEYS } from '@/types/settings';

type Message = { type: 'success' | 'error'; text: string } | null;

const DEFAULT_ALLOWED_SEVERITIES = ['high', 'critical'];

const DEFAULT_AUTO_RCA_CONFIG: AutoRCAConfig = {
    enabled: false,
    rate_limit: 10,
    period: 60,
    allowed_severities: [...DEFAULT_ALLOWED_SEVERITIES],
};

/**
 * 系统使用标准化后的四级严重度（low/medium/high/critical）。
 * Alert 实体在后端会将 info/warning/error 等同归并到上述四级，
 * 因此设置页仅提供四级选项以与实体保持一致。
 */
const severityOptions = [
    { value: 'critical', label: '致命 (Critical)' },
    { value: 'high', label: '高 (High)' },
    { value: 'medium', label: '中 (Medium)' },
    { value: 'low', label: '低 (Low)' },
];

/**
 * 获取各告警级别对应的颜色标签样式类
 * 用于在系统设置页面的“允许自动分析的告警级别”复选项旁显示颜色标识
 */
function getSeverityTagClass(severity: string): string {
    switch (severity) {
        case 'critical':
            return 'bg-red-600 dark:bg-red-500';
        case 'high':
            return 'bg-orange-500 dark:bg-orange-400';
        case 'medium':
            return 'bg-yellow-400 dark:bg-yellow-300';
        case 'low':
            return 'bg-blue-400 dark:bg-blue-300';
        case 'error':
            return 'bg-rose-600 dark:bg-rose-500';
        case 'warning':
            return 'bg-amber-500 dark:bg-amber-400';
        case 'info':
        default:
            return 'bg-sky-500 dark:bg-sky-400';
    }
}

export default function SystemSettingsPage() {
    const [loading, setLoading] = useState(false);
    const [saving, setSaving] = useState(false);
    const [autoRCAConfig, setAutoRCAConfig] = useState<AutoRCAConfig>(DEFAULT_AUTO_RCA_CONFIG);
    const [message, setMessage] = useState<Message>(null);

    const normalizeConfig = (value: AutoRCAConfig | null | undefined): AutoRCAConfig => {
        if (!value) {
            return { ...DEFAULT_AUTO_RCA_CONFIG, allowed_severities: [...DEFAULT_ALLOWED_SEVERITIES] };
        }
        return {
            ...DEFAULT_AUTO_RCA_CONFIG,
            ...value,
            allowed_severities:
                Array.isArray(value.allowed_severities) && value.allowed_severities.length > 0
                    ? value.allowed_severities
                    : [...DEFAULT_ALLOWED_SEVERITIES],
        };
    };

    const loadSettings = async () => {
        setLoading(true);
        setMessage(null);
        try {
            const settings = await listSettings();
            const autoRCASetting = settings.find((setting) => setting.key === SETTING_KEYS.AUTO_RCA);
            if (autoRCASetting?.value) {
                setAutoRCAConfig(normalizeConfig(autoRCASetting.value as AutoRCAConfig));
            } else {
                setAutoRCAConfig(normalizeConfig(null));
            }
        } catch (error) {
            console.error('加载设置失败:', error);
            setMessage({ type: 'error', text: `无法加载系统设置: ${error instanceof Error ? error.message : '未知错误'}` });
        } finally {
            setLoading(false);
        }
    };

    const handleSeverityToggle = (severity: string) => {
        setAutoRCAConfig((prev) => {
            const exists = prev.allowed_severities.includes(severity);
            const next = exists
                ? prev.allowed_severities.filter((item) => item !== severity)
                : [...prev.allowed_severities, severity];
            return { ...prev, allowed_severities: next };
        });
    };

    const saveAutoRCAConfig = async () => {
        setSaving(true);
        setMessage(null);
        try {
            await updateSetting(SETTING_KEYS.AUTO_RCA, {
                value: autoRCAConfig,
                description: 'Auto-RCA 自动根因分析配置',
            });
            setMessage({ type: 'success', text: 'Auto-RCA 配置已成功保存' });
            await loadSettings();
        } catch (error) {
            console.error('保存设置失败:', error);
            setMessage({ type: 'error', text: `无法保存 Auto-RCA 配置: ${error instanceof Error ? error.message : '未知错误'}` });
        } finally {
            setSaving(false);
        }
    };

    useEffect(() => {
        loadSettings();
    }, []);

    return (
        <div className="space-y-6">
            <div className="flex flex-col gap-2">
                <div className="flex items-center gap-2">
                    <SettingsIcon className="h-6 w-6 text-primary" />
                    <div>
                        <h1 className="text-2xl font-semibold">系统设置</h1>
                        <p className="text-sm text-muted-foreground">
                            配置系统的全局设置和功能开关
                        </p>
                    </div>
                </div>
            </div>

            {message && (
                <Alert variant={message.type === 'error' ? 'destructive' : 'default'}>
                    {message.type === 'success' ? (
                        <CheckCircle2 className="h-4 w-4" />
                    ) : (
                        <AlertCircle className="h-4 w-4" />
                    )}
                    <AlertDescription>{message.text}</AlertDescription>
                </Alert>
            )}

            <Card>
                <CardHeader>
                    <CardTitle className="text-lg">Auto-RCA 自动根因分析</CardTitle>
                    <CardDescription>
                        当告警触发时自动启动 RCA 分析，帮助快速定位问题根因
                    </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                    {loading ? (
                        <div className="flex items-center justify-center py-10">
                            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                        </div>
                    ) : (
                        <>
                            <div className="flex items-center justify-between">
                                <div className="space-y-0.5">
                                    <Label htmlFor="auto-rca-enabled" className="text-base">
                                        启用 Auto-RCA
                                    </Label>
                                    <p className="text-sm text-muted-foreground">
                                        自动对新触发的告警进行根因分析
                                    </p>
                                </div>
                                <Switch
                                    id="auto-rca-enabled"
                                    checked={autoRCAConfig.enabled}
                                    onCheckedChange={(checked) => setAutoRCAConfig({ ...autoRCAConfig, enabled: checked })}
                                />
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="rate-limit">速率限制（次数）</Label>
                                <Input
                                    id="rate-limit"
                                    type="number"
                                    min="1"
                                    max="100"
                                    value={autoRCAConfig.rate_limit}
                                    onChange={(event) =>
                                        setAutoRCAConfig({
                                            ...autoRCAConfig,
                                            rate_limit: parseInt(event.target.value, 10) || DEFAULT_AUTO_RCA_CONFIG.rate_limit,
                                        })
                                    }
                                />
                                <p className="text-sm text-muted-foreground">
                                    在指定周期内允许的最大 RCA 分析次数
                                </p>
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="period">周期（秒）</Label>
                                <Input
                                    id="period"
                                    type="number"
                                    min="10"
                                    max="3600"
                                    value={autoRCAConfig.period}
                                    onChange={(event) =>
                                        setAutoRCAConfig({
                                            ...autoRCAConfig,
                                            period: parseInt(event.target.value, 10) || DEFAULT_AUTO_RCA_CONFIG.period,
                                        })
                                    }
                                />
                                <p className="text-sm text-muted-foreground">
                                    速率限制的时间窗口（秒）
                                </p>
                            </div>

                            <div className="space-y-2">
                                <Label>允许自动分析的告警级别</Label>
                                <p className="text-sm text-muted-foreground">
                                    仅勾选的级别会自动触发 RCA，若同时排队会优先处理严重级别
                                </p>
                                <div className="grid grid-cols-2 gap-2">
                                    {severityOptions.map((option) => (
                                        <label key={option.value} className="flex items-center gap-2 rounded-md border border-border/50 p-2 text-sm">
                                            <Checkbox
                                                checked={autoRCAConfig.allowed_severities.includes(option.value)}
                                                onCheckedChange={() => handleSeverityToggle(option.value)}
                                            />
                                            <span className="inline-flex items-center gap-2">
                                                <span
                                                    className={`inline-block w-2.5 h-2.5 rounded-full ${getSeverityTagClass(option.value)}`}
                                                    aria-hidden="true"
                                                />
                                                <span>{option.label}</span>
                                            </span>
                                        </label>
                                    ))}
                                </div>
                                <p className="text-xs text-muted-foreground">
                                    队列按严重度排序（Critical &gt; High &gt; Medium &gt; Low）
                                </p>
                            </div>

                            <div className="rounded-lg bg-muted p-3 text-sm">
                                <p className="font-medium mb-1">当前配置：</p>
                                <p className="text-muted-foreground">
                                    每 {autoRCAConfig.period} 秒最多执行 {autoRCAConfig.rate_limit} 次 RCA 分析
                                    {autoRCAConfig.period === 60 && (
                                        <span className="text-xs ml-1">
                                            （约 {(autoRCAConfig.rate_limit / (autoRCAConfig.period / 60)).toFixed(1)} 次/分钟）
                                        </span>
                                    )}
                                </p>
                                <p className="text-muted-foreground mt-1">
                                    自动分析级别：{autoRCAConfig.allowed_severities.length > 0 ? autoRCAConfig.allowed_severities.join('、') : '未选择（不会触发自动分析）'}
                                </p>
                            </div>

                            <div className="flex items-center gap-2 pt-2">
                                <Button onClick={saveAutoRCAConfig} disabled={saving} className="flex items-center gap-2">
                                    {saving ? (
                                        <>
                                            <Loader2 className="h-4 w-4 animate-spin" />
                                            保存中...
                                        </>
                                    ) : (
                                        <>
                                            <Save className="h-4 w-4" />
                                            保存配置
                                        </>
                                    )}
                                </Button>
                                <Button
                                    variant="outline"
                                    onClick={loadSettings}
                                    disabled={loading || saving}
                                    className="flex items-center gap-2"
                                >
                                    <RefreshCw className="h-4 w-4" />
                                    刷新
                                </Button>
                            </div>
                        </>
                    )}
                </CardContent>
            </Card>
        </div>
    );
}
