'use client';

import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Alert, AlertDescription } from '@/components/ui/alert';
import {
  Download,
  Eye,
  FileText,
  Loader2,
  AlertCircle,
  Copy,
  Check,
  Sparkles
} from 'lucide-react';
import { minioClient } from '@prototypes/alerts/lib/minio-client';
import { RawPayloadData, RawDataViewMode } from '@prototypes/alerts/types/alerts';
import { Finding } from '@prototypes/alerts/types/enrichment';
import { parseFinding } from '@prototypes/alerts/lib/enrichment-parser';
import { EnrichmentRenderer } from '@prototypes/alerts/components/enrichment/EnrichmentRenderer';
import { copyTextToClipboard } from '@/lib/clipboard';
import { appConfig } from '@/config';

interface RawPayloadViewerProps {
  rawPayloadKey: string;
  alertId: string;
  alertTitle?: string;
  className?: string;
}

export function RawPayloadViewer({
  rawPayloadKey,
  alertId,
  className
}: RawPayloadViewerProps) {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<RawPayloadData | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<RawDataViewMode>('formatted');
  const [copied, setCopied] = useState(false);
  const [finding, setFinding] = useState<Finding | null>(null);

  // 加载原始数据
  const loadRawData = async () => {
    if (!rawPayloadKey && !alertId) return;

    setLoading(true);
    setError(null);

    try {
      let rawData: RawPayloadData | null = null;

      if (alertId) {
        // 优先使用告警 ID 直接从后端 API 获取
        const response = await fetch(`${appConfig.apiBaseUrl}/alerts/${alertId}/raw-payload`);

        if (response.ok) {
          const contentType = response.headers.get('content-type') || 'application/json';
          const contentLength = response.headers.get('content-length');
          const payloadKey = response.headers.get('x-raw-payload-key') || rawPayloadKey || '';

          let data: any;
          if (contentType.includes('application/json')) {
            data = await response.json();
          } else {
            data = await response.text();
          }

          rawData = {
            key: payloadKey,
            data,
            contentType,
            size: contentLength ? parseInt(contentLength) : 0,
            lastModified: new Date(),
          };
        }
      } else if (rawPayloadKey) {
        // 回退到使用 MinIO 客户端
        rawData = await minioClient.getRawPayload(rawPayloadKey);
      }

      if (rawData) {
        setData(rawData);

        // 尝试解析为 Finding
        try {
          console.log('Raw data:', rawData.data);
          const parsed = parseFinding(rawData.data);
          console.log('Parsed finding:', parsed);
          console.log('Enrichments:', parsed.enrichments);
          console.log('Enrichments length:', parsed.enrichments?.length);
          setFinding(parsed);
        } catch (parseErr) {
          console.error('Failed to parse as Finding:', parseErr);
          setFinding(null);
        }
      } else {
        setError('无法获取原始数据');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '获取数据时发生错误');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadRawData();
  }, [rawPayloadKey]);

  // 复制到剪贴板
  const copyToClipboard = async (text: string) => {
    const success = await copyTextToClipboard(text);
    if (success) {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } else {
      // 浏览器不支持复制或拒绝访问剪贴板
    }
  };

  // 下载原始数据
  const downloadRawData = () => {
    if (!data) return;

    const content = typeof data.data === 'string'
      ? data.data
      : JSON.stringify(data.data, null, 2);

    const blob = new Blob([content], { type: data.contentType });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `alert-${alertId}-raw-data.${getFileExtension(data.contentType)}`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  // 获取文件扩展名
  const getFileExtension = (contentType: string): string => {
    if (contentType.includes('json')) return 'json';
    if (contentType.includes('yaml')) return 'yaml';
    if (contentType.includes('xml')) return 'xml';
    return 'txt';
  };

  // 格式化显示数据
  const formatData = (data: any, mode: RawDataViewMode): string => {
    if (typeof data === 'string') {
      return data;
    }

    switch (mode) {
      case 'json':
        return JSON.stringify(data, null, 2);
      case 'yaml':
        // 简单的 JSON 到 YAML 转换
        return JSON.stringify(data, null, 2)
          .replace(/"/g, '')
          .replace(/,$/gm, '')
          .replace(/^\s*{\s*$/gm, '')
          .replace(/^\s*}\s*$/gm, '');
      case 'formatted':
        return JSON.stringify(data, null, 2);
      default:
        return String(data);
    }
  };

  if (!rawPayloadKey) {
    return null;
  }

  return (
    <Card className={className}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg flex items-center gap-2">
            <FileText className="h-5 w-5" />
            原始数据
          </CardTitle>
          <div className="flex items-center gap-2">
            {data && (
              <>
                <Badge variant="outline" className="text-xs">
                  {data.contentType}
                </Badge>
                <Badge variant="outline" className="text-xs">
                  {(data.size / 1024).toFixed(1)} KB
                </Badge>
              </>
            )}
          </div>
        </div>
      </CardHeader>

      <CardContent>
        {loading && (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-6 w-6 animate-spin mr-2" />
            <span>加载原始数据中...</span>
          </div>
        )}

        {error && (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {data && !loading && (
          <div className="space-y-4">
            {/* 操作按钮 */}
            <div className="flex items-center gap-2 flex-wrap">
              <Button
                variant="outline"
                size="sm"
                onClick={() => copyToClipboard(formatData(data.data, viewMode))}
                className="flex items-center gap-1"
              >
                {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                {copied ? '已复制' : '复制'}
              </Button>

              <Button
                variant="outline"
                size="sm"
                onClick={downloadRawData}
                className="flex items-center gap-1"
              >
                <Download className="h-4 w-4" />
                下载
              </Button>

              <Button
                variant="outline"
                size="sm"
                onClick={loadRawData}
                className="flex items-center gap-1"
              >
                <Eye className="h-4 w-4" />
                刷新
              </Button>
            </div>

            {/* 数据展示 - 使用 Tabs 切换视图 */}
            <Tabs value={viewMode} onValueChange={(value) => setViewMode(value as RawDataViewMode)}>
              <TabsList className="grid w-full grid-cols-4">
                <TabsTrigger value="formatted" className="flex items-center gap-1">
                  <Sparkles className="h-3 w-3" />
                  美化视图
                </TabsTrigger>
                <TabsTrigger value="json">JSON</TabsTrigger>
                <TabsTrigger value="yaml">YAML</TabsTrigger>
                <TabsTrigger value="raw">原始</TabsTrigger>
              </TabsList>

              {/* 美化视图 - 仅显示解析后的 Enrichments */}
              <TabsContent value="formatted" className="mt-4">
                {/* 调试信息 */}
                <div className="mb-4 p-4 bg-muted rounded text-xs space-y-1">
                  <div>Finding: {finding ? '✅' : '❌'}</div>
                  <div>Enrichments: {finding?.enrichments ? '✅' : '❌'}</div>
                  <div>Enrichments length: {finding?.enrichments?.length || 0}</div>
                  {finding?.enrichments && finding.enrichments.length > 0 && (
                    <div>First enrichment title: {finding.enrichments[0].title}</div>
                  )}
                </div>

                {finding && finding.enrichments && finding.enrichments.length > 0 ? (
                  <div className="space-y-4">
                    {/* 仅显示 Enrichments 列表，不显示 Finding 基本信息 */}
                    {finding.enrichments.map((enrichment, idx) => (
                      <EnrichmentRenderer
                        key={idx}
                        enrichment={enrichment}
                        index={idx}
                      />
                    ))}
                  </div>
                ) : (
                  <Alert>
                    <AlertCircle className="h-4 w-4" />
                    <AlertDescription>
                      无法解析为结构化数据，请切换到其他视图查看原始内容
                    </AlertDescription>
                  </Alert>
                )}
              </TabsContent>

              {/* JSON 视图 */}
              <TabsContent value="json" className="mt-4">
                <div className="relative">
                  <pre className="bg-slate-950 text-slate-50 p-4 rounded-lg overflow-auto max-h-[600px] text-xs">
                    <code>{formatData(data.data, 'json')}</code>
                  </pre>
                </div>
              </TabsContent>

              {/* YAML 视图 */}
              <TabsContent value="yaml" className="mt-4">
                <div className="relative">
                  <pre className="bg-slate-950 text-slate-50 p-4 rounded-lg overflow-auto max-h-[600px] text-xs">
                    <code>{formatData(data.data, 'yaml')}</code>
                  </pre>
                </div>
              </TabsContent>

              {/* 原始视图 */}
              <TabsContent value="raw" className="mt-4">
                <div className="relative">
                  <pre className="bg-slate-950 text-slate-50 p-4 rounded-lg overflow-auto max-h-[600px] text-xs">
                    <code>{formatData(data.data, 'raw')}</code>
                  </pre>
                </div>
              </TabsContent>
            </Tabs>

            {/* 元数据信息 */}
            <div className="text-sm text-muted-foreground space-y-1 pt-2 border-t">
              <div>最后修改: {data.lastModified.toLocaleString()}</div>
              <div>存储键: <code className="bg-muted px-1 rounded text-xs">{data.key}</code></div>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
