'use client';

import React, { useState, useEffect, useRef } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import {
  Clock,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Info,
  ExternalLink,
  Tag,
  Server,
  Brain,
  Loader2
} from 'lucide-react';
import { Alert, RCAStatus } from '@prototypes/alerts/types/alerts';
import { RawPayloadViewer } from './RawPayloadViewer';
import { EnhancedHolmesGPTChat } from '@/components/alerts/EnhancedHolmesGPTChat';
import { appConfig } from '@/config';

interface AlertDetailProps {
  alert: Alert;
  className?: string;
}

export function AlertDetail({ alert, className }: AlertDetailProps) {
  const [cachedRCAResult, setCachedRCAResult] = useState<any>(null);
  const [loadingCache, setLoadingCache] = useState(false);
  const [rcaStatus, setRCAStatus] = useState<RCAStatus | null>(null);
  const [rcaRunId, setRCARunId] = useState<string | null>(null);
  const [cacheRequestKey, setCacheRequestKey] = useState<string | null>(null);
  const [loadingStatus, setLoadingStatus] = useState(false);
  const streamRef = useRef<EventSource | null>(null);

  // 检查是否有正在运行的 RCA
  const runningRCA = alert.rca_runs?.find(run =>
    run.status === 'running' || run.status === 'pending' || run.status === 'queued'
  );

  // 检查是否有已完成的 RCA
  const completedRCA = alert.rca_runs?.find(run => run.status === 'completed');

  // 主动查询最新的 RCA 状态
  const fetchLatestRCAStatus = async () => {
    if (!alert.id) return;

    setLoadingStatus(true);
    try {
      const basePath = appConfig.basePath || '';
      const url = `${basePath}/api/v1/rca/${alert.id}`;
      const response = await fetch(url);

      if (response.ok) {
        const data = await response.json();
        const rcaRuns = data.data || [];

        console.log('✅ 主动查询 RCA 状态成功:', rcaRuns);

        // 找到最新的运行记录
        const latestRun =
          rcaRuns.find((run: any) => run.status === 'running' || run.status === 'pending' || run.status === 'queued') ||
          rcaRuns.find((run: any) => run.status === 'completed') ||
          rcaRuns[0];

        if (latestRun) {
          setRCAStatus(latestRun.status);
          setRCARunId(latestRun.id);
          console.log('📊 更新 RCA 状态:', { status: latestRun.status, runId: latestRun.id });
        }
      } else {
        console.warn('⚠️ 查询 RCA 状态失败:', response.status);
      }
    } catch (error) {
      console.error('❌ 查询 RCA 状态失败:', error);
    } finally {
      setLoadingStatus(false);
    }
  };

  // 页面打开时，先从 alert.rca_runs 获取初始状态，然后主动查询最新状态
  useEffect(() => {
    setCachedRCAResult(null);
    setRCAStatus(null);
    setRCARunId(null);
    setCacheRequestKey(null);

    // 1. 先从 alert.rca_runs 获取初始状态（可能不是最新的）
    const runs = alert.rca_runs || [];
    const preferredRun =
      runs.find(run => run.status === 'running' || run.status === 'pending' || run.status === 'queued') ||
      runs.find(run => run.status === 'completed') ||
      runs[0];

    if (preferredRun) {
      setRCAStatus(preferredRun.status);
      setRCARunId(preferredRun.id);
    }

    // 2. 主动查询最新状态（覆盖初始状态）
    void fetchLatestRCAStatus();
  }, [alert.id]);

  const loadRCACacheResult = async (runId?: string) => {
    setLoadingCache(true);
    try {
      const basePath = appConfig.basePath || '';
      const url = runId
        ? `${basePath}/api/v1/rca/${alert.id}/cache?run_id=${runId}`
        : `${basePath}/api/v1/rca/${alert.id}/cache`;
      const response = await fetch(url);

      if (response.ok) {
        const data = await response.json();
        console.log('✅ 加载RCA缓存成功:', data);
        setCachedRCAResult(data.data || data);
      } else {
        console.warn('⚠️ 加载RCA缓存失败:', response.status);
      }
    } catch (error) {
      console.error('❌ 加载RCA缓存失败:', error);
    } finally {
      setLoadingCache(false);
    }
  };

  useEffect(() => {
    if (streamRef.current) {
      streamRef.current.close();
      streamRef.current = null;
    }
    if (!alert.id) {
      console.log('🔴 AlertDetail: 没有 alert.id，跳过 SSE 连接');
      return;
    }

    const apiBase = appConfig.apiBaseUrl || `${appConfig.backendBaseUrl.replace(/\/+$/, '')}/api/v1`;
    const streamUrl = `${apiBase.replace(/\/+$/, '')}/rca/${alert.id}/stream`;

    console.log('🔵 AlertDetail: 建立 RCA SSE 连接', {
      alertId: alert.id,
      streamUrl,
      apiBase,
      backendBaseUrl: appConfig.backendBaseUrl,
      apiBaseUrl: appConfig.apiBaseUrl
    });

    const source = new EventSource(streamUrl, { withCredentials: true });
    streamRef.current = source;

    const handleStatusPayload = (payload: any) => {
      if (!payload) return;
      const nextStatus = (payload.status || payload.data?.status) as RCAStatus | undefined;
      const nextRunId = payload.run_id || payload.data?.run_id;

      if (!nextStatus) return;
      console.log('📊 AlertDetail: 收到 RCA 状态更新', {
        prevStatus: rcaStatus,
        nextStatus,
        prevRunId: rcaRunId,
        nextRunId,
        payload
      });

      // 更新状态
      setRCAStatus(prev => prev === nextStatus ? prev : nextStatus);

      // 更新 runId（如果有）
      if (nextRunId && nextRunId !== rcaRunId) {
        setRCARunId(nextRunId);
        console.log('🆔 AlertDetail: 更新 RCA Run ID', { runId: nextRunId });
      }

      if (nextStatus === 'completed' || nextStatus === 'failed' || nextStatus === 'timeout') {
        console.log('✅ AlertDetail: RCA 已结束，关闭 SSE 连接', { status: nextStatus });
        source.close();
        streamRef.current = null;
      }
    };

    const handleEvent = (event: MessageEvent) => {
      console.log('📨 AlertDetail: 收到 SSE 事件', {
        data: event.data,
        type: event.type
      });
      try {
        const parsed = JSON.parse(event.data);
        handleStatusPayload(parsed);
      } catch (error) {
        console.warn('⚠️ AlertDetail: 解析 RCA 流事件失败:', error, event.data);
      }
    };

    source.addEventListener('status', handleEvent as EventListener);
    source.onmessage = handleEvent;

    source.onopen = () => {
      console.log('✅ AlertDetail: SSE 连接已建立', { streamUrl });
    };

    source.onerror = (error) => {
      console.error('❌ AlertDetail: SSE 连接错误', {
        error,
        streamUrl,
        readyState: source.readyState,
        // EventSource.CONNECTING = 0, EventSource.OPEN = 1, EventSource.CLOSED = 2
        readyStateText: source.readyState === 0 ? 'CONNECTING' :
          source.readyState === 1 ? 'OPEN' : 'CLOSED'
      });
      source.close();
      streamRef.current = null;
    };

    return () => {
      console.log('🔴 AlertDetail: 清理 SSE 连接', { alertId: alert.id });
      source.close();
      streamRef.current = null;
    };
  }, [alert.id]);

  useEffect(() => {
    if (rcaStatus !== 'completed' || loadingCache || cachedRCAResult) {
      return;
    }

    const targetRunId = rcaRunId || completedRCA?.id || runningRCA?.id;
    const requestKey = targetRunId || 'no-run-id';
    if (cacheRequestKey === requestKey) {
      return;
    }

    setCacheRequestKey(requestKey);
    void loadRCACacheResult(targetRunId);
  }, [rcaStatus, loadingCache, cachedRCAResult, rcaRunId, completedRCA?.id, runningRCA?.id, cacheRequestKey]);

  // 获取严重程度颜色
  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical': return 'destructive';
      case 'high': return 'destructive';
      case 'medium': return 'default';
      case 'low': return 'secondary';
      default: return 'outline';
    }
  };

  // 获取状态图标
  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'active':
      case 'firing': return <AlertTriangle className="h-4 w-4" />;
      case 'resolved': return <CheckCircle className="h-4 w-4" />;
      case 'acknowledged': return <Info className="h-4 w-4" />;
      default: return <XCircle className="h-4 w-4" />;
    }
  };

  // 获取状态颜色
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
      case 'firing': return 'destructive';
      case 'resolved': return 'default';
      case 'acknowledged': return 'secondary';
      default: return 'outline';
    }
  };

  // 获取显示用的状态文本
  const getStatusText = (status: string) => {
    switch (status) {
      case 'firing': return '触发中';
      case 'active': return '活跃';
      case 'resolved': return '已解决';
      case 'acknowledged': return '已确认';
      default: return status.toUpperCase();
    }
  };

  // 获取 RCA 状态徽章
  const getRCAStatusBadge = (status?: RCAStatus | null) => {
    switch (status) {
      case 'running':
        return (
          <Badge variant="default" className="flex items-center gap-1">
            <Loader2 className="h-3 w-3 animate-spin" />
            RCA 分析中
          </Badge>
        );
      case 'pending':
      case 'queued':
        return (
          <Badge variant="secondary" className="flex items-center gap-1">
            <Clock className="h-3 w-3" />
            RCA 排队中
          </Badge>
        );
      case 'completed':
        return (
          <Badge variant="default" className="flex items-center gap-1 bg-green-600">
            <CheckCircle className="h-3 w-3" />
            RCA 已完成
          </Badge>
        );
      case 'failed':
        return (
          <Badge variant="destructive" className="flex items-center gap-1">
            <XCircle className="h-3 w-3" />
            RCA 失败
          </Badge>
        );
      default:
        return null;
    }
  };

  const effectiveRCAStatus = rcaStatus || runningRCA?.status || completedRCA?.status || null;
  const shouldShowRCASection = !!effectiveRCAStatus;

  return (
    <div className={`space-y-6 ${className}`}>
      {/* 基本信息 */}
      <Card>
        <CardHeader>
          <div className="flex items-start justify-between">
            <div className="space-y-2">
              <CardTitle className="text-xl">{alert.title}</CardTitle>
              {alert.description && (
                <p className="text-muted-foreground">{alert.description}</p>
              )}
            </div>
            <div className="flex items-center gap-2">
              <Badge variant={getSeverityColor(alert.severity)}>
                {alert.severity.toUpperCase()}
              </Badge>
              <Badge variant={getStatusColor(alert.status)} className="flex items-center gap-1">
                {getStatusIcon(alert.status)}
                {getStatusText(alert.status)}
              </Badge>
              {/* 显示 RCA 状态 */}
              {getRCAStatusBadge(effectiveRCAStatus)}
            </div>
          </div>
        </CardHeader>

        <CardContent className="space-y-4">
          {/* 时间信息 */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="flex items-center gap-2">
              <Clock className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm">
                创建时间: {new Date(alert.created_at).toLocaleString()}
              </span>
            </div>
            {alert.cluster && (
              <div className="flex items-center gap-2">
                <Server className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm">
                  集群: {alert.cluster.name}
                </span>
              </div>
            )}
          </div>

          {/* 来源信息 */}
          <div className="flex items-center gap-2">
            <ExternalLink className="h-4 w-4 text-muted-foreground" />
            <span className="text-sm">来源: {alert.source}</span>
            {alert.generatorURL && (
              <a
                href={alert.generatorURL}
                target="_blank"
                rel="noopener noreferrer"
                className="text-blue-600 hover:text-blue-800 text-sm"
              >
                查看详情
              </a>
            )}
          </div>

          {alert.fingerprint && (
            <div className="text-sm text-muted-foreground">
              指纹: <code className="bg-muted px-1 rounded">{alert.fingerprint}</code>
            </div>
          )}
        </CardContent>
      </Card>

      {/* RCA 分析区域 */}
      {shouldShowRCASection && (
        <>
          <Separator />
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Brain className="h-5 w-5" />
                RCA 根因分析
              </CardTitle>
            </CardHeader>
            <CardContent>
              <EnhancedHolmesGPTChat
                alert={alert as any}
                showCard={false}
                cachedResult={cachedRCAResult}
                loadingCache={loadingCache}
              />
            </CardContent>
          </Card>
        </>
      )}

      {/* 标签和注释 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* 标签 */}
        {alert.labels && Object.keys(alert.labels).length > 0 && (
          <Card className="h-full">
            <CardHeader className="pb-3">
              <CardTitle className="text-base font-medium flex items-center gap-2">
                <Tag className="h-4 w-4" />
                标签
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex flex-wrap gap-2">
                {Object.entries(alert.labels).map(([key, value], index) => {
                  // Generate a consistent color based on the key length to add some variety
                  const colors = [
                    "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300 border-blue-200",
                    "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300 border-green-200",
                    "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300 border-purple-200",
                    "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300 border-orange-200",
                    "bg-pink-100 text-pink-800 dark:bg-pink-900/30 dark:text-pink-300 border-pink-200",
                  ];
                  const colorClass = colors[key.length % colors.length];

                  return (
                    <div key={key} className={`inline-flex items-center rounded-md border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 ${colorClass}`}>
                      <span className="opacity-70 mr-1">{key}:</span>
                      <span>{value}</span>
                    </div>
                  );
                })}
              </div>
            </CardContent>
          </Card>
        )}

        {/* 注释 */}
        {alert.annotations && Object.keys(alert.annotations).length > 0 && (
          <Card className="h-full">
            <CardHeader className="pb-3">
              <CardTitle className="text-base font-medium flex items-center gap-2">
                <Info className="h-4 w-4" />
                注释
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex flex-wrap gap-2">
                {Object.entries(alert.annotations)
                  .filter(([key]) => key !== 'enrichment_keys' && key !== 'investigate_uri')
                  .map(([key, value]) => (
                    <div key={key} className="inline-flex items-center rounded-md border border-muted-foreground/20 bg-muted/50 px-2.5 py-0.5 text-xs font-medium text-muted-foreground transition-colors hover:bg-muted/70">
                      <span className="font-semibold text-foreground mr-1">{key}:</span>
                      <span className="truncate max-w-[300px]">{value}</span>
                    </div>
                  ))}
              </div>
            </CardContent>
          </Card>
        )}
      </div>

      {/* 原始数据展示 */}
      {alert.raw_payload_key && (
        <>
          <Separator />
          <RawPayloadViewer
            rawPayloadKey={alert.raw_payload_key}
            alertId={alert.id}
            alertTitle={alert.title}
          />
        </>
      )}
    </div>
  );
}
