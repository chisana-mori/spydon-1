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
  Code, 
  FileText, 
  Loader2, 
  AlertCircle,
  Copy,
  Check
} from 'lucide-react';
import { minioClient } from '@prototypes/alerts/lib/minio-client';
import { RawPayloadData, RawDataViewMode } from '@prototypes/alerts/types/alerts';

interface RawPayloadViewerProps {
  rawPayloadKey: string;
  alertId: string;
  alertTitle: string;
  className?: string;
}

export function RawPayloadViewer({ 
  rawPayloadKey, 
  alertId, 
  alertTitle,
  className 
}: RawPayloadViewerProps) {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<RawPayloadData | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<RawDataViewMode>('formatted');
  const [copied, setCopied] = useState(false);

  // 加载原始数据
  const loadRawData = async () => {
    if (!rawPayloadKey && !alertId) return;

    setLoading(true);
    setError(null);

    try {
      let rawData: RawPayloadData | null = null;

      if (alertId) {
        // 优先使用告警 ID 直接从后端 API 获取
        const apiBaseUrl = process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:8080/api/v1';
        const response = await fetch(`${apiBaseUrl}/alerts/${alertId}/raw-payload`);

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
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('复制失败:', err);
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

            {/* 数据展示 */}
            <Tabs value={viewMode} onValueChange={(value) => setViewMode(value as RawDataViewMode)}>
              <TabsList className="grid w-full grid-cols-4">
                <TabsTrigger value="formatted">格式化</TabsTrigger>
                <TabsTrigger value="json">JSON</TabsTrigger>
                <TabsTrigger value="yaml">YAML</TabsTrigger>
                <TabsTrigger value="raw">原始</TabsTrigger>
              </TabsList>
              
              <TabsContent value={viewMode} className="mt-4">
                <div className="relative">
                  <pre className="bg-muted p-4 rounded-lg overflow-auto max-h-96 text-sm">
                    <code>{formatData(data.data, viewMode)}</code>
                  </pre>
                </div>
              </TabsContent>
            </Tabs>

            {/* 元数据信息 */}
            <div className="text-sm text-muted-foreground space-y-1">
              <div>最后修改: {data.lastModified.toLocaleString()}</div>
              <div>存储键: <code className="bg-muted px-1 rounded">{data.key}</code></div>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
