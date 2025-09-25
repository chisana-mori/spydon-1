'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  Download,
  FileText,
  Loader2,
  AlertCircle,
  Copy,
  Check,
  RefreshCw,
  Code,
  Eye,
  FileCode
} from 'lucide-react';

interface RawPayloadViewerProps {
  rawPayloadKey?: string;
  alertId: string;
  alertTitle?: string;
  className?: string;
}

type DataFormat = 'markdown' | 'json' | 'yaml' | 'xml' | 'text';

export function RawPayloadViewer({
  rawPayloadKey,
  alertId,
  alertTitle,
  className
}: RawPayloadViewerProps) {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<any>(null);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [activeTab, setActiveTab] = useState<string>('formatted');

  // 检测数据格式
  const detectDataFormat = (content: string): DataFormat => {
    const trimmed = content.trim();

    // 检测 Markdown
    if (trimmed.includes('# ') || trimmed.includes('## ') ||
        trimmed.includes('### ') || trimmed.includes('**') ||
        trimmed.includes('*') || trimmed.includes('- ') ||
        trimmed.includes('```')) {
      return 'markdown';
    }

    // 检测 JSON
    if ((trimmed.startsWith('{') && trimmed.endsWith('}')) ||
        (trimmed.startsWith('[') && trimmed.endsWith(']'))) {
      try {
        JSON.parse(trimmed);
        return 'json';
      } catch {
        // 继续检测其他格式
      }
    }

    // 检测 YAML
    if (trimmed.includes(':\n') || trimmed.includes(': ') ||
        trimmed.includes('---\n') || trimmed.match(/^[a-zA-Z_][a-zA-Z0-9_]*:/m)) {
      return 'yaml';
    }

    // 检测 XML
    if (trimmed.startsWith('<') && trimmed.endsWith('>') &&
        trimmed.includes('</')) {
      return 'xml';
    }

    return 'text';
  };

  // 加载原始数据
  const loadRawData = useCallback(async () => {
    if (!rawPayloadKey && !alertId) return;

    setLoading(true);
    setError(null);

    try {
      // 使用后端 API 获取原始数据
      const apiBaseUrl = process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:8080/api/v1';
      const response = await fetch(`${apiBaseUrl}/alerts/${alertId}/raw-payload`);

      if (response.ok) {
        const contentType = response.headers.get('content-type') || 'text/plain';
        const payloadKey = response.headers.get('x-raw-payload-key') || rawPayloadKey || '';

        // 始终以文本形式获取数据
        const responseData = await response.text();

        const detectedFormat = detectDataFormat(responseData);

        setData({
          key: payloadKey,
          data: responseData,
          contentType,
          format: detectedFormat,
          size: new Blob([responseData]).size,
        });
      } else {
        setError(`获取数据失败: ${response.status} ${response.statusText}`);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '获取数据时发生错误');
    } finally {
      setLoading(false);
    }
  }, [alertId, rawPayloadKey]);

  useEffect(() => {
    if (alertId) {
      loadRawData();
    }
  }, [alertId, loadRawData]);

  // 复制到剪贴板
  const copyToClipboard = async () => {
    if (!data) return;

    try {
      await navigator.clipboard.writeText(data.data);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('复制失败:', err);
    }
  };

  // 下载原始数据
  const downloadRawData = () => {
    if (!data) return;

    const blob = new Blob([data.data], { type: data.contentType || 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `alert-${alertId}-raw-data.txt`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  // 格式化 JSON 数据
  const formatJSON = (content: string): string => {
    try {
      const parsed = JSON.parse(content);
      return JSON.stringify(parsed, null, 2);
    } catch {
      return content;
    }
  };

  // 渲染 Markdown 内容（简单版本）
  const renderMarkdown = (content: string): React.ReactNode => {
    // 简单的 Markdown 渲染，可以后续集成专业的 Markdown 库
    const lines = content.split('\n');
    return (
      <div className="prose prose-sm max-w-none">
        {lines.map((line, index) => {
          // 标题
          if (line.startsWith('### ')) {
            return <h3 key={index} className="text-lg font-semibold mt-4 mb-2">{line.slice(4)}</h3>;
          }
          if (line.startsWith('## ')) {
            return <h2 key={index} className="text-xl font-semibold mt-4 mb-2">{line.slice(3)}</h2>;
          }
          if (line.startsWith('# ')) {
            return <h1 key={index} className="text-2xl font-bold mt-4 mb-2">{line.slice(2)}</h1>;
          }

          // 代码块
          if (line.startsWith('```')) {
            return <div key={index} className="bg-gray-100 p-2 rounded font-mono text-sm my-2">{line}</div>;
          }

          // 列表
          if (line.startsWith('- ')) {
            return <li key={index} className="ml-4">{line.slice(2)}</li>;
          }

          // 粗体文本
          const boldText = line.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');

          // 普通段落
          if (line.trim()) {
            return <p key={index} className="mb-2" dangerouslySetInnerHTML={{ __html: boldText }} />;
          }

          return <br key={index} />;
        })}
      </div>
    );
  };

  // 获取格式化的数据
  const getFormattedData = (format: DataFormat, content: string): React.ReactNode => {
    switch (format) {
      case 'markdown':
        return renderMarkdown(content);
      case 'json':
        return <pre className="whitespace-pre-wrap font-mono text-sm">{formatJSON(content)}</pre>;
      case 'yaml':
      case 'xml':
      case 'text':
      default:
        return <pre className="whitespace-pre-wrap font-mono text-sm">{content}</pre>;
    }
  };

  // 如果没有原始数据键且没有告警ID，不显示组件
  if (!rawPayloadKey && !alertId) {
    return null;
  }

  return (
    <Card className={className}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg flex items-center gap-2">
            <FileText className="h-5 w-5" />
            原始数据
            {alertTitle && (
              <span className="text-sm font-normal text-gray-500">
                - {alertTitle}
              </span>
            )}
          </CardTitle>
          <div className="flex items-center gap-2">
            {data && (
              <>
                <Badge variant="outline" className="text-xs">
                  {data.format?.toUpperCase() || 'TEXT'}
                </Badge>
                <Badge variant="outline" className="text-xs">
                  {data.size} 字节
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
          <Alert variant="destructive" className="mb-4">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>
              {error}
              <Button 
                variant="outline" 
                size="sm" 
                onClick={loadRawData}
                className="ml-2"
              >
                <RefreshCw className="h-4 w-4 mr-1" />
                重试
              </Button>
            </AlertDescription>
          </Alert>
        )}

        {data && !loading && (
          <div className="space-y-4">
            {/* 操作按钮 */}
            <div className="flex items-center gap-2 flex-wrap">
              <Button
                variant="outline"
                size="sm"
                onClick={copyToClipboard}
                className="flex items-center gap-2"
              >
                {copied ? (
                  <>
                    <Check className="h-4 w-4 text-green-600" />
                    已复制
                  </>
                ) : (
                  <>
                    <Copy className="h-4 w-4" />
                    复制
                  </>
                )}
              </Button>
              
              <Button
                variant="outline"
                size="sm"
                onClick={downloadRawData}
                className="flex items-center gap-2"
              >
                <Download className="h-4 w-4" />
                下载
              </Button>
              
              <Button
                variant="outline"
                size="sm"
                onClick={loadRawData}
                className="flex items-center gap-2"
              >
                <RefreshCw className="h-4 w-4" />
                刷新
              </Button>
            </div>

            {/* 数据展示区域 - 使用 Tabs */}
            <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
              <TabsList className="grid w-full grid-cols-3">
                <TabsTrigger value="formatted" className="flex items-center gap-2">
                  <Eye className="h-4 w-4" />
                  格式化显示
                </TabsTrigger>
                <TabsTrigger value="raw" className="flex items-center gap-2">
                  <Code className="h-4 w-4" />
                  原始数据
                </TabsTrigger>
                <TabsTrigger value="preview" className="flex items-center gap-2">
                  <FileCode className="h-4 w-4" />
                  预览
                </TabsTrigger>
              </TabsList>

              <TabsContent value="formatted" className="mt-4">
                <div className="border rounded-lg overflow-hidden">
                  <div className="p-4 text-sm bg-gray-50 max-h-96 overflow-auto">
                    {getFormattedData(data.format, data.data)}
                  </div>
                </div>
              </TabsContent>

              <TabsContent value="raw" className="mt-4">
                <div className="border rounded-lg overflow-hidden">
                  <pre className="p-4 text-sm bg-gray-50 whitespace-pre-wrap break-words max-h-96 overflow-auto font-mono">
                    {data.data}
                  </pre>
                </div>
              </TabsContent>

              <TabsContent value="preview" className="mt-4">
                <div className="border rounded-lg overflow-hidden">
                  <div className="p-4 text-sm bg-white max-h-96 overflow-auto">
                    {data.format === 'markdown' ? (
                      renderMarkdown(data.data)
                    ) : (
                      <div className="text-gray-500 text-center py-8">
                        此格式不支持预览模式
                      </div>
                    )}
                  </div>
                </div>
              </TabsContent>
            </Tabs>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
